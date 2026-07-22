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
	eventBuffer
}

// NewSandbox mints a new sandbox in the CREATING state.
func NewSandbox(id string) *Sandbox {
	s := &Sandbox{
		ID: id,
		Status: SandboxCreating,
		CreatedAt: time.Now().UTC(),
		eventBuffer: eventBuffer{aggregateID: id},
	}
	s.record("sandbox.created", s.CreatedAt)
	return s
}

// MarkReady moves the sandbox to READY. Legal from CREATING (came up)
// or ACTIVE (a run finished).
func (s *Sandbox) MarkReady() error {
	if s.Status != SandboxCreating && s.Status != SandboxActive {
		return fmt.Errorf("sandbox %s: cannot mark ready from %s", s.ID, s.Status)
	}
	s.Status = SandboxReady
	s.record("sandbox.ready", time.Now().UTC())
	return nil
}

// MarkActive moves the sandbox to ACTIVE. Legal only from READY —
// a run is starting. Enforces one live run per sandbox.
func (s *Sandbox) MarkActive() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot mark active from %s", s.ID, s.Status)
	}
	s.Status = SandboxActive
	s.record("sandbox.active", time.Now().UTC())
	return nil
}

// MarkDeleted moves the sandbox to DELETED (explicit destroy).
// Legal only from READY — an active run resolves to READY first.
func (s *Sandbox) MarkDeleted() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot mark deleted from %s", s.ID, s.Status)
	}
	s.Status = SandboxDeleted
	s.record("sandbox.deleted", time.Now().UTC())
	return nil
}

// MarkExpired moves the sandbox to EXPIRED (TTL ended it).
// Legal only from READY.
func (s *Sandbox) MarkExpired() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot expire from %s", s.ID, s.Status)
	}
	s.Status = SandboxExpired
	s.record("sandbox.expired", time.Now().UTC())
	return nil
}

// MarkError moves the sandbox to ERROR (infrastructure fault).
// Legal only from any live state: CREATING, READY, or ACTIVE.
func (s *Sandbox) MarkError() error {
	switch s.Status {
	case SandboxCreating, SandboxReady, SandboxActive:
		s.Status = SandboxError
		s.record("sandbox.error", time.Now().UTC())
		return nil
	default:
		return fmt.Errorf("sandbox %s: cannot error from %s", s.ID, s.Status)
	}
}
