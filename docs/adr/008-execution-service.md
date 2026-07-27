# ADR-008: Execution Service

Status: Accepted
Date: 2026-07-24

## Context

A mutating operation is more than one step. Create-sandbox is: mint id,
construct the aggregate, `registry.Add`, then drain its events and hand
them to the sink (AdR-007). ADR-005 already named this orchestrator
"the caller" — the thing that holds the aggregate long enough to drain
it. That caller now needs a home.

ADR-007 left one rule for here: all drain-and-record must route through
one helper, so write-ahead ordering is a single later edit, not eleven.
A helper that must hold on every mutating path cannot live in the HTTP
handler — the TTL reaper (Tier 1 teardown) fires `MarkExpired` with no
request in sight. If the orchestration lives in the handler, the reaper
re-implements ot or can't record. The rule needs a home that isn't any
one entry point.

## Decision

An `internal/service` package holding a concrete `*ExecutionService`,
constructed with a registry and a sink.

	type ExecutionService struct {
		reg 	*registry.Registry
		sink	evidence.Sink
	}

- **A service, not handler-direct.** Handler and reaper both call in
  through the service, so drain-and-record cannot be bypassed by adding
  a second entry point. Handler-direct is fewer files today and wrong
  by Tier 1.
- **A type, not free functions.** The registry and sink are stable,
  process-lifetime state shared across every operation and the private
  helper. Injected once as fields, not threaded through every signature.
  Free functions win when there's nothing shared to carry; that isn't
  this.
- **First operation: `CreateSandbox`.** Mints the id (`id.New`),
  constructs the aggregate, `reg.Add`, then drains the records — the
  smallest real operation that exercises the whole path.
- **One private `drainAndRecord` helper.** The single home ADR-007
  required. Every operation drains through it; none hand-rolls it.

## Consequences

- The drain-and-record rule now holds for every caller, HTTP or not.
- **Write-ahead ordering stays deferred** (ADR-007). Create's only
  effect is `reg.Add`, which cannot realistically fail on a fresh
  128-bit id — no failable effect yet to be ahead of. When one exists,
  the fix is one edit in `drainAndRecord`.
- **Deferred, not decided here:** route wiring, error→status mapping,
  and the rest of the method surface. Each lands with its route or its
  own ADR.
- Id minting lives in the service, the "caller" ADR-005 pointed at — a
  rand failure still surfaces at this boundary, not from storage.
