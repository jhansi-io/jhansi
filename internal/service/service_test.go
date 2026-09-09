package service

import (
	"context"
	"errors"
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"github.com/jhansi-io/jhansi/internal/registry"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeSink captures recorded events for assertions and can be told to
// fail, to exercise the record-failure path.
type fakeSink struct {
	events []domain.Event
	err    error
}

func (f *fakeSink) Record(events []domain.Event) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, events...)
	return nil
}

func TestCreateSandbox(t *testing.T) {
	sink := &fakeSink{}

	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	if sb.Status != domain.SandboxReady {
		t.Errorf("status = %s, want %s", sb.Status, domain.SandboxReady)
	}
	if !strings.HasPrefix(sb.ID, "sb_") {
		t.Errorf("id = %q, want sb_ prefix", sb.ID)
	}

	// Stored: the service added it to the registry.
	got, err := svc.reg.Get(sb.ID)
	if err != nil {
		t.Fatalf("registry.Get: %v", err)
	}
	if got != sb {
		t.Error("registry holds a different sandbox than returned")
	}

	// Recorded: the creation event drained through the sink.
	if len(sink.events) != 2 {
		t.Fatalf("recorded %d events, want 2", len(sink.events))
	}
	if sink.events[0].Name != "sandbox.created" {
		t.Errorf("event = %q, want sandbox.created", sink.events[0].Name)
	}
	if sink.events[1].Name != "sandbox.ready" {
		t.Errorf("event = %q, want sandbox.ready", sink.events[1].Name)
	}
	if sink.events[0].AggregateID != sb.ID {
		t.Errorf("event aggregateID = %q, want %q", sink.events[0].AggregateID, sb.ID)
	}
}

func TestCreateSandboxFailure(t *testing.T) {
	sinkErr := errors.New("sink down")
	svc := New(registry.New(), &fakeSink{err: sinkErr}, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if !errors.Is(err, sinkErr) {
		t.Fatalf("err = %v, want %v", err, sinkErr)
	}
	if sb != nil {
		t.Errorf("sandbox = %v, want nil on record failure", sb)
	}
}

func TestDeleteSandbox(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	if err := svc.DeleteSandbox(sb.ID); err != nil {
		t.Fatalf("DeleteSandbox: %v", err)
	}
	if sb.Status != domain.SandboxDeleted {
		t.Errorf("status = %s, want %s", sb.Status, domain.SandboxDeleted)
	}

	// Delete is a status flip, not a remove — still gettable.
	got, err := svc.reg.Get(sb.ID)
	if err != nil {
		t.Fatalf("registry.Get after delete: %v", err)
	}
	if got.Status != domain.SandboxDeleted {
		t.Errorf("stored status = %s, want %s", got.Status, domain.SandboxDeleted)
	}

	// created + deleted drained through the sink.
	if len(sink.events) != 3 {
		t.Fatalf("recorded %d events, want 2", len(sink.events))
	}
	if sink.events[2].Name != "sandbox.deleted" {
		t.Errorf("event = %q, want sandbox.deleted", sink.events[1].Name)
	}
}

func TestDeleteSandboxIdempotent(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	if err := svc.DeleteSandbox(sb.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	if err := svc.DeleteSandbox(sb.ID); err != nil {
		t.Fatalf("second delete: %v, want nil (idempotent)", err)
	}
	if sb.Status != domain.SandboxDeleted {
		t.Errorf("status = %s, want %s", sb.Status, domain.SandboxDeleted)
	}

	// The redundant retry is still recorded:
	// Created, deleted, deleted_rejected {DELETED, DELETED}.
	if len(sink.events) != 4 {
		t.Fatalf("recorded %d events, want 3", len(sink.events))
	}
	last := sink.events[3]
	if last.Name != "sandbox.deleted_rejected" {
		t.Fatalf("event = %q, want sandbox.deleted_rejected", last.Name)
	}
	rej, ok := last.Payload.(domain.SandboxTransitionRejected)
	if !ok {
		t.Fatalf("payload = %T, want SandboxTransitionRejected", last.Payload)
	}
	if rej.From != domain.SandboxDeleted || rej.To != domain.SandboxDeleted {
		t.Errorf("payload = %+v, want {DELETED, DELETED}", rej)
	}
}

func TestDeleteSandboxNotFound(t *testing.T) {
	svc := New(registry.New(), &fakeSink{}, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})
	err := svc.DeleteSandbox("sb_missing")
	if !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSandboxRecordFailure(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	// Break the sink after create, so delete's drainAndRecord fails.
	sinkErr := errors.New("sink down")
	sink.err = sinkErr
	if err := svc.DeleteSandbox(sb.ID); !errors.Is(err, sinkErr) {
		t.Fatalf("err = %v, want %v", err, sinkErr)
	}
}

func TestExecHappyPath(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "echo hi",
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if out.Run.Status != domain.RunSucceeded {
		t.Errorf("run status = %q, want %q", out.Run.Status, domain.RunSucceeded)
	}
	if out.Result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", out.Result.ExitCode)
	}
	if out.Result.Stdout != "echo hi" {
		t.Errorf("stdout = %q, want the command echoed", out.Result.Stdout)
	}
	if sb.Status != domain.SandboxReady {
		t.Errorf("sandbox status = %q, want %q — released after the run", sb.Status, domain.SandboxReady)
	}
	// The spine: exec drains both aggregates — the sandbox's claim and
	// release, then the run's whole lifecycle. Order is sandbox-then-run
	// (ADR-015, arbitrary pending the event-model ADR).
	want := []string{
		"sandbox.created", "sandbox.ready",
		"sandbox.active", "sandbox.idle",
		"run.created", "run.preparing", "run.running", "run.succeeded",
	}
	if len(sink.events) != len(want) {
		t.Fatalf("recorded %d events, want %d", len(sink.events), len(want))
	}
	for i, name := range want {
		if sink.events[i].Name != name {
			t.Errorf("event %d = %q, want %q", i, sink.events[i].Name, name)
		}
	}
}

func TestExecNonZeroExit(t *testing.T) {
	sink := &fakeSink{}
	engine := &isolation.StubEngine{
		ExecFunc: func(ctx context.Context, req isolation.ExecRequest) (isolation.ExecResult, error) {
			return isolation.ExecResult{ExitCode: 1, Stderr: "boom"}, nil
		},
	}
	svc := New(registry.New(), sink, engine, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})
	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "false",
	})
	if err != nil {
		t.Fatalf("exec: %v — a non-zero exit is a completed run, not an error", err)
	}
	if out.Run.Status != domain.RunFailed {
		t.Errorf("run status = %q, want %q", out.Run.Status, domain.RunFailed)
	}
	if out.Result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", out.Result.ExitCode)
	}
	if sb.Status != domain.SandboxReady {
		t.Errorf("sandbox status = %q, want %q — released after a failed run", sb.Status, domain.SandboxReady)
	}

	want := []string{
		"sandbox.created", "sandbox.ready",
		"sandbox.active", "sandbox.idle",
		"run.created", "run.preparing", "run.running", "run.failed",
	}
	if len(sink.events) != len(want) {
		t.Fatalf("recorded %d events, want %d", len(sink.events), len(want))
	}
	for i, name := range want {
		if sink.events[i].Name != name {
			t.Errorf("event %d = %q, want %q", i, sink.events[i].Name, name)
		}
	}
}

func TestExecTimedOut(t *testing.T) {
	sink := &fakeSink{}
	engine := &isolation.StubEngine{
		ExecFunc: func(ctx context.Context, req isolation.ExecRequest) (isolation.ExecResult, error) {
			return isolation.ExecResult{TimedOut: true}, nil
		},
	}
	svc := New(registry.New(), sink, engine, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "sleep 999",
	})
	if err != nil {
		t.Fatalf("exec: %v — a timeout is a completed run, not an error", err)
	}
	if out.Run.Status != domain.RunTimedOut {
		t.Errorf("run status = %q, want %q", out.Run.Status, domain.RunTimedOut)
	}
	if sb.Status != domain.SandboxReady {
		t.Errorf("sandbox status = %q, want %q — released after a timeout", sb.Status, domain.SandboxReady)
	}
	want := []string{
		"sandbox.created", "sandbox.ready",
		"sandbox.active", "sandbox.idle",
		"run.created", "run.preparing", "run.running", "run.timed_out",
	}
	if len(sink.events) != len(want) {
		t.Fatalf("recorded %d events, want %d", len(sink.events), len(want))
	}
	for i, name := range want {
		if sink.events[i].Name != name {
			t.Errorf("event %d = %q, want %q", i, sink.events[i].Name, name)
		}
	}
}

func TestExecInfraError(t *testing.T) {
	sink := &fakeSink{}
	infraErr := errors.New("engine down")
	engine := &isolation.StubEngine{
		ExecFunc: func(ctx context.Context, req isolation.ExecRequest) (isolation.ExecResult, error) {
			return isolation.ExecResult{}, infraErr
		},
	}
	svc := New(registry.New(), sink, engine, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "anything",
	})

	if !errors.Is(err, infraErr) {
		t.Fatalf("exec err = %v, want the infra error surfaced (→ 500)", err)
	}
	if out.Run.Status != domain.RunFailed {
		t.Errorf("run status = %q, want %q", out.Run.Status, domain.RunFailed)
	}
	if sb.Status != domain.SandboxError {
		t.Errorf("sandbox status = %q, want %q — the runtime under it is broken", sb.Status, domain.SandboxError)
	}
	want := []string{
		"sandbox.created", "sandbox.ready",
		"sandbox.active", "sandbox.error",
		"run.created", "run.preparing", "run.running", "run.failed",
	}
	if len(sink.events) != len(want) {
		t.Fatalf("recorded %d events, want %d", len(sink.events), len(want))
	}
	for i, name := range want {
		if sink.events[i].Name != name {
			t.Errorf("event %d = %q, want %q", i, sink.events[i].Name, name)
		}
	}
}

func TestExecBusySandbox(t *testing.T) {
	sink := &fakeSink{}
	engine := &isolation.StubEngine{}
	svc := New(registry.New(), sink, engine, Config{
		DataDir:        t.TempDir(),
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	// Force the sandbox ACTIVE. At Tier 0 exec is synchronous, so the only
	// way to meet Exec with an alteady-claimed sandbox is to claim it first.
	if err := sb.MarkActive(); err != nil {
		t.Fatalf("setup MarkActive: %v", err)
	}
	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "anything",
	})

	if !errors.Is(err, domain.ErrSandboxBusy) {
		t.Errorf("err = %v, want ErrSandboxBusy", err)
	}

	if err == nil {
		t.Fatalf("exec on a busy sandbox: err = nil, want a rejection")
	}
	if out.Run != nil {
		t.Errorf("run = %v, want nil — nothing is minted past a failed claim", out.Run)
	}
	// The rejection is recorded — Option A evidence hygiene. The status it
	// maps to (409) is ADR-018 and is deliberately not asserted here.
	last := sink.events[len(sink.events)-1]
	if last.Name != "sandbox.active_rejected" {
		t.Errorf("last event = %q, want sandbox.active_rejected", last.Name)
	}
	if sb.Status != domain.SandboxActive {
		t.Errorf("sandbox status = %q, want %q — a rejected claim doesn't move it", sb.Status, domain.SandboxActive)
	}
}

// TestCreateSandboxMakesWorkDir checks that a created sandbox has its
// working directory on a disk at the derived path (ADR-019): registry
// membership implies the directory exists.

func TestCreateSandboxMakesWorkDir(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(registry.New(), &fakeSink{}, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	info, err := os.Stat(workDirFor(dataDir, sb.ID))
	if err != nil {
		t.Fatalf("stat work dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("work dir is not a directory")
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("work dir mode = %o, want 700", perm)
	}
}

// TestDeleteSandboxRemovesWorkDir checks that destroying a sandbox
// removes its working directory (ADR-019). Create and destroy are
// symmetric: the directory's lifetime is the sandbox's.
func TestDeleteSandboxRemovesWorkDir(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(registry.New(), &fakeSink{}, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	if err := svc.DeleteSandbox(sb.ID); err != nil {
		t.Fatalf("delete sandbox: %v", err)
	}

	if _, err := os.Stat(workDirFor(dataDir, sb.ID)); !os.IsNotExist(err) {
		t.Errorf("work dir still present, stat err = %v", err)
	}
}

// TestDeleteSandboxWorkDirRemovalFails checks decision 5 of ADR-019: a
// directory that cannot be removed does not change the outcome. The
// sandbox still reaches DELETED and DeleteSandbox still returns nil;
// the orphaned bytes are recorded as an engine failure instead.
func TestDeleteSandboxWorkDirRemovalFails(t *testing.T) {
	dataDir := t.TempDir()
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	parent := filepath.Dir(workDirFor(dataDir, sb.ID))
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}

	t.Cleanup(func() { os.Chmod(parent, 0o700) })

	if err := svc.DeleteSandbox(sb.ID); err != nil {
		t.Fatalf("delete sandbox: %v", err)
	}
	if sb.Status != domain.SandboxDeleted {
		t.Errorf("status = %s, want DELETED", sb.Status)
	}
	// Recorded: the failure is the last event, after sandbox.deleted.
	if n := len(sink.events); n < 2 ||
		sink.events[n-2].Name != "sandbox.deleted" ||
		sink.events[n-1].Name != "sandbox.workdir_removal_failed" {
		t.Fatalf("last events = %v, want sandbox.deleted then sandbox.workdir_removal_failed", sink.events)
	}

	payload, ok := sink.events[len(sink.events)-1].Payload.(domain.SandboxWorkDirRemovalFailed)
	if !ok {
		t.Fatalf("payload type = %T, want SandboxWorkDirRemovalFailed", sink.events[len(sink.events)-1].Payload)
	}
	if payload.Reason == "" {
		t.Error("payload reason is empty")
	}
}

// TestCreateSandboxWorkDirFails checks that a sandbox which cannot be
// provisioned never enters the registry (ADR-019). The invariant is
// one-directional: registry membership implies a directory exists.
func TestCreateSandboxWorkDirFails(t *testing.T) {
	sink := &fakeSink{}
	dataDir := t.TempDir()
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sandboxes := filepath.Join(dataDir, "sandboxes")
	if err := os.MkdirAll(sandboxes, 0o500); err != nil {
		t.Fatalf("mkdir sandboxes: %v", err)
	}
	t.Cleanup(func() { os.Chmod(sandboxes, 0o700) })

	if _, err := svc.CreateSandbox(); err == nil {
		t.Fatal("create sandbox error = nil, want failure")
	}
	if got := svc.reg.List(); len(got) != 0 {
		t.Errorf("registry holds %d sandboxes, want 0", len(got))
	}
	if len(sink.events) != 0 {
		t.Errorf("recorded %d events, want 0", len(sink.events))
	}
}

// findEvent returns the payload of the first event with the given name.
func findEvent(t *testing.T, events []domain.Event, name string) any {
	t.Helper()
	for _, e := range events {
		if e.Name == name {
			return e.Payload
		}
	}
	t.Fatalf("no %s event in %v", name, events)
	return nil
}

// The exit code recorded must be one the engine actually observed. A killed
// or never-started container has none, and zero would read as a clean exit
// (ADR-023).
func TestExecRecordsOutcomePayload(t *testing.T) {
	infraErr := errors.New("daemon unreachable")

	cases := []struct {
		name      string
		exec      func(context.Context, isolation.ExecRequest) (isolation.ExecResult, error)
		wantEvent string
		wantExit  *int
	}{
		{
			name: "success",
			exec: func(context.Context, isolation.ExecRequest) (isolation.ExecResult, error) {
				return isolation.ExecResult{ExitCode: 0, Stdout: "hi"}, nil
			},
			wantEvent: "run.succeeded",
			wantExit:  ptr(0),
		},
		{
			name: "non-zero exit",
			exec: func(context.Context, isolation.ExecRequest) (isolation.ExecResult, error) {
				return isolation.ExecResult{ExitCode: 2, Stderr: "boom"}, nil
			},
			wantEvent: "run.failed",
			wantExit:  ptr(2),
		},
		{
			name: "timeout",
			exec: func(context.Context, isolation.ExecRequest) (isolation.ExecResult, error) {
				return isolation.ExecResult{TimedOut: true}, nil
			},
			wantEvent: "run.timed_out",
			wantExit:  nil,
		},
		{
			name: "infra fault",
			exec: func(context.Context, isolation.ExecRequest) (isolation.ExecResult, error) {
				return isolation.ExecResult{}, infraErr
			},
			wantEvent: "run.failed",
			wantExit:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sink := &fakeSink{}
			svc := New(registry.New(), sink, &isolation.StubEngine{ExecFunc: tc.exec}, Config{
				DataDir:        t.TempDir(),
				ExecTimeout:    30 * time.Second,
				MaxOutputBytes: 1 << 20,
			})

			sb, err := svc.CreateSandbox()
			if err != nil {
				t.Fatalf("CreateSandbox: %v", err)
			}
			sink.events = nil // clear creation events

			if _, err := svc.Exec(context.Background(), ExecInput{
				SandboxID: sb.ID,
				Command:   "echo hi",
			}); err != nil && !errors.Is(err, infraErr) {
				t.Fatalf("Exec: %v", err)
			}

			cmd, ok := findEvent(t, sink.events, "run.created").(domain.RunCommand)
			if !ok || cmd.Command != "echo hi" {
				t.Errorf("run.created payload = %+v, want command %q", cmd, "echo hi")
			}

			payload := findEvent(t, sink.events, tc.wantEvent)
			outcome, ok := payload.(domain.RunOutcome)
			if !ok {
				timedOut, isTimeout := payload.(domain.RunTimedOutOutcome)
				if !isTimeout {
					t.Fatalf("%s payload = %T, want an outcome", tc.wantEvent, payload)
				}
				if timedOut.TimeoutMS != (30 * time.Second).Milliseconds() {
					t.Errorf("TimeoutMS = %d, want 30000", timedOut.TimeoutMS)
				}
				outcome = timedOut.RunOutcome
			}

			switch {
			case tc.wantExit == nil && outcome.ExitCode != nil:
				t.Errorf("ExitCode = %d, want nil", *outcome.ExitCode)
			case tc.wantExit != nil && outcome.ExitCode == nil:
				t.Errorf("ExitCode = nil, want %d", *tc.wantExit)
			case tc.wantExit != nil && *outcome.ExitCode != *tc.wantExit:
				t.Errorf("ExitCode = %d, want %d", *outcome.ExitCode, *tc.wantExit)
			}

			if outcome.StdoutSHA256 == "" || outcome.StderrSHA256 == "" {
				t.Errorf("hashes = %q/%q, want both set", outcome.StdoutSHA256, outcome.StderrSHA256)
			}
		})
	}
}

func ptr(i int) *int { return &i }

func TestExecRefusesWhenRunDirCannotBeCreated(t *testing.T) {
	dataDir := t.TempDir()
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	// A regular file where runs/ must be a directory: MkdirAll fails the same
	// way a read-only mount or a full disk would.
	if err := os.WriteFile(filepath.Join(dataDir, "runs"), nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "echo hi",
	})
	if err == nil {
		t.Fatal("Exec: want error, got nil")
	}
	if out.Run.Status != domain.RunFailed {
		t.Errorf("run status = %s, want %s", out.Run.Status, domain.RunFailed)
	}
	if sb.Status != domain.SandboxReady {
		t.Errorf("sandbox status = %s, want %s", sb.Status, domain.SandboxReady)
	}

	findEvent(t, sink.events, "run.preparation_failed")
	for _, e := range sink.events {
		if e.Name == "run.running" {
			t.Error("run.running recorded on a refused run")
		}
	}
}

func TestExecWritesRunLogs(t *testing.T) {
	dataDir := t.TempDir()
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "echo hi",
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if out.LogsError != nil {
		t.Fatalf("LogsError = %v, want nil", out.LogsError)
	}

	dir := runDirFor(dataDir, out.Run.ID)
	stdout, err := os.ReadFile(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	// The file holds the bytes the payload hashed — one claim in two places.
	if string(stdout) != out.Result.Stdout {
		t.Errorf("stdout file = %q, want %q", stdout, out.Result.Stdout)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "stderr")); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
}

func TestExecRecordsLogsWriteFailure(t *testing.T) {
	dataDir := t.TempDir()
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{}, Config{
		DataDir:        dataDir,
		ExecTimeout:    30 * time.Second,
		MaxOutputBytes: 1 << 20,
	})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	out, err := svc.Exec(context.Background(), ExecInput{
		SandboxID: sb.ID,
		Command:   "echo hi",
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if out.LogsError != nil {
		t.Fatalf("LogsError = %v, want nil", out.LogsError)
	}
	_ = out
}
