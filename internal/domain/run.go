package domain

import (
	"fmt"
	"time"
)

// RunStatus is the lifecycle state of a single execution.
type RunStatus string

const (
	RunQueued    RunStatus = "QUEUED"
	RunPreparing RunStatus = "PREPARING"
	RunRunning   RunStatus = "RUNNING"
	RunSucceeded RunStatus = "SUCCEEDED"
	RunFailed    RunStatus = "FAILED"
	RunTimedOut  RunStatus = "TIMED_OUT"
	RunCancelled RunStatus = "CANCELLED"
)

type RunTransitionRejected struct {
	From RunStatus
	To   RunStatus
}

type Run struct {
	ID        string
	SandboxID string
	Status    RunStatus
	CreatedAt time.Time
	eventBuffer
}

func NewRun(id, sandboxID string) *Run {
	r := &Run{
		ID:          id,
		SandboxID:   sandboxID,
		Status:      RunQueued,
		CreatedAt:   time.Now().UTC(),
		eventBuffer: eventBuffer{aggregateID: id},
	}
	r.record("run.created", r.CreatedAt)
	return r
}

// MarkPreparing moves the run to PREPARING. Legal only from QUEUED.
func (r *Run) MarkPreparing() error {
	if r.Status != RunQueued {
		r.recordWith("run.preparing_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunPreparing,
		})
		return fmt.Errorf("run %s: cannot mark preparing from %s", r.ID, r.Status)
	}
	r.Status = RunPreparing
	r.record("run.preparing", time.Now().UTC())
	return nil
}

// MarkRunning moves the run to Running. Legal only from PREPARING.
func (r *Run) MarkRunning() error {
	if r.Status != RunPreparing {
		r.recordWith("run.running_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunRunning,
		})
		return fmt.Errorf("run %s: cannot mark running from %s", r.ID, r.Status)
	}
	r.Status = RunRunning
	r.record("run.running", time.Now().UTC())
	return nil
}

// MarkSucceeded moves the run to Succeeded. Legal only from RUNNING.
func (r *Run) MarkSucceeded() error {
	if r.Status != RunRunning {
		r.recordWith("run.succeeded_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunSucceeded,
		})
		return fmt.Errorf("run %s: cannot mark succeeded from %s", r.ID, r.Status)
	}
	r.Status = RunSucceeded
	r.record("run.succeeded", time.Now().UTC())
	return nil
}

// MarkFailed moves the run to FAILED. Legal only from RUNNING.
func (r *Run) MarkFailed() error {
	if r.Status != RunRunning {
		r.recordWith("run.failed_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunFailed,
		})
		return fmt.Errorf("run %s: cannot mark failed from %s", r.ID, r.Status)
	}
	r.Status = RunFailed
	r.record("run.failed", time.Now().UTC())
	return nil
}

// MarkTimedOut moves the run to TIMED_OUT. Legal only from RUNNING.
func (r *Run) MarkTimedOut() error {
	if r.Status != RunRunning {
		r.recordWith("run.timed_out_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunTimedOut,
		})
		return fmt.Errorf("run %s: cannot mark timed_out from %s", r.ID, r.Status)
	}
	r.Status = RunTimedOut
	r.record("run.timed_out", time.Now().UTC())
	return nil
}

// MarkCancelled moves the run to CANCELLED (explicitly stopped).
// Legal from any non-terminal state: QUEUED, PREPARING or RUNNING.
func (r *Run) MarkCancelled() error {
	switch r.Status {
	case RunQueued, RunPreparing, RunRunning:
		r.Status = RunCancelled
		r.record("run.cancelled", time.Now().UTC())
		return nil
	default:
		r.recordWith("run.cancelled_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunCancelled,
		})
		return fmt.Errorf("run %s: cannot mark cancelled from %s", r.ID, r.Status)
	}
}
