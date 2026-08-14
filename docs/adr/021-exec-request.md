# ADR-021: ExecRequest on the isolation seam

Status: Accepted
Date: 2026-08-14

## Context

The isolation seam is `Exec(ctx, sandboxID, command string) (ExecResult, error)`.
It was shaped for `StubEngine`, which needs nothing but the command.

`DockerEngine` needs more. It bind-mounts the sandbox working directory into the
container, and the seam hands it a sandbox ID, not a path. ADR-019 gave the service
ownership of that directory: the service creates it, removes it, and derives its
path with `workDirFor`. The driver cannot ask anyone for the path, because nothing
exposes it.

## Decision

`Exec` takes a request struct:

```go Exec(ctx context.Context, req ExecRequest) (ExecResult, error)```

`ExecRequest` carries `SandboxID`, `WorkDir`, and `Command`. The service fills `WorkDir` from `workDirFor`.

### Why the path is passed, not derived.

The alternative is to give `DockerEngine` its own `dataDir` and let it call the
same derivation. That needs no seam change, and it puts knowledge of the on-disk
layout in two packages that must agree forever. When the layout changes, one of
them is wrong and nothing fails to compile. ADR-019 put that knowledge in the 
service; passing the path keeps it there.

The seam stays free of the domain either way. A path is a primitive, like the ID
beside it.

### Why a struct, not a fourth parameter

`Exec(ctx, sandboxID, workDir, command string)` compiles cleanly when two of
those strings are transposed, and the failure surfaces as a container running in
the wrong directory.

The struct also absorbs the fields already known to be coming — per-exec limits
in ADR-022, an image once a sandbox can carry one — without changing the seam again.
This is not speculative extensibility extensibility: ADR-022 is thhe next ADR.

### Image is not on the request yet

`DockerEngine` has a `defaultImage` and `createBody` falls back to it. Nothing
else in jhansi stores an image: `domain.Sandbox` has no field for one and
`POST /v1/sandboxes` does not accept one. Putting `Image` on `ExecRequest` now
would plumb an always-empty string though the service. It joins the request when
there is a value to put in it.

## Consequences

- The service remains the only component that knows the sandbox directory layout.
- `StubEngine` ignores `WorkDir`, which keeps every existing service test
unchanged in behaviour.
- Adding a per-exec input is an additive struct field, not a signature change and
not a call-site edit.
- Every implementation of the same seam takes the same shape, so a second one
(gVisor) starts from the request rather than from a parameter list.
- ADR-020's deferred section points at ADR-021 and ADR-022 for timeout and Docker 
unavailability. Those pointers shift by one.

## Testing

Covered by existing tests. The change is behaviour-preserving. `StubEngine` and `ExecutionService.Exec` produce the same results, and the compiler finds every call
site.

## Deferred

- **`Image` on the request** — gated on a sandbox that can carry one.
- **Per-exec timeout and output cap** — ADR-022. These land as request fields.
- **A result struct change** — `ExecResult` is untouched here.
