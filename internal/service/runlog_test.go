package service

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteRunLogs checks both streams land in the run's directory, byte for
// byte, when the directory exists.
func TestWriteRunLogs(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(runDirFor(dataDir, "run_abc"), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := writeRunLogs(dataDir, "run_abc", "out", "err"); err != nil {
		t.Fatalf("writeRunLogs: %v", err)
	}

	for name, want := range map[string]string{"stdout": "out", "stderr": "err"} {
		got, err := os.ReadFile(filepath.Join(runDirFor(dataDir, "run_abc"), name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

// TestWriteRunLogsMissingDir checks the write reports failure rather than
// silently losing output when the run directory is not there.
func TestWriteRunLogsMissingDir(t *testing.T) {
	if err := writeRunLogs(t.TempDir(), "run_abc", "out", "err"); err == nil {
		t.Fatal("writeRunLogs: want error for a missing run directory, got nil")
	}
}
