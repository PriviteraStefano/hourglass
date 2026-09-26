# Research Summary — Ontology Extension (Origins, Tickets & Coverage + Direction)

**Date:** 2026-08-02
**Authoritative source:** `hourglass-vault/research/2026-08-01 — Origins, Tickets & Coverage — Ontology Extension Research.md` (Parts 1–15, decisions D-A … D-AA, all closed — no open questions)
**Status:** Research complete before milestone definition; this summary points at the vault note. Earlier research round (2026-07-28 work ontology) superseded by this one.

## Key findings

**Cardinal principle:** Captured effort is a fact. Coverage is a decision. Direction is a plan. The decision/plan never rewrites the fact.

**Three orthogonal planes:**
1. **Direction** (plan, mutable, manager/self-owned) — *before* the work: "what should you work on?"
2. **Facts** (time entries, immutable after approval) — *during*: "what did you work on?"
3. **Coverage** (money label, mutable, snapshot-protected) — *after*: "who pays for it?"

**Decisions that shape the build (full set D-A…D-AA in the note):**
- **Tickets**: first-class, internal-only; lifecycle open→triage→planned→in_progress→resolved→closed + reopen (D-A); kinds question/bug/change/evolution closed set (Q2); chain ticket→activity→entries (revised P-003); dismissal guard — blocked while linked activity has logged hours (D-M)
- **Origins**: type + reference set on activities, single fact at creation (D-D); proposal approval mirrors activity routing (D-G); refs stored directly on activities with additive derivation fallback from first direction record (R4)
- **Coverage**: allocation ledger with Σ = entry hours invariant (P-012); proposals computed on read, never stored (D-I); one-step manager confirm (D-L); editable indefinitely, cutoff = reporting snapshot not lock (D-F); snapshot mechanics backend-only (F)
- **Funding sources**: contract budget (default for billable) · support bucket (hours, carry-over, no expiry — D-P) · service request = zero-value contract (D-J) · internal absorption (mandatory reason) · cross-project transfer (explicit justification). Beneficiary unit collapsed with absorbing unit (D-B), nullable on activity, inherited downward
- **Direction**: one entity, mode derived (D-R); managers + self both first-class, self no approval (D-S); WG queued-only + claim model (D-T); lifecycle draft→active→superseded/cancelled, done/lapsed/claimed derived (D-V); per-day storage, partial days first-class (D-W/D-AA); org policy configurable (D-X); scheduler warns on P-008 absences, never blocks (D-Y); direction-coverage read-model (D-Z); plan-adherence aggregate-only (D-U)
- **Sold hours**: contracts carry `sold_hours` in v0.2 (D-N); V5 mines sold vs Σ actual

## Watch out for

- **The 4+4 split test**: coverage must never be applied by editing/duplicating time entries — that corrupts V5 analytics forever
- **Terminology collision**: "allocation" = coverage allocation (money label) vs direction (work direction) — P-012 drafts with "coverage allocation", never the bare word
- **Expense coverage is schema-ready only** (D-K): polymorphic entry, `time` only in v0.2 — cost the validation branch in the BE ADR, revisit if dead weight
- **Plan-adherence surveillance risk** (D-C/D-U philosophy): aggregate-only, per-period; the report is the control, not a block
- **No legacy-data migration** — pre-deploy; only append-only migrations per ADR-BE-004

## Build order (Part 15, agreed)

1. **Foundations** — schema + origins + ticket linkage (D-D, D-M, revised P-003 chain)
2. **Coverage** — the allocation loop (D-F, D-I, D-K, D-L, D-P, F)
3. **Direction** — the plan plane backend (D-Q…D-AA)
4. **Surfaces** — all UI, prototype-driven (4a: allocation screen/to-cover/own-coverage · 4b: buckets/per-unit report · 4c: Today both shapes · 4d: scheduler/queue/read-model)

ADRs drafted as phases land: P-012 (draft exists in vault, Proposed) → P-003 rev → P-013 → P-014 → P-015 → BE encoding. UI-last rationale: prototypes run against the complete backend with real data.

## Milestone impact (requirements mapping)

- 47 v0.2 requirements across 8 categories (FND/TICK/COV/DIR/AVAIL/UXFD/SURF/POLS) — see REQUIREMENTS.md
- 16 phases (11–26) — see ROADMAP.md
- Availability (AVAIL) kept from original v0.2 scope; UX polish phases retained trailing with UAT debt folded per page
