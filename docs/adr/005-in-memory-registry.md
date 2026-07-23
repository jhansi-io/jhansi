# ADR-005: In-memory registry

Status: Accepted
Date: 2026-07-23

## Context

Tier 0 needs create / get / list to survive between requests. Something
must hold sandboxes in process. Persistence (registry + events surviving
restart) is Tier 1, so today's store is in-memory — but the surface it
exposes now is what the persistent version will have to satisfy.

Two questions had to be settled before tyoing it: whether the store
hdes behind an interface from day one, and whether it mints ids.

## Decision

A `internal/registry` package holding a concrete `*Registry` — a map
guarded by a `sync.RWMutex`, since handlers are concurrent from the
first route.

- **No interface.** The four seams (isolation, auth, secret backend,
event sink) are the deliberate swap points; internal storage is
explicitly not one of them. In Go the interface belongs to the
consumer, so adding one when persistence lands costs one file — 
whereas defining it now means designing the persistence contract
before the persistence model exists.
- **Dumb store.** `Add`, `Get`, `List`. The caller mints the id via
  `id.New`, constructs the aggregate, then hands it over — so the
  caller still holds the aggregate to drain its events, and a rand
  failure surfaces at the request boundary rather than from a storage
  method.
- **`Get(id) (*domain.Sandbox, error)`** with a sentinel `ErrNotFound`,
  not `(*Sandbox, bool)`. A persistent store can fail for reasons other
  than absence; `error` survives that swap, `bool` forces a call-site
  rewrte. It also separates 404 from 500 at the route.
- **`Add` returns an error** with a sentinel `ErrAlreadyExists`. The
  risk isn't collision (128-bit random ids) — it's a caller bug
  re-adding a live id, silently overwriting a live sandbox and losing
  its unflushed teardwn events. The registry guards its own key
  invariant rather than trusting callers.
- **No Remove.** Destroy is `MarkDeleted()` on the aggregate; the entry
  stays in the registry in its terminal state. Evidence-native means a
  destroyed sandbox is still gettable, not erased.

## Consequences

- The store is swappable later, but nothing pays for that today.
- Callers handle two sentinels; both map cleanly to HTTP status.
- The map grows unbounded — DELETED and EXPIRED sandboxes are never
  reclaimed. Acceptable in-memory at Tier 0; retention and reclamation
  get decided with persistence, not guessed at now.
