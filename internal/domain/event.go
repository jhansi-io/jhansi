package domain

import "time"

// Event is a record of something that happened to an aggregate.
// The envelope is minimal by ADR-002; payload and a typed name
// taxonomy are deferred to the event-model ADR.
type Event struct {
	Name        string
	At          time.Time
	AggregateID string
}
