package domain

import "time"

// Event is a record of something that happened to an aggregate.
// Name, At and AggregateID are the envelope (ADR-002). Payload is nil
// unless the event carries data (ADR-006). A typed name taxonomy and
// versioning remain deferred.
//
// Rejection events are named as the exact twin of the success event
// they refused: sandbox.deleted -> sandbox.deleted_rejected. Payload
// Action carries the same stem.
type Event struct {
	Name        string
	At          time.Time
	AggregateID string
	Payload     any
}
