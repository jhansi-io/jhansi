package service

import (
	"path/filepath"
	"testing"
)

// TestWorkDirFor checks that a sandbox's working directory is the sandbox ID
// under a "sandboxes" collection inside the data directory.
func TestWorkDirFor(t *testing.T) {
	got := workDirFor("/var/lib/jhansi", "sb_abc123")
	want := filepath.Join("/var/lib/jhansi", "sandboxes", "sb_abc123")
	if got != want {
		t.Errorf("workDirFor() = %q, want %q", got, want)
	}
}

// TestRunDirFor checks that a run's working directory is the run ID
// under a "runs" collection inside the data directory.
func TestRunDirFor(t *testing.T) {
	got := runDirFor("/var/lib/jhansi", "run_abc123")
	want := filepath.Join("/var/lib/jhansi", "runs", "run_abc123")
	if got != want {
		t.Errorf("runDirFor() = %q, want %q", got, want)
	}
}
