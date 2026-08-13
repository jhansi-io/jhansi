# ADR-019: Sandbox working directory

Status: Accepted
Date: 2026-08-07

A sandbox at v0.1.0 is an identity and a state machine. It contains nothing.
There is no path to upload a file, no directory a process could run against,
and no decision about where a sandbox's bytes would live if it had any.

ADR-016 chose `data-dir` (default `./jhansi-data`, mode 0700) and placed
`events.jsonl` under it, anticipating that the tree would later hold captured
process output. It did not decide the layout below that root.

Three pending pieces of work each need that layout to exist first: the Docker
driver needs a host path to mount, the filesystem API needs somewhere to read
and write, and the run log directory wants the same root. Deciding it once,
now, keeps those three from each inventing an answer.

A sandbox's contents are ephemeral by design. It is a bundle assembled for
execution, not a place to store things: inputs arrive from somewhere durable,
outputs are fetched to somewhere durable, and the directory between them is 
disposable. That is why the directory can be removed on destroy without
ceremony, and why its loss is an operational event rather than a data-loss
incident.

## Decision

**1. Layout.** A sandbox's working directory is `<data-dir>/sandboxes/<sandbox_id>`,
created mode 0700, sibling to the planned `<data-dir>/runs/<run_id>`. Flat: one
segment for the collection, one for the identity.

**2. The service owns it.** Creation and removal happen in the service layer
alongside the existing registry and event calls. This is not a fifth seam. A 
filesystem abstraction earns an interface when a second implementation is asked
for; local disk is the only one on the board.

**3. The domain holds no path.** `Sandbox` gains no `WorkDir` field. The path is
derived where it is needed, by an unexported helper in the service:

	workDirFor(dataDir, sandboxID string) string

The identity is intrinsic to the sandbox and travels with it. The path is a 
consequence of one operator's `-data-dir` on one machine — it goes stale the
moment that flag changes, and a stored copy can disagree with the function that
computes it. Deriving removes the possibility of disagreement. It also keeps
`domain.NewSandbox` free of configuration, and keeps the persistence honest later:
what gets written to disk is an ID, and the path is recomputed on load.

**4. Created on create, not lazily.** Every sandbox has a directory from the
moment it exists. Under lazy creation, whether a sandbox has a directory
depends on whether anything has happened to write to it yet — two sandboxes in
the same state are indistinguishable through the API but differ on disk, and an
operator listing `<data-dir>/sandboxes` sees fewer directories than sandboxes
with no way to tell whether that is expected. Lazy creation also means every operation that touches the directory must first check whether it exists,
since none of them knows whether it is the first. Eager creation buys one
invariant: if a sandbox is in the registry, its directory exists.

**5. Removal failure is an engine failure, not a run outcome.** If the directory
cannot be removed on destroy, the sandbox still reaches `DELETED` and the route
still returns `204`. The failure is emitted as its own event. The sandbox is gone
as far as the domain is concerned; orphaned bytes are the engine's problem and
are recorded as such. Silence is the unacceptable outcome, not failure.

**6. The path is passed down, never re-derived.** The service computes it once
per operation and hands it to whatever needs it. Nothing downstream calls
`workDirFor` for itself, so there is exactly one place that knows the layout.

## Consequences

The Docker driver and the filesystem API both receive a host path rather than
comouting one, so neither depends on the layout decision beyond its signature.

`workDirFor` is a pure function of its arguments and is trivially testable
without touching disk.

Persistence (later) stores identity and recomputes paths, so restoring a 
registry onto a host with a different `-data-dir` is correct rather than
dangling.

**Known gap: nothing reaps orphans.** A sandbox whose process is killed, or
whose host crashes, or which is simply never destroyed, leaves it directory
behind with nothing to clean it up. Sandbox TTL and the reaper are separate
work; until they land, `<data-dir>/sandboxes/` grows monotonically under
abnormal termination. Recorded here as accepted for now, not overlooked.

## Testing

- `workDirFor` returns the expected join; pure, no filesystem.
- Create leaves a directory at the derived path with mode 0700.
- Destroy removes it.
- Destroy with removal failing still transitions to `DELETED`, still returns
  204, and drains an engine-failure event.
- Create failing to make the directory does not leave a registered sandbox.

## Deferred

**Ownership hierarchy.** Sandboxes are flat under one root. When a sandbox
acquires an owner — a project, a tenant — the domain gains the identity field
and `workForDir` grows an argument; the layout becomes hierarchical and the
path stays derived. The rule does not change, only its inputs. The trigger is
the first route that accepts an owner, and that route is its own ADR: it has to
settle defaults, naming, collisions, and whether existing sandboxes are
assigned retrospectively. None of that is a struct field.

**Operator-supplied path.** A sandbox on a different volume, or at a path the
operator names, breaks derivation and woud need a stored field. No such
request exists. Config-shaped, and a migration rather than a rewrite if it
arrives.

**Reaping orphaned directories.** Belong with sandbox TTL.

**Directory contents.** This ADR decides where a sandbox's directory is and who
manages its lifecycle. What goes in it — uploads, mount layout, whether the
container sees it at a fixed guest path — is decided by the Docker driver and
filesystem API ADRs.
