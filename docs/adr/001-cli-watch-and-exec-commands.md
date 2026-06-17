# ADR 001 - CLI watch and exec commands

## Status
Accepted

## Context

Developers building with jhansi need a tight local dev loop. Manually uploading files and passing sandbox IDs on every run creates friction. As jhansi is used to build jhansi itself, this gap is felt immediately.

## Decision

Add three CLI commands to the `jhansi` SDK:

- `jhansi watch <dir>` - creates a sandbox, uploads project files, watches for changes, syncs on save. Writes sandbox ID tp `.jhansi` in the project root. On restart, reconnects to the existing sandbox if still alive, or silently recreates it and resyncs.
- `jhansi exec <command>` - reads sandbox ID from `.jhansi`, runs the command against the active sandbox via the exec endpoint. If the sandbox is expired or invalid, silently recreates it, resyncs files, and retries.
- `jhansi unwatch` - tears down the sandbox and deletes `.jhansi`. Explicit opt-out only.

## Consequences

- Developers never manage sandbox IDs manually.
- Sandboxes persist across terminal sessions until explicitly unwatched.
- The `.jhansi` file should be added to `.gitignore`.
- No changes required to Petri.
- CLI entry point added to `pyproject.toml` via `[tool.poetry.scripts]`.
