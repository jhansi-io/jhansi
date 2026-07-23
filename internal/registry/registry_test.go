package registry

import (
	"errors"
	"github.com/jhansi-io/jhansi/internal/domain"
	"testing"
)

func TestAdd(t *testing.T) {
	r := New()
	s := domain.NewSandbox("sb_1")

	if err := r.Add(s); err != nil {
		t.Fatalf("Add: unexpected error %v", err)
	}

	if r.sandboxes["sb_1"] != s {
		t.Errorf("sandbox not stored under its id")
	}
}

func TestAddDuplicate(t *testing.T) {
	r := New()
	first := domain.NewSandbox("sb_1")
	if err := r.Add(first); err != nil {
		t.Fatalf("setup Add: %v", err)
	}

	second := domain.NewSandbox("sb_1")
	err := r.Add(second)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Add duplicate: got %v, want ErrAlreadyExists", err)
	}
	if r.sandboxes["sb_1"] != first {
		t.Errorf("duplicate Add overwrote the stored sandbox")
	}
}

func TestGet(t *testing.T) {
	r := New()
	s := domain.NewSandbox("sb_1")
	if err := r.Add(s); err != nil {
		t.Fatalf("setup Add: %v", err)
	}
	got, err := r.Get("sb_1")
	if err != nil {
		t.Fatalf("Get: unexpected error %v", err)
	}
	if got != s {
		t.Errorf("Get returned a  different sandbox")
	}
}

func TestGetMissing(t *testing.T) {
	r := New()

	got, err := r.Get("sb_nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing: got %v, want ErrNotFound", err)
	}
	if got != nil {
		t.Errorf("Get missing returned %v, want nil", got)
	}
}

func TestListEmpty(t *testing.T) {
	r := New()

	got := r.List()
	if got == nil {
		t.Fatalf("List on empty registry returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("List = %v, want empty", got)
	}
}

func TestList(t *testing.T) {
	r := New()
	for _, id := range []string{"sb_1", "sb_2"} {
		if err := r.Add(domain.NewSandbox(id)); err != nil {
			t.Fatalf("setup Add %s: %v", id, err)
		}
	}

	got := r.List()
	if len(got) != 2 {
		t.Fatalf("List returned %d sandboxes, want 2", len(got))
	}

	seen := map[string]bool{}
	for _, s := range got {
		seen[s.ID] = true
	}
	if !seen["sb_1"] || !seen["sb_2"] {
		t.Errorf("List = %v, want both sandboxes", got)
	}
}
