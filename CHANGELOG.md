# Changelog

All notable user-facing changes to jhansi are recorded here.
Internal and behaviour-preserving changes are not.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.0] - 2026-09-09

### Added
- Run events in `events.jsonl` now carry the execution itself, not just its timeline. `run.created` records the command as submitted; `run.succeeded`, `run.failed` and `run.timed_out` record the exit code, duration, retained output sizes and SHA-256 hashes, and whether output was truncated. `run.timed_out` also records the timeout applied.
- Output hashes commit to the bytes jhansi retained, which `output_truncated` reports may be fewer than the command wrote.
- A run that was killed on timeout or never started records no exit code, rather than zero.
- Run output is now kept on disk. Each run's retained stdout and stderr are written to `<data-dir>/runs/<run_id>/`, byte for byte the content the run's event hashes commit to. `runs/` sits outside `sandboxes/`, so deleting a sandbox does not remove its runs' output.
- An exec is refused if jhansi cannot create the run's log directory — a full disk, a read-only data directory, or wrong permissions. Nothing runs, and the run is recorded as `FAILED` with `run.preparation_failed` and the reason. A refused run has no `run.running` event.
- `logs_retained` on the exec response reports whether the run's output was kept. When it is false, `logs_error` gives the reason and `run.logs_write_failed` records it: the command's own outcome is unaffected.

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
