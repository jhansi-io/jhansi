# Changelog

All notable user-facing changes to jhansi are recorded here.
Internal and behaviour-preserving changes are not.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.2.0] - 2026-09-08

### Added
- Real execution. A command submitted to a sandbox now runs in a Docker container rather than a stub; the run's recorded outcome describes an execution that actually happened.
- Each sandbox gets its own working directory under `<data-dir>/sandboxes/`, created on create and removed on delete.
- `-docker-socket` — path to the Docker daemon socket (default `/var/run/docker.sock`).
- `-default-image` — image used when a sandbox does not specify one (default `python:3.12-slim`).
- `-exec-timeout` — maximum duration of a single command (default `5m`). A command that outlives it is killed and its run recorded as `TIMED_OUT`.
- `-max-output-bytes` — maximum output retained per stream (default `1MiB`). The tail is kept, and truncation is reported in the exec response.
- `timeout_seconds` on the exec request body — optional per-command override of `-exec-timeout`. Must be positive; 400 otherwise.

## [0.1.0] - 2026-08-06

### Added
- `POST /v1/sandboxes` — create a sandbox and get back its id and status.
- `GET /v1/sandboxes/{id}` — fetch a sandbox by id; 404 if unknown.
- `GET /v1/sandboxes` — list all sandboxes.
- `DELETE /v1/sandboxes/{id}` — delete a sandbox by id; idempotent, 404 if unknown.
- `POST /v1/sandboxes/{id}/exec` — run a command in a sandbox; returns the run's outcome (status, exit code, stdout, stderr). 409 if the sandbox is busy with a live run.
- `jhansi server` — run the engine as a single binary; `-addr` and `-data-dir` flags, events written to `<data-dir>/events.jsonl`.
