# Feature brief — Sandbox TTL and reaper

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

A sandbox lives until someone deletes it. If a caller creates one and crashes, or forgets, or simply walks away, it stays — holding a workdir on disk, and once warm containers exist, a container too.

A long-running server accumulates abandoned sandboxes until something runs out. Nothing complains, so nobody notices until it is already a problem. This is the item that makes jhansi safe to leave running.

## Scope

### Idle TTL (the default)

A sandbox is reaped once it has been in `ready` longer than its TTL.

`ready` covers both cases that matter — a sandbox that has finished a run and returned to it, and a sandbox that was created and never used at all. They are indistinguishable to the reaper, which is correct: both are doing nothing. No separate creation clock, no invented idle state.

A sandbox executing a run is not `ready`, so **a sandbox cannot expire mid-run under idle TTL**. File operations and status reads do not touch the clock either, so a client polling status cannot keep a dead sandbox alive indefinitely.

### Absolute TTL (optional ceiling)

A hard maximum lifetime regardless of activity. Off or generous by default.

Idle TTL alone leaves one hole: a sandbox kept continuously busy lives forever, which is the same leak in slower motion. The absolute ceiling closes it, and it is the only path by which a sandbox can expire mid-run. In that case the run is **killed** — a ceiling that waits politely is not a ceiling.

### The reaper

A background loop that wakes periodically, finds expired sandboxes, and destroys them **through the same destroy path a caller-initiated delete uses**. Not a shortcut. Two ways to end a sandbox, only one of them properly evidenced, is exactly the hole this product cannot have.

### Terminal event

Expiry emits its own terminal event, distinct from user-initiated deletion. "The system reclaimed this" and "a user asked for this" are different facts, and an auditor cares which one happened.

### Evidence survives the reaper

Reaping removes the container and the workdir. It must not remove run logs. This is the item where `runs/` living outside `sandboxes/` earns its keep, and it needs an explicit test rather than an assumption.

## Not in scope

**Retention and cleanup of run logs.** The reaper reclaims sandboxes, not evidence. What eventually deletes run logs, and on what policy, is a separate question tied to the paid retention story.

**Warm container pooling.** TTL interacts with it, but pooling is its own item and does not exist yet.

**Per-caller quotas or limits on sandbox count.** Reaping handles abandonment; capping how many a caller may create is a different concern, and one that needs auth to mean anything.

**Configurable reaper interval as a headline feature.** It needs a value, and probably a flag, but tuning it is not the point of the item.

## Depends on

- Sandbox state transitions and their events — shipped.
- The existing destroy path, which the reaper reuses rather than duplicating.
- Run log directory, if built first, for the survives-the-reaper test to be meaningful.

## Trigger

Tier-1 position 6. This is an operator-facing item: it matters for anyone running jhansi as a service for weeks, and barely at all for someone evaluating it for an afternoon.

## Open questions for the ADR

- **What the terminal event is actually called, and whether the domain has an `idle` status at all.** Earlier notes list `idle` among sandbox events; if the real domain only has `ready`, the event name must match what exists rather than what an old list said. Check the tree before naming it.
- Default TTL value, and whether it is per-sandbox on create, server-wide, or both with a server ceiling — the same shape as the ADR-022 timeout override, and worth deciding consistently with it.
- Reaper interval, and whether a sandbox can be reaped up to one interval late (almost certainly yes, and worth stating).
- What happens if destroy fails during a reap — retry, mark, or emit and move on.
- Whether the reaper is one goroutine scanning the registry, or something driven off expiry times.
