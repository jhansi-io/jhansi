package service

import "path/filepath"

// workDirFor returns the filesystem path of a sandbox' working directory.
// The path is derived from the data directory and the sandbox ID rather than
// stored on the aggregate, so it stays correct when the data directory moves.
func workDirFor(dataDir, sandboxID string) string {
	return filepath.Join(dataDir, "sandboxes", sandboxID)
}

// runDirFor returns the filesystem path of a run's log directory. It sits at
// the data-dir root rather than under sandboxes/, because evidence outlives
// the sandbox that produced it (ADR-024).
func runDirFor(dataDir, runID string) string {
	return filepath.Join(dataDir, "runs", runID)
}
