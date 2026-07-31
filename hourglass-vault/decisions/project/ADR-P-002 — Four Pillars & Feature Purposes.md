# ADR-P-002 — Four Pillars & Feature Purpose Assignment

---
tags: ["adr", "idea-layer", "vision"]

---

# ADR-P-002 — Four Pillars & Feature Purpose Assignment

**Status:** Proposed (rows revised by [[ADR-P-007 — Activity Ontology]], 2026-07-29)
**Date:** 2026-07-28
**Operationalizes:** [[VISION]] §4–§5

## Context

Before the restructure, the vault described features by mechanics (tables, endpoints, CRUD) with no stated purpose. That made every new idea look equally valid and removed any test for rejecting one. [[VISION]] introduced four pillars; this ADR ratifies the pillar model and fixes each existing feature's pillar and purpose as the reference all feature docs cite.

## Decision

Ratify the four pillars — **Capture → Structure → Control → Insight** — as the organizing model for all features. Each feature belongs to exactly one pillar and carries a one-sentence purpose tied to the three questions.

The assignment below is binding (mirrors [[VISION]] §5):

| Feature | Pillar | Purpose |
|---------|--------|---------|
| Time entries | Capture | The atomic unit of "what work actually happened" |
| Expenses | Capture | The atomic unit of "what the work cost beyond hours" |
| Org hierarchy / units | Structure | Who is accountable for whom — approval routing, visibility |
| Working groups | Structure | Who actually works together right now (execution; see [[ADR-P-001 — Units vs Working Groups]]) |
| Activities | Structure | The recursive container work belongs to, at any granularity — makes time attributable (replaces *Projects* per [[ADR-P-007 — Activity Ontology]]) |
| Contracts | Structure | The economic boundary of work — optionally linked to activities, inherited downward; economics only, never work decomposition |
| Approval workflows | Control | Converts captured time into *trusted* time |
| Governance models | Control | Per-activity definition of whose approval counts |
| Invitations / bootstrap | Control | Controlled entry into an org's structure |
| Adoption (shared resources) | Control | Reuse of standard work definitions (activities) across orgs |
| Exports | Insight | Today's primitive insight — proof captured data is queryable and meaningful |

## Consequences

* Every feature doc declares its pillar + purpose at the top (enforced by `01-Features/_TEMPLATE.md`).
* New ideas are slotted into one pillar; an idea that fits none fails the steering test.
* Insight is acknowledged as the under-developed pillar — the post-v0.1 roadmap concentrates there (V1–V6).
* A feature moving between pillars requires a vision revision, not just a doc edit.

## Related

* [[VISION]] (governing document)
* [[ADR-P-001 — Units vs Working Groups]]
* `01-Features/_TEMPLATE.md`
