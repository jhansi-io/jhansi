# ADR-002: Aggregate-owns-emission for domain events

Status: Accepted
Date: 2026-07-22

## Context

The engine is evidence-native: structured events must exist from the
first transition, never retrofitted later as an "audit feature".
A state transition is the only place that knows state changed. This
forces a fork in where event emission lives:

- **Aggregate-owns-emission**: transition methods record the event inline.
- **Separate recorder**: transitions change state; a separate component
  records events in a second step.

The run aggregate (seven transitions) is next to build, and Sandbox
already has five. The decision cannot be deferred — every pure transition
written now becomes retrofit debt if the answer turns out to be
aggregate-owns.

## Decision

Transitions record events inline. State change and emissions are one
atomic in-memory step — it is structurally impossible to transition
without producing the event.

The aggregate **buffers** events in memory (an `[]Event` field). It does
no I/O and never knows the sink. A downstream drainer pulls buffered
events to the sink later, keeping the event-sink seam intact.

Every event carries a minimal, irreducible envelope:
- Name (string): what happened, e.g. "sandbox.created".
- At (time.Time): when it happened.
- AggregateID (string): which aggregate emitted it.

## Consequences

- Sandbox's five existing transitions are retrofitted to emit — its own
  commit, after this ADR.
- Run is built emitting from the start.
- The event *schema* (payloads, a typed name taxonomy, versioning,
  causation) is out of scope here — deferred to the event-model ADR. The
  three-field envelope is enough to compile the buffer and emit today.
- Aggregates gain a drain method (return + clear the buffer) — an
  implementation detail, not fixed by this ADR.
