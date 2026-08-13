package service

import "path/filepath"

// workDirFor returns the filesystem path of a sandbox' working directory.
// The path is derived from the data directory and the sandbox ID rather than
// stored on the aggregate, so it stays correct when the data directory moves.
func workDirFor(dataDir, sandboxID string) string {
	return filepath.Join(dataDir, "sandboxes", sandboxID)
}
