package domain

import "time"

// eventBuffer holds an aggregate's unflushed domain events. Embedded in
// each aggregate so record and DrainEvents promote onto it directly.
type eventBuffer struct {
	aggregateID string
	events		[]Event
}

// record appends an event, stamping the buffer's aggregateID. The caller
// supplies at so New() can match CreatedAt and clock injection stays open.
func (b *eventBuffer) record(name string, at time.Time) {
	b.events = append(b.events, Event{
		Name:			name,
		At:				at,
		AggregateID:	b.aggregateID,
	})
}

// DrainEvents returns the buffered events and clears the buffer.
func (b *eventBuffer) DrainEvents() []Event {
	events := b.events
	b.events = nil
	return events
}
