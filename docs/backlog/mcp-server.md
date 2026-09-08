# Feature brief — MCP server

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

Trying jhansi today means writing HTTP calls. For the beachhead — developers building private AI applications — that is a real barrier at exactly the wrong moment, before they have seen it work.

MCP removes it. Point Claude, or any MCP client, at jhansi and it runs code in a sandbox on your own machine. This is the first-hour experience, and it is the only item on the board that turns jhansi from something someone reads about into something someone tries.

Every other backlog item improves jhansi for people who have already installed it.

## Scope

An MCP server exposing the engine's operations as tools:

- create sandbox
- exec
- delete sandbox
- file operations (write batch, read, list)

The three lifecycle tools existed in the archived Python engine, so the shape is known. File tools are included because this item lands after the filesystem API, and a demo loop that cannot move a script in or a result out is a weak demo.

**Lives inside the engine binary**, not as a separate service. Single binary is the distribution promise; an MCP server that needs its own process breaks it.

**A thin translation layer over the existing HTTP handlers**, not a parallel implementation. The engine's API is deliberately small and stable; MCP is a second surface over the same operations, and if the two drift there are two contracts to keep honest instead of one.

## Not in scope

**`read_file` as a distinct tool.** A Python-era roadmap item that was never built. If the filesystem tools cover it, it does not need to exist separately. Reconsider on demand, not by default.

**Tools with no HTTP equivalent.** MCP exposes what the API already does. A tool that has no route behind it is a second contract by another name.

**Transport beyond the simplest thing that works.** Whatever the standard client expects; no multi-transport support until someone needs it.

**Agent identity.** See below — deliberately deferred, not overlooked.

## Known gap: the caller is anonymous

An MCP client is an agent, and the agent is the untrusted party. Today a tool call arrives with no identity attached — jhansi cannot say *which* agent asked for what.

That is acceptable for a demo and it is a hole in the evidence claim. The record can say a sandbox was created and code was run; it cannot say who asked. The auth seam and the capability manifest (G1) eventually close it.

Recorded here so that MCP is not mistaken for complete, and so the gap is a known deferral rather than something discovered by the first person who cares about it.

## Depends on

- The create → exec → destroy loop being real — shipped. This was the trigger the roadmap set, and it has been met.
- **Filesystem API**, for the file tools and for the demo to be worth giving.

## Trigger

After the filesystem API.

Worth pulling ahead of the operator block (health signals, TTL, persistence) if that block starts to run long. Those three serve an operator running jhansi for weeks; MCP serves the person deciding whether to install it at all.

## Open questions for the ADR

- Whether the MCP server runs in the same process as the HTTP API by default, or behind a flag.
- How tool errors map to the engine's error taxonomy. "Good errors are a feature" applies here too, and an MCP client surfaces errors straight to a human reading a chat window.
- How exec output — potentially large, potentially truncated — is presented through a tool result, where the consumer is a language model with a context limit rather than a program.
- Whether tool calls emit events distinct from the equivalent HTTP calls, or whether the spine is indifferent to which surface a request arrived on.
