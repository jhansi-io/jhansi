# ADR-012: SandboxEngine seam — Exec only, with an honest stub default

Status: Accepted
Date: 2026-07-28

## Context

exec is the first route that runs code, and the root everything above it
sits on must exist before exec can call it: the isolation seam. The
extensibility catalog names this seam `SandboxEngine` — create / exec /
destroy against an opaque runtime — with ephemeral Docker as the default.
This ADR does not build all of that. It builds the smallet coherent
root: the `Exec` method of the seam and one working default, so the exec
choreography can be typed and go green before a real container backend
exists.

Scope is deliberately `Exec` only. `Create` and `Destroy` are real seam
methods, but they have no caller yet. `Create` / provisioning-timing
arrives with the choreography ADR that needs it; `Destroy` / teardown
arrives when it's wired into the already-shipped `DeleteSandbox`. Defining
them now would shape two methods from imagination — the seam-geting-wrong
failure the extensibility doc warns against. With only `Exec`, there is no
prior `Create` handing back a handle, so the seam is forced into the
simple sandbox-id-keyed shape and the `Sandbox` aggregate grows no handle
field yet.

## Decision

- **Package and interface.** New `internal/isolation`. Interface
  `SandboxEngine` with one method:
  `Exec(ctx context.Context, sandboxID, command string) (ExecResult, error)`.
  The seam is keyed by a primitive `sandboxID`, not the `*Sandbox`
  aggregate — a swappable backend must not depend on the domain model.
  (Contrast `evidence.Sink`, which imports domain because events *are* its
  payload; here the payload is the command to run, not the aggregate.)

- **The payload is a bare command, no language field.** The command
  already names its interpreter — `python3 -c "..."`, `node -e "..."`,
  `ls -la`. A `language` field would be a second, less-direct way to
  specify the same thing, so the engine stays dumb: it runs the string,
  it does not know the language. This also keeps the MCP to one tool with one
  shape instead of code / file / project modes to choose between. The
  honest limit: Tier 0 runs interpreted, inline-able languages only.
  Compiled-from-file languages (Go, Java) need a file in the sandbox
  first — that is the Tier 1 filesystem API, after which `go run main.go`
  is the sane one command primitive, still no language field.

- **The result carries the run's outcome; error is infra only.**
  `ExecResult{ ExitCode int; Stdout, Stderr string; TimedOut bool}`.
  A returned `error` means the engine could not run the command at all — 
  daemon down, image pull failed — and maps to the sandbox's infra-fault
  path. A completed run reports through the result: `ExitCode == 0`,
  non-zero exit, or `TimedOut`. This keeps the choreography from sniffing
  sentinel errors to tell "run TIMED_OUT" from "sandbox ERROR": error is
  infra, result is outcome. The three result outcomes line up 1:1 with
  `Run`'s SUCCEEDED / FAILED / TIMED_OUT, which already exist.

- **`TimedOut` ships now though nothing enforces a timeout in Tier 0.**
  Per-exec timeout *enforcement* is Tier 1. The *field* is here from day
  one because `Run.TIMED_OUT` already exists and the stub must be able to
  produce every outcome to be an honest double. This mirrors the domain's
  own stages, not an imagined feature.

- **The default is an honest stub, not a happy-path echo.** `StubEngine`
  is the Tier-0 default and the permanent test double. Its zero value
  returns `ExitCode: 0` with `Stdout` echoing the command — a working default
  so the product runs out of the box. But all four outcomes (exit 0,
  non-zero exit timed out, infra error) are reachable via an injectable
  knob, so tests drive the full failure surface. That reachability is what
  keeps the seam shaped for Docker. Ephemeral Docker is a later drop-in
  second implementation of this same interface (its own ADR); nothing
  above the seam changes when it lands.

## Consequences

- **Nothing is wired yet.** This ADR adds a package, an interface, a
  result type, and a stub. No service method, no route, no domain change.
  The choreography ADR (next) wires `Exec` through the service and adds
  `POST /v1/sandboxes/{id}/exec`, driving the `Run` lifecycle and
  READY⇄ACTIVE — the first producttion callers of `MarkReady` /
  `MarkActive`.

- **Docker is not next; the choreography is.** Because the stub is a real
  backend, the whole exec path ships and goes green on it with no daemon.
  Docker slots in afterward as a second impl of `SandboxEngine`, and the
  choreography is unaffected — the payoff of stub-first.

- **Write-ahead, teardown, and 409 stay deferred, sharpened.** They attach
  when a *failable external effect* exists. The stub's `Exec` can fail
  (infra error), so the choreography ADR is where record-before-effect
  first becomes real, and where the reachable ACTIVE state makes
  delete-an-ACTIVE a genuine 409. Docker / teardown on `Destroy` is later
  still.

- **Stub knob mechanism settled when typed.** Whether the injectable
  outcome is a func field or a next-result pair is an implementation
  detail of the stub, decided at code time (func-field leaning, since it
  can honor a ctx deadline for timeout tests). Not an interface decision.

- **Deferred, unchanged:** auth, server bootstrap / `main`, the mapping
  layer (still inline; trigger still >=3 distinct client statuses),
  `Create` / `Destroy` seam methods, resource-limit enforcement (Tier 1).
