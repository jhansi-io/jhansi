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
