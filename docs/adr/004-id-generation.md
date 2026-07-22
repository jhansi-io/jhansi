# ADR-004: ID generation

Status: Accepted
Date: 2026-07-22

## Context

Both domain constructors take `id string` — the aggregate does not mint
its own id (ADR-001 kept the domain pure and deterministic). The
registry, and then the first route, now need to *produce* ids before
calling NewSandbox / NewRun. Sandbox and Run need the same generator;
two copies would drift. IDs also surface in the evidence spine and
become registry keys, so the format has to be fixed once, deliberately.

## Decision

A standalone `internal/id` package, outside the domain (domain
unchanged). One function generates `prefix + "_" + hex(16 random bytes)`
using crypto/rand: `sb_` for sandboxes, `run_` for runs.

- 16 bytes → 128 bits, collision-free at any realistic scale.
- Hex (stdlib encoding/hex), lowercase — no ambiguous characters,
  greppable in logs and evidence bundles.
- Signature `New(prefix string) (string, error)`: crypto/rand. Read can
  fail, and the callers that mint ids already return errors, so the
  failure is surfaced honestly rather than hidden behind a panic.

## Consequences

- Domain stays pure and its tests keep using fixed ids (sb_test).
- The generator is thesingle place id format lives; changing
  length/ encoding later is one edit.
- Callers must handle an "impossible" rand error — accepted as the
  honest cost.
