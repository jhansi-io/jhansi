package isolation

import (
	"encoding/binary"
	"testing"
)

// TestDemuxLogs checks that interleaved frames are split back into two
// streams in the order the process wrote them.
func TestDemuxLogs(t *testing.T) {
	var raw []byte
	raw = append(raw, logFrame(logStreamStdout, "hello ")...)
	raw = append(raw, logFrame(logStreamStderr, "warning")...)
	raw = append(raw, logFrame(logStreamStdout, "world")...)

	stdout, stderr, _ := demuxLogs(raw, 0)

	if string(stdout) != "hello world" {
		t.Errorf("stdout = %q, want %q", stdout, "hello world")
	}

	if string(stderr) != "warning" {
		t.Errorf("stderr = %q, want %q", stderr, "warning")
	}
}

// TestDemuxLogsTruncated checks that a partial trailing frame is dropped and
// the complete frames before it are still returned.
func TestDemuxLogsTruncated(t *testing.T) {
	raw := logFrame(logStreamStdout, "complete")
	raw = append(raw, logFrame(logStreamStdout, "cut off")...)
	raw = raw[:len(raw)-3]

	stdout, stderr, _ := demuxLogs(raw, 0)

	if string(stdout) != "complete" {
		t.Errorf("stdout = %q, want %q", stdout, "complete")
	}
	if len(stderr) != 0 {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestCreateBodyDefaults checks that an empty image falls back to the engine's
// default and that the mount and working directory agree.
func TestCreateBodyDefaults(t *testing.T) {
	e := NewDockerEngine("/var/run/docker.sock", "python:3.12-slim")

	body := e.createBody("", "/data/sandboxes/sb_1", []string{"python", "main.py"})

	if body.Image != "python:3.12-slim" {
		t.Errorf("Image = %q, want the default", body.Image)
	}
	if body.WorkingDir != containerWorkDir {
		t.Errorf("WorkingDir = %q, want %q", body.WorkingDir, containerWorkDir)
	}

	want := "/data/sandboxes/sb_1:" + containerWorkDir
	if len(body.HostConfig.Binds) != 1 || body.HostConfig.Binds[0] != want {
		t.Errorf("Binds = %v, want [%q]", body.HostConfig.Binds, want)
	}
}

// TestCreateBodyExplicitImage checks that a caller's image wins over the
// engine's default, which is what keeps jhansi language-neutral.
func TestCreateBodyExplicitImage(t *testing.T) {
	e := NewDockerEngine("/var/run/docker.sock", "python:3.12-slim")

	body := e.createBody("golang:1.26", "/data/sandboxes/sbx_1", []string{"go", "run", "."})

	if body.Image != "golang:1.26" {
		t.Errorf("Image = %q, want the caller's image", body.Image)
	}
}

// logFrame builds one Docker log frame: an 8-byte header carrying the stream
// type and payload length, followed by the payload.
func logFrame(stream byte, payload string) []byte {
	frame := make([]byte, logHeaderLen, logHeaderLen+len(payload))
	frame[0] = stream
	binary.BigEndian.PutUint32(frame[4:logHeaderLen], uint32(len(payload)))
	return append(frame, payload...)
}
