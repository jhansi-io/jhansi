package domain

import (
	"fmt"
	"time"
)

//RunStatus is the lifecycle state of a single execution.
type RunStatus string

const (
	RunQueued		RunStatus = "QUEUED"
	RunPreparing	RunStatus = "PREPARING"
	RunRunning		RunStatus = "RUNNING"
	RunSucceeded	RunStatus = "SUCCEEDED"
	RunFailed		RunStatus = "FAILED"
	RunTimedOut		RunStatus = "TIMED_OUT"
	RunCancelled	RunStatus = "CANCELLED"
)

type Run struct {
	ID			string
	SandboxID	string
	Status		RunStatus
	CreatedAt	time.Time
	events		[]Event
}

func NewRun(id, sandboxID string) *Run {
	r := &Run{
		ID:			id,
		SandboxID:	sandboxID,
		Status:		RunQueued,
		CreatedAt:	time.Now().UTC(),
	}
	r.events = append(r.events, Event{
		Name:		"run.created",
		At:			r.CreatedAt,
		AggregateID:r.ID,
	})
	return r
}

// MarkPreparing moves the run to PREPARING. Legal only from QUEUED.
func (r *Run) MarkPreparing() error {
	if r.Status != RunQueued {
		return fmt.Errorf("run %s: cannot mark preparing from %s", r.ID, r.Status)
	}
	r.Status = RunPreparing
	r.events = append(r.events, Event{
		Name:			"run.preparing",
		At:				time.Now().UTC(),
		AggregateID:	r.ID,
	})
	return nil
}

//MarkRunning moves the run to Running. Legal only from PREPARING.
func (r *Run) MarkRunning() error {
	if r.Status != RunPreparing {
		return fmt.Errorf("run %s: cannot mark running from %s", r.ID, r.Status)
	}
	r.Status = RunRunning
	r.events = append(r.events, Event{
		Name:			"run.running",
		At:				time.Now().UTC(),
		AggregateID:	r.ID,
	})
	return nil
}

// MarkSucceeded moves the run to Succeeded. Legal only from RUNNING.
func (r *Run) MarkSucceeded() error {
	if r.Status != RunRunning {
		return fmt.Errorf("run %s: cannot mark succeeded from %s", r.ID, r.Status)
	}
	r.Status = RunSucceeded
	r.events = append(r.events, Event{
		Name:			"run.succeeded",
		At:				time.Now().UTC(),
		AggregateID:	r.ID,
	})
	return nil
}

// MarkFailed moves the run to FAILED. Legal only from RUNNING.
func (r *Run) MarkFailed() error {
	if r.Status != RunRunning {
		return fmt.Errorf("run %s: cannot mark failed from %s", r.ID, r.Status)
	}
	r.Status = RunFailed
	r.events = append(r.events, Event{
		Name:			"run.failed",
		At:				time.Now().UTC(),
		AggregateID:	r.ID,
	})
	return nil
}

// MarkTimedOut moves the run to TIMED_OUT. Legal only from RUNNING.
func (r *Run) MarkTimedOut() error {
	if r.Status != RunRunning {
		return fmt.Errorf("run %s: cannot mark timed_out from %s", r.ID, r.Status)
	}
	r.Status = RunTimedOut
	r.events = append(r.events, Event{
		Name:			"run.timed_out",
		At:				time.Now().UTC(),
		AggregateID:	r.ID,
	})
	return nil
}

// MarkCancelled moves the run to CANCELLED (explicitly stopped).
// Legal from any non-terminal state: QUEUED, PREPARING or RUNNING.
func (r *Run) MarkCancelled() error {
	switch r.Status {
	case RunQueued, RunPreparing, RunRunning:
		r.Status = RunCancelled
		r.events = append(r.events, Event{
			Name:			"run.cancelled",
			At:				time.Now().UTC(),
			AggregateID:	r.ID,
		})
		return nil
	default:
		return fmt.Errorf("run %s: cannot mark cancelled from %s", r.ID, r.Status)
	}
}

//DrainEvents returns the buffered events and clears the buffer.
func (r *Run) DrainEvents() []Event {
	events := r.events
	r.events = nil
	return events
}
