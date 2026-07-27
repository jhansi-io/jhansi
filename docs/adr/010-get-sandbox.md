# ADR-010: GET /v1/sandboxes/{id} — first reachable client error

Status: Accepted
Date: 2026-07-27

## Context

`POST /v1/sandboxes` (ADR-009) can only fail internally, so it maps every
error to 500. `GET /v1/sandboxes/{id}` is the first route with a
**reachable client error**: an unknown id must be `404`, not `500`.
ADR-009 named this route as the moment the error→status mapping question
comes due.

## Decision

- **Route.** `GET /v1/sandboxes/{id}`, same stdlib `net/http.ServeMux` /
  Go 1.22 method routing. Id read via `r.PathValue("id")`.
- **Service.** A new `GetSandbox(id string) (*domain.Sandbox, error)`
  that passes through to `registry.Get`. A read — no mutation, so no
  events, no `drainAndRecord`, no sink.
- **Failure mapping stays inline, no mapping layer.** The handler
  branches directly: `errors.Is(err, registry.ErrNotFound)` → `404`,
  anything else → `500`.
- **Response.** Reuses the existing `sandbosResponse` DTO. `200 OK` on
  hit.

 ## Consequences

 - **Mapping layer stays deferred, with a sharper trigger.** One handler
   with two outcomes is one `if`. A mapping abstraction earns itself when
   the *same* `ErrNotFound → 404` decision is duplicated across handlers — 
   `exec` and `destroy` will both hit it. At the second duplicated call
   site the extraction is obvious; not before.
- **The handler imports `registry` to check its sentinel, reaching past
  `service`.** `errors.Is` over a wrapped sentinel is the idiomatic tool,
  so this is acceptable. If `service` later needs to own its own error
  vocabulary (translate `registry.ErrNotFound` into a service-level)
  error), that's a separate ADR — not paid for by one route.
- **Deferred, unchanged:** auth, server bootstrap / `main`, and
  `created_at` on the DTO.
