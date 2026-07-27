# ADR-009: First Route — POST /v1/sandboxesone.

Status: Accepted
Date: 2026-07-27

# Context:

`CreateSandbox` (ADR-008) has no HTTP caller yet. This is its first one.
It pulls in two things ADR-008 deferred: route wiring and error→status
mapping.

## Decision

- **Route.** `POST /v1/sandboxes`, stdlib `net/http.ServeMux` with Go
  1.22 method-pattern routing (`mux.HandleFunc("POST /v1/sandboxes", ...)`).
  No framework.
- **Handler home.** A new `internal/httpapi` package holding a concrete
  handler struct constructed with `*service.ExecutionService`. Same
  reasoning as ADR-008's service: the dependency is injected once as a
  field, not threaded through signatures.
- **Response DTO.** A small `sandboxResponse` struct in `httpapi`,
  `{id, status}`, marshalled to JSON. The domain `Sandbox` stays free of
  wire (`json:`) tags — the engine owns the wire contract separately, so
  a domain rename is never a wire break.
- **Success.** `201 Created`, `Content-Type: application/json`, DTO body.
- **Failure.** Any error from `CreateSandbox` → `500`, generic body, real
  error logged. Not a mapping layer — just "write 500."

## Consequences

- **error→status mapping stays deferred.** Only 500 is reachable on
  create (rand glitch, sink write). `ErralreadyExists` can't fire on a
  fresh 128-bit id; there's no request body, so no 400. A mapping layer
  would be one live row plus a dead branch. It earns itself at the frist
  *reachable* client error ('GET /v1/sandboxes/{id}' → 404).
- **Response is minimal by choice:** `id` + `status`. `created_at` can be
  added later — adding a field is non-breaking; removing one would break.
- **Deferred, not decided here:** auth middleware (the auth seam),
  request-body / validation (create takes no input yet), and server
  bootstrap / `main`. Each lands with its route or its own ADR.
