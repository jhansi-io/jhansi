package service

import (
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/id"
	"github.com/jhansi-io/jhansi/internal/registry"
)

type eventSource interface {
	DrainEvents() []domain.Event
}

// ExecutionService orchestrates mutating operations: It holds the
// registry and sink, and routes every drain-and-record through one
// helper so write-ahead ordering stays a single later edit (ADR=008).
type ExecutionService struct {
	reg  *registry.Registry
	sink evidence.Sink
}

// New constructs an ExecutionService over a registry and sink.
func New(reg *registry.Registry, sink evidence.Sink) *ExecutionService {
	return &ExecutionService{reg: reg, sink: sink}
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

// drainRecord drains an aggregate's buffered events and hands them
// to the sink. The single home ADR-008 required: every operation
// records through here, so write-ahead ordering becomes one later edit
// rather than a change at every call site (ADR-007).
func (s *ExecutionService) drainAndRecord(src eventSource) error {
	events := src.DrainEvents()
	return s.sink.Record(events)
}
