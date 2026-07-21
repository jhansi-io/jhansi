# ADR-001: Domain model - Sandbox and Run as seperate aggregates

Status: Accepted
Date: 2026-07-21

## Context

The engine's core domain is two things: a sandbox (an isolated place
code can run) and a run (an execution inside it). We need to decide
whether these are one entity or two, and what lifecycle each has.

A sandbox outlives any single execution — an agent creates one, then
runs code in it repeatedly (iterating on failures) before it is torn
down. A run is one such execution. Their lifecycles are different
shapes and their failures mean different things.

## Decision

Sandbox and Run are **seperate aggregates**, each with its own state
machine. A Run's terminal state never propagates to its Sandbox.

**Sandbox lifecycle:**

    CREATING → READY ⇄ ACTIVE
    CREATING → ERROR
    READY    → EXPIRED | DELETED | ERROR
    ACTIVE   → READY | ERROR

- READY: exists, idle, no run in flight.
- ACTIVE: a run is executing. Enforces one live run per sandbox, serial.
- READY ⇄ ACTIVE cycles for each run.
- EXPIRED: TTL ended it. DELETED: explicit destroy. ERROR: infrastructure
  fault (container died, runtime available).

**Run lifecycle:**

	QUEUED → PREPARING → RUNNING → SUCCEEDED | FAILED | TIMED_OUT | CANCELLED

- QUEUED: accepted, awaiting sandbox.
- PREPARING: materialising the execution (code in, setup).
- RUNNING: process executing.
- SUCCEEDED: process exited 0.
- FAILED: process exited nonzero. This is normal and expected — an agent
  iterating on syntax errors or test failures produces FAILED runs
  routinely. A FAILED run does NOT put the sandbox into ERROR.
- TIMED_OUT: per-exec timeout killed it. CANCELLED: explicitly stopped.

## Consequences

- The two aggregates are decoupled: only an infrastructure fault moves a
  sandbox to ERROR, never the exit code of the code it ran.
- "The command failed" (Run FAILED) and "the sandbox broke" (Sandbox 
  ERROR) are deliberately different words for different facts.
- Terminal error/failure states will carry a *reason*; the shared reason
  taxonomy is deferred to a later ADR, not defined here.
