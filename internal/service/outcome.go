package service

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/jhansi-io/jhansi/internal/domain"
	"github.com/jhansi-io/jhansi/internal/isolation"
	"time"
)

// sha256Hex returns the hex SHA-256 of s. It commits to the bytes jhansi
// retained, not to everything the command wrote (ADR-023). An empty stream
// hashes to the digest of the empty string, which records that nothing was
// retained.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// runOutcome maps an engine result and the observed elapsed time onto the
// event payload. exitCode is nil on an infra fault, where the engine never
// ran the command and no exit was observed (ADR-023).
func runOutcome(result isolation.ExecResult, elapsed time.Duration, exitCode *int) domain.RunOutcome {
	return domain.RunOutcome{
		ExitCode:        exitCode,
		DurationMS:      elapsed.Milliseconds(),
		StdoutBytes:     len(result.Stdout),
		StderrBytes:     len(result.Stderr),
		StdoutSHA256:    sha256Hex(result.Stdout),
		StderrSHA256:    sha256Hex(result.Stderr),
		OutputTruncated: result.OutputTruncated,
	}
}
