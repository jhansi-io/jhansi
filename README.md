# jhansi

**A self-hosted runtime for executing untrusted and AI-generated code.**

Simple to run. Provable by design.

This repo is the engine: a Go single binary with a small HTTP API and a CLI. It is the product — client SDKs are separate repos and wrap this API.

## Status

Early, and honest about it: there is no install and no working execution loop yet. The engine is being typed from zero, in dependency order, starting with the domain model.

Built so far:

- `internal/domain` — `Sandbox` and `Run` aggregates, state machines, event emission
- `internal/id` — prefixed identifiers
- `internal/registry` — in-memory aggregate store
- `internal/evidence` — append-only event sink (JSON Lines)

Next: the execution service, then the first route (`POST /v1/sandboxes`).

Progress and changelog: [jhansiio.featurebase.app](https://jhansiio.featurebase.app)

## Design

**Evidence-native, no retrofit.** Every state transition — including failures and rejected transitions — emits a structured event, from the first operation. Evidence bolted onto an opaque runtime is a rewrite; evidence built into the spine is free.

**Capabilities, not credentials.** Executing code is granted the use of a secret, never its value.

**Four seams.** Isolation, auth, secret backend, and event sink are interfaces with exactly one working default each. A second implementation gets built when a real user needs it, not before.

**Stdlib only.** No web framework, no ORM, no DI container. For a single-binary product with a small route surface, a framework is a liability.

**One machine is a first-class deployment**, not a cut-down cluster install.

## Decisions

Every significant decision is recorded as an ADR in [`docs/adr/`](docs/adr/) before the code is written. Start there to understand why the code looks the way it does.

## Building

```bash
go build ./...
go test ./...
```

Go 1.26 or later.

## Licence

Apache-2.0.
