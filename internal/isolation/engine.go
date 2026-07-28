package isolation

import (
	"context"
)

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
	Exec(ctx context.Context, sandboxID, command string) (ExecResult, error)
}
