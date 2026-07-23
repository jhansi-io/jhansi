package domain

import "testing"

func TestNewSandbox(t *testing.T) {
	s := NewSandbox("sb_1")
	if got := s.DrainEvents(); len(got) != 1 || got[0].Name != "sandbox.created" {
		t.Fatalf("expected sandbox.created, got %v", got)
	}

	if s.ID != "sb_1" {
		t.Errorf("ID = %q, want %q", s.ID, "sb_1")
	}

	if s.Status != SandboxCreating {
		t.Errorf("Status = %q, want %q", s.Status, SandboxCreating)
	}
}

func TestMarkReady(t *testing.T) {
	tests := []struct {
		name    string
		from    SandboxStatus
		wantErr bool
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
				if events := s.DrainEvents(); len(events) != 1 || events[0].Name != "sandbox.ready_rejected" {
					t.Errorf("events = %v, want one sandbox.ready_rejected", events)
				}
				return
			}
			if err != nil {
				t.Errorf("MarkReady from %s: unexpected error %v", tt.from, err)
			}
			if s.Status != SandboxReady {
				t.Errorf("Status = %s, want READY", s.Status)
			}
		})
	}
}

func TestMarkActive(t *testing.T) {
	tests := []struct {
		name    string
		from    SandboxStatus
		wantErr bool
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
				if events := s.DrainEvents(); len(events) != 1 || events[0].Name != "sandbox.active_rejected" {
					t.Errorf("events = %v, want one sandbox.active_rejected", events)
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
		name      string
		from      SandboxStatus
		mark      func(*Sandbox) error
		want      SandboxStatus
		wantEvent string
		wantErr   bool
	}{
		{"delete from ready", SandboxReady, (*Sandbox).MarkDeleted, SandboxDeleted, "sandbox.deleted", false},
		{"delete from active", SandboxActive, (*Sandbox).MarkDeleted, "", "sandbox.delete_rejected", true},
		{"expire from ready", SandboxReady, (*Sandbox).MarkExpired, SandboxExpired, "sandbox.expired", false},
		{"expire from creating", SandboxCreating, (*Sandbox).MarkExpired, "", "sandbox.expire_rejected", true},
		{"error from creating", SandboxCreating, (*Sandbox).MarkError, SandboxError, "sandbox.error", false},
		{"error from active", SandboxActive, (*Sandbox).MarkError, SandboxError, "sandbox.error", false},
		{"error from deleted", SandboxDeleted, (*Sandbox).MarkError, "", "sandbox.error_rejected", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sandbox{ID: "sb_1", Status: tt.from}
			err := tt.mark(s)

			if tt.wantErr {
				if err == nil {
					t.Errorf("want error, got nil")
				}

				events := s.DrainEvents()
				if len(events) != 1 || events[0].Name != tt.wantEvent {
					t.Errorf("events = %v, want one %s", events, tt.wantEvent)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if s.Status != tt.want {
				t.Errorf("Status = %s, want %s", s.Status, tt.want)
			}
			events := s.DrainEvents()
			if len(events) != 1 || events[0].Name != tt.wantEvent {
				t.Errorf("events = %v, want one %s", events, tt.wantEvent)
			}
		})
	}
}

func TestSandboxEventSequence(t *testing.T) {
	s := NewSandbox("sb_1")
	if err := s.MarkReady(); err != nil {
		t.Fatalf("MarkReady: %v", err)
	}
	if err := s.MarkActive(); err != nil {
		t.Fatalf("MarkActive: %v", err)
	}
	events := s.DrainEvents()
	want := []string{"sandbox.created", "sandbox.ready", "sandbox.active"}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %v", len(events), len(want), events)
	}
	for i, name := range want {
		if events[i].Name != name {
			t.Errorf("events[%d].Name = %q, want %q", i, events[i].Name, name)
		}
	}
}

func TestSandboxRejectionPayload(t *testing.T) {
	s := &Sandbox{ID: "sb_1", Status: SandboxActive}
	if err := s.MarkDeleted(); err == nil {
		t.Fatalf("want error, got nil")
	}

	events := s.DrainEvents()
	if len(events) != 1 {
		t.Fatalf("events = %v, want one", events)
	}
	got, ok := events[0].Payload.(SandboxTransitionRejected)
	if !ok {
		t.Fatalf("payload = %T, want SandboxTransitionRejected", events[0].Payload)
	}
	if got.From != SandboxActive || got.Action != "delete" {
		t.Errorf("payload = %v, want {ACTIVE delete}", got)
	}
}
