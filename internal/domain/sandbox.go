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
}

// NewSandbox mints a new sandbox in the CREATING state.
func NewSandbox(id string) *Sandbox {
	return &Sandbox{
		ID: id,
		Status: SandboxCreating,
		CreatedAt: time.Now().UTC(),
	}
}

// MarkReady moves the sandbox to READY. Legal from CREATING (came up)
// or ACTIVE (a run finished).
func (s *Sandbox) MarkReady() error {
	if s.Status != SandboxCreating && s.Status != SandboxActive {
		return fmt.Errorf("sandbox %s: cannot mark ready from %s", s.ID, s.Status)
	}
	s.Status = SandboxReady
	return nil
}

// MarkActive moves the sandbox to ACTIVE. Legal only from READY —
// a run is starting. Enforces one live run per sandbox.
func (s *Sandbox) MarkActive() error {
	if s.Status != SandboxReady {
		return fmt.Errorf("sandbox %s: cannot mark active from %s", s.ID, s.Status)
	}
	s.Status = SandboxActive
	return nil
}
