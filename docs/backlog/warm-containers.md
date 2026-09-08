# Feature brief — Warm containers

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

> **Status: deferred.** Not queued, not scheduled. Waiting on a demand signal that does not exist yet. Written down so the idea is captured with its reasoning intact, and so it is not quietly pulled forward.

---

## Problem (as it would be, if it were real)

Every exec starts a fresh container and destroys it afterwards. That is a few hundred milliseconds of startup on every run, paid every time, even for a sequence of ten quick commands in the same sandbox.

Warm containers would keep one alive per sandbox, so the second exec onwards starts near-instantly.

## Why this is deferred rather than queued

**Nobody has complained.** No one outside the project has installed jhansi, let alone found it slow. This is the exact shape of item the standing rule warns about: build-what-excites-me is a tiebreaker between equally user-valuable options, never a reason to pull something forward.

**It is explicitly a non-goal.** Sub-100ms startup obsession is on the strategy doc's non-goals list. It is E2B's game, and competing on cold-start speed against Firecracker microVMs is not a winnable fight or a differentiating one.

**Per-exec containers buy something real.** Clean isolation between runs, and a lifecycle with no live state to manage between execs. Warm containers give both of those up.

**The cost lands on three items already on the board.** A persistent container per sandbox must be tracked, health-checked, reaped when the sandbox expires, and reconciled after a restart — touching TTL and reaper, persistence and reconciliation, and the isolation seam. It also weakens the isolation story, since two runs in the same sandbox would share a process space.

## Scope (when it happens)

One long-lived container per sandbox, created on first exec, reused by subsequent execs, destroyed with the sandbox.

Behind the `SandboxEngine` seam. The architecture already anticipates this — the seam exists precisely so isolation strategy can change without the lifecycle code knowing.

## Not in scope

**A pool of pre-warmed containers shared across sandboxes.** A different and much larger idea, with a cross-tenant isolation problem attached. Not this.

**Anything framed as a startup-latency benchmark.** If this is ever built, it is because a user's workload made it necessary, not to post a number.

## Depends on

TTL and reaper, and persistence with reconciliation. A warm container is exactly the resource those two items are designed to not leak, so building this before them creates the orphan problem twice.

## Trigger

**A real user says exec startup is too slow for their workload.**

Not a benchmark, not an intuition, not a competitor comparison. A specific person with a specific workload. Until then this file is the whole of the work.

## Open questions for the ADR

- What happens to a warm container that dies unexpectedly — recreate silently, or surface it.
- Whether run isolation within a sandbox needs any guarantee at all, or whether sharing a process space between runs in the same sandbox is acceptable by definition.
- How a warm container interacts with per-exec resource limits from ADR-022, which currently apply at container creation.
