# ADR-015: Exec choreography — the happy path, end to end

Status: Accepted
Date: 2026-07-30

## Context:

Every piece the exec path needs now exists and none of them touch each
other. The Sandbox has its claim/ release pair (ADR-014). The Run has a
full lifecycle with no production caller at all. The isolation seam has
`Exec` and a working stub (ADR-012). The service has a registry, a sink,
and one drain helper.

This ADR writes them unto one route: `POST /v1/sandboxes/{id}/exec`. It is
the first operation that drives two aggregates in a single call, and the
first production caller of `MarkActive`, `MarkIdle`, and the entire `Run`
state machine.

Scope is the happy path only — the engine returns exit 0. EVery failure
mode is ADR-017. Splitting them keeps this ADR about *choreography*: what
happens, in what order, on which aggregate. Mixing the failure surface in
would bury that under a mapping table.

## Decision

### The walk

1. Look up the sandbox. Unknown id → `registry.ErrNotFound` → 404.
2. Mint the run id: `id.New("run")`.
3. Claim the sandbox: `MarkActive` (READY→ACTIVE).
4. `NewRun(runID, sandboxID)` → QUEUED.
5. `MarkPreparing` → `MarkRunning`.
6. `engine.Exec(ctx, sandboxID, command)`.
7. `MarkSucceeded` (happy path: exit 0).
8. Release the sandbox: `MakIdle` (ACTIVE→READY)
9. `drainAndRecord` the sandbox, then the run.

### The mint happens before the claim

`id.New` is the only fallible step that is not a domain transition, and
it is ordered above `MarkActive` deliberately. If it ran after the claim,
a `crypto/rand` blip would leave the sandbox ACTIVE with no run to
explain it — and `MarkExpired` is READY-only (ADR-013), so nothing would
ever reap it. MInting first makes every pre-claim failure a clean 500
with the sandbox untouched.

### The service holds the engine

`service.New(reg, sink)` widens to `service.New(reg, sink, engine)`,
storing an `isolation.SandboxEngine`. Same shape as the registry and sink:
process-lifetime state, shared across operations (ADR-008).

### The domain never sees `ExecResult`

The service maps the engine's result onto a Run terminal; `internal/domain`
does not import `internal/isolation`. A backend type must never reach the
aggregate. The mapping itself is one line here — exit 0 → `MarkSucceeded`
— because the happy path has exactly one outcome. The full table is
ADR-017.

### The Run is not stored

It is minted, transitioned, drained, and dropped when the handler
returns. Its only durable trace is its events in the sink.

This is correct for Tier 0, not an oversight. The sandbox's ACTIVE state
already enforces "one live run, serial", so there is nothing to look a
Run up for: no `GET /v1/runs/{id}`, no cancel, no list. Storing it would
be a registry entry with no reader. It becomes real when a route needs to
address a run after its handler returned — concurrent runs, async
exec, or cancellation. That is Tier 1, and it earns itw own ADR.

Corollary: the response is assembled from `ExecResult` plus the run id,
never from a lookup.

### Wire shapes

Request:
```json
{"command": "echo hello"}
```

One field. The seam takes a bare command string and no language field
(ADR-012); the DTO mirrors it exactly.

Response:

```json
{
	"run_id": "run_...",
	"status": "SUCCEEDED",
	"exit_code": 0,
	"stdout": "echo hello",
	"srderr": ""
}
```

`execRequest` / `execResponse` live in `internal/httpapi`, unexported,
beside `sandboxResponse`. They are not `ExecResult`: that type is the
seam's contract, it carries `TimedOut` which never goes on the wire, and
it has never heard of a run. Returning it directly would make `httpapi`
import `isolation` — the exact dependency the seam exists to prevent, and
it would put any field a future Docker backend adds into the public API by
accident.

`status` ships now although it is constant. Without it the first client
keys off `exit_code == 0` to mean success, and ADR-017 adds TIMED_OUT,
where there is no meaningful exit code at all.

## Consequences

- `service.New` gains a third parameter. Existing call sites in
  `service_test.go` and `handler_test.go` move with it.
- `MarkIdle` gets the caller ADR-014 promised it.
- The `Run` aggregate leaves the test suite and enters production.
- The stub's zero value echoes the command on stdout, so the happy-path
  assertion is `stdout == command` with no knob set.
- `POST /v1/sandboxes/{id}/exec` joins `Routes()`.
- Three client statuses on this route: 400 on a malformed body, 404 on
  an unknown sandbox, 500 otherwise. Mapping stays inline — a decode
  failure is a DTO concern the handler answers itself, not a service
  error needing translation. ADR-017's 409 is the first status that
  comes back through the service, and that is the one that earns a
  mapping layer.
  
## Deferred

- **ADR-017, the failure surface:** non-zero exit → FAILED, `TimedOut` → 
  TIMED_OUT, infra error → sandbox ERROR, and a typed 409 on an
  already-ACTIVE sandbox.
- **Write-ahead ordering:** its own ADR. `engine.Exec` is a failable
  external effect, so record-before-effect is now genuinely reachable — 
  but it is one edit inside `drainAndRecord`, not a choreography change.
- **Output as evidence:** stdout/stderr go on the wire only. They are
  never recorded raw in the sink — events will carry a commitment instead
  (byte count + SHA-256), with the content itself in a run log directory
  at Tier 1. Payload ADR.
- **Storing runs:** Tier 1, gated on a route that addresses a run.
- **Unchanged deferrals:** Docker behind the seam, `Create` / `Destroy`
  seam methods, resource-limit enforcement, auth, server bootstrap.


