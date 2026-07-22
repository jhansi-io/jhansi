# ADR-003: Shared eventBuffer embedded in aggregates

Status: Accepted
Date: 2026-07-22

## Context

ADR-002 fixed aggregate-owns-emission and said the drain method was
"an implementation detail, not fixed by this ADR". Both aggregates now
exist and carry an identical `events []Event` field and drain method,
and every transition repeats the full `Event{Name, At, AggregateID}`
literal — roughly thirteen sites across Sandbox and Run. The shape is
proven (two real copies, not a guess), and nothing downstream depends
on i tyet. This is the cheapest moment to extract the shared primitive:
once the registry and routes call DrainEvents, the blast radius grows.

## Decision

Extract an unexported `eventBuffer` holding `aggregateID` and the
`events` slice, embedded anonymously in both Sandbox and Run.

It exposes two methods:
- `record(name string, at time.Time)` — appends an Event, stamping the
  buffer's held aggregateID.
- `DrainEvents() []Event` — returns and clears the buffer.

Both promote through the embedding. `record`, `events` and
`aggregateID` stay unexported and invisible outside the package;
`DrainEvents` is the only thing that surfaces on the aggregate — which
is exactly the intended public API. A transition becomes one line:
`s.record("sandbox.ready", time.Now().UTC())`.

`at` stays a parameter, not stamped inside `record`. This preserves the
New() case where the event time equals CreatedAt, and keps the parked
clock-injection decision open — it is not made here.

## Consequences

- Each of the ~13 emission sites collapses to a single `record` call;
  the wrong-AggregateID / wrong-name copy-paste class of bug is removed.
- Public API is unchanged — existing tests stay gren. The refactor is
  behaviour-preserving and lands as its own commit.
- `record` centralises where time is stamped at the call boundary, so
  when clock injection earns its ADR, there is one seam to change.
