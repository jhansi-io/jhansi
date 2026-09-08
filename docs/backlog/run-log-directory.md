# Feature brief — Run log directory

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

Event payloads commit to a sha256 of stdout and stderr, but nothing on disk holds the bytes that hash covers. A hash with no content to check it against proves nothing.

Captured output today exists only in the HTTP response to the caller who ran the exec. Once that response is gone, the run's output is gone. The sandbox workdir is removed on delete, so it is not a place output can live either.

## Scope

### The directory

A per-run directory holding the captured streams:

```
<data-dir>/runs/<run_id>/stdout
<data-dir>/runs/<run_id>/stderr
```

Two separate files, one per stream, never merged. Merging loses which stream was which, and the payload hashes them separately.

The bytes written are **byte-for-byte identical to the content the event payload hashed** — same retained bytes, same truncation applied. If they diverge, the evidence claim is broken.

`runs/` sits at the data-dir root, deliberately **not** under `sandboxes/`. Evidence outlives the sandbox that produced it. A sandbox deletion must never remove a run record.

### Preflight

Before the container starts, verify the run directory is writable.

If it is not, **refuse the exec** and return an error saying so. At that point nothing has executed, so refusing costs nothing and the caller gets a clear failure instead of a silent evidence gap. This catches the cases that actually occur — full disk, read-only remount, exhausted inodes, changed permissions — all of which are true before the run as well as after.

### Mid-run failure

If the write fails after the container has run, the run cannot be un-executed. That case produces an **engine-failure event in the sink** — not `run.failed`. The run itself may have succeeded; what failed is jhansi's ability to retain its output. Silence is the only unacceptable outcome.

## Not in scope

**A fifth seam.** This is a data-directory convention, not a swappable boundary. There are four seams and this is not one of them. Remote or object storage for run logs is a later question; answering it early would invent a seam nobody has asked for.

**A read API.** Nothing here exposes a route for fetching a run's output. The files exist on disk for the operator and for verification. An API for reading them is its own item, gated on someone needing it.

**Retention and cleanup.** Run logs accumulate. What deletes them, and when, belongs with sandbox TTL and the retention story — not here. This item only writes.

**Configurable strictness.** Whether refusal is the default, or whether an operator can choose to execute without evidence retention, is deferred. Default-open suits the beachhead; default-closed suits a regulated buyer. `auditd` and SQL Server Audit both made this configurable rather than picking, and that decision needs a real user behind it.

**Proactive monitoring.** Free space, disk health, and anything that warns before a failure belongs to the operator health signals item.

## Depends on

- **Event payloads.** The hashes have to exist before there is anything for these files to be the counterpart of. Built in that order, the two land as one coherent claim.
- **ADR-022** (shipped) for the cap that defines what "retained" means.

## Trigger

Tier-1 position 4. Immediately after event payloads.

## Open questions for the ADR

- **Whether streaming capture comes in here or separately.** ADR-022 notes the cap bounds retained output, not memory during the read, and flags this work as where streaming becomes possible. Streaming to file while hashing bytes in flight would fix it — but it rewrites the capture path ADR-022 touched, which makes it a second moving part and probably its own ADR.
- **Whether a mid-run write failure also surfaces in the exec response.** Leaning yes: the health signals item can tell an operator the disk is filling, but only the response knows that *this run* lost its evidence, and on a first install nobody is watching the sink.
- Directory permissions, and whether they follow the 0700 convention from ADR-019.
- What happens on a partial write — a file that exists but is short. Whether the engine-failure event is enough, or whether the partial file needs marking.
- Whether the run directory is created at run start or lazily on first write. The preflight may settle this by forcing creation up front.
