package service

import (
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/evidence"
	"github.com/jhansi-io/jhansi/internal/id"
	"github.com/jhansi-io/jhansi/internal/registry"
)

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
	if err := s.reg.Add(sb); err != nil {
		return nil, err
	}

	if err := s.drainAndRecord(sb); err != nil {
		return nil, err
	}
	return sb, nil
}

// drainRecord drains an aggregate's buffered events and hands them
// to the sink. The single home ADR-008 required: every operation
// records through here, so write-ahead ordering becomes one later edit
// rather than a change at every call site (ADR-007).
func (s *ExecutionService) drainAndRecord(sb *domain.Sandbox) error {
	events := sb.DrainEvents()
	return s.sink.Record(events)
}
