# Feature brief — Operator health signals

*Backlog item. Not an ADR. Scope and boundaries are settled here; code shape is not.*

---

## Problem

jhansi has no way to tell an operator that anything is wrong until something fails.

Run logs make this concrete: every exec writes two files that are never cleaned up, so the data directory grows continuously and free space becomes a real operational concern rather than a theoretical one.

The preflight in the run log directory item catches a broken or full data dir at the moment of use — but by then a caller's exec is already being refused. Nothing surfaces the condition beforehand, and nothing tells an operator at startup that their data dir was misconfigured.

## Scope

Expose the facts an operator's existing monitoring can consume:

- data directory writable
- data directory free space and total space
- isolation engine reachable

Plus a **startup preflight**: a misconfigured or unwritable data dir fails at boot with a clear message, rather than surfacing at the first exec.

## Not in scope

**Alerting.** jhansi exposes signals; it does not alert. No thresholds, no notification channels, no email, no dashboard. The operator already runs Prometheus or whatever their shop uses. Our job is to expose the number honestly and let them threshold it where it makes sense for their disk — picking 70% would be wrong for a 30GB disk and wrong for a 30TB one.

**A paid gate.** These signals ship in the OSS install. Self-hosted means complete: an operator who cannot tell whether their own install is healthy does not have a finished product, they have a demo with a paywall on the part that makes it operable — the exact shape criticised in the competitors. Gating this also hits the beachhead at the precise moment they were about to depend on it.

> **Where the paid line sits instead:** one node observing itself is free. Anything aggregating across nodes, or retaining history, is commercial — a hosted view of thirty installs over ninety days is a different product, not a crippled version of this one.

**`jhansi doctor`.** Stays on Tier-5. It reads these same facts and presents them to a human on demand; this item is the machine-readable, always-on counterpart.

**Cleanup and retention.** Knowing the disk is filling is not the same as doing something about it. Retention belongs with sandbox TTL.

## Depends on

- **Run log directory.** That is what makes data-directory growth real. Before it, the data dir barely grows and this solves a problem that does not yet exist.

## Trigger

After run log directory. Pullable earlier if a real user asks for it — being separate from the Tier-5 operability block is partly what makes that possible.

## Open questions for the ADR

- **One endpoint or two.** A load balancer asking "are you alive" and an operator asking "is anything degrading" are different consumers wanting different answers. Merging them produces a health check that fails because the disk is nearly full, which is a well-known way to make a service restart-loop over a non-fatal condition.
- **Prometheus format or plain JSON.** Tier-5 already lists Prometheus metrics; whether this item anticipates that format or ships something simpler and converges later.
- **Whether the endpoint sits behind auth.** Free space is mild, but it is still information about the host.
- What "isolation engine reachable" costs to check, and whether it is checked on request or cached.
