package domain

import "testing"

func TestNewSandbox(t *testing.T) {
	s := NewSandbox("sb_1")

	if s.ID != "sb_1" {
		t.Errorf("ID = %q, want %q", s.ID, "sb_1")
	}

	if s.Status != SandboxCreating {
		t.Errorf("Status = %q, want %q", s.Status, SandboxCreating)
	}
}
