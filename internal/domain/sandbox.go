package domain

import "time"

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
