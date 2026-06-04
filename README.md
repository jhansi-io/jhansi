# Jhansi.io

> **Build it. Run it. Ship it.**
>
> Cloud sandbox for running AI-generated code safely.

---

## The problem

Every AI coding tool generates code. None of them answer:

> *"Where does that code actually run, and what can it touch?"*

Paste it into your terminal and hope? That's `eval()` with access to your `.env`, your AWS creds, your customer database.

In a startup that's risky. In fintech that's an FCA conversation you do not want to have.

Jhansi.io is the answer.

---

## What it is

A cloud sandbox platform with three layers:

```
┌─────────────────────────────────────────────┐
│                  Jhansi                      │
│         Product · SDK · Cloud UI            │
└───────────────────┬─────────────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
┌───────▼────────┐   ┌──────────▼──────────┐
│     Petri      │   │     TenantVault      │
│ Execution      │   │  Secrets · Audit     │
│ Engine         │   │  Layer               │
└────────────────┘   └─────────────────────┘
```

| Layer | What it does |
|---|---|
| **Jhansi** | SDK, cloud UI, and the API your app talks to. This repo. |
| **Petri** | Runs code in isolated Docker containers. Spins up, executes, tears down. Zero state left behind. |
| **TenantVault** | Per-tenant secrets injection. AI agents can use secrets but can't read or exfiltrate them. |

---

## Repositories

| Repo | Status | Description |
|---|---|---|
| [jhansi-io/jhansi](https://github.com/jhansi-io/jhansi) | 🟢 Active | SDK — `pip install jhansi` |
| [jhansi-io/petri](https://github.com/jhansi-io/petri) | 🟢 Active | Execution engine — FastAPI, Docker isolation, multi-language |
| `jhansi-io/tenantVault` | 🔜 Coming soon | Secrets management — envelope encryption, per-tenant KMS |

---

## Install

```bash
pip install jhansi
```

---

## Quick start

```python
from jhansi import Sandbox

with Sandbox(language="python") as sb:
    result = sb.exec("print('hello from isolation')")
    print(result.output)
```

> SDK is under active development. Follow this repo for releases.

---

## Petri API

```
POST   /v1/sandboxes            → Create sandbox, get sb_<id>
POST   /v1/sandboxes/{id}/exec  → Run code, get output
GET    /v1/sandboxes/{id}       → Check status
DELETE /v1/sandboxes/{id}       → Destroy it. Gone.
```

Multi-language: `python` · `node` · `go`

---

## Business model — Open Core

| Tier | Details |
|---|---|
| **Open Source** | Self-hosted, Apache 2.0. Run it in your own VPC. |
| **SaaS — Pro** | Managed cloud. No infra to run. |
| **SaaS — Enterprise** | SOC2, TenantVault, audit streaming, SSO, DPA. |

---

## Roadmap

Full roadmap with target dates: **[jhansi.io/roadmap](https://jhansi.io/roadmap)**

No vaporware. If it's not on the roadmap with a date, we're not building it yet.

---

## Why Jhansi.io?

- Competitors (E2B, Modal, Daytona) are SaaS only — Jhansi.io is open core, self-hostable
- Nobody else combines execution sandbox + secrets vault + compliance audit in one platform
- Built by someone with 18 years in banking who has lived the compliance problem firsthand
- Regulation (FCA, SOC2, EU AI Act) requires audit of AI actions — Jhansi.io provides it

---

## Follow the build

Building in public. Technical deep-dives on [Dev.to](https://dev.to) and the [Jhansi.io blog](https://jhansi.io).

Star this repo to follow along. ⭐

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
