package evidence

import (
	"encoding/json"
	"github.com/jhansi-io/jhansi/internal/domain"
	"os"
	"sync"
)

var _ Sink = (*FileSink)(nil)

// FileSink is the default sink: an append-only JSON Lines log, one
// event object per line, fsynced per Record (ADR-007).
type FileSink struct {
	mu sync.Mutex
	f  *os.File
}

// NewFileSink opens path for appending, creating it if absent, and
// holds the handle open for the life of the sink.
func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &FileSink{f: f}, nil
}

// Record appends each event as a JSON line, then fsyncs. Synchronous
// and fatal: any failure returns and the caller must fail the request
// (ADR-007).
func (s *FileSink) Record(events []domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range events {
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := s.f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return s.f.Sync()
}
