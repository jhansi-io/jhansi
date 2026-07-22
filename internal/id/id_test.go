package id

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	got, err := New("sb")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if !strings.HasPrefix(got, "sb_") {
		t.Errorf("missing prefix: got %q", got)
	}
	body := strings.TrimPrefix(got, "sb_")
	b, err := hex.DecodeString(body)
	if err != nil {
		t.Errorf("body not valid hex: %q", body)
	}
	if len(b) != 16 {
		t.Errorf("want 16 bytes, got %d", len(b))
	}
}

func TestNewUnique(t *testing.T) {
	a, _ := New("sb")
	b, _ := New("sb")
	if a == b {
		t.Errorf("ids collided: %q", a)
	}
}
