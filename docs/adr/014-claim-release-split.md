# ADR-014: Claim/release is a distinct pair, not two meanings of MarkReady

Status: Accepted
Date: 2026-07-30

## Context

MarkReady currently accepts two sources: CREATING (came up) and ACTIVE
(a run finished). Both emit sandbox.ready. IN an evidence-native product
that is a bug in the log: an auditor sees sandbox.ready twice meaning two
different facts — "provisioning finished" and "a run released the sandbox"
— and cannot tell them apart without replaying prior state. The spine
exists precisly to make a row answerable from its own name; one name for
two facts breaks that.

The dual source in MarkReady's own doc-comment is the tell. A sandbox is
claimed for a run (READY→ACTIVE) and released when the run ends
(ACTIVE→READY). That is a matched pair. Naming the release MarkReady
conflated it with provisioning.

## Decision

Split the pair. ACTIVE→READY stops being MarkReady.

- MarkActive → sandbox.active. Claim: a run starts. Legal from READY.
  Unchanged.
- MarkIdle → sandbox.idle. Release: a run finished, sandbox free again.
  Legal only from ACTIVE. New.
- MarkReady → sandbox.ready. Reserved for CREATING→READY only —
  provisioning done. Its ACTIVE source is removed.

Rename, not a new sibling: nothing should ever reach READY from ACTIVE
again, so leaving that path legal would be a trap. Remove it outright.

Result in the log: sandbox.ready fires once per sandbox lifetime;
sandbox.active / sandbox.idle fire once per run. Every row is anwerable
from its own name.

## Consequences

- MarkReady's ACTIVE branch and its rejection semantics narrows to a
  single legal source (CREATING). Its doc-comment losesthe dual meaning.
- MarkIdle joins the aggregate with its own rejection event,
  sandbox.idle_rejected {From, To}, following the ADR-006 convention.
- No caller exists for MarkIdle yet. It earns its first caller in the
  exec choreography (next ADR, step: release after a run ends). Addint it
  now with no caller is justified only because it is the corrective half
  of an existing method's overload, not an imagined future method.
- Domain-only change. No service, route, or DTO touched.

## Deferred

Exec choreography (the first MarkIdle caller) and the failure surface
are the following two ADRs.
