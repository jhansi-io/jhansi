# ADR-024: Run log directory

Status: Accepted
Date: 2026-09-09

## Context

ADR-023 records a SHA-256 of stdout and stderr on the run's terminal event.
Nothing on disk holds the bytes those hashes cover, so the commitment cannot
be checked against anything. A hash with no content behind it proves nothing.

Captured output currently exists only in the HTTP response to the caller who
ran the exec. Once that response is gone the output is gone. The sandbox
working directory (ADR-019) is removed on delete, so it is not a place output
can live either.

This ADR settles where the bytes are written and what happens when that write
fails. The preflight that establishes the directory is writable before any
container starts is a change to the run state machine and is decided
separately.

## Decision

### 1. One directory per run, two files, at the data-dir root

    <data-dir>/runs/<run_id>/stdout
    <data-dir>/runs/<run_id>/stderr

The streams are never merged. Merging loses which stream was which, and
ADR-023 hashes them separately.

`runs/` sits at the data-dir root, deliberately not under `sandboxes/`.
Evidence outlives the sandbox that produced it, and a sandbox deletion must
never remove a run record.

Directories are created with mode 0700, following the convention ADR-019 set
for sandbox working directories.

### 2. The bytes written are the bytes hashed

The files hold the retained output, byte-for-byte identical to the content the
event payload hashed — same truncation applied, same tail kept. The hash and
the file are one claim in two places. If they can diverge, the claim is false.

### 3. The write happens after the engine returns, before the terminal event

The engine produces the output; the terminal event commits to it. Writing
between the two means the bytes are on disk before the record that vouches for
them exists, so the spine never asserts a hash for content jhansi failed to
keep.

### 4. A failed write is recorded, and surfaces in the response

Once the container has run, the run cannot be un-executed. If the write fails
at that point, jhansi records a distinct event stating that the run's output
could not be retained, carrying the reason.

That event is not `run.failed`. The command may have succeeded; what failed is
jhansi's ability to retain what it produced. Conflating the two would put a
failed run in the spine for a command that worked.

The failure also surfaces in the exec response. Only the caller knows that
*this* run lost its evidence, and on a first install nobody is watching the
sink.

Silence is the only unacceptable outcome.

## Consequences

Run logs accumulate and nothing deletes them. Retention belongs with sandbox
TTL and the retention story, not here.

No route reads these files. They exist on disk for the operator and for
verification; a read API is its own item, gated on someone needing it.

Streaming capture stays out. ADR-022 noted that the output cap bounds retained
output rather than memory during the read, and named this work as where
streaming becomes possible. It rewrites the capture path ADR-022 just touched,
and no user has hit the ceiling. It earns its own ADR when one does.

This is not a fifth seam. It is a data-directory convention. Remote or object
storage for run logs is a later question, and answering it now would invent a
boundary nobody has asked for.
