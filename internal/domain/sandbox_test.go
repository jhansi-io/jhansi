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

func TestMarkReady(t *testing.T) {
	tests := []struct {
		name	string
		from	SandboxStatus
		wantErr	bool
	}{
		{"from creating", SandboxCreating, false},
		{"from active", SandboxActive, false},
		{"from deleted", SandboxDeleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sandbox{ID: "sb_1", Status: tt.from}
			err := s.MarkReady()

			if tt.wantErr {
				if err == nil {
					t.Errorf("MarkReady from %s: want error, got nil", tt.from)
				}
				return 
			}
			if err != nil {
				t.Errorf("MarkReady from %s: unexpected error %v", tt.from, err)
			}
			if s.Status != SandboxReady {
				t.Errorf("Sttaus = %s, wan READY", s.Status)
			}
		})
	}
}

func TestMarkActive(t *testing.T) {
	tests := []struct {
		name	string
		from 	SandboxStatus
		wantErr	bool	
	}{
		{"from ready", SandboxReady, false},
		{"from active", SandboxActive, true},
		{"from creating", SandboxCreating, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sandbox{ID: "sb_1", Status: tt.from}
			err := s.MarkActive()

			if tt.wantErr {
				if err == nil {
					t.Errorf("MarkActive from %s: want error, got nil", tt.from)
				}
				return
			}
			if err != nil {
				t.Errorf("MarkActive from %s: unexpected error %v", tt.from, err)
			}
			if s.Status != SandboxActive {
				t.Errorf("Status = %s, want ACTIVE", s.Status)
			}
		})
	}
}

func TestSandboxTerminals(t *testing.T) {
	tests := []struct {
		name    string
		from    SandboxStatus
		mark    func(*Sandbox) error
		want    SandboxStatus
		wantErr bool
	}{
		{"delete from ready", SandboxReady, (*Sandbox).MarkDeleted, SandboxDeleted, false},
		{"delete from active", SandboxActive, (*Sandbox).MarkDeleted, "", true},
		{"expire from ready", SandboxReady, (*Sandbox).MarkExpired, SandboxExpired, false},
		{"expire from creating", SandboxCreating, (*Sandbox).MarkExpired, "", true},
		{"error from creating", SandboxCreating, (*Sandbox).MarkError, SandboxError, false},
		{"error from active", SandboxActive, (*Sandbox).MarkError, SandboxError, false},
		{"error from deleted", SandboxDeleted, (*Sandbox).MarkError, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sandbox{ID: "sb-1", Status: tt.from}
			err := tt.mark(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if s.Status != tt.want {
				t.Errorf("Status = %s, want %s", s.Status, tt.want)
			}
		})
	}
}
