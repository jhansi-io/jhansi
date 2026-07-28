package service

import (
	"errors"
	"github.com/jhansi-io/jhansi/internal/domain"
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
	svc := New(registry.New(), sink)

	sb, err := svc.CreateSandbox()
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	if sb.Status != domain.SandboxCreating {
		t.Errorf("status = %s, want %s", sb.Status, domain.SandboxCreating)
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
	if len(sink.events) != 1 {
		t.Fatalf("recorded %d events, want 1", len(sink.events))
	}
	if sink.events[0].Name != "sandbox.created" {
		t.Errorf("event = %q, want sandbox.created", sink.events[0].Name)
	}
	if sink.events[0].AggregateID != sb.ID {
		t.Errorf("event aggregateID = %q, want %q", sink.events[0].AggregateID, sb.ID)
	}
}

func TestCreateSandboxFailure(t *testing.T) {
	sinkErr := errors.New("sink down")
	svc := New(registry.New(), &fakeSink{err: sinkErr})

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
	svc := New(registry.New(), sink)

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
	if len(sink.events) != 2 {
		t.Fatalf("recorded %d events, want 2", len(sink.events))
	}
	if sink.events[1].Name != "sandbox.deleted" {
		t.Errorf("event = %q, want sandbox.deleted", sink.events[1].Name)
	}
}

func TestDeleteSandboxIdempotent(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink)

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
	if len(sink.events) != 3 {
		t.Fatalf("recorded %d events, want 3", len(sink.events))
	}
	last := sink.events[2]
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
	svc := New(registry.New(), &fakeSink{})
	err := svc.DeleteSandbox("sb_missing")
	if !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDeleteSandboxRecordFailure(t *testing.T) {
	sink := &fakeSink{}
	svc := New(registry.New(), sink)

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
