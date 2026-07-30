# ADR-013: Sandbox becomes READY on create

Status: Accepted
Date: 2026-07-30

## Context

A sandbox is READY when it can run code. In a real backend that
means provisioning has finished — container up, image pulled. That
step takes time and can fail, which is what CREATING is for.

Tier 0 has no backend. Provisioning is a no-op, so sandbox is
runnable the instant it is minted.

## Decision

CreateSandbox moves the sandbox CREATING → READY before returning,
via MarkReady right after NewSandbox. The rule is "READY = provisioned";
Tier - provisioning is empty, so it fires immediately. CREATING stays
in the machine as the slot a future async backend will occupy.

## Consequences

- Idle sandboxes are reapable. MarkExpired is READY-only; a sandbox
  parked in CREATING could never be TTL-reaped and would leak. This
  rules out the lazy-on-exec alternative.
- Exec is uniform: always READY → ACTIVE → READY, no CREATING branch.
- No SandboxEngine.Create needed. MarkReady is a pure domain flip.
- The create response status changes CREATING → READY.

## Deferred

When a real backend lands, MarkReady moves onto SandboxEngine.Create
returning ok. The rule is unchanged; only its location moves.
