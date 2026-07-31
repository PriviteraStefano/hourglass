# Hourglass Documentation Vault

This vault documents **why Hourglass exists, what we're building, and how it's built** — in that order.

**Start here:** [[VISION]] → [[00-Index]]

---

## The Order Matters

```
VISION  →  decisions/project/  →  01-Features/  →  03-Schema/
(why)      (what & why ADRs)      (user docs)      (contracts)
                  ↓
         decisions/backend/  →  02-Technical/
         (how-we-build ADRs)    (guides)
```

1. **[[VISION]]** — mission, the three questions, the four pillars, the steering test. Read first; cite often.
2. **`decisions/project/`** — idea-layer ADRs: what we build and *why*, pillar by pillar.
3. **`decisions/backend/`** — technical ADRs **specific to this repo**, recorded only where they differ from or extend the global knowledge vault (`knowledge/adr/`). The global vault is the main source — never duplicate it, link it.
4. **`01-Features/`** — user-facing docs; every feature declares its pillar and purpose.
5. **`02-Technical/`**, **`03-Schema/`** — implementation guides and design contracts.
6. **`legacy/`** — the previous vault, frozen. Reference only; do not extend.

## Rules

* A feature idea enters the roadmap only after passing the **steering test** in [[VISION]] §7.
* Feature docs follow `01-Features/_TEMPLATE.md` and declare a **pillar** and **purpose**.
* ADRs are append-only. Supersede via status + links, never rewrite silently.
* `legacy/` is read-only history. New work never goes there.
* Mermaid for all workflows/state machines.

## Automation

From the repo root:

```bash
./scripts/docs-check.sh           # completeness check
./scripts/validate-mermaid.sh     # diagram syntax
```
