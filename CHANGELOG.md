# Changelog

All notable user-facing changes to jhansi are recorded here.
Internal and behaviour-preserving changes are not.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0] - 2026-08-06

### Added
- `POST /v1/sandboxes` — create a sandbox and get back its id and status.
- `GET /v1/sandboxes/{id}` — fetch a sandbox by id; 404 if unknown.
- `GET /v1/sandboxes` — list all sandboxes.
- `DELETE /v1/sandboxes/{id}` — delete a sandbox by id; idempotent, 404 if unknown.
- `POST /v1/sandboxes/{id}/exec` — run a command in a sandbox; returns the run's outcome (status, exit code, stdout, stderr). 409 if the sandbox is busy with a live run.
- `jhansi server` — run the engine as a single binary; `-addr` and `-data-dir` flags, events written to `<data-dir>/events.jsonl`.
