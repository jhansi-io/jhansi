package service

import (
	"context"
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/id"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"github.com/jhansi-io/jhansi/internal/registry"
	"os"
	"time"
)

type eventSource interface {
	DrainEvents() []domain.Event
}

// Config holds the services's tunable settings, separate from its
// collaborators. The defaults here apply to every eexc that does not carry
// its own (ADR-022).
type Config struct {
	// DataDir is the root under which sandbox working directories live.
	DataDir string

	// ExecTimeout bounds how long a single command may run.
	ExecTimeout time.Duration

	// MaxOutputBytes bounds how much of each stream is retained.
	MaxOutputBytes int64
}

// ExecutionService orchestrates mutating operations: It holds the
// registry and sink, and routes every drain-and-record through one
// helper so write-ahead ordering stays a single later edit (ADR-008).
type ExecutionService struct {
	reg    *registry.Registry
	sink   evidence.Sink
	engine isolation.SandboxEngine

	// cfg holds the tunable settings applied to every exec.
	cfg Config
}

// New constructs an ExecutionService over a registry and sink.
func New(reg *registry.Registry, sink evidence.Sink, engine isolation.SandboxEngine, cfg Config) *ExecutionService {
	return &ExecutionService{
		reg:    reg,
		sink:   sink,
		engine: engine,
		cfg:    cfg}
}

// CreateSandbox mints an id, constructs a sandbox, stores it, and
// records its creation event. The order is deliberate (ADR-008):
// registry.Add is the effect, drainAndRecord follows it. Add cannot
// realistically fail on a fresh 128-bit id, so there is no failable
// effect for the record to be ahead of — write-ahead stays deferred.
func (s *ExecutionService) CreateSandbox() (*domain.Sandbox, error) {
	sbID, err := id.New("sb")
	if err != nil {
		return nil, err
	}

	sb := domain.NewSandbox(sbID)

	if err := os.MkdirAll(workDirFor(s.cfg.DataDir, sbID), 0o700); err != nil {
		return nil, err
	}

	if err := sb.MarkReady(); err != nil {
		return nil, err
	}
	if err := s.reg.Add(sb); err != nil {
		return nil, err
	}

	if err := s.drainAndRecord(sb); err != nil {
		return nil, err
	}
	return sb, nil
}

// ExecInput is what a caller asks for Exec. Timeout is nil when the caller
// did not supply one, and the service applies its configured default —
// the default lives here so a later operator ceiling has one place to clamp
// (ADR-022).
type ExecInput struct {
	SandboxID string
	Command   string
	Timeout   *time.Duration
}

// Exec runs a command in a sandbox: claims it, drives a Run through its
// lifecycle, calls the isolation seam, then releases the sandbox and
// records both aggregates (ADR-015). The switch below maps the engine's
// result to a terminal state — success, non-zero exit, timeout, or infra
// fault (ADR-017).
//
// The run id is minted before MarkActive deliberately: id.New can fail,
// and MarkExpired is READY-only, so claiming first would leak a
// permanently unreapable ACTIVE sandbox on a rand blip.
//
// The Run is not stored. It is minted, transitioned, drained and dropped -
// its only durable trace is its events in the sink.
func (s *ExecutionService) Exec(ctx context.Context, in ExecInput) (*domain.Run, isolation.ExecResult, error) {

	sb, err := s.reg.Get(in.SandboxID)
	if err != nil {
		return nil, isolation.ExecResult{}, err
	}

	runID, err := id.New("run")
	if err != nil {
		return nil, isolation.ExecResult{}, err
	}

	if err := sb.MarkActive(); err != nil {
		if drainErr := s.drainAndRecord(sb); drainErr != nil {
			return nil, isolation.ExecResult{}, drainErr
		}
		return nil, isolation.ExecResult{}, err
	}
	run := domain.NewRun(runID, in.SandboxID, in.Command)
	if err := run.MarkPreparing(); err != nil {
		panic(err) // unreachable: a fresh run is QUEUED, MarkPreparing is legal from QUEUED
	}
	// Preflight: creating the run's log directory is the check that the host
	// can retain this run's output. Failing here costs nothing — no code has
	// executed — so jhansi refuses rather than run unrecorded (ADR-025).
	if err := os.MkdirAll(runDirFor(s.cfg.DataDir, runID), 0o700); err != nil {
		sb.MarkIdle()
		run.MarkPreparationFailed(err.Error())
		if drainErr := s.drainAndRecord(sb); drainErr != nil {
			return run, isolation.ExecResult{}, drainErr
		}
		if drainErr := s.drainAndRecord(run); drainErr != nil {
			return run, isolation.ExecResult{}, drainErr
		}
		return run, isolation.ExecResult{}, err
	}

	if err := run.MarkRunning(); err != nil {
		panic(err) // unreachable: MarkPreparing left it PREPARING, MarkRunning is legal from PREPARING
	}
	timeout := s.timeoutFor(in.Timeout)

	started := time.Now()
	result, err := s.engine.Exec(ctx, isolation.ExecRequest{
		SandboxID:      in.SandboxID,
		WorkDir:        workDirFor(s.cfg.DataDir, in.SandboxID),
		Command:        in.Command,
		Timeout:        timeout,
		MaxOutputBytes: s.cfg.MaxOutputBytes,
	})
	elapsed := time.Since(started)

	switch {
	case err != nil:
		sb.MarkError()
		run.MarkFailed(runOutcome(result, elapsed, nil))
	case result.TimedOut:
		sb.MarkIdle()
		run.MarkTimedOut(domain.RunTimedOutOutcome{
			RunOutcome: runOutcome(result, elapsed, nil),
			TimeoutMS:  timeout.Milliseconds(),
		})
	case result.ExitCode != 0:
		sb.MarkIdle()
		run.MarkFailed(runOutcome(result, elapsed, &result.ExitCode))
	default:
		sb.MarkIdle()
		run.MarkSucceeded(runOutcome(result, elapsed, &result.ExitCode))
	}

	if drainErr := s.drainAndRecord(sb); drainErr != nil {
		return run, result, drainErr
	}
	if drainErr := s.drainAndRecord(run); drainErr != nil {
		return run, result, drainErr
	}
	return run, result, err
}

// DeleteSandbox marks a sandbox DELETED and records the transition.
// Idempotent: deleting an already-DELETED sandbox is success, since the
// desired state is reached. MarkDeleted and drainAndRecord run whatever
// the state — the rejection row is the one an auditor reads (ADR-011).
func (s *ExecutionService) DeleteSandbox(id string) error {
	sb, err := s.reg.Get(id)
	if err != nil {
		return err
	}
	markErr := sb.MarkDeleted()

	if err := os.RemoveAll(workDirFor(s.cfg.DataDir, id)); err != nil {
		sb.RecordWorkDirRemovalFailed(err.Error())
	}
	if err := s.drainAndRecord(sb); err != nil {
		return err
	}
	if markErr != nil && sb.Status != domain.SandboxDeleted {
		return markErr
	}
	return nil
}

// GetSandbox returns the sandbox stored under id, or an error.
// A read: no mutation, no events, no sink (ADR-010).
func (s *ExecutionService) GetSandbox(id string) (*domain.Sandbox, error) {
	return s.reg.Get(id)
}

// ListSandboxes returns every stored sandbox. A read: no mutation,
// no events, no sink.
func (s *ExecutionService) ListSandboxes() []*domain.Sandbox {
	return s.reg.List()
}

// drainAndRecord drains an aggregate's buffered events and hands them
// to the sink. The single home ADR-008 required: every operation
// records through here, so write-ahead ordering becomes one later edit
// rather than a change at every call site (ADR-007).
func (s *ExecutionService) drainAndRecord(src eventSource) error {
	events := src.DrainEvents()
	return s.sink.Record(events)
}

// timeoutFor resolves a caller's optional timeout against the configured
// default. Nil means the caller did not ask for one.
func (s *ExecutionService) timeoutFor(t *time.Duration) time.Duration {
	if t == nil {
		return s.cfg.ExecTimeout
	}
	return *t
}
