# ADR-023: Run event payloads

Status: Accepted
Date: 2026-09-09

## Context

Run events carry `Name`, `At` and `AggregateID` and nothing else. The spine
records that a run was created, prepared, started and reached a terminal
state. It does not record what command ran, what it returned, or which bytes
it produced. That is a timeline, not evidence.

ADR-006 settled the mechanism: `Payload any`, nil by default, a small typed
struct defined next to the transition that emits it. Nothing in that ADR is
reopened here. This ADR supplies the structs for the run lifecycle.

ADR-022 named this ADR as the place where truncation must land, and stated the
constraint it must honour: an output hash commits to what jhansi captured, not
to what the command wrote.

## Decision

### 1. The command is recorded on `run.created`, as the caller sent it

`NewRun` gains a command argument and emits `run.created` carrying
`RunCommand{Command string}`.

Recording at creation rather than at `run.running` puts the intended command
in the spine before any container starts. A run that fails during preparation
still has a record of what was attempted; recording at `run.running` would
leave those runs describing nothing.

The command is the caller's string, not the `sh -c` invocation the engine
builds. The shell wrapper is jhansi's own invocation detail, identical for
every run and already stated in ADR-020. Recording the assembled form would
make historical records shift meaning if that wrapper ever changed, and
answers a question nobody asks of an execution record.

### 2. `run.succeeded` and `run.failed` share one outcome payload; `run.timed_out` embeds it

    RunOutcome:  ExitCode (*int), DurationMS, StdoutBytes, StderrBytes,
                 StdoutSHA256, StderrSHA256, OutputTruncated
    RunTimedOutOutcome: RunOutcome (embedded), TimeoutMS

ADR-006's pattern is a struct per transition, and three near-identical structs
is the literal reading of it. It is rejected here: the three events describe
one thing — how a command ended — with three outcomes. Duplicating seven
fields across three declarations means a field added later must be added three
times or silently diverge, and divergence between `run.succeeded` and
`run.failed` is exactly the kind of drift an evidence record cannot absorb.

The cost is accepted knowingly. Sharing couples the three events to one type,
so a field meaningful only to `run.failed` cannot be added without appearing
on the other two. Nothing needs such a field today; when one arrives, that is
the trigger to split.

`run.timed_out` carries `RunTimedOutOutcome`, which embeds `RunOutcome` rather
than repeating it. The name avoids the existing `RunTimedOut` status constant. Embedded structs flatten under
`encoding/json`, so a reader sees one flat object with one additional field
rather than a nested one.

A single struct with an optional timeout field was rejected. On two of the
three events that field would be a zero value carrying no meaning, and a
reader could not distinguish "no timeout applied" from "a timeout of zero".

`ExitCode` is a pointer, nil when no exit was observed. Two paths reach a
terminal event without one: an infra fault, where the engine never ran the
command, and a timeout, where the container was killed rather than seen
exiting — `waitContainer` returns an error on the deadline, so `ExecResult`
carries its zero value. Recording `0` on either would put a clean exit in the
spine for a run that never had one. That is the precise failure this ADR
exists to prevent, and a sentinel such as `-1` only moves the problem to a
reader who does not know the convention.

### 3. Hashes and byte counts are computed in the service, over retained output

`service.Exec` maps `ExecResult` into the payload: it takes SHA-256 over the
retained `Stdout` and `Stderr` strings, records their lengths in bytes, and
passes the results to the domain. Raw output never crosses into the domain
package and never enters the spine.

Hashing after capture rather than during it is the only honest option under
ADR-022, which reads the stream and trims to the tail afterwards. Hashing as
bytes arrive would commit to bytes that are subsequently discarded, so the
hash would not describe what jhansi holds. When streaming capture lands with
the run log directory, the computation moves but the claim does not change.

An empty stream hashes to the SHA-256 of the empty string. That is a
commitment that nothing was retained, which is itself a fact worth recording,
and it keeps every outcome event uniformly shaped.

The alternative was a domain constructor taking the raw streams and hashing
them, which would stop a future call site passing an incorrect hash. It is
rejected because it requires handing raw stdout and stderr into the domain
package. Raw output not entering the spine is a standing rule, and the
cheapest way to keep it true is for the domain never to hold the bytes.

`OutputTruncated` is carried through from `ExecResult` unchanged and sits
beside the hashes, so the record never overstates what it covers.

### 4. Durations are recorded in milliseconds, with the unit in the field name

`DurationMS` and `TimeoutMS` are `int64` milliseconds.

`FileSink` marshals events with `encoding/json`, which renders a
`time.Duration` as raw nanoseconds. An integer with no unit in a record
intended for a reader who did not write the code is a defect. Milliseconds is
proportionate: container startup dominates any execution jhansi runs, so
sub-millisecond precision describes nothing real.

### 5. Duration is measured in the service around the engine call

`ExecResult` carries no duration. The service already brackets the call to
`SandboxEngine.Exec` and already owns the timeout default, so it measures wall
time across that call and puts the result on the payload. The engine keeps its
primitives-only contract and gains no new field.

## Consequences

`NewRun`, `MarkSucceeded`, `MarkFailed` and `MarkTimedOut` change signature.
`service.Exec` and the domain tests are the call sites.

`Run` gains a `Command` field. The constructor takes the command in order to
record it, and a parameter consumed once and discarded reads as an oversight;
an aggregate that cannot state its own command also blocks anything that later
wants to expose it. The field is not on the API response — that is a separate
choice.

`MarkPreparing`, `MarkRunning` and `MarkCancelled` keep nil payloads — no
content distinguishes those moments beyond the fact that they occurred.
Rejection events keep `RunTransitionRejected` unchanged.

Sandbox events are untouched.

The spine now commits to output that lives only in the caller's HTTP response.
The bytes themselves are not stored anywhere until the run log directory item
lands, so a hash is verifiable only by whoever already holds the output. This
is a real limit on the evidence claim and must not be described as more than
it is.

The measured duration spans jhansi's call to the engine, which includes
container creation and teardown, not solely the command's own execution. It is
the interval jhansi can observe and attest to.

## Deferred

**Sandbox event payloads.** A separate and smaller question; folding it in
would blur what this ADR decides.

**Versioning of payload structs.** Nothing consumes the wire format yet. The
event model item decides it, and persistence changes the calculus.

**Isolation backend in the execution record.** Flagged by the gVisor brief.
There is one backend today, so the field would record a constant.

**Splitting `RunOutcome`.** Triggered by the first field meaningful to one
terminal event and not the others.
