package domain

import (
	"fmt"
	"time"
)

//SandboxStatus is the lifecycle state of a sandbox.

type SandboxStatus string

const (
	SandboxCreating SandboxStatus = "CREATING"
	SandboxReady	SandboxStatus = "READY"
	SandboxActive	SandboxStatus = "ACTIVE"
	SandboxExpired	SandboxStatus = "EXPIRED"
	SandboxDeleted	SandboxStatus = "DELETED"
	SandboxError	SandboxStatus = "ERROR"
)

type Sandbox struct {
	ID			string
	Status		SandboxStatus
	CreatedAt	time.Time
	events		[]Event
}

// NewSandbox mints a new sandbox in the CREATING state.
func NewSandbox(id string) *Sandbox {
	s := &Sandbox{
		ID: id,
		Status: SandboxCreating,
		CreatedAt: time.Now().UTC(),
	}
	s.events = append(s.events, Event{
		Name:			"sandbox.created",
		At:				s.CreatedAt,
		AggregateID:	s.ID, 		
	})
	return s
}

// MarkReady moves the sandbox to READY. Legal from CREATING (came up)
// or ACTIVE (a run finished).
func (s *Sandbox) MarkReady() error {
	if s.Status != SandboxCreating && s.Status != SandboxActive {
		return fmt.Errorf("sandbox %s: cannot mark ready from %s", s.ID, s.Status)
	}
	s.Status = SandboxReady
	s.events = append(s.events, Event{
		Name:			"sandbox.ready",
		At:				time.Now().UTC(),
		AggregateID:	s.ID,
	})
	return nil
}

// MarkActive moves the sandbox to ACTIVE. Legal only from READY —
// a run is starting. Enforces one live run per sandbox.
func (s *Sandbox) MarkActive() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot mark active from %s", s.ID, s.Status)
	}
	s.Status = SandboxActive
	s.events = append(s.events, Event{
		Name:			"sandbox.active",
		At:				time.Now().UTC(),
		AggregateID:	s.ID,
	})
	return nil
}

// MarkDeleted moves the sandbox to DELETED (explicit destroy).
// Legal only from READY — an active run resolves to READY first.
func (s *Sandbox) MarkDeleted() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot mark deleted from %s", s.ID, s.Status)
	}
	s.Status = SandboxDeleted
	s.events = append(s.events, Event{
		Name:			"sandbox.deleted",
		At:				time.Now().UTC(),
		AggregateID:	s.ID,
	})
	return nil
}

// MarkExpired moves the sandbox to EXPIRED (TTL ended it).
// Legal only from READY.
func (s *Sandbox) MarkExpired() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot expire from %s", s.ID, s.Status)
	}
	s.Status = SandboxExpired
	s.events = append(s.events, Event{
		Name:			"sandbox.expired",
		At:				time.Now().UTC(),
		AggregateID:	s.ID,
	})
	return nil
}

// MarkError moves the sandbox to ERROR (infrastructure fault).
// Legal only from any live state: CREATING, READY, or ACTIVE.
func (s *Sandbox) MarkError() error {
	switch s.Status {
	case SandboxCreating, SandboxReady, SandboxActive:
		s.Status = SandboxError
		s.events = append(s.events, Event{
			Name:			"sandbox.error",
			At:				time.Now().UTC(),
			AggregateID:	s.ID,
		})
		return nil
	default:
		return fmt.Errorf("sandbox %s: cannot error from %s", s.ID, s.Status)
	}
}

// DrainEvents returns the buffered events and clears the buffer.
func (s *Sandbox) DrainEvents() []Event {
	events := s.events
	s.events = nil
	return events
}
