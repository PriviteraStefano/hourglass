# ADR-P-005 — Insight Pillar Roadmap (V3–V6)

---
tags: ["adr", "idea-layer", "insight", "roadmap"]

---

# ADR-P-005 — Insight Pillar Roadmap (V3–V6)

**Status:** Proposed
**Date:** 2026-07-28
**Operationalizes:** [[VISION]] §6 · **Depends on:** [[ADR-P-003 — Tickets as the Second Capture Layer|ADR-P-003]], [[ADR-P-004 — The Today View|ADR-P-004]]

## Context

v0.1 ships Capture + Structure + Control; Insight holds only exports. The vision features V3–V6 all live in Insight (plus Structure enrichment), and all tempt simultaneous development — analytics, knowledge profiles, finance dashboards all sound equally "next". Building them in parallel would scatter effort across four features that have hard data dependencies on each other.

## Decision

Sequence the Insight pillar strictly by data dependency — each step requires the data maturity of the previous one:

| Order | Feature | Pillar | What it answers | Hard dependency |
|-------|---------|--------|-----------------|-----------------|
| 1 | **V1 Today view** + **V2 tickets** | Insight + Capture | What should I be working on? | v0.1 stable (deployed) |
| 2 | **V3 Employee knowledge** | Structure→Insight | Who is good at what, and how loaded are they? | Accumulated entry history; ADR-P-001 |
| 3 | **V4 Project knowledge + change requests** | Structure + Control | What is this project's living state? | V2 (change requests are tickets) |
| 4 | **V5 Contract pricing analytics** | Insight | What did similar work actually cost? | Sufficient history across contracts |
| 5 | **V6 Real-time project finance** | Insight | Is this project burning as planned? | Budget fields on contracts; dashboard infra from V5 |

Enforcement rules:

* **No V(n) SPEC until V(n−1) is deployed** — not merged, deployed. Analytics on immature data produces confident nonsense.
* **Shared dashboard infrastructure is extracted at V5, not before** — V1's Today view deliberately avoids charts so no premature dashboard framework is built.
* Each step's SPEC must name the data it consumes and prove that data exists in sufficient volume (a measurement, not an assumption).

## Consequences

* Insight work is never parallelized against its own dependencies — one deep feature at a time.
* The roadmap gains an explicit "data maturity gate": features are blocked by evidence, not enthusiasm.
* V3/V4 enrich Structure *in service of* Insight — they are not standalone profile/wiki products (see [[VISION]] §8).
* ⚠️ Contract/project schema will need budget/target fields before V6 — deferred until V4's design, not guessed now.

## Related

* [[VISION]] §6 (dependency chain), §7 (steering test)
* [[ADR-P-001 — Units vs Working Groups]] (V3's home)
* [[ADR-P-004 — The Today View]] (step 1)
