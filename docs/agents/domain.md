# Domain Docs

How the engineering skills consume this repo's domain documentation. Single-context layout.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root — the shared language: three-plane ontology, vocabulary, roles and states, hard constraints.
- **`hourglass-vault/decisions/`** — the ADRs (`ADR-BE-001 … ADR-BE-018`, index at `decisions/backend/_index.md`). Read the ADRs that touch the area you're about to work in. There is no `docs/adr/` in this repo; do not create one.
- **`docs/codebase/`** — architecture, structure, stack, conventions, testing, concerns (the current-state map, not decisions).

For any `web/src` change to UI, tokens, components, copy, or layout, also open `docs/design/INDEX.md` first (repo rule in `AGENTS.md`); `docs/design/LANGUAGE.md` is the authority on design vocabulary.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. `/domain-modeling` (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates `CONTEXT.md` entries and ADRs lazily when terms or decisions actually get resolved.

## File structure

```
/
├── CONTEXT.md                     ← shared language (root)
├── docs/
│   ├── agents/                    ← this skill-set's config (tracker, labels, domain)
│   ├── codebase/                  ← current-state codebase map
│   ├── design/                    ← design authority stack (INDEX → LANGUAGE → CHROME → workflows)
│   └── history/planning/          ← frozen pre-migration records
└── hourglass-vault/
    ├── decisions/backend/         ← ADRs (ADR-BE-NNN — <title>.md) + _index.md
    ├── 01-Features/               ← feature specs (F05…F13)
    ├── 03-Schema/                 ← schema/domain-model/API/state-machine docs
    └── research/                  ← dated research notes
```

## Naming new ADRs

Follow the vault convention: `hourglass-vault/decisions/backend/ADR-BE-NNN — <Title>.md`, next free number, and add the entry to `_index.md`. ADRs written mid-conversation by `/grill-with-docs` or `/domain-modeling` follow the same shape as the existing files.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids (e.g. "project" for an activity, "budget" for coverage, "task" for a ticket).

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-BE-017 (coverage encoding), but worth reopening because…_
