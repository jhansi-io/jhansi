# Changelog

All notable user-facing changes to jhansi are recorded here.
Internal and behaviour-preserving changes are not.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- `POST /v1/sandboxes` — create a sandbox and get back its id and status.
- `GET /v1/sandboxes/{id}` — fetch a sandbox by id; 404 if unknown.
- `GET /v1/sandboxes` — list all sandboxes.
- `DELETE /v1/sandboxes/{id}` — delete a sandbox by id; idempotent, 404 if unknown.
- `jhansi server` — run the engine as a single binary; `-addr` and `-data-dir` flags, events written to `<data-dir>/events.jsonl`.
