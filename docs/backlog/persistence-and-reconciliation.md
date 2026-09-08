# Feature brief — Persistence and startup reconciliation

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

The registry is in memory. Restart the server and every sandbox jhansi knew about disappears — while the containers and workdirs those records described are still sitting on the host. Real resources, nothing tracking them.

This is tolerable while one person restarts deliberately. The moment jhansi runs as a service, a deploy or a crash silently strands everything it was managing.

## Scope

### Survive a restart

The registry (sandboxes and runs) and the event log persist across restarts. Storage lives in the data directory. SQLite is the presumed choice — it was proven in the archived Python engine and there is no reason to relitigate it.

### Startup reconciliation

Storage is the easy half. Reconciliation is the work.

After a restart, the truth on disk and the truth in Docker can disagree in both directions:

- **A sandbox we have a record for, whose container is gone.** Mark it terminated; do not pretend it is live.
- **A container running that we have no record of.** Destroy it — but only if it carries jhansi's label. An adopted container has no record of its image, its limits, or what secrets it was granted, so adopting it puts a sandbox in the spine that jhansi cannot account for. That is worse than losing a container. Nothing is holding the container after a restart anyway; the client session died too.
- **Anything unlabelled.** Leave it alone. Another workload on the same Docker host is none of jhansi's business. This requires labelling containers at creation.

### Closing dangling runs

A run recorded as active cannot still be running — the process watching it died with the server. Such a run has no terminal event, so the spine shows a run that started and never ended.

Startup closes it out with a terminal event recording that the server died mid-run. Leaving it dangling is a gap in the record, and the evidence claim does not survive gaps that jhansi knew about and ignored.

## Not in scope

**Storage as a fifth seam.** jhansi picks its store and writes to the data directory. No `Store` interface, no pluggable backend, nobody swapping in Postgres. Same argument as run logs: a data-directory convention, not a boundary. If a real user needs Postgres, that is when it becomes a question.

**Schema migrations.** Migration tooling exists because users have data you cannot destroy. There are no users. Today the answer to a schema change is delete the file and start again. Building migration machinery now is building for a problem that does not exist — but note that it *becomes* real the moment someone outside the project installs jhansi and keeps it.

**Clustering or shared state across nodes.** Directly against "one machine is a first-class deployment." That is Fleet, it is deferred, and it must not leak into how a single node stores its data.

**Run log retention.** Persistence keeps records; it does not decide when they are deleted.

## Depends on

- Sandbox and run domain models, and their events — shipped.
- Container labelling at creation, which reconciliation requires and which may not exist yet. Worth checking the tree; if absent, it is part of this item.

## Trigger

Tier-1 position 7, last of the tier. Operator-facing: it matters to anyone running jhansi as a service, and not at all to someone evaluating it for an afternoon.

## Open questions for the ADR

- Whether the registry and the event log share one store or stay separate. The event log is append-only JSONL today, and putting it in SQLite changes what "append-only" means and how easily an auditor reads it.
- What the dangling-run terminal event is called, and whether it is distinct from timeout and failure. It probably must be — "the server died" is a different fact from "the command failed."
- Whether reconciliation runs before the API starts accepting requests, or concurrently. Serving requests against an unreconciled registry is a race.
- What happens when reconciliation itself fails — refuse to start, or start degraded and say so.
- Write durability: whether every event append fsyncs, and what that costs on the exec path.
