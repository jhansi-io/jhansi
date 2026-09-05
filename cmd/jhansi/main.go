package main

import (
	"flag"
	"fmt"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/httpapi"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"github.com/jhansi-io/jhansi/internal/registry"
	"github.com/jhansi-io/jhansi/internal/service"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Config holds the settings supplied at bootstrap. The fields mirror the
// server subcommand's flag. Grouping them keeps newServer's signature
// stable as later work adds defaults, and makes each value named at the
// call site rather than positional.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string

	// DataDir is the directory under which evidence and sandbox
	// working directories are created. Resolved to an absolute path
	// by newServer.
	DataDir string

	// ExecTimeout bounds how long a single command may run.
	ExecTimeout time.Duration

	// MaxOutputBytes bounds how much of each stream is retained.
	MaxOutputBytes int64
}

// newServer wires the engine and returns a configured *http.Server without
// starting it. Construction is separated from listening so tests can drive
// the real handler — and the real FileSink — without binding a port.
func newServer(cfg Config, engine isolation.SandboxEngine) (*http.Server, error) {
	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	sink, err := evidence.NewFileSink(filepath.Join(dataDir, "events.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("open event sink: %w", err)
	}

	svc := service.New(registry.New(), sink, engine, service.Config{
		DataDir:        dataDir,
		ExecTimeout:    cfg.ExecTimeout,
		MaxOutputBytes: cfg.MaxOutputBytes,
	})
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(svc).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}, nil
}

// runServer parses the flags for the "server" subcommand and starts the
// engine. Flags are parsed with a dedicated FlagSet over the caller's args
// because package-level flag.Parse stops at the first non-flag argument.
func runServer(args []string) error {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "address to listen on")
	dataDir := fs.String("data-dir", "./jhansi-data", "directory for evidence and run data")
	socket := fs.String("docker-socket", "/var/run/docker.sock", "path to the Docker daemon socket")
	image := fs.String("default-image", "python:3.12-slim", "image used when a sandbox does not specify one")
	execTimeout := fs.Duration("exec-timeout", 5*time.Minute, "maximum duration of a single command")
	maxOutput := fs.Int64("max-output-bytes", 1<<20, "maximum output retained per stream")

	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := Config{
		Addr:           *addr,
		DataDir:        *dataDir,
		ExecTimeout:    *execTimeout,
		MaxOutputBytes: *maxOutput,
	}
	srv, err := newServer(cfg, isolation.NewDockerEngine(*socket, *image))
	if err != nil {
		return err
	}
	fmt.Printf("jhansi listening on %s, data dir %s\n", *addr, *dataDir)
	return srv.ListenAndServe()
}

// main dispatches on the subcommand in os.Args[1]. Exactly one subcommand
// exists today; an unknown or missing verb is a usage error, not a default.
func main() {
	if len(os.Args) < 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "server":
		err = runServer(os.Args[2:])
	default:
		usage()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "jhansi: %v\n", err)
		os.Exit(1)
	}
}

// usage prints the command surface to stderr and exits 2.
func usage() {
	fmt.Fprint(os.Stderr, "usage: jhansi server [flags]\n")
	os.Exit(2)
}
