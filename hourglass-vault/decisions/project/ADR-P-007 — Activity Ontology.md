# ADR-P-007 — Activity Ontology

---

tags: ["adr", "idea-layer", "structure", "capture", "ontology"]

---

# ADR-P-007 — Activity Ontology: One Recursive Work Entity

**Status:** Accepted
**Date:** 2026-07-29 (accepted) · 2026-07-29 (revised: D-7 billability, D-8 personal activities)
**Operationalizes:** [[VISION]] §4 (Capture, Structure) · **Basis:** [[research/2026-07-28 — Work Ontology — Full Picture & Revision Proposal]] · **Revises rows of:** [[ADR-P-002 — Four Pillars & Feature Purposes]] · **Implemented by:** ADR-BE-014 (routing) + schema migration

---

## Context

The 0.1.0 MVP built work decomposition as **two tables, one concept**: `projects` and `subprojects` have the same shape (name, description, governance_model, created_by_org_id, is_shared, adoption tables, is_active) at exactly two depths. The full-capture chain was rigid — logging one hour required `Customer → Contract → Project → Subproject → WorkingGroup` to exist, with all four FKs `NOT NULL` on `time_entries` — while expenses linked asymmetrically (nullable `project_id`/`customer_id`, **no `wg_id`**). Five ontology problems (O1–O5) are documented with code citations in the research note. The accepted [[ADR-P-001 — Units vs Working Groups]] decision ("expenses route like time") was **blocked by O3**: expenses cannot route by WG when no WG FK exists.

Stefano's framing settled the direction: *"projects are activities — we may need a better ontology."*

## Decision

**Work is one recursive entity: the `Activity`.** Projects and subprojects are replaced by a single `activities` table. This ontology **ships in v0.1** — the revision lands before first deploy; there is no "deploy rigid-chain v0.1 first" step.

The three concerns the old chain mashed together are separated:

| Concern | Question | Entity |
|---------|----------|--------|
| **Commercial** | What is the work worth, to whom? | Customer, Contract (unchanged role) |
| **Work** | What is there to do, at any granularity? | **Activity** (new) |
| **Execution** | Who does it, who approves? | Working Group (re-anchored to Activity) |

### D-1 — Naming: `activity`

The table, the domain type, and the user-facing word are **activity** (`activities`). Not `work_item`; the legacy word "project" survives only in `project_managers` (governance role name, addressed in Consequences) and historical references.

### D-2 — Structure: recursion without level semantics

* `activities.parent_id` is **nullable and self-referencing** — nesting is available when work is genuinely nested, never required. Flat is the default; depth is data, not schema.
* **`kind` is a free label from an org-level catalog** (`activity_kinds` table), seeded with `engagement`, `phase`, `task`, `internal`. Kinds carry **no inherent level, ordering, or depth semantics** — a kind does not say *where* an activity sits, only *what sort of work* it is. Orgs extend the catalog with their own kinds.
* There is **no level ladder** and no constraint between kind and depth. (Rejected alternative: a closed `CHECK (kind IN (...))` enum with fixed level meaning — that's the two-table rigidity re-born as a constraint.)

### D-3 — Commercial context is optional and inherited downward

* `activities.contract_id` is **nullable**. Activities without a contract are **internal work, first-class** — training, pre-sales, events, helping a colleague. No fake customer/contract scaffolding to log them.
* Commercial context is **derived, not stored**: an entry's commercial chain is resolved by walking `parent_id` upward to the nearest ancestor with a `contract_id` (recursive CTE, the same pattern as the units subtree). Descendants inherit; nothing is denormalized onto children.

### D-4 — Capture links through one required FK

* **Both `time_entries` and `expenses` reference exactly one `activity_id`, and it is `NOT NULL` on both.** The pointed-to activity may sit at **any** depth — coarse or fine, the chain upward is derived (D-3).
* `time_entries`: the four-FK chain (`project_id`, `subproject_id`, `wg_id`, `unit_id` all required) collapses to `activity_id` (required) + `unit_id` (required, accountability pin per ADR-P-001). `wg_id` is **dropped as a column** — the WG is resolved via the activity's anchored WG (D-5), not pinned on the entry.
* `expenses`: `activity_id` becomes **required**; `customer_id` is **dropped** — the customer is derived through the activity's contract. Symmetric with time, per the originally accepted ADR-P-001 Q1 ("expenses match time entries").
* Required-activity is what makes "expenses route like time" implementable: **every entry routes through its activity → anchored WG → manager/delegate**, with **no fallback path** (the no-activity approval fallback is eliminated by making the state unrepresentable — see D-3: an `internal` activity always exists for non-commercial work).

### D-5 — Working groups anchor at any depth, when a team exists

`working_groups.activity_id` replaces `working_groups.subproject_id`. A WG anchors to an activity at any level — engagement-level team, phase-level squad, task-level pair. Team formation is no longer hostage to work-breakdown depth (O4). **A WG is the execution structure for group work** — it is not mandatory on activities where the "team" is one person (see D-8).

### D-6 — Migration: big-bang, pre-deploy

One migration replaces `projects`/`subprojects` with `activities` (+ `activity_kinds`, + `activity_adoptions`), rewrites all FKs (`time_entries`, `expenses`, `working_groups`, `project_managers` → `activity_managers`, `financial_cutoff_periods`), and drops `enforce_unit_tuple` (already decided in ADR-P-001 Q3). **Data migration is trivial** (MVP seed only) — existing projects become activities with the seeded `engagement` kind, subprojects become children. Per ADR-BE-004, applied history is immutable: this lands as **new migration files**, not edits to `000_full_schema.up.sql`.

### Table sketch (reference shape)

```sql
CREATE TABLE activity_kinds (                       -- D-2: free catalog, org-extensible
    org_id   UUID NOT NULL REFERENCES organizations(id),
    name     VARCHAR(50) NOT NULL,                  -- 'engagement','phase','task','internal', + org's own
    is_seed  BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (org_id, name)
);

CREATE TABLE activities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(id),
    parent_id        UUID REFERENCES activities(id) ON DELETE RESTRICT,   -- D-2: nullable, no level meaning
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    kind             VARCHAR(50) NOT NULL,          -- FK to activity_kinds (catalog), NOT a CHECK enum
    contract_id      UUID REFERENCES contracts(id) ON DELETE RESTRICT,    -- D-3: nullable = internal work
    governance_model VARCHAR(50) NOT NULL DEFAULT 'creator_controlled'
                     CHECK (governance_model IN ('creator_controlled','unanimous','majority')),
    created_by_org_id UUID NOT NULL REFERENCES organizations(id),
    is_shared        BOOLEAN NOT NULL DEFAULT FALSE,
    billable         BOOLEAN,                       -- D-7: NULL = inherit from contract link / nearest ancestor
    budget_amount    DECIMAL(12,2),
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- activity_adoptions mirrors project_adoptions (sharing preserved)
-- working_groups.activity_id replaces subproject_id (D-5)
-- time_entries.activity_id NOT NULL; wg_id column dropped (D-4)
-- expenses.activity_id NOT NULL; customer_id column dropped (D-4)
-- financial_cutoff_periods.activity_id replaces project_id
```

*(Billability defaults by activity class — D-7/D-8)*

| Class | `contract_id` | Default `billable` | WG | Examples |
|-------|---------------|--------------------|----|----------|
| **Project activity** | set | **billable** | anchored (team exists) | engagement, phase, task under a contract |
| **Internal group activity** | NULL | **non-billable** | anchored (group executing) | briefing, teaching session, company event |
| **Personal activity** | NULL | **non-billable** | **none** (one person) | learning, certifications, self-study |

### D-7 — Billability is a property of the work, defaulted by commercial link

An activity is **billable or non-billable**, independent of its kind. Rationale: kind says *what sort of work* it is (free catalog, D-2) — billability says *whether a customer pays*, and the same kind can fall on either side (a `phase` under an internal R&D engagement vs. under a customer contract; a `task` that is warranty work).

* **Default follows the contract link:** activities under a contract default to billable; activities with no contract (internal, personal) default to non-billable.
* **Nullable tri-state on the row:** `billable NULL` = inherit from the nearest ancestor that sets it, or from the contract link default; an explicit `TRUE`/`FALSE` **overrides** and cascades to descendants that don't override.
* **Not on the entry.** Time entries and expenses carry no billable flag — the invoicing question is asked of the *work* ("how much of this contract is billable?"), and entry-level flags rot the moment work is reclassified. Reclassifying an activity must reinterpret its history, not strand it.
* This adds the missing piece to [[VISION]]: contracts "know what they're worth" (§2) and approval makes data "usable for billing" (§5) both presupposed a billability property that was never stated. It now exists, on the activity, where the commercial boundary lives.

### D-8 — Personal activities: WG optional, accountability via the unit

An activity whose execution is **one person** (learning, certifications, self-study, individual research) **does not anchor a working group.** Forcing a one-person "team" is meta-work — the exact thing the product exists to remove (VISION §2).

* **Approval routing falls back to the unit tree** — the submitter's unit manager (per `unit_memberships.role = 'manager'`, the same role ADR-P-001 already gave subtree *visibility* over these entries). The approver is therefore the person who already sees the entry — consistent, and no self-approval by construction.
* **This is not a hole in "expenses route like time."** Both entry types resolve through the same precedence chain (activity → WG if anchored → unit manager otherwise → finance); personal activities simply take the second step's fallback. Symmetry is preserved.
* **No personal-activity flag.** "Personal" is not a `kind` and not a column — it is *observed* (an activity with no anchored WG), keeping the kind catalog free (D-2) and the schema honest. An `internal` kind + no WG *is* a personal activity.
* **Information completeness is structural, not flag-based** (Stefano: "keep all the possible informations about it"). Every entry always pins: **who** (owner), **what** (`activity_id` → its kind, its chain, its contract-or-not, its billability), **accountability** (`unit_id`), **when** (dates), **how much** (hours/amount), **who approved** (audit). Personal vs. group changes *who approves*, never *what is captured*.
* **Lifecycle note:** a personal activity can grow a WG later (a study group becomes a teaching session) — anchoring is additive, no data rewrite.

## Problem resolution map

| # | Problem (research note §1.3) | Resolution |
|---|------------------------------|------------|
| O1 | Project/subproject = one concept, two tables | One recursive `activities` (D-1, D-2); one repo, one endpoint set |
| O2 | Rigid 4-FK capture chain | One `activity_id`; internal work needs no commercial scaffolding (D-3, D-4) |
| O3 | Time/expense link asymmetrically; expenses can't route by WG | Both entry types link identically; `wg_id` resolved via activity (D-4, D-5) — **unblocks ADR-P-001 Q1** |
| O4 | WG anchored to deepest level only | WG anchors to any activity depth (D-5) |
| O5 | Commercial/work boundaries entangled | Contract = economics only; Activity = work only; relation via nullable `contract_id`, inheritance derived (D-3) |

## Consequences

* **Capture gets easier** (pillar: Capture): one reference instead of a four-FK chain — directly serves "remove meta-work."
* **Internal work is first-class** (pillar: Structure): an `internal`-kinded activity with no contract is a normal citizen, not a workaround.
* **Routing simplifies** (pillar: Control): every entry resolves through one activity chain; the BE-014 fallback question (research §2.6) is **closed by construction** — no-activity entries cannot exist.
* **Insight improves** (pillar: Insight): V5 pricing analytics compares activities by kind/contract/customer with real depth, not the flattened two-level model.
* ⚠️ **`project_managers` is renamed `activity_managers`** (references `activity_id`). The *word* "project" survives in the governance role name for now; a rename of the governance concept itself is a separate, cosmetic decision — not v0.1-blocking.
* ⚠️ **Every layer built on projects/subprojects is touched in one sweep**: repositories (two collapse into one), endpoint sets (two collapse into one), domain types, seed data, `IsPeriodLocked` (key changes org+project → org+activity), frontend routes/types. Big-bang is accepted *because* pre-deploy makes the blast radius seed-data-only.
* ⚠️ **Cycle safety:** `parent_id` recursion needs cycle prevention (path check on insert/update) — implementation detail for the migration/service, not an idea-layer question.
* ⚠️ **ADR-P-002 rows change** (done in the same commit): *Projects* → *Activities*; *Contracts* row notes the activity linkage; *Adoption* now applies to activities.

### Revision log

| Date | Change | Reason |
|------|--------|--------|
| 2026-07-29 | Accepted (D-1…D-6) | Stefano's five calls on the research note's Part 3 |
| 2026-07-29 | +D-7 (billability), +D-8 (personal activities); D-5 amended (WG "when a team exists") | Stefano's probe: "what if an activity is personal — learning, briefing?" exposed (a) the unstated billability property the vision already assumed, (b) that WG-anchoring can't be universal without forcing one-person teams |

## Resolved-by (implementation)

* **ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution** (routing through the activity chain, subtree visibility, D-11 skip, `enforce_unit_tuple` drop).
* **Schema migration** per D-6 (new files per ADR-BE-004; includes the `enforce_unit_tuple` drop).

## Related

* [[VISION]] §4 (Capture, Structure), §5
* [[ADR-P-001 — Units vs Working Groups]] (units/WG split, expense-routing decision this unblocks)
* [[ADR-P-002 — Four Pillars & Feature Purposes]] (rows revised by this ADR)
* [[ADR-BE-004 — Database Migrations]] (migration mechanics)
* [[research/2026-07-28 — Work Ontology — Full Picture & Revision Proposal]] (evidence base, O1–O5)
* Code: `migrations/000_full_schema.up.sql`, `internal/core/domain/{project,subproject,time_entry,expense,working_group}/`
