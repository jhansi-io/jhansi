package evidence

import "github.com/jhansi-io/jhansi/internal/domain"

// Sink records drained domain events. One of the four seams: the
// default is a local append-only log; SIEM and remote sinks are second
// implementations, on demand (ADR-007).
//
// Record is synchronous and fatal — a failed write must fail the
// request, so an execution nobody could record is never claimed to
// have happened.
type Sink interface {
	Record(events []domain.Event) error
}
