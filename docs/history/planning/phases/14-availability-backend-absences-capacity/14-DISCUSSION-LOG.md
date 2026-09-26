# Phase 14: Availability Backend — Absences + Capacity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-08
**Phase:** 14-availability-backend-absences-capacity
**Areas discussed:** Confirmation routing & approvers, Absence lifecycle & schema gap, Overlap rejection semantics, Capacity query contract, Read-visibility rules, Capacity fallback & validity filtering, Endpoint & role-gate shapes, Contract-type management

---

## Confirmation routing & approvers

| Option | Description | Selected |
|--------|-------------|----------|
| Full ADR routing per kind | holiday = unit manager AND WG manager; permit/medical/unavailable record-only | |
| One-step manager confirm/reject | any declared window confirmable by unit manager or HR | |
| Hybrid | auto-confirm record-only kinds, holiday requires confirmation | |
| **Free text** | **"holiday is Unit related, so only that manager. medical is record only"** | ✓ |

**User's choice:** Free text — holiday is unit-related, only that manager confirms; medical is record-only
**Notes:** Drops ADR-P-008 D-1a's WG-manager second confirm line.

Follow-up: all kinds need manager confirm — permit and unavailable confirmed like holiday; medical stays record-only. Manager resolution = unit-tree walk, self-confirm allowed. HR curates, never confirms. Medical requires certificate_ref at declare; employee provides it.

| Option | Description | Selected |
|--------|-------------|----------|
| Declare optional + HR editable | employee may provide certificate_ref optionally | |
| HR sets it exclusively | only HR sets the ref | |
| Required at declare | employee MUST provide the protocol number | ✓ |

| Option | Description | Selected |
|--------|-------------|----------|
| Required at declare, ref-only | protocol number only, document stays outside (ADR D-5) | |
| Required + store the document | store the image/pdf in Hourglass | ✓ |
| Optional, HR enriches | ref optional, HR enriches later | |

| Option | Description | Selected |
|--------|-------------|----------|
| DB-backed attachment table | PostgreSQL attachment table per window | ✓ |
| File storage + path column | disk/object store + path | |
| External link, no storage | URL only | |

**Notes:** The employee sends the document images/pdf of medical leave — required. This overrides ADR-P-008 D-5's no-document-storage boundary (GDPR flag noted). Storage = DB-backed (no object store in stack).

## Absence lifecycle & schema gap

| Option | Description | Selected |
|--------|-------------|----------|
| Terminal, new window to retry | rejected is terminal; new window to re-attempt | ✓ |
| Editable/re-submittable | rejected rows editable back to declared | |
| Terminal + withdraw option | terminal + employee withdraw before manager acts | |

| Option | Description | Selected |
|--------|-------------|----------|
| No withdraw; edit while declared | edit dates/kind while declared | |
| Withdraw declared allowed | withdraw (delete) own declared window | |
| Withdraw declared + cancel confirmed | plus manager-gated cancel of confirmed | |
| **Free text** | **"an employee can change its absence window by withdrawing the first one and creating a new one (that goes under the same flow as a normal one)"** | ✓ |

**User's choice:** Free text — withdraw the first window, create a new one through the normal flow

| Option | Description | Selected |
|--------|-------------|----------|
| Declared-only, status not delete | withdrawn terminal status, audit-logged | ✓ |
| Hard delete | no trace | |
| Also withdraw confirmed | confirmed windows withdrawable | |

| Option | Description | Selected |
|--------|-------------|----------|
| Only HR edits windows | HR curator correction power | |
| Nobody edits | no in-place edits at all | ✓ |
| HR + employee note edits | HR plus employee note/hours edits | |

| Option | Description | Selected |
|--------|-------------|----------|
| Medical withdrawable anytime | withdrawal as medical correction path | |
| Immutable, HR compensating window | HR creates compensating window | |
| HR may edit medical only | HR edits medical dates + certificate_ref directly | ✓ |

| Option | Description | Selected |
|--------|-------------|----------|
| Full in-tx audit trail | every window event audit-logged in-tx | ✓ |
| State transitions only | only confirm/reject/withdraw logged | |
| None this phase | no audit logging | |

**Notes:** Lifecycle = declared → confirmed/rejected/withdrawn; rejected and withdrawn terminal. Nobody edits except HR on medical (the medical correction path, since medical is record-only and never withdrawable).

## Overlap rejection semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Active-only, kind-blind | declared+confirmed count, withdrawn/rejected excluded | ✓ |
| All statuses count | terminal states still block overlap | |
| Same-kind only | overlap only among same kind | |

| Option | Description | Selected |
|--------|-------------|----------|
| Date-range only | range intersection; same-day partials count | ✓ |
| Partial windows composable | partials never conflict | |
| Hours-aware capacity check | partial vs full combined-hours check | |

| Option | Description | Selected |
|--------|-------------|----------|
| Service in-tx check | CR-01 pattern, no new extensions | ✓ |
| DB EXCLUDE constraint | btree_gist extension | |
| Both | service + DB backstop | |

**Notes:** Schema stores hours but no time-of-day → date-range-only overlap. Kind-blind: holiday overlapping medical still rejects.

## Capacity query contract

| Option | Description | Selected |
|--------|-------------|----------|
| Org setting × 5 days | planning_daily_hours × 5 workdays | |
| Own constant | availability's own 40h/week constant | |
| Per-employee hours | per-person weekly hours | |
| **Free text** | **"we should have different capacities... a smart move could be having a work week definition where we define: hours by month or by week, how many hours a day, how many hours a week/month, fixed days or dynamic"** | ✓ |

**User's choice:** Free text — work-schedule definition per employment contract (cadence, hours/day, hours/period, fixed vs dynamic days)

| Option | Description | Selected |
|--------|-------------|----------|
| Type table + membership override | contract_types template + per-employee day-hours override | ✓ |
| One table, shared/fork flag | work_schedules with shared flag + forks | |
| Type only, no overrides | employees inherit type exactly | |

**Notes:** Two-tier model confirmed — shared contract type + per-employee override for different days/hours on the same day ("the most complex case").

| Option | Description | Selected |
|--------|-------------|----------|
| Derived per-day | monthly hours ÷ working days in month | ✓ |
| Pattern + monthly ceiling | default pattern + total as ceiling | |
| Always explicit matrix | every monthly contract stores day matrix | |

| Option | Description | Selected |
|--------|-------------|----------|
| Subtree entries, per employee | recursive CTE, per-employee grouping | ✓ |
| Direct + one level | no deep recursion | |
| Approved entries only | submitted excluded | |

| Option | Description | Selected |
|--------|-------------|----------|
| Scope-param endpoint | one endpoint, scope params like Phase 13 D-13-25 | ✓ |
| Per-entity endpoints | separate per activity/WG | |
| Per-employee only | no aggregation server-side | |

| Option | Description | Selected |
|--------|-------------|----------|
| Confirmed only, declared advisory | confirmed subtracts, declared advisory field | ✓ |
| Confirmed + declared subtracted | conservative pessimistic view | |
| Confirmed only, strict | no declared info at all | |

**Notes:** Confirmed-only subtraction closes Phase 13 D-13-29 (direction warnings switch to confirmed-only).

## Read-visibility rules

| Option | Description | Selected |
|--------|-------------|----------|
| Hierarchical scopes + medical restriction | employee/manager/HR scopes, medical restricted | |
| Same + finance org-wide | finance also reads org-wide | |
| Org-wide calendar view | all members see all windows | |
| **Free text** | **"absence concerns everyone in the org, so we need to show it without limits (only privacy wise)"** | ✓ |

**User's choice:** Free text — org-wide visibility, privacy-restricted medical data only

| Option | Description | Selected |
|--------|-------------|----------|
| Kind+dates public, ref/doc private | certificate_ref + documents restricted to hr + unit manager | ✓ |
| Everything public except doc | ref public, document private | |
| Confirmed public, declared limited | declared hidden org-wide | |

**Notes:** Kind label + dates + status public org-wide (including declared). certificate_ref + documents = hr + employee's unit manager only (ADR D-1a). Server-side field filtering.

## Capacity fallback & validity filtering

| Option | Description | Selected |
|--------|-------------|----------|
| Override → type → org default → 8×5 | full chain with hard fallback | ✓ |
| No hard fallback, zero capacity | unconfigured = zero capacity | |
| 8×5 constant only this phase | contract_types deferred | |

| Option | Description | Selected |
|--------|-------------|----------|
| Excluded entirely | outside validity = not in capacity responses | ✓ |
| Included, flagged | greyed with zero capacity | |
| No filtering this phase | pure schedule math | |

**Notes:** User asked what "employment validity" means — explained ADR-P-008 D-2 (valid_from/valid_until, open-ended nullable). Chose exclusion entirely, parity with Phase 13 D-13-31.

## Endpoint & role-gate shapes

| Option | Description | Selected |
|--------|-------------|----------|
| REST under /availability | declare/withdraw/confirm/reject/HR-edit/certificate/windows/capacity/contract-types | ✓ |
| Single actions endpoint | transition-matrix style | |
| Minimal set now | declare/confirm/reject/list/capacity only | |

| Option | Description | Selected |
|--------|-------------|----------|
| As discussed + document gates | declare self/HR; withdraw owner; confirm/reject unit manager; HR medical + doc write; reads org-member | ✓ |
| Doc read: hr + unit mgr + self | stricter doc read scope | |
| Capacity manager-gated | capacity numbers restricted | |

**Notes:** Certificate document READ = hr + unit manager (ADR D-1a scope, per discussion).

## Contract-type management

| Option | Description | Selected |
|--------|-------------|----------|
| HR-owned CRUD this phase | create/edit/delete hr-gated, managers read-only | ✓ |
| Seed data only this phase | management deferred to Phase 16 | |
| Manager + HR CRUD | both roles manage | |

| Option | Description | Selected |
|--------|-------------|----------|
| Deactivate, not delete | soft-deactivate, existing members keep serving | |
| Hard delete if unused | FK-blocks in-use types | ✓ |
| Immutable, new type to change | strictest, no edits | |

| Option | Description | Selected |
|--------|-------------|----------|
| Membership endpoint extension | contract_type_id + override on existing org/membership endpoints | ✓ |
| Dedicated schedule endpoint | /availability/schedules/{user_id} | |
| Defer override setting | Phase 16 attaches overrides | |

---

## the agent's Discretion

- Exact endpoint URL shapes and route registration within the `/availability` REST surface
- `rejection_reason` column shape; confirmed_by/rejected_by/timestamps on row vs audit-only
- Certificate document storage details (BYTEA vs chunking; size limits; MIME allowlist)
- Org default schedule representation (flagged contract_type vs org_settings key)
- Day-hours override storage shape (rows table vs JSONB on membership)
- Windows list read-model filters/pagination; capacity period format
- Contract-types CRUD endpoint shapes; org-default schedule routing
- Test layout for the new availability domain package

## Deferred Ideas

- **Payroll export** (ADR-P-008 D-1c): confirmed windows feed Phase 25 Exports payroll view
- **work_permit_expires_at**: not consumed by Phase 14; stays for Phase 13 warning path
- **Block-vs-nag soft policy** (D-X): UI-decided in Phase 19
- **Absence balances/accruals** (ADR D-5): still rejected; work schedules are a capacity basis, not entitlements
