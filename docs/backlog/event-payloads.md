# Feature brief — Event payloads

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Already decided — do not reopen

**ADR-006 (accepted 2026-07-23)** settled the mechanism. `Event` carries `Payload any`, nil by default; events with data hold a small typed struct defined next to the transition that emits it; `map[string]any` was considered and rejected.

So this item is **not** about where payloads live on the type. That exists. This item is about the run lifecycle events not yet carrying the data that makes them evidence.

## Problem

Run events carry a name, a timestamp, and the aggregate they belong to. Someone reading `events.jsonl` can see that a run started and that it succeeded. They cannot see what command ran, what it returned, or whether the output they are holding is the output that was produced.

The spine records the shape of an execution but says nothing about its content. That makes it a timeline, not evidence.

## Scope

Payload structs for the run lifecycle events, following the ADR-006 pattern.

**`run.started`** carries the command as executed.

**`run.succeeded`** and **`run.failed`** carry:
- exit code
- duration
- byte count of stdout and stderr, as retained
- sha256 of each retained stream
- whether output was truncated by the cap from ADR-022

**`run.timed_out`** carries the same result fields, plus the timeout value that was applied.

The hashes commit to the **retained** bytes, not to everything the command produced. Where the ADR-022 cap has bitten, the truncation flag sits next to the hash so the record does not overstate what it covers.

## Not in scope

**Raw stdout and stderr.** These never enter the spine. The log is append-only, so a leaked secret written into it cannot be removed, and a chatty command would grow it without bound. Hash commitments only.

**Sandbox event payloads.** Sandbox events get nothing new here. A separate and smaller question; folding it in would blur what this item decides.

**Ordering, versioning, causation, name taxonomy.** All belong to the event model item, which runs ahead of this one. This item adds payload structs to existing events; it does not touch how they are sequenced or versioned.

**The stored output itself.** The bytes the hashes point at live in the run log directory — a separate item.

## Depends on

- **Real container execution** (ADR-020, ADR-021 — shipped). Payloads describing a StubEngine run would be evidence of nothing.
- **ADR-022** (shipped). The truncation flag and retained byte counts only mean something once a cap exists.
- **Event model item**, which runs first. If it changes ordering or adds envelope fields, this item should be written against the result rather than ahead of it.

## Trigger

After the event model item.

## Open questions for the ADR

- **Whether the hash is computed during capture or after the fact.** Constrained by ADR-022's noted limitation that output is currently read fully into memory before the cap is applied. The run log directory item may settle this by introducing streaming capture — worth checking which lands first.
- Whether the command is recorded as the caller sent it, or as it was handed to the shell.
- Whether one payload struct is shared across `run.succeeded`, `run.failed` and `run.timed_out`, or each gets its own. They carry nearly identical fields, and ADR-006's pattern is a struct per transition.
