# ADR-018: Busy-sandbox 409 and the error mapping layer

Status: Accepted
Date: 2026-08-06

## Context

ADR-017 made `service.Exec` return the claim rejection cleanly but stopped
short of a status. Today a busy claim surfaces as a plain `fmt.Errorf` from
`Sandbox.MarkActive`, indistinguishable from any other domain error. The
handler's `errors.Is` chain knows only `registry.ErrNotFound`; everything else
falls through to 500. So exec against an already-ACTIVE sandbox returns 500 —
wrong. A concurrent or repeat caller asking for a sandbox that is busy is a
`409 Conflict`, the textbook "the resource is in a state that forbids this",
not a serverfault.

Two things have to be decided, and 017 explicitly handed both here.

**Where the discriminator lives.** `MarkActive` currently collapses *every*
non-READY status into one generic error. But "busy" is one specific case:
`From == ACTIVE`. The other reachable non-READY states — DELETED
(exec-after-delete; the registry keeps entries for audit, so lookup still
finds them), ERROR, EXPIRED, CREATING — are *not* busy and must not become
409. A service-side "the claim failed, therefore busy" inference is wrong the
moment exec meets a deleted sandbox. The domain, not the service, is the
authority on *why* a transition rejected.

**Whether to extract a mapping layer.** The `ErrNotFound → 404, else → 500`
block is copy-pasted identically in three handlers — GetSandbox, DeleteSandbox,
Exec. 017 named the second distinct service error as the trigger to decide:
extract, or stay inline once more? The trigger is met by duplication that
already exists, not by a hypothetical third case.

## Decision

**1. A new sentinel `domain.ErrSandboxBusy`, wrapped only for the ACTIVE case.**
`MarkActive` records the rejection exactly as it does today (unchanged from 
017), then chooses its return by the from-state:

- `From == ACTIVE` → `fmt.Errorf("sandbox %s: already active: %w", s.ID, ErrSandboxBusy)`
- any other non-READY → today's generic error, unchanged.

The `%w` wrap keeps the sandbox id in the operator-facing message while
`errors.Is(err, ErrSandboxBusy)` matches for mapping. Wrapping, not a bare
sentinel, because the id is worth more in a log line than the two characters it
costs.

The sentinel lives in `domain`, beside the aggregates — not in `registry`.
This is a state-machine rejection reason, not a lookup miss. `ErrNotFound`
belongs to the registry because a missing key is a registry fact; a busy
sandbox is a domain fact.

**2. The service does not change.** `Exec` already returns the claim error
verbatim (the post-017 drain-then-return on the failed-claim path). The wrapped
sentinel propagates through untouched; `errors.Is` sees through the `%w`. This
ADR touches `domain` and `httpapi` only.

**3. Extract `statusFor(err) int` in `httpapi`.** One unexported helper
replaces the three copy-pasted blocks:

- `registry.ErrNotFound → 404`
- `domain.ErrSandboxBusy → 409`
- `default → 500`

Each handlers's error branch becomes `http.Error(w, msg, statusFor(err))`. The
extraction is justified by the duplication it deletes today, not by the arrival
of a third case in the abstract — three identical blocks were already a smell;
adding a fourth status inline would have deepened it.

**4. No new events.** Evidence is already complete — 017 records
`sandbox.active_rejected` on the busy claim. 018 is status mapping only.

## Consequences

- Exec against a busy sandbox is a `409`, carrying the sandbox id in its
message. Every other terminal-state rejection stays `500` — an explicit
deferral, not an oversight (see Deferred). Nothing in Tier 0 auto-expires,
so exec-after-delete is the only other reachable claim rejection, and it does
not earn a distinction status yet.
- The 500-by-default posture is preserved for genuinely unclassified errors.
  `statusFor` is deny-by-default: an error nobody has mapped is a server fault
  until proven otherwise, which is the safe direction.
- `domain` now exports two names an outside package matches on
  (`ErrSandboxBusy` here; the aggregares already). The handler imports `domain`
  for status mapping as well as for `SandboxResponse` — no new dependency edge,
  the edge already exists.
- The mapping layer is now a real, if tiny, seam. A future status (the payload
  ADR's cause discriminator does not need one; a Tier 1 `409` on a second
  concurrent claim might) is one `case`, added in one place, asserted once.

## Testing

- `TestExecBusySandbox` (service) gains the assertion 017 deliberately omitted:
  `errors.Is(err, domain.ErrSandboxBusy)`. The event and status-quo assertions
  it already makes stay.
- A new handler test drives exec against a claimed sandbox through 
  `httptest`  and asserts `409`, closing the loop the service test cannot see.

## Deferred

- **A distinct status for exec-after-delete and other terminal-state claim
  rejections.** They stay 500. Revisit when a reaper or real backend makes
  a non-busy claim rejection routine.
- **Cause discriminator on `run.failed`** — the payload ADR, unchanged from
  017. Unrelated to claim rejection.
- **Write-ahead-ordering, cross-aggregate event ordering** — their own ADRs,
  unchanged.
- Custom error bodies — standard status text (http.StatusText) for now; friendlier    per-error messages deferred, no user reads a 404 body today.

## After this ADR

The Tier 0 slice is complete.
