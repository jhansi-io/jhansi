# Feature brief — gVisor isolation

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

> **Status: demand-gated.** Currently sits at Tier 2 on the roadmap. This brief argues it is misplaced there and should move behind the operator block, gated on a real user for whom Docker is not enough.

---

## Problem

Docker is the weakest isolation on the board. A container escape puts untrusted code on the host, and jhansi's stated purpose is running untrusted and AI-generated code. That is the one claim the product had better be able to defend.

gVisor closes it by interposing a user-space kernel between the workload and the host, so a syscall from inside the sandbox does not reach the real kernel directly.

## Why this is demand-gated rather than next

**The beachhead is not the adversary case.** The first user is a developer running jhansi on their own machine, executing code their own agent wrote. That is not hostile code from a stranger — it is their code, on their box. Docker is proportionate for that threat model.

**It costs the door.** gVisor is another thing to install and configure. The distribution promise is `curl … | sh` and `jhansi server`, and a runtime that also needs runsc set up is a materially harder first hour. Some workloads also do not run cleanly under it, which turns a simple product into a troubleshooting surface.

**Nobody has asked.** Same discipline applied to every other item on this board. The `SandboxEngine` seam exists precisely so isolation can change when someone needs it to — that is the seam earning its keep, not a reason to exercise it early.

**The regulated buyer is downstream by design.** Strategy says they adopt later, once smaller users have proven jhansi is dependable. Doing hard isolation work before anyone has installed the product at all inverts that sequence.

## Scope (when it happens)

A second implementation behind the `SandboxEngine` seam, selectable by configuration, with ephemeral Docker remaining the default.

Firecracker is a further step beyond this and is not part of it.

## Not in scope

**Replacing Docker as the default.** The default stays Docker unless a user's evidence says otherwise. Two implementations, one default — the standing extensibility rule.

**Firecracker.** A separate and larger item. Noted so this brief does not quietly become "microVMs".

**A hardening guide or threat model document.** Tier-5 operability material. Related, not this.

## Depends on

The `SandboxEngine` seam — already defined, with `DockerEngine` as its working implementation. Nothing else. This item is genuinely swappable, which is the whole point of having defined the seam early.

## Trigger

**A user tells us Docker is not sufficient for their workload.**

That conversation is itself the signal that a regulated or multi-tenant buyer has arrived, which is the transition the strategy anticipates. Until it happens, the seam existing *is* the answer to the question.

Worth noting the counter-argument honestly: if the first serious enterprise conversation stalls on isolation, this becomes urgent quickly. The seam means that urgency is survivable rather than a rewrite.

## Open questions for the ADR

- Whether gVisor is selected per-sandbox or per-server. Per-sandbox is more flexible and complicates the evidence record, since two runs in one deployment would then have different isolation guarantees — which the record must state.
- Whether the isolation backend in use is recorded in the execution record. It probably must be: "what ran this, and under what containment" is exactly an auditor's question.
- How resource limits from ADR-022 map onto runsc, which may not accept them identically.
- What jhansi does when configured for gVisor and runsc is absent — refuse to start, or fall back to Docker. Falling back silently would be the worst of both.
