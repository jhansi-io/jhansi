# jhansi.io

> **Build it. Run it. Ship it.**
>
> Run AI-generated code in isolated sandboxes. Open source. Self-hostable.

---

## The problem

Every AI coding tool generates code. None of them answer:

> *"Where does that code actually run, and what can it touch?"*

Paste it into your terminal and hope? That's `eval()` with access to your `.env`, your AWS creds, your customer database.

In a startup that's risky. In fintech that's an FCA conversation you do not want to have.

jhansi.io is the answer.

---

## What it is

A sandboxed execution platform with three layers:

| Layer | Status | What it does |
|---|---|---|
| **jhansi SDK** | 🟢 Available | Python SDK — `pip install jhansi` |
| **Petri** | 🟢 Available | Execution engine — Docker isolation, multi-language, self-hostable |
| **TenantVault** | 🔜 Coming soon | Per-tenant secrets injection, envelope encryption, audit log |

---

## Quick start

```bash
# Start Petri
git clone https://github.com/jhansi-io/petri.git
cd petri
docker compose up

# Install the SDK
pip install jhansi
```

```python
from jhansi import Sandbox

with Sandbox(language="python") as sb:
    sb.upload_file("main.py")
    result = sb.exec("python main.py")
    print(result["output"])
```

Full docs at [docs.jhansi.io](https://docs.jhansi.io).

---

## Repositories

| Repo | Description |
|---|---|
| [jhansi-io/jhansi](https://github.com/jhansi-io/jhansi) | SDK — `pip install jhansi` |
| [jhansi-io/petri](https://github.com/jhansi-io/petri) | Execution engine — FastAPI, Docker isolation, multi-language |

---

## Why jhansi.io?

- Competitors (E2B, Modal, Daytona) are SaaS only — jhansi.io is open core, self-hostable
- Nobody else combines execution sandbox + secrets vault + compliance audit in one platform
- Built by someone with 18 years in banking who has lived the compliance problem firsthand
- Regulation (FCA, SOC2, EU AI Act) requires audit of AI actions — jhansi.io provides it

---

## Roadmap

Full roadmap: [jhansiio.featurebase.app](https://jhansiio.featurebase.app)

---

## Follow the build

Building in public. Technical deep-dives on [Dev.to](https://dev.to/thearun85).

Star this repo to follow along. ⭐

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
