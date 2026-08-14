package isolation

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
)

var _ SandboxEngine = (*DockerEngine)(nil)

// dockerAPIVersion pins the Docker API version jhansi requests.
//
// The daemon serves older versions than its own, so pinning keeps jhansi's
// requests stable across daemon upgrades.
const dockerAPIVersion = "v1.43"

// containerWorkDir is the path the sandbox working directory is mounted at
// inside every container, and the working directory command run from.
//
// Mount point and working directory are deliberately the same path.
const containerWorkDir = "/workspace"

// Docker log frame header layout. Each frame is an 8-byte header followed by
// its payload: one type byte, three zero bytes, then a big-endian uint32
// length.
const (
	logHeaderLen    = 8
	logStreamStdout = 1
	logStreamStderr = 2
)

// DockerEngine executes commands in ephemeral Docker containers.
//
// One container is creted per Exec and removed once its result has been
// read. Everything the container writes to the sandbox working directory
// survives, because that directory is bind-mounted from jhansi's disk.
type DockerEngine struct {
	client       *http.Client
	defaultImage string
}

// containerConfig is the container half of Docker's create request body.
//
// Field names match the Docker API exactly, which is why they are capitalised
// in the JSON tags.
type containerConfig struct {
	Image      string     `json:"Image"`
	Cmd        []string   `json:"Cmd"`
	WorkingDir string     `json:"WorkingDir"`
	User       string     `json:"User"`
	HostConfig hostConfig `json:"HostConfig"`
}

// hostConfig is the host half of Docker's create request body.
//
// Binds carries the sandbox working directory mount in Docker's
// "hostPath:containerPath" form.
type hostConfig struct {
	Binds []string `json:"Binds"`
}

// createResponse is Docker's reply to POST /containers/create.
//
// Warnings is ignored for now; a warning does not stop the container from
// having been created.
type createResponse struct {
	ID string `json:"id"`
}

// waitResponse is Docker's reply to POST /containers/{id}/wait.
//
// StatusCode is the exited process's exit code, not an HTTP status.
type waitResponse struct {
	StatusCode int `json:"StatusCode"`
}

// NewDockerEngine returns a DockerEngine that talks to the Docker daemon on
// the local host.
//
// The daemon must be local because the sandbox working directory is
// bind-mounted, and bind mounts are resolved by the daemon rather than the
// client. defaultImage is used for sandboxes created without one.
func NewDockerEngine(socketPath, defaultImage string) *DockerEngine {
	return &DockerEngine{
		client:       newSocketClient(socketPath),
		defaultImage: defaultImage,
	}
}

// newSocketClient returns an HTTP client that sends every request to the
// Docker daemon's unix socket, ignoring the host in the request URL.
//
// The Docker API is HTTP over a unix socket, so requests are addressed to a
// placeholder host and the dialer redirects them to socketpath.
func newSocketClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// Exec runs one command in a fresh container over the sandbox's working
// directory, then removes the container. The command is handed to a shell,
// so shell syntax work and the image must contain /bin/sh.
func (e *DockerEngine) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	cmd := []string{"sh", "-c", req.Command}

	id, err := e.createContainer(ctx, "", req.WorkDir, cmd)
	if err != nil {
		return ExecResult{}, err
	}
	defer e.removeContainer(context.WithoutCancel(ctx), id)

	if err := e.startContainer(ctx, id); err != nil {
		return ExecResult{}, err
	}

	code, err := e.waitContainer(ctx, id)
	if err != nil {
		return ExecResult{}, err
	}

	raw, err := e.fetchLogs(ctx, id)
	if err != nil {
		return ExecResult{}, err
	}

	stdout, stderr := demuxLogs(raw)
	return ExecResult{
		ExitCode: code,
		Stdout:   string(stdout),
		Stderr:   string(stderr),
	}, nil
}

// createContainer creates a container for one exec and returns its ID.
//
// The container is created but not started. Any non-201 reply is returned as
// an error carrying Docker's status, which ADR-022 will turn into a
// distinguishable failure.
func (e *DockerEngine) createContainer(ctx context.Context, image, hostDir string, cmd []string) (string, error) {
	body, err := json.Marshal(e.createBody(image, hostDir, cmd))
	if err != nil {
		return "", fmt.Errorf("marshall create body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiURL("/containers/create"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create container: docker returned %s", resp.Status)
	}

	var created createResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}

	return created.ID, nil
}

// createBody builds the request body for POST /containers/create.
//
// image is the caller's choice, or the engine's default when empty. hostDir
// is the sandbox working directory on jhansi's disk, mounted at
// containerWorkDir inside the container.
func (e *DockerEngine) createBody(image, hostDir string, cmd []string) containerConfig {
	if image == "" {
		image = e.defaultImage
	}
	return containerConfig{
		Image:      image,
		Cmd:        cmd,
		WorkingDir: containerWorkDir,
		User:       currentUser(),
		HostConfig: hostConfig{
			Binds: []string{bindFor(hostDir)},
		},
	}
}

// startContainer starts a created container.
//
// Docker replies 204 on success and 304 if the container is already running.
// which cannot happen for a per-exec container.
func (e *DockerEngine) startContainer(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiURL("/containers/"+id+"/start"), nil)
	if err != nil {
		return fmt.Errorf("build start request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("start container: docker returned %s", resp.Status)
	}
	return nil
}

// waitContainer blocks until the container exits and returns its exit code.
//
// Docker holds the response open for the container's whole lifetime, so this
// call is what makes an exec synchronous.
func (e *DockerEngine) waitContainer(ctx context.Context, id string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiURL("/containers/"+id+"/wait"), nil)
	if err != nil {
		return 0, fmt.Errorf("build wait request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("wait for container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("wait for container: docker returned %s", resp.Status)
	}

	var waited waitResponse
	if err := json.NewDecoder(resp.Body).Decode(&waited); err != nil {
		return 0, fmt.Errorf("decode wait response: %w", err)
	}
	return waited.StatusCode, nil
}

// fetchLogs returns the container's raw log stream after it has exited.
//
// With TTY off, Docker multiplexes stdout and stderr into one stream with a
// frame header per chunk, so the bytes here still need demultiplexing.
func (e *DockerEngine) fetchLogs(ctx context.Context, id string) ([]byte, error) {
	url := e.apiURL("/containers/" + id + "/logs?stdout=1&stderr=1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build logs request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch logs: docker returned %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read logs: %w", err)
	}

	return raw, nil
}

// removeContainer deletes a container once its result has been read.
//
// AutoRemove is deliberately not used, because it races the log read and can
// destroy output the engine about to record.
func (e *DockerEngine) removeContainer(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, e.apiURL("/containers/"+id), nil)
	if err != nil {
		return fmt.Errorf("build remove request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remove container: docker returned %s", resp.Status)
	}
	return nil
}

// apiURL builds a Docker API URL for the given path.
//
// The host is a placeholder. The client's dialer sends every request to the
// unix socket regardless of what the URL says.
func (e *DockerEngine) apiURL(path string) string {
	return "http://docker/" + dockerAPIVersion + path
}

// bindFor returns the Docker bind specification that mounts a sandbox's
// working directory into the container.
func bindFor(hostDir string) string {
	return hostDir + ":" + containerWorkDir
}

// currentUser returns the running process's UID and GID in Docker's
// "uid:gid" form.
//
// Containers run as this user so that files written into the bind-mounted
// working directory are owned by jhansi and can be removed when the sandbox
// is deleted.
func currentUser() string {
	return strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
}

// demuxLogs splits Docker's multiplexed log stream into stdout and stderr.
//
// Frames are read in order and appended to their stream, so each stream
// reproduces exactly what the process wrote to it. A truncated final frame is
// ignored rathet than treated as an error, because partial output is still
// worth returning.
func demuxLogs(raw []byte) (stdout, stderr []byte) {
	for len(raw) >= logHeaderLen {
		length := int(binary.BigEndian.Uint32(raw[4:logHeaderLen]))
		if len(raw) < logHeaderLen+length {
			break
		}

		payload := raw[logHeaderLen : logHeaderLen+length]
		switch raw[0] {
		case logStreamStdout:
			stdout = append(stdout, payload...)
		case logStreamStderr:
			stderr = append(stderr, payload...)
		}

		raw = raw[logHeaderLen+length:]
	}
	return stdout, stderr
}
