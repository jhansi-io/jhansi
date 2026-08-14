package isolation

import "context"

var _ SandboxEngine = (*StubEngine)(nil)

// StubEngine is the Tier 0 default SandboxEngine and the permanent test
// double. Its zero value echoes the command back on stdout with exit 0.
// Set ExecFunc to drive any other outcome — non-zero exit, TimedOut, or an
// infra error — so tests exercise the full failure surface.
type StubEngine struct {
	ExecFunc func(ctx context.Context, req ExecRequest) (ExecResult, error)
}

func (e *StubEngine) Exec(ctx context.Context, req ExecRequest) (ExecResult, error) {
	if e.ExecFunc != nil {
		return e.ExecFunc(ctx, req)
	}
	return ExecResult{ExitCode: 0, Stdout: req.Command}, nil
}
