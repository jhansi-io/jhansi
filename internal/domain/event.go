package domain

import "time"

// Event is a record of something that happened to an aggregate.
// Name, At abd AggregateID are the envelope (ADR-002). Payload is nil
// unless the event carries data (ADR-006). A typed name taxonomy and
// versioning remain deferred.
type Event struct {
	Name        string
	At          time.Time
	AggregateID string
	Payload     any
}
