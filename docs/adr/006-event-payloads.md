# ADR-006: Event payloads

Status: Accepted
Date: 2026-07-23

## Context

ADR-002 locked a minimal three-field envelope (Name, At, AggregateID)
and deferred the event schema to a later ADR. Two things now force the
smallest part of that schema.

Failures must be audited, not only successes — a rejected transition is
exactly the row an auditor reads. But `sandbox.delete_rejected` with no
from-state and no attempted action tells them nothing. Auditing failures
is impossible on a three-field envelope.

The event sink is next, and its default is an append-only log. Deciding
what an event *contains* after buulding the thing that writes events
down is retrofit in miniature.

## Decision

`Event` gains one field: `Payload any`, defaulting to nil.

Events that carry data hold a small struct defined next to the
transition that emits it. Events that carry none leave it nil.

Rejected: `map[string]any`. It adds no types and no safety — a year on,
the only way to learn what a given event carries is to grep its emit
sites. For a product whose output is handed to an auditor, "what is in
this record" must be answerable from a type declaration.

Deliberately out of scope, each deferred with a reason:

- **Versioning** — nothing consumes the wire format yet. Versioning an
  unread schema is ceremony.
- **Typed name taxonomy** — eleven names exist. A taxonomy for eleven
  strings is a rule looking for a problem.
- **Causation IDs and cross-aggregate ordering** — Sandbox and Run do
  not co-occur until exec. See below.
- **Request-level failures with no aggregate** (malformed JSON, auth
  denial) — no HTTP layer exists to reject anything, and these may not
  be domain events at all. Decide when the route does.

## Consequences

- One field, no behaviour change. The eleven existing emit sites stay
  as they are and their tests pass unmodified.
- Failure events on rejected transitions become possible — its own
  commit, immediately after this one.
- The sink is unblocked: the envelope is settled enough to write down.
- Payload is `any`, so the sink must marshall it. `encoding/json` sorts
  struct fields deterministically, which the eventual signed bundle
  depends on. Not exercised yet.
- Cross-aggregate ordering stays open. The envelope orders events
  *within* an aggregate by append order, but At (wall clock) cannot
  totally order events across Sandbox abd Run. The fix — a monotonic
  sequence stamped at drain-to-sink, or causation IDs — is a small
  change to the sink whenever exec makes it real. Known, accepted,
  not built.
