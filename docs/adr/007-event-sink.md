# ADR-007: Event sink

Status: Accepted
Date: 2026-07-24

## Context

Aggregates buffer events and `DrainEvents` returns them. Nothing
consumes them — drained events are dropped on the floor. The spine
claims events are recorded from the first transition; today they exist
only in memory and die with the process.

The sink is one of the four declared seams. Its default ships complete
(local append-only log), the interface is defined now, and second
implementations (SIEM, S3-WORM, async/remote) wait for a real user.

## Decision

`internal/evidence` holds the seam and its one default.

- **`Sink` interface, `Record([]domain.Event) error`.** Batch, because
  `DrainEvents` returns slice and a transition's events belong
  together. The interface lives in the seam package, not with its
  consumer — the registry (ADR-005) go no interface because internal
  storage is explicitly not a seam. This is.

- **Synchronous and fatal.** `Record` blocks, and a failed `Record`
  fails the request. An execution nobody could record must not be
  claimed to have happened. The alternative — log the sink error and
  carry on — makes the evidence spine best-effort, which is the one
  property this product cannot sell.

- **Default `FileSink`, JSON Lines.** One JSON object per line, stdlib
  `encoding/json`, file opened `O_APPEND` and held. Greppable,
  tailable, appendable with no format ceremony. Marshalling is
  per-record, so a struct payload (ADR-006) serialises with
  deterministic field order — what the signed bundle will need.

- **fsync per `Record`.** "Fatal" is a durability claim. Returning nil
  after handing bytes to the page cache means a power cut loses
  evidence the caller was told was safe. A few ms per exec is not worth
  a false claim. Batching without sync is a config change the day
  throughput hurts; retrofitting durability into a claim already made
  is not.

## Consequences

- Events survive the process.

- `Record` is deliberately slow. Under per-exec workloads that is
  invisible; under high-frequency workloads it will be the first thing
  to hurt — and that is the signal for async, not a reason to
  pre-build one.

- No caller yet. Nothing constructs a sink or drains into it; that
  arrives with the service layer. The sink ships with the tests and no
  production call site.

- **Write-ahead ordering deferred.** Events are recorded after the
  transition, not before. There is no service, no route and no
  container — nothing for a write be ahead of. Cheap to add later:
  a call-site rule in the service layer, no domain or sink change. To
  keep it cheap, the service layer must route all drain-and-record
  through one helper instead of hand-rolling it per method. Noted here,
  decided in the service-layer ADR.

- **Two-phase emission left open, and it is the expensive one.**
  Ordering only swaps a hole (effect happened, nothing recorded) for a
  lie (record written, effect the failed). Closing the lie needs
  intent-then-outcome events — two per transition — which breaks
  ADR-002's one-record-per-transition rule and touches all eleven emit
  sites and their tests. Undecidable until a real irreversible effect
  can fail. Its own ADR then.

- The log file grows unbounded. Rotation and retention are not decided
  here — same posture as ADR-005 on registry growth.
  
