# ADR-P-008 — Availability & Employment Validity

---\ntags: ["adr", "idea-layer", "structure", "staffing"]\n---

# ADR-P-008 — Availability & Employment Validity: Who Can Take This Work

**Status:** Proposed
**Date:** 2026-07-29 (proposed) · 2026-07-29 (revised: typed absence windows + holiday confirmation + payroll export)
**Operationalizes:** [[VISION]] §4 (Structure), §6 (V3) · **Sharpens:** [[VISION]] §8 HR anchor (carve, not removal) · **Carves:** the 2026-07-29 entries in [[ADR-P-006 — Out-of-Scope Enforcement]] · **Feeds:** V3 employee knowledge profile, the payroll export (the wages link for Finance + HR)

---

## Context

The blanket rejection of "work permits & holidays" (ADR-P-006 log, 2026-07-29) lumped two different things together: the **question** — *"is this person available / legally employed?"* — and the **machinery** — leave balances, request/approval workflows, permit document storage.

Stefano's design-partner case: **a manager adding people to activities must be able to check availability and employability at assignment time.** Assigning work to someone who is away, or whose engagement ends mid-activity, silently breaks plans — that serves "is the work on track?" and passes the §7 belonging test. The machinery does not: it answers HR's questions, not the product's three.

## Decision

### D-1 — Absence windows are typed; absence is captured, routed, exported — never accounted

`availability_windows` carry a **`kind`: `holiday` / `permit` / `medical` / `unavailable`** (the generic catch-all from the original proposal). Kinds are required because **wages treat absences differently** — ferie, permessi and malattia are paid differently, so payroll needs the classification, not just "away". Note the GDPR correction to the earlier draft: recording the *kind* and dates of an absence is ordinary lawful employer processing; special-category data is medical *content*, which Hourglass never stores (D-5).

The boundary that keeps this from becoming an HR product — **absence is captured and routed here; it is never *accounted* here:**

* **No balances, no accruals, no carry-over, no entitlement counters, no "days remaining" display.** Payroll computation happens outside (§8 payroll anchor). The payroll link is satisfied by **export** (D-1c): approved entries *plus* confirmed absence windows, aligned to cutoff periods — the monthly input a payroll provider/consulente consumes.
* Declared by the person, their unit manager, **or the org's HR role** (D-4). Partial-day permits are supported via optional `hours`.
* Data quality is declaration-of-intent. Accepted: staffing needs intent, not legal truth.

### D-1a — Confirmation routing per kind (the employee → HR channel, without building "communications")

| Kind | Confirmation | Documentation | Payroll feed |
|------|-------------|---------------|--------------|
| `holiday` | **Both lines confirm:** the unit manager (accountability, ADR-P-001) **and** one WG manager from the person's active working groups (execution). Both must confirm for the window to enter the export. | none | paid leave days |
| `permit` | **None** — record only (no WG formation around a permit) | none | paid hours (`hours` column) |
| `medical` | **None** — it is notification, not a request | **`certificate_ref`** — the INPS certificate protocol number (free-text reference, **never the document**), visible to `hr` + the unit manager only | sick-pay days |
| `unavailable` | None | none | unpaid/absent |

This is the same routing doctrine as ADR-P-001, applied to a different object: holiday touches **accountability** (unit manager) *and* **execution** (the WG whose work the absence displaces); medical and permit are facts to record, not work to approve. HR confirms nothing and consumes everything (D-4) — the channel is *data*, not messages, so §8's communication anchor is respected.

### D-1b — One window ↔ one wage code

Each window maps to exactly one payroll treatment via its `kind`. A `holiday` window that is half vacation / half sickness is **two windows** — the export stays a flat `(person, date range, kind, hours)` table, which is what payroll systems and consulenti actually consume.

### D-1c — The payroll export

The v0.1 **Exports** feature gains a payroll view: approved, cutoff-locked entries **plus** confirmed absence windows (kind, dates, hours, `certificate_ref` for medical), per person per period. This is the wages link for both Finance and HR — Hourglass produces *trusted data*; the payslip is produced elsewhere (§8: "Hourglass produces *trusted data* for those systems; it does not become them").

### D-2 — Employment validity = status dates, not document management

`organization_memberships` gains **`valid_from` / `valid_until`** (nullable = open-ended) — "is this person engaged with us right now" — plus optional **`work_permit_expires_at`** for non-EU employees and contractors. **No document storage**: no scans, no certificate tracking, no renewal workflows. Status and dates only.

### D-3 — Surfaced at assignment time only

The single consumption point in this scope: **forming a WG or assigning someone to an activity** surfaces warnings — *"away 10–21 Aug"*, *"membership ends 1 Sep"*, *"permit expires 15 Aug"*. Read-only elsewhere. Explicitly **not** used to block time-entry submission: reality is messy (people work during declared absence), and blocking creates exceptions and meta-work (VISION §2).

### D-4 — HR is in the loop as curator and consumer, never as approver

A new **`hr` org role** (on `organization_memberships`, alongside employee/manager/finance/customer) brings HR into Hourglass directly — as the natural *maintainer* of this data, not as a workflow stage:

* **Curator:** HR can record and correct availability windows and validity/permit dates **for anyone in the org**. This upgrades D-1/D-2 data from "declaration of intent" toward "maintained record" — the single biggest data-quality lever available, and the thing that genuinely *eases HR's work* (staffing answers become self-service for managers instead of questions routed to HR).
* **Consumer:** org-wide read of windows and validity dates (same visibility scope as the org-role `manager`/`finance`, R-4), plus the natural *maintainer surfaces*: window lists per unit/person, expiry queues ("which memberships/permits expire in the next 60 days"), and **exports of approved, cutoff-locked entries** for payroll prep — trusted data *for* HR/payroll systems, exactly the vision's framing (§8 payroll anchor).
* **Never an approver:** HR is **not** a stage in entry routing — BE-014's two-stage chain (manager → finance) is untouched. The moment HR approves entries, Hourglass starts absorbing the HR workflow; that is the machinery D-5 keeps out.
* HR holds **no leave-management powers in Hourglass**: requests, approvals, balances, accruals, documents all remain rejected (D-5). The `hr` role exists to maintain *structure data*, not to run absence administration.

### D-5 — What stays rejected (reaffirmed)

**Balances, accruals, carry-over, entitlement counters, "days remaining" displays** · **medical documentation** (certificate uploads, diagnoses, illness sub-types — only the `certificate_ref` protocol number is stored) · **permit document storage** · **HR as an approval stage on entries** · **payroll computation** and anything payslip-adjacent · **multi-step HR approval chains** beyond the D-1a confirmation routing. These remain in the ADR-P-006 rejection log under the §8 HR anchor, reopenable only via vision revision.

*(Absence **kinds** were removed from this list 2026-07-29 — payroll needs them; see revision log. The earlier "no leave-type taxonomy" position is reversed.)*

### Schema sketch (additive, zero-coupled to the activity ontology)

```sql
CREATE TABLE availability_windows (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id),
    user_id        UUID NOT NULL REFERENCES users(id),
    kind           VARCHAR(20) NOT NULL DEFAULT 'unavailable'
                   CHECK (kind IN ('holiday','permit','medical','unavailable')),  -- D-1
    starts_on      DATE NOT NULL,
    ends_on        DATE NOT NULL CHECK (ends_on >= starts_on),
    hours          DECIMAL(4,2),                    -- D-1: partial-day permits
    certificate_ref VARCHAR(100),                   -- D-1a: medical only, INPS protocol nº; hr + unit mgr visibility
    note           TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'declared'
                   CHECK (status IN ('declared','confirmed')),  -- D-1a: holiday→confirmed when both lines confirm
    created_by     UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE organization_memberships
    ADD COLUMN valid_from DATE,
    ADD COLUMN valid_until DATE,              -- NULL = open-ended
    ADD COLUMN work_permit_expires_at DATE;   -- NULL = not applicable
-- organization_memberships.role CHECK extended: + 'hr' (D-4)
```

## Consequences

* **§8 sharpened, not weakened:** HR *accounting* machinery stays out; absence *capture + routing + export* comes in, and HR itself comes in as that data's curator (D-4). Boundary now explicit instead of vibes.
* **V3 gets its fourth profile attribute as an installment:** skills, current load, project history, **availability**. This ADR is the employee-knowledge profile's first landing.
* **Sequencing:** schema is one table + three columns + one enum value with no FK coupling to the activity rewrite — it **rides the big-bang migration batch** (ADR-P-007 D-6) without expanding its blast radius. Window surfacing at assignment and the payroll export land with their respective UI/export passes (v0.1 if pre-deploy, otherwise v0.2).
* ⚠️ **The scope magnet on this feature is the strongest in the product.** The slide windows → balances → accruals → CCNL rules is how this becomes an HR suite. D-1's "never *accounted* here" is the load-bearing sentence; any balance/counter/"days remaining" proposal fails the steering test on sight (ADR-P-006).
* ⚠️ **GDPR, correctly scoped:** kinds and dates are ordinary employer processing; `certificate_ref` is a reference number, not health content — but it is still visibility-restricted (D-1a). No medical documents, no diagnoses, no sub-types.
* ⚠️ **No enforcement coupling:** an entry logged during an absence window is legal data (people work during declared absence). Patterns matter to Insight later, not as a Capture constraint now.

### Revision log

| Date | Change | Reason |
|------|--------|--------|
| 2026-07-29 | Proposed (D-1…D-5, schema, HR role in D-4) | Availability/validity admitted as staffing structure data |
| 2026-07-29 | D-1 revised: absence kinds (`holiday`/`permit`/`medical`/`unavailable`) + `hours` + `certificate_ref`; D-1a confirmation routing (holiday → unit mgr + WG mgr; permit → none; medical → certificate ref); D-1b one-window-one-wage-code; D-1c payroll export; D-5 re-scoped | Stefano: wages treat absences differently and HR needs the kind + holiday confirmation + medical documentation. GDPR objection to kinds was mis-scoped (kinds/dates = ordinary processing; only medical *content* is special-category) |

## Related

* [[VISION]] §4 (Structure), §6 (V3), §8 (sharpened anchor)
* [[ADR-P-006 — Out-of-Scope Enforcement]] (the carve this resolves)
* [[ADR-P-007 — Activity Ontology]] (the assignment target this feeds)
* [[ADR-P-001 — Units vs Working Groups]] (unit manager = who declares/validates for their subtree)
