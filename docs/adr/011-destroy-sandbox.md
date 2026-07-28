# ADR-011: DELETE /v1/sandboxes/{id} — first write-path mutation, idempotent

Status: Accepted
Date: 2026-07-28

# Context:

The first route that mutates an *existing* aggregate. `POST` (ADR-009)
created; `GET`/`LIST` (ADR-010) read. Destroy is the first route to run a
domain transition and `drainAndRecord` on a stored sandbox, and the first
that must be **idempotent**: deleting twice is a 204, not an error.

The plan assumed the delete path is READY→DELETED. It isn't. No backend
fires `MarkReady` in Tier 0, so a sandbox lives in **CREATING**. The real
path is CREATING→DELETED, which READ-only `MarkDeleted` refuses. It
records a rejection and leaves the status as CREATING. The plan's next
step — translate the rejection to 204 — would then report the sandbox
gone while it's still there. The 204 is a lie.  204 is honest only when
the sandbox actually ends up DELETED (or already was), which is what
forces `MarkDeleted` to accept CREATING.

## Decision

- **State machine.** `MarkDeleted` becomes legal from **CREATING and 
  READY**. Deleting a still-provisioning sandbox is a real, permanent
  operation — sharper once exec makes CREATING a real Docker-pull window.
  The teardown obligation (kill the half-provisioned container) attaches
  when a backend exists; flagged, not built.
- **Route.** `DELETE /v1/sandboxes/{id}`, same stdlib mux / Go 1.22
  method routing, id via `r.PathValue("id")`.
- **Service.** New `DeleteSandbox(id string) error`: `Get` → `MarkDeleted`
  → `drainAndRecord`, in that order. `MarkDeleted` and `drainAndRecord`
  run **unconditionally** — the redundant retry records
  `{From: DELETED, To: DELETED}` down the spine. Record, don't
  short-circuit; the rejection row is precisely what an auditor reads.
- **204-truth is the resulting status, not the error.** After the op,
  `Status == DELETED` → success (nil). That covers both a real delete and
  an already-DELETED retry (idempotent). A rejection that leaves status
  *not* DELETED is a genuine conflict and is returned as an error — never
  swallowed as 204.
- **Failure mapping stays inline.** Handler: `ErrNotFound` → 404, any
  other error → 500, else 204 (no body).

## Consequences

- **Conflict (409) deferred — unreachable, not hand-waved.** ACTIVE /
  EXPIRED / ERROR are the only From≠To rejections, and none is reachable
  in Tier 0 (no exec, no reaper, no infra). Their genuine conflict path
  returns an error → 500 today, but no client can trigger it. When exec
  lands, a typed conflict error + 409 mapping is its own ADR.
- **Mapping layer stays deferred, trigger reaffirmed.** Destroy is the
  "second `ErrNotFound`" ADR-010 predicted — but a second copy of one
  check isn't the trigger. The layer earns itself at **>=3 distinct client
  statuses**. Destroy adds none (reuses 404; 204/500 are trivial). Stay
  inline.
- **The entry stays in the registry (no `Remove`) — but that's not the
  audit mechanism.** Durable audit is the event log on disk (the sink),
  which survives delete regardless. Keeping the object gettable serves
  two live conveniences only: the idempotent 204 retry (the row must
  still exist to re-reject) and a GET that returns the final `DELETED`
  state. In Tier 0 the registry is in-memory and cleared on restart, so
  terminals can't grow unbounded across a deployment yet.
- **Retention / archive / eviction of terminal sandboxes is a
  persistence-layer concenr (Tier 1), deferred there.** Once the registry
  is storage-backed, growth becomes real and the answer is a retention
  policy — live us archived state, eviction, possibly filtering terminals
  out of `List`. The persistence ADR's decision, not destroy's; flagged
  so it isn't dropped.
- **Write-ahead ordering still deferred.** `MarkDeleted` is an in-memory
  flip with no failable external effect to be ahead of — same as create.
  When teardown makes CREATING→DELETED a failable effect, write-ahead and
  teardown ordering becomes real. Deferred, flagged.
- **Deferred, unchanged:** auth, server bootstrap / `main`, conflict/409,
  the mapping layer.
