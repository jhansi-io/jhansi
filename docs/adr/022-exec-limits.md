# ADR-022: Per-exec timeout and captured-output cap

Status: Accepted
Date: 2026-09-04

## Context

`DockerEngine` runs a command and waits for the container to exit. Nothing
bounds either how long that wait lasts or how much output it reads back.

A command that never terminates holds its sandbox forever. ADR-018 makes a
sandbox with a live run reject further execs with 409, so one hung container
takes the sandbox out of service permanently. It also holds the inbound HTTP
request open, which is why ADR-016 left `WriteTimeout` unset.

A command that writes without bound is read into memory in full by
`fetchLogs`. Output is a string on `ExecResult`, so the ceiling on jhansi's
memory is whatever the untrusted code decides to print.

Both are limits on a single command, not on a sandbox. Within one sandbox
`pip install` and `print(2+2)` have nothing in common: a limit sized for the
first gives the second no protection at all. This rules out the sandbox
aggregate as their home.

## Decision

### 1. The limits are fields on `ExecRequest`, defaulted by the service.

`ExecRequest` gains `Timeout time.Duration` and `MaxOutputBytes int64`. 

The service fills both on every call, exactly as it fills `WorkDir` 
(ADR-021). The exec request body gains an optional `timeout_seconds`; when
present the service uses it, otherwise its own default. `MaxOutputBytes` has
no request-body field yet — no caller has needed to vary it.

A `timeout_seconds` that is absent uses thedefault. A value that is present
must be positive; zero or negative is rejected with 400. Zero in particular
reads as "n timeout" to the caller, which is the thing this ADR exists to
refuse. Values above the default are accepted — the ceiling that would cap
them is deferred with the auth seam.

The engine applies what it is told. It does not hold the defaults.

The alternative was constructor arguments on `DockerEngine`, which is simpler
today and was rejected on where the values must eventually be read together.
A caller-supplied timeout is safe now because the caller is the operator's own
application: the untrusted party is the code inside the sandbox, and that code 
cannot reach the API. When auth seam lands and the caller may be a
different party, an operator ceiling clamping the callers's request becomes
necessary. That clamp can only happen somewhere that sees both values, which
is the service. Siting the defaults there now means the clamp has a home later
with nothing to move.

### 2. Timeout is enforced by killing the container.

`DockerEngine.Exec` wraps the wait in `context.WithTimeout(ctx, req.Timeout)`.
On expiry it issues `POST /containers/{id}/kill`, then reads the logs, the
returns `ExecResult{TimedOut: true}`.

The context alone cancels only jhansi's HTTP request to the daemon. The
container keeps running, keeps its workfir mounted, and keeps the sandbox
occupied. The kill is what makes the limit real. This is ADR-020's reason for
choosing the daemon socket over CLI shell-out, cashing out: the socket gives
direct lifecycle control over a container jhansi did not fork.

Kill and remove stay separate operations. A killed container still exists, and
its logs are still readable — which is required, because a timed-out runs's
partial output is evidence. Docker's `force` removal would kill and delete in
one call and destroy that output before it is read.

### 3. Capture is bounded to the tail of the output.

`fetchLogs` retains the last `req.MaxOutputBytes` of each stream, discarding
earlier bytes as it reads. `ExecResult` gains `OutputTruncated bool`, set when
anything was discarded.

The cap bounds what jhansi captures, not what the command writes.

The tail is kept rather than the head because a failing command explains
itself at the end. A build that emits megabytes of compiler output and then
dies on one error must not have that error discarded — a head-bounded capture
returns the chatter and drops the reason, which is close to useless to the
caller. The head is the cheaper loss: the command itself is already known from
the request and recorded on `run.started`.

Keeping both ends was rejected. Signalling the omitted middle requires
inserting a marker line into the captured output, and that output is hashed
and presented as what the command produced. Nothing synthetic may enter
content that jhansi commits to. Tail-only keeps the claim clean: every byte
captured came from the command.

Silently discarding output would put a claim in the evidence spine that is not
true of the execution. Truncation is a fact about the run, and the caller that
cannot see it will misread an incomplete capture as a complete one — parsing a 
cut-off document as malformed data and repairing the wrong thing.
`OutputTruncated` is what distinguishes those cases.

`OutputTruncated` is the first `ExecResult` field describing jhansi's handling
of a command rather than the command's outcome. This is deliberate: the
result is whatthe caller learns about the execution, and how it was bounded
is part of that.

The cap applies per stream on the demultiplexed content, not to the raw
framed stream. Docker's log endpoint returns 8-byte frame headers interleaved
with payload; bounding the raw stream would cut mid-frame. The demux reads
frames whole and the retained window holds payload bytes only.

### 4. Defaults come from server flags

`-exec-timeout` (default `5m`) and `max-outout-bytes` (default `1048576`) are
added to the server subcommand, carried on `Config`, and passed to
`service.New`.

Five minutes covers the slow tail of ordinary agent work — dependency
installation, moderate builds — while still bounding a hung container. Thirty
seconds was rejected: a limit that fires during normal use gets raised
globally by the first user who trips it, and then protects nothing. Callers
needing longer pass `timeout_seconds`.

If callers routinely override the default, the default is wrong. That is a 
signal to revisit, not a reason to guess higher now.

## Consequences

The outcome mapping is already decided and unchanged. ADR-017 maps
`ExecResult.TimedOut` to run `TIMED_OUT`; `service.Exec`'s four-way switch
handles it today. This ADR supplies a `TimedOut` that was previously only
reachable from `StubEngine`.

A hung container now costs one exec-timeout, after which the sandbox returns
to service. The 409-busy window is bounded.

`removeContainer` continues to run from its existing `defer` with
`context.WithoutCancel`. The timeout path is a new way for `Exec` to return,
not a new cleanup path: the deferred removal already fires on it and already
survives the cancellation that caused it. No second removal call is needed.

`OutputTruncated` has no event payload yet — the payload ADR carries no fields
at all today. When it lands, truncation is a fact it must include, and the
payload ADR must state that an output hash commits to what jhansi captured,
not to what the command wrote.

The cap bounds retained output, not memory during the read. `demuxLogs`
accumulates the full stream and trims afterwards, so a command writing
gigabytes still forces jhansi to hold them before discarding all but the tail.
The sandbox is protected — its 409-busy window ends — but the process is not.
Bounding memory properly needs a ring buffer filled as frames arrive, which is
a change to how logs are read rather than to whatis kept, and is deferred.

`WriteTimeout` becomes settable. ADR-016 left it unset because an exec could
run without bound; that reason no longer holds. Choosing the value is a
decision about the HTTP server rather than about limits, and gets its own ADR.

## Deferred

**Operator ceiling on caller-supplied timeouts.** Needs a caller identity to
hold to a ceiling, and the auth seam does not exist. The service already holds
the default, so the clamp lands there without moving anything.

**Caller-supplied output cap.** Additive — `OutputTruncated` stays true either way.
Blocked on deciding how a synthetic marker can exist inside hashed content, or
on carrying the omitted-byte count as a separate field instead.

**CPU and memory limits.** A different mechanism: container create options,
not values the service passes per exec. Nothing forces them now.

**`WriteTimeout`.** Unblocked here, decided elsewhere.

**Async exec.** Packed with triggers recorded separately. A timeout is a
ceiling on how long a command may run; async is about how its result is
delivered. Async does not remove the need for this ADR.

**Streaming capture.** Retaining the tail as frames arrive, rather than
accumulating and trimming, is what actually bounds jhansi's memory.
`OutputTruncated` and the retained bytes are identical either way, so this is
an internal change with no contract impact. Blocked on nothing; waiting for a 
reason.
