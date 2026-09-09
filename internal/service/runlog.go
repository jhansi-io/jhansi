package service

import (
	"os"
	"path/filepath"
)

// writeRunLogs writes a run's retained output to its log directory. The bytes
// are the ones the event payload hashed — same truncation, same tail — so the
// hash and the file are one claim in two places (ADR-024). The directory
// already exists: the preflight created it (ADR-025).
func writeRunLogs(dataDir, runID, stdout, stderr string) error {
	dir := runDirFor(dataDir, runID)
	if err := os.WriteFile(filepath.Join(dir, "stdout"), []byte(stdout), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "stderr"), []byte(stderr), 0o600)
}
