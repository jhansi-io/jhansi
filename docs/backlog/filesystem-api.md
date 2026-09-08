# Feature brief — Filesystem API

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

The only way to get code into a sandbox today is to embed it in the exec call as a string. That works for one-liners and fails immediately after — a real script has quotes, newlines, and fifty lines of body. And whatever the code produces cannot come back out at all; there is no route for it.

Without file operations, jhansi runs commands. With them, it runs work. This is the item that makes the engine usable by someone other than its author.

The working loop it enables:

```
create sandbox
write script.py + data.csv          (one call)
exec  python script.py
read  output.csv
delete sandbox
```

## Scope

Three operations against a sandbox's workdir:

**Write** — accepts **multiple files in one call**. The primary caller is an agent that has just generated a set of files and knows exactly what it made; it should not need one round trip per file. A single-file write is a batch of one, so there is one route and one event shape, not two.

**Read** — fetch one file out.

**List** — enumerate what is in the workdir. Needed when the caller does not know what the run produced.

Files live in the sandbox workdir on the host. Containers are per-exec and ephemeral, so the workdir is the only thing that persists between runs in the same sandbox. It is removed when the sandbox is deleted — which is why run logs live outside `sandboxes/`.

### Events

A batch write emits **one event for the batch**, carrying the list of paths, sizes, and a hash per file. Not the contents — same reasoning as run output: the log is append-only, files carry secrets, and contents would grow the spine without bound.

One event per batch rather than one per file. It matches how the caller thinks about the action ("I uploaded my project" is one thing, not two hundred), keeps the spine legible, and costs one append regardless of batch size.

Reads emit nothing. A read changes no state, and a client polling for output would flood the log without adding to the claim.

### Path safety

Every route takes a caller-supplied path. Traversal (`../`), absolute paths, and symlinks pointing outside the workdir are the same bug wearing three hats. This is the bulk of the work and the bulk of the tests, and it is the reason this item is larger than "three routes" suggests.

## Not in scope

**Delete, move, chmod.** Not needed for the first working loop. Added on demand.

**`jhansi watch`.** Belongs to the SDK, not the engine. Watching a folder, diffing it, and pushing what changed is a client composing these primitives — the engine must not know what a project is. Hash-on-list, which the sync pattern would need, is deferred with it rather than being built speculatively now.

**Read auditing.** Reads emit nothing today. Noted honestly: reading a file out is how data *leaves* a sandbox, and in a regulated setting that is exactly what an auditor would want recorded. This is out of scope for now, not out of scope forever. If a user asks for it, that is a real requirement rather than something to refuse.

**Streaming very large files.** Cap and revisit. The first user is moving scripts and result files, not disk images.

## Depends on

Real container execution and a persisting sandbox workdir — both shipped.

Nothing in this item depends on event payloads or run logs; the batch-write event is its own shape. It can move independently in the ordering if a user pulls it forward.

## Trigger

Tier-1 position 5. Strong candidate for pulling earlier: this is the item that makes the engine usable by an outside developer, and every other Tier-1 item serves the operator rather than the first user.

## Open questions for the ADR

- Wire format for batch write — multipart, JSON with base64 bodies, or something else. Affects how large a batch can practically get.
- Whether list is recursive by default, and whether it returns directories.
- Per-file and per-batch size caps, and where they are enforced.
- Whether the write event records file *modes* alongside paths and hashes.
- How path validation is structured so it is provably applied to all three routes rather than repeated per handler.

## Assumption on record

Agents are expected to be the source of most traffic; a human working in a watched folder is expected to be rare but influential, since `watch` is a first-hour evaluation tool. Nobody outside the project has installed jhansi yet, so this is a reasonable guess and not a finding. It should be revisited once real usage exists.
