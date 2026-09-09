package domain

import "testing"

func TestNewRun(t *testing.T) {
	r := NewRun("run_1", "sb_1", "echo hi")
	if got := r.DrainEvents(); len(got) != 1 || got[0].Name != "run.created" {
		t.Fatalf("expected run.created, got %v", got)
	}

	if r.ID != "run_1" {
		t.Errorf("ID = %q, want %q", r.ID, "run_1")
	}

	if r.Status != RunQueued {
		t.Errorf("Status = %q, want %q", r.Status, RunQueued)
	}
}

func TestRunHappyChain(t *testing.T) {
	r := NewRun("run_1", "sb_1", "echo hi")

	if err := r.MarkPreparing(); err != nil {
		t.Fatalf("MarkPreparing: %v", err)
	}

	if err := r.MarkRunning(); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}

	want := []string{"run.created", "run.preparing", "run.running"}
	got := r.DrainEvents()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %v", len(got), len(want), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("event[%d] = %q, want %q", i, got[i].Name, name)
		}
	}
}

func TestMarkSucceeded(t *testing.T) {
	// happy: RUNNING → SUCCEEDED
	r := NewRun("run_1", "sb_1", "echo hi")
	if err := r.MarkPreparing(); err != nil {
		t.Fatalf("setup MarkPreparing: %v", err)
	}

	if err := r.MarkRunning(); err != nil {
		t.Fatalf("setup MarkRunning: %v", err)
	}
	r.DrainEvents() // clear setup events

	if err := r.MarkSucceeded(RunOutcome{}); err != nil {
		t.Fatalf("MarkSucceeded: unexpected error %v", err)
	}
	if r.Status != RunSucceeded {
		t.Errorf("Status = %s, want SUCCEEDED", r.Status)
	}
	got := r.DrainEvents()
	if len(got) != 1 || got[0].Name != "run.succeeded" {
		t.Errorf("events = %v, want one run.succeeded", got)
	}

	// reject: from QUEUED, emits nothing
	q := NewRun("run_2", "sb_1", "echo hi")
	q.DrainEvents() // clear run.created
	if err := q.MarkSucceeded(RunOutcome{}); err == nil {
		t.Errorf("MarkSucceeded from QUEUED: want error, got nil")
	}
	if got := q.DrainEvents(); len(got) != 1 || got[0].Name != "run.succeeded_rejected" {
		t.Errorf("rejected transition emitted %v, want one run.succeeded_rejected", got)
	}
}

func TestMarkFailed(t *testing.T) {
	// RUNNING → FAILED
	r := NewRun("run_1", "sb_1", "echo hi")
	if err := r.MarkPreparing(); err != nil {
		t.Fatalf("setup MarkPreparing: %v", err)
	}
	if err := r.MarkRunning(); err != nil {
		t.Fatalf("setup MarkRunning: %v", err)
	}
	r.DrainEvents() // clear setup events
	if err := r.MarkFailed(RunOutcome{}); err != nil {
		t.Fatalf("MarkFailed: unexpected error %v", err)
	}
	if r.Status != RunFailed {
		t.Errorf("Status = %s, want FAILED", r.Status)
	}
	got := r.DrainEvents()
	if len(got) != 1 || got[0].Name != "run.failed" {
		t.Errorf("events = %v, want one run.failed", got)
	}
	// reject: from QUEUED, emits nothing
	q := NewRun("run_2", "sb_1", "echo hi")
	q.DrainEvents() // clear run.created
	if err := q.MarkFailed(RunOutcome{}); err == nil {
		t.Errorf("MarkFailed from QUEUED: want error, got nil")
	}
	if got := q.DrainEvents(); len(got) != 1 || got[0].Name != "run.failed_rejected" {
		t.Errorf("rejected transition emitted %v, want one run.failed_rejected", got)
	}
}

func TestMarkTimedOut(t *testing.T) {
	// RUNNING → TIMED_OUT
	r := NewRun("run_1", "sb_1", "echo hi")
	if err := r.MarkPreparing(); err != nil {
		t.Fatalf("setup MarkPreparing: %v", err)
	}
	if err := r.MarkRunning(); err != nil {
		t.Fatalf("setup MarkRunning: %v", err)
	}
	r.DrainEvents() //clear setup events

	if err := r.MarkTimedOut(RunTimedOutOutcome{}); err != nil {
		t.Fatalf("MarkTimedOut: unexpected error: %v", err)
	}

	if r.Status != RunTimedOut {
		t.Errorf("Status: %s, want TIMED_OUT", r.Status)
	}
	got := r.DrainEvents()
	if len(got) != 1 || got[0].Name != "run.timed_out" {
		t.Errorf("events = %v, want one run.timed_out", got)
	}

	// reject: from QUEUED, emits nothing
	q := NewRun("run_2", "sb_1", "echo hi")
	q.DrainEvents() // clear run.created
	if err := q.MarkTimedOut(RunTimedOutOutcome{}); err == nil {
		t.Errorf("MarkTimedOut from QUEUED: want error, got nil")
	}
	if got := q.DrainEvents(); len(got) != 1 || got[0].Name != "run.timed_out_rejected" {
		t.Errorf("rejected transition emitted %v, want one run.timed_out_rejected", got)
	}
}

func TestMarkCancelled(t *testing.T) {
	// permissive: legal from QUEUED, PREPARING, RUNNING
	froms := []struct {
		name string
		to   func(*Run) // drive a fresh run to the from-state
	}{
		{"from queued", func(r *Run) {}},
		{"from preparing", func(r *Run) { r.MarkPreparing() }},
		{"from running", func(r *Run) { r.MarkPreparing(); r.MarkRunning() }},
	}

	for _, tt := range froms {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRun("run_1", "sb_1", "echo hi")
			tt.to(r)
			r.DrainEvents() // clear setup events

			if err := r.MarkCancelled(); err != nil {
				t.Fatalf("MarkCancelled %s: unexpected error %v", tt.name, err)
			}
			if r.Status != RunCancelled {
				t.Errorf("Status = %s, want CANCELLED", r.Status)
			}
			got := r.DrainEvents()
			if len(got) != 1 || got[0].Name != "run.cancelled" {
				t.Errorf("events = %v, want one run.cancelled", got)
			}
		})
	}
	// reject: from a terminal state, emits nothing
	q := NewRun("run_2", "sb_1", "echo hi")
	q.MarkPreparing()
	q.MarkRunning()
	q.MarkSucceeded(RunOutcome{})
	q.DrainEvents()
	if err := q.MarkCancelled(); err == nil {
		t.Errorf("MarkCancelled from SUCCEEDED: want error, got nil")
	}
	if got := q.DrainEvents(); len(got) != 1 || got[0].Name != "run.cancelled_rejected" {
		t.Errorf("rejected transition emitted %v, want one run.cancelled_rejected", got)
	}
}

func TestRunRejectionPayload(t *testing.T) {
	r := NewRun("run_1", "sb_1", "echo hi")
	if err := r.MarkPreparing(); err != nil {
		t.Fatalf("setup MarkPreparing: %v", err)
	}
	r.DrainEvents() // clear setup events

	if err := r.MarkPreparing(); err == nil {
		t.Fatalf("Mark Preparing from PREPARING: want error, got nil")
	}
	events := r.DrainEvents()
	if len(events) != 1 || events[0].Name != "run.preparing_rejected" {
		t.Fatalf("events = %v, want one run.preparing_rejected", events)
	}
	got, ok := events[0].Payload.(RunTransitionRejected)
	if !ok {
		t.Fatalf("payload = %T, want RunTransitionRejected", events[0].Payload)
	}
	if got.From != RunPreparing || got.To != RunPreparing {
		t.Errorf("payload = %+v, want {RunPreparing}", got)
	}
}

func TestMarkRunningRejected(t *testing.T) {
	r := NewRun("run_1", "sb_1", "echo hi")
	r.DrainEvents() // clear run.created

	if err := r.MarkRunning(); err == nil {
		t.Fatal("MarkRunning from QUEUED: want error, got nil")
	}
	if r.Status != RunQueued {
		t.Errorf("Status = %s, want QUEUED", r.Status)
	}
	if got := r.DrainEvents(); len(got) != 1 || got[0].Name != "run.running_rejected" {
		t.Errorf("events = %v, want one run.running_rejected", got)
	}
}

func TestRunCreatedCarriesCommand(t *testing.T) {
	r := NewRun("run_1", "sb_1", "python train.py")

	events := r.DrainEvents()
	if len(events) != 1 || events[0].Name != "run.created" {
		t.Fatalf("events = %v, want one run.created", events)
	}
	got, ok := events[0].Payload.(RunCommand)
	if !ok {
		t.Fatalf("payload = %T, want RunCommand", events[0].Payload)
	}
	if got.Command != "python train.py" {
		t.Errorf("Command = %q, want %q", got.Command, "python train.py")
	}
}

func TestTerminalEventCarriesOutcome(t *testing.T) {
	r := NewRun("run_1", "sb_1", "echo hi")
	r.MarkPreparing()
	r.MarkRunning()
	r.DrainEvents() // clear setup events

	exit := 0
	want := RunOutcome{
		ExitCode:        &exit,
		DurationMS:      1200,
		StdoutBytes:     3,
		StderrBytes:     0,
		StdoutSHA256:    "abc",
		StderrSHA256:    "def",
		OutputTruncated: true,
	}
	if err := r.MarkSucceeded(want); err != nil {
		t.Fatalf("MarkSucceeded: %v", err)
	}

	events := r.DrainEvents()
	got, ok := events[0].Payload.(RunOutcome)
	if !ok {
		t.Fatalf("payload = %T, want RunOutcome", events[0].Payload)
	}
	if got != want {
		t.Errorf("payload = %+v, want %+v", got, want)
	}
}

// A timed-out run has no exit code: the container was killed rather than
// observed exiting, so nil is the only truthful value (ADR-023).
func TestTimedOutEventCarriesTimeoutAndNoExitCode(t *testing.T) {
	r := NewRun("run_1", "sb_1", "sleep 999")
	r.MarkPreparing()
	r.MarkRunning()
	r.DrainEvents() // clear setup events

	if err := r.MarkTimedOut(RunTimedOutOutcome{
		RunOutcome: RunOutcome{DurationMS: 5000},
		TimeoutMS:  5000,
	}); err != nil {
		t.Fatalf("MarkTimedOut: %v", err)
	}

	events := r.DrainEvents()
	got, ok := events[0].Payload.(RunTimedOutOutcome)
	if !ok {
		t.Fatalf("payload = %T, want RunTimedOutOutcome", events[0].Payload)
	}
	if got.TimeoutMS != 5000 {
		t.Errorf("TimeoutMS = %d, want 5000", got.TimeoutMS)
	}
	if got.ExitCode != nil {
		t.Errorf("ExitCode = %d, want nil", *got.ExitCode)
	}
}
