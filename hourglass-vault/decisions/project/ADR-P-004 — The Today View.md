# ADR-P-004 — The "Today" View

---
tags: ["adr", "idea-layer", "insight", "today"]

---

# ADR-P-004 — The "Today" View

**Status:** Proposed
**Date:** 2026-07-28
**Operationalizes:** [[VISION]] §6 V1 · **Depends on:** [[ADR-P-003 — Tickets as the Second Capture Layer]] *(softened 2026-07-30 by [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-2: a ticketless Today ships pre-deployment as the landing surface — approvals/entries composition only; tickets join at v0.2)*

## Context

The first of the three questions — *"What should I be working on?"* — currently has no answer in the product. v0.1's home route redirects to `/time-entries`, a recording surface. There is no screen that answers "what now" for an individual.

This is the defining feature of the Insight pillar: without it, Hourglass is a place work gets *recorded*, not a place that *directs* work.

## Decision

Build a **Today view** as the authenticated landing surface — one screen answering "what should I be working on?" from already-captured data:

* **My open tickets by priority** (from ADR-P-003) — the demand assigned to me
* **My pending approvals** (if I'm a manager/finance) — work blocked *on me*
* **My draft/submitted entries this week** — capture state at a glance
* **Fallbacks when empty** — the view must never be blank: no tickets → suggest next actions (submit drafts, review pending); nothing at all → onboarding pointers

Design rules:

* **Read-only composition** — the view introduces no new state; it composes existing queries. It is pure Insight over Capture + Control data.
* **One answer, not a dashboard** — it ranks "what now"; it does not show charts, KPIs, or analytics (those belong to V5/V6 dashboard infrastructure).
* **Replaces the `/` redirect** — landing becomes Today, not time-entries.

## Consequences

* Question #1 of the vision gets a concrete home.
* Establishes the Insight pillar's first real feature and its design constraint: insight composes, never duplicates, captured state.
* The empty-state fallbacks make the product self-directing for new users (directly attacks "what am I working on today?").
* ⚠️ Landing-route change affects `_authenticated/index.tsx`; low risk, but E2E expectations that `/` → time-entries must be updated.

## Related

* [[VISION]] §3 Q1, §6 V1
* [[ADR-P-003 — Tickets as the Second Capture Layer]] (primary data source)
* Future: V5/V6 dashboard infrastructure (this view must not grow into it)
