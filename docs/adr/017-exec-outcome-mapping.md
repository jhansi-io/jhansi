# ADR-017: Exec outcome mapping

Status: Accepted
Date: 2026-07-31

## Context

ADR-015 shipped the exec choreography for the happy path only. Every other
outcome is currently either wrong or a leak:

- A non-zero exit is reported as SUCCEEDED. The engine's `ExecResult` carries
  `ExitCode`, and nothing reads it.
- `ExecResult`.TimedOut is on the seam's contract and is never mapped.
- An infra error from `engine.Exec` returns early. The sandbox is left ACTIVE
  with its buffered events never drained — a permanently unreapable sandbox,
  because `MarkExpired` is READY-only. This is the same leak the
  mint-before-claim ordering in ADR-015 was designed to avoid, still live on a
  different path.

ADR-012 established the split this ADR makes real: a returned **error** means
the engine could not run the command at all (infra fault); a returned
**ExecResult** means it ran and the fields report the outcome.

One question has no existing answer. On an infra fault the sandbox goes ERROR,
but the Run is mid-flight in RUNNING and its terminals are
`SUCCEEDED | FAILED TIMED_OUT | CANCELLED`. None of them means "the runtime
broke".

## Decision

**1. Non-zero exit → `run.MarkFailed`.** The command ran and did not succeed.

**2. `ExecResult.TimedOut` → `run.MarkTimedOut`.** The mapping ships now.
Tier 0 enforces no timeout, so no `StubEngine` default produces it; enforcement
is Tier 1. Shipping the mapping ahead of the enforcement keeps the wire status
honest from the first release, so no client ever learns to infer a timeout from
something else.

**3. Infra error → sandbox `MarkError`, run `MarkFailed`.** The sandbox is
unusable and goes to its  ERROR terminal. The run reports FAILED.

**4. The Run's terminal on an infra fault is FAILED, not CANCELLED.

CANCELLED is reserved. It has a known future owner — Tier 3 policy denial and
G4 human approval, where a run is topped because something *decided* it should
not proceed. Spending it on infra faults means either living with a polluted
status when the real canceller arrives, or adding a fifth terminal then anyway.

The intuition that the run "never truly ran" is a fact about the *cause*, not
about the *outcome*, and a Run's status is an outcome field.

The distinction is not lost: `sandbox.error` is recorded in the same sink at the
same moment, on the sandbox that owned the run. At Tier 0 that correlation is
unambigous, because ACTIVE means exactly one run in flight.

**5. No failure path returns before its events are drained.** Every terminal
outcome — success, non-zero exit, timeout, infra fault — reaches
`drainAndRecord` for both aggregates. A failure that leaves no evidence is the
one unacceptable outcome in a product whose thesis is the evidence spine.

## Consequences

- `execResponse.Status` stops being the constant `"SUCCEEDED"` and carries the
  run's real terminal. Clients that read it get the correct outcome without
  interpreting `exit_code`, which is why ADR-015 shipped the field early.
- A non-zero exit and a timeout are **200 responses**. The HTTP call succeeded;
  the command did not. Only the infra fault is a 500.
- A sandbox that hits ERROR is terminal and cannot serve another exec. That is
  correct — the runtime under it is broken — but it means an infra blip costs
  the caller a sandbox. Acceptable at Tier 0 with a stub engine; revisit when a
  real Docker backend makes infra faults routine.
- Every early return in `service.Exec` is rewritten. The method stops being a
  straight line of `if err != nil { return }` and gains a single exit path that
  records.
- Two transitions ate the exception and deliberately so: `run.MarkPreparing`
  and `run.MarkRunning` are invariant by construction (a fresh run is QUEUED;
  each transition is legal from the state the previous one leaves), so they
  panic rather than drain-and-return. They are assertions, not failure path —
  which is why point 5's "no failure path returns before draining" holds with
  no exception rather than in spite of them. This does not soften the
  `id.New` -does-not-panic rule (ADR-004): that guards a recoverable external
  transient; a broken state machine is an unrecoverable coding error, and
  limping into an impossible state is worse than dying loudly.
- `StubEngine.ExecFunc` finally earns its existence: it is how the tests drive
  non-zero exits, `TimedOut`, and infra errors.

## Deferred

- **Busy-sandbox 409 and the error mapping layer** — ADR-018. Claiming an
  already-ACTIVE sandbox is a rejection that must not be a 500. This ADR must
  return that rejection cleanly from `Exec` but must not stop it to a status; the
  mapping layer is 018's decision and does not belong inside an outcome ADR.
- **A cause discriminator on `run.failed`** — the payload ADR. A non-zero exit
  and an infra fault currently produce the same row, distinguishable only by the
  absence of an exit code and the presence of a neighbouring `sandbox.error`.
  That implicit signal must not stay load-bearing.
- **Timeout enforcement** — Tier 1, with the per-exec timeout and the
  `WriteTimeout` deferred in ADR-016
- **Write-ahead ordering** — its own ADR, unchanged
- **Sandbox recovery from ERROR** — not a Tier 0 concern.

  
