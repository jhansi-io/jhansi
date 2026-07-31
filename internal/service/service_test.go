package service

import (
	"context"
	"errors"
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"github.com/jhansi-io/jhansi/internal/registry"
	"strings"
	"testing"
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

	svc := New(registry.New(), sink, &isolation.StubEngine{})

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
	svc := New(registry.New(), &fakeSink{err: sinkErr}, &isolation.StubEngine{})

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
	svc := New(registry.New(), sink, &isolation.StubEngine{})

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
	svc := New(registry.New(), sink, &isolation.StubEngine{})

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
	svc := New(registry.New(), &fakeSink{}, &isolation.StubEngine{})
	err := svc.DeleteSandbox("sb_missing")
	if !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSandboxRecordFailure(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink, &isolation.StubEngine{})

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
	svc := New(registry.New(), sink, &isolation.StubEngine{})

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	run, result, err := svc.Exec(context.Background(), sb.ID, "echo hello")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if run.Status != domain.RunSucceeded {
		t.Errorf("run status = %q, want %q", run.Status, domain.RunSucceeded)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "echo hello" {
		t.Errorf("stdout = %q, want the command echoed", result.Stdout)
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
		ExecFunc: func(ctx context.Context, sandboxID, command string) (isolation.ExecResult, error) {
			return isolation.ExecResult{ExitCode: 1, Stderr: "boom"}, nil
		},
	}
	svc := New(registry.New(), sink, engine)
	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	run, result, err := svc.Exec(context.Background(), sb.ID, "false")
	if err != nil {
		t.Fatalf("exec: %v — a non-zero exit is a completed run, not an error", err)
	}
	if run.Status != domain.RunFailed {
		t.Errorf("run status = %q, want %q", run.Status, domain.RunFailed)
	}
	if result.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", result.ExitCode)
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
		ExecFunc: func(ctx context.Context, sandboxID, command string) (isolation.ExecResult, error) {
			return isolation.ExecResult{TimedOut: true}, nil
		},
	}
	svc := New(registry.New(), sink, engine)

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	run, _, err := svc.Exec(context.Background(), sb.ID, "sleep 999")
	if err != nil {
		t.Fatalf("exec: %v — a timeout is a completed run, not an error", err)
	}
	if run.Status != domain.RunTimedOut {
		t.Errorf("run status = %q, want %q", run.Status, domain.RunTimedOut)
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
		ExecFunc: func(ctx context.Context, sandboxID, command string) (isolation.ExecResult, error) {
			return isolation.ExecResult{}, infraErr
		},
	}
	svc := New(registry.New(), sink, engine)

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	run, _, err := svc.Exec(context.Background(), sb.ID, "anything")

	if !errors.Is(err, infraErr) {
		t.Fatalf("exec err = %v, want the infra error surfaced (→ 500)", err)
	}
	if run.Status != domain.RunFailed {
		t.Errorf("run status = %q, want %q", run.Status, domain.RunFailed)
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
	svc := New(registry.New(), sink, engine)

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	// Force the sandbox ACTIVE. At Tier 0 exec is synchronous, so the only
	// way to meet Exec with an alteady-claimed sandbox is to claim it first.
	if err := sb.MarkActive(); err != nil {
		t.Fatalf("setup MarkActive: %v", err)
	}
	run, _, err := svc.Exec(context.Background(), sb.ID, "anything")
	if err == nil {
		t.Fatalf("exec on a busy sandbox: err = nil, want a rejection")
	}
	if run != nil {
		t.Errorf("run = %v, want nil — nothing is minted past a failed claim", run)
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
