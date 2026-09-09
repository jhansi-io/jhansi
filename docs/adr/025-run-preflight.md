# ADR-025: Preflight before a run executes

Status: Accepted
Date: 2026-09-09

## Context

ADR-024 puts a run's retained output on disk. It does not say what happens if
the disk cannot take it.

The failures that actually occur — a full disk, a read-only remount, exhausted
inodes, changed permissions — are all true before a run as well as after. That
makes them catchable at a point where refusing costs nothing, because no code
has executed yet. Refusing after execution is not available: the run cannot be
un-executed.

`Exec` today mints a run, drives it to RUNNING, and calls the engine. There is
no point in that sequence at which jhansi can decline to execute.

## Decision

### 1. Creating the run directory is the preflight

    MkdirAll(<data-dir>/runs/<run_id>, 0700)

Its success or failure is the check. It fails on precisely the conditions
above, and it leaves behind the thing the run needs anyway.

No separate probe file is written. A probe proves that a probe could be
written; it adds a failure mode of its own and answers nothing the directory
creation has not already answered.

### 2. The preflight runs between `MarkPreparing` and `MarkRunning`

That is what PREPARING is for: the work done to make a run possible, before it
is running. Placing the check earlier — before the sandbox is claimed — would
leave a refusal with no run to attach it to, and so no record.

### 3. The preflight owns directory creation

ADR-024's write finds the directory already present and never creates it. One
owner, so the mode and location are decided in one place.

### 4. On failure the run does not enter RUNNING

A new domain method, legal only from PREPARING, moves the run to FAILED and
emits `run.preparation_failed` carrying the reason.

`MarkFailed` is not reused: it is legal only from RUNNING, and widening it
would let a run reach FAILED from PREPARING through the event that means the
command ended.

### 5. The status is FAILED, not a new terminal state

The run did fail. A fourth terminal leaf would make every reader of the status
field learn a new word for no gain, and CANCELLED is reserved for policy and
approval denial rather than infra faults.

What separates this from a run that executed and failed is structural: a run
that reached FAILED with no `run.running` event in the spine never executed
code. That is a property of the event history, not a field an auditor has to
interpret.

This matters because a nil `ExitCode` under ADR-023 already carries two
meanings — an infra fault and a timeout. Making it carry a third would put the
distinction beyond recovery.

### 6. The sandbox returns to idle

The sandbox is intact. What is broken is the data directory. Marking the
sandbox ERROR would retire a healthy sandbox for a condition that has nothing
to do with it.

### 7. The route returns a server error

The condition belongs to jhansi and its host, not to the caller's request.

## Consequences

The default is to refuse. Whether an operator may opt out — execute without
retention — stays deferred. `auditd` and SQL Server Audit both made this
configurable rather than choosing, and the choice needs a real user behind it:
default-open suits the beachhead, default-closed suits a regulated buyer.

A mid-run write failure remains possible and is handled as ADR-024 describes.
The preflight narrows that window; it does not close it.

Every refused run still costs a run ID and a pair of events. That is the
intended shape — a refusal is an evidenced outcome, not a silent rejection.
