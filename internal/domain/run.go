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

// RunCommand is the payload on run.created. It records the command as the
// caller sent it, not the shell invocation the engine builds around it.
type RunCommand struct {
	Command string
}

// RunOutcome is the payload on run.succeeded and run.failed. It describes how
// a command ended. The hashes and byte counts commit to the output jhansi
// retained, which OutputTruncated reports may be less than the command wrote.
type RunOutcome struct {
	// ExitCode is nil when the engine could not run the command at all and
	// no exit was observed. Zero would read as a clean exit (ADR-023).
	ExitCode        *int
	DurationMS      int64
	StdoutBytes     int
	StderrBytes     int
	StdoutSHA256    string
	StderrSHA256    string
	OutputTruncated bool
}

// RunTimedOutOutcome is the payload on run.timed_out: the same outcome fields
// plus the timeout that was applied. Embedded rather than repeated, so it
// flattens to one object under encoding/json.
type RunTimedOutOutcome struct {
	RunOutcome
	TimeoutMS int64
}

// RunPreparationFailed is the payload on run.preparation_failed. It states why
// the run could not be prepared. A run carrying this event never entered
// RUNNING, so no code executed (ADR-025).
type RunPreparationFailed struct {
	Reason string
}

// RunLogsWriteFailed is the payload on run.logs_write_failed. Reason is the
// error text. It records that jhansi could not retain the run's output — not
// that the command failed, which is a separate outcome (ADR-024).
type RunLogsWriteFailed struct {
	Reason string
}

type Run struct {
	ID        string
	SandboxID string
	Command   string
	Status    RunStatus
	CreatedAt time.Time
	eventBuffer
}

// NewRun creates a run in QUEUED and records run.created carrying the command
// as the caller sent it (ADR-023).
func NewRun(id, sandboxID, command string) *Run {
	r := &Run{
		ID:          id,
		SandboxID:   sandboxID,
		Command:     command,
		Status:      RunQueued,
		CreatedAt:   time.Now().UTC(),
		eventBuffer: eventBuffer{aggregateID: id},
	}
	r.recordWith("run.created", r.CreatedAt, RunCommand{Command: command})
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

// MarkPreparationFailed moves the run to FAILED without it ever entering
// RUNNING, recording why preparation could not complete (ADR-025). Legal only
// from PREPARING: a run that reached FAILED with no run.running event in the
// spine never executed code.
func (r *Run) MarkPreparationFailed(reason string) error {
	if r.Status != RunPreparing {
		r.recordWith("run.preparation_failed_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunFailed,
		})
		return fmt.Errorf("run %s: cannot mark preparation failed from %s", r.ID, r.Status)
	}
	r.Status = RunFailed
	r.recordWith("run.preparation_failed", time.Now().UTC(), RunPreparationFailed{Reason: reason})
	return nil
}

// RecordLogsWriteFailed records that the run's output could not be written to
// disk. It changes no state: the command's own outcome stands, and what failed
// is jhansi's retention of what it produced (ADR-024).
func (r *Run) RecordLogsWriteFailed(reason string) {
	r.recordWith("run.logs_write_failed", time.Now().UTC(), RunLogsWriteFailed{Reason: reason})
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

// MarkSucceeded moves the run to SUCCEEDED and records the outcome (ADR-023).
// Legal only from RUNNING.
func (r *Run) MarkSucceeded(outcome RunOutcome) error {
	if r.Status != RunRunning {
		r.recordWith("run.succeeded_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunSucceeded,
		})
		return fmt.Errorf("run %s: cannot mark succeeded from %s", r.ID, r.Status)
	}
	r.Status = RunSucceeded
	r.recordWith("run.succeeded", time.Now().UTC(), outcome)
	return nil
}

// MarkFailed moves the run to FAILED and records the outcome (ADR-023).
// Legal only from RUNNING.
func (r *Run) MarkFailed(outcome RunOutcome) error {
	if r.Status != RunRunning {
		r.recordWith("run.failed_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunFailed,
		})
		return fmt.Errorf("run %s: cannot mark failed from %s", r.ID, r.Status)
	}
	r.Status = RunFailed
	r.recordWith("run.failed", time.Now().UTC(), outcome)
	return nil
}

// MarkTimedOut moves the run to TIMED_OUT and records the outcome together
// with the timeout that was applied (ADR-023). Legal only from RUNNING.
func (r *Run) MarkTimedOut(outcome RunTimedOutOutcome) error {
	if r.Status != RunRunning {
		r.recordWith("run.timed_out_rejected", time.Now().UTC(), RunTransitionRejected{
			From: r.Status,
			To:   RunTimedOut,
		})
		return fmt.Errorf("run %s: cannot mark timed_out from %s", r.ID, r.Status)
	}
	r.Status = RunTimedOut
	r.recordWith("run.timed_out", time.Now().UTC(), outcome)
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
