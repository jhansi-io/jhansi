package isolation

import (
	"context"
)

// ExecRequest is one command to run in a sandbox.
// It carries primitives only, so a backend never depends on the domain model.
// WorkDir is the sandbox's directory on the host, derived and owned by the
// service; a backend makes it available to the command but does not create it.
type ExecRequest struct {
	SandboxID string
	WorkDir   string
	Command   string
}

// ExecResult is the outcome of running in a sandbox.
// A returned error means the engine could not run it at all (infra fault);
// A completed run reports through these fields.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

// SandboxEngine runs commands in an opaque isolation backend.
// It is keyed by a primitive sandboxID, not the domain aggregate, so a
// backend never depends on the domain model.

type SandboxEngine interface {
	Exec(ctx context.Context, req ExecRequest) (ExecResult, error)
}
