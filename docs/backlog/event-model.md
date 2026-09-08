# Feature brief — Event model

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

*Structural, not a feature. Little or nothing user-visible ships from this item.*

---

## Already decided — do not reopen

Two parts of the event model are settled and in the tree. They were parked as "the event-model ADR" in the roadmap, which is why they read as open; they are not.

**ADR-002 (accepted 2026-07-22) — who emits.** Aggregate-owns-emission. Transition methods change state and record the event in one atomic in-memory step, so an unrecorded transition is structurally impossible. The aggregate buffers events in an `[]Event` field, does no I/O, and never knows the sink; a downstream drainer pulls to the sink, keeping the event-sink seam intact. Envelope: `Name`, `At`, `AggregateID`.

**ADR-006 (accepted 2026-07-23) — what an event is as a type.** `Event` carries `Payload any`, nil by default. Events with data hold a small typed struct defined next to the transition that emits it. `map[string]any` was considered and rejected: for a product whose output goes to an auditor, "what is in this record" must be answerable from a type declaration rather than by grepping emit sites.

## Problem

What ADR-006 deferred has since become live.

It parked cross-aggregate ordering with the note that the fix becomes real "whenever exec makes it real" — **exec is now real**. It parked request-level failures until a route existed to reject anything — **routes now exist**. Both deferrals were correctly reasoned and both have now expired.

Meanwhile four backlog items each assume something about events:

- **Filesystem API** requires one event to describe a batch of many files.
- **Run log directory** requires an engine-failure event that is not a run failure and belongs to no aggregate transition.
- **Persistence** asks what "append-only" means once events live in a store rather than a JSONL file, and requires a terminal event for runs interrupted by a restart.
- **gVisor** (deferred) would need the execution record to state which isolation backend ran the code.

Deciding these one at a time, inside items that are really about something else, is how the model becomes four half-decisions that disagree.

## Scope

**Cross-aggregate ordering.** ADR-006 records the problem precisely: append order totals events *within* an aggregate, but wall-clock `At` cannot totally order events across Sandbox and Run. The noted fix is a monotonic sequence stamped at drain-to-sink, or causation IDs. Small change to the sink; now unblocked and now needed.

**Request-level failures with no aggregate.** Malformed JSON, auth denial, and — from the run log brief — a failure to write evidence after a successful run. ADR-006 flagged that these may not be domain events at all. That question is now answerable.

**Versioning.** Deferred in ADR-006 because nothing consumed the wire format. Still nothing does, so this may stay deferred — but persistence changes the calculus, since a stored log outlives the code that wrote it. Decide explicitly rather than by omission.

**Typed name taxonomy.** Deferred at eleven names as "a rule looking for a problem." Worth recounting: the backlog adds batch-write, expiry-distinct-from-deletion, engine-failure and interrupted-run events. If the count has roughly doubled, the earlier reasoning may no longer hold.

## Not in scope

**Signing and tamper-evidence.** Tier-4 (G2). The model must not preclude it — ADR-006 already notes that `encoding/json` orders struct fields deterministically, which a signed bundle depends on — but building it here is premature.

**External sinks.** The event sink seam. Interface early, second implementation on demand.

**Replay.** G3, deferred.

**Backward compatibility with logs on disk today.** No outside users, so existing logs are disposable.

## Depends on

Exec and routes both existing — shipped. Those were the two triggers ADR-006 named.

## Trigger

Next ADR, ahead of event payloads.

## Open questions for the ADR

The scope above *is* the open set. Constraints already imposed by other briefs:

- One event must describe a batch of many files (filesystem API).
- Engine-failure events must be expressible without being aggregate lifecycle events (run log directory).
- Expiry and deletion must be distinguishable (TTL).
- "Append-only" must survive the move into persistent storage and stay readable by an auditor (persistence).
- The isolation backend in use may need to be in the execution record (gVisor).

## Note on splitting

Four questions is more moving parts than one ADR should carry. Ordering and versioning probably travel together as a serialisation decision; request-level failures is self-contained; the taxonomy may resolve to "still not yet." Expect two or three ADRs, and treat that as the rule working rather than a failure to scope.
