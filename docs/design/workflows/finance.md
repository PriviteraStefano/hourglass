# Hourglass Finance Role Contract

## Purpose

This contract names the jobs a user acting as **finance** performs in Hourglass, and the surfaces those jobs need. Jobs are workflows, not routes.

- **Authority split:** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary; `docs/design/CHROME.md` wins on the frame, sidebar, and page anatomy. This file adds jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **Not authorization:** every visibility statement here is UX scoping. The server's routing and role checks are authoritative; where enforcement is missing it is listed as a gap, never as a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added finance role contract

## Actor

- **Role string:** `finance`, carried by the current membership (`GET /auth/me` → memberships) and minted into the JWT claim from `organization_memberships.role`. Switching organizations switches the active membership and therefore the role.
- **Derived hats:** a finance user may also hold the **working-group manager or delegate** hat (WG `manager_id` / `delegate_ids`). That hat is not a membership role and is not `manager`; it makes that group's stage-1 queue visible (`manager.md` `M-01`), but it does not admit the approve/reject action: those routes accept only the `manager`/`finance` JWT role and the stage-1 transition accepts only `manager`, so a finance membership cannot execute stage 1 either (`G12`). On the approvals surface the client derives stage-2 from the membership role and stage-1 from the hat (`deriveApprovalStages`).
- **What finance is not:** not a stage-1 approver by role, not an allocator of coverage, not a closer of periods, not an employee capturing their own time, not HR curating availability or employment validity, and `MUST NOT` be presented as approving its own submissions — self-approval is structurally impossible for entries and expenses, and the coverage allocation owner is forbidden outright.
- **No Admin:** membership and organization-settings maintenance below is a finance job; there is no Admin/Settings role or surface in v0.2.1 (SC2), and `isAdminVisible` returns false for every role.

## Scope

Per-role rows (the assembled role×surface matrix is Phase 20, D-19-08):

| Surface | Finance sees | Evidence today |
|---|---|---|
| Approvals | the stage-2 queue (`pending_finance`) for entries and expenses | `web/src/routes/_authenticated/approvals/` |
| Contracts | the whole commercial record: create, adopt, edit, mileage recalculation, delete, bucket balance | `web/src/routes/_authenticated/contracts/` |
| Customers | customer records, finance-gated on create/update/delete | `web/src/routes/_authenticated/customers/` |
| Activities | assignment and customer-ticket origins, edit/delete, proposal approval | `web/src/routes/_authenticated/activities/` |
| Coverage money labels | proposals, allocation history, to-cover queue, bucket balances, snapshots — read-only | `F-05` (no client route yet) |
| Membership | member list, role assignment, deactivation, legacy org settings | `web/src/routes/_authenticated/org-hierarchy/` |
| Exports | timesheet / expense / combined rows and counts, CSV + XLSX | `web/src/routes/_authenticated/exports/` |
| Approving stage 1 as finance | **never** — stage 1 is the `manager` membership role (`M-01`); the WG manager/delegate hat does not admit the action (`G12`) | `web/src/lib/role-visibility.ts:25` |
| Closing a period | never — the close action is the manager job `M-05`; finance owns only the snapshot read (`F-08`) | `M-05` |
| Employee capture (own time, own expenses) | never — `E-01`, `E-02` | `employee.md` |
| HR curation (availability, employment validity) | never — `H-01`, `H-02` | `hr.md` |
| Admin / Settings | no such surface exists (SC2) | — |

Finance `MUST NOT` be offered stage-1 approval as a role, coverage allocation writes, or the period-close action. **Caveat:** these rows are UX scoping only. Hiding a queue or a control is never authorization; the server's role checks decide, and where they do not exist the gap is named below.

## Jobs

### F-01 — Approve the second stage

| field | value |
|---|---|
| Actor | finance |
| Trigger | entries and expenses have cleared stage 1 and now wait on the commercial decision |
| Steps | 1. Open the approvals surface. 2. Read the finance-stage queue. 3. Inspect the record and its approval history. 4. Approve to settle it, or reject with a reason. |
| Surfaces | Approvals (finance stage) · entry/expense detail |
| API interactions | `POST /time-entries/{id}/approve`, `POST /time-entries/{id}/reject`, `POST /expenses/{id}/approve`, `POST /expenses/{id}/reject`, `GET /time-entries/pending`, `GET /expenses/pending` |
| Current state | exists — `web/src/routes/_authenticated/approvals/` |
| Backend authority | Stage 2 is settled by the org role `finance` on a `pending_finance` record; approve/reject are gated `manager`\|`finance` with self-approval forbidden. The pending-queue reads admit `manager`\|`finance`\|`wg_manager`, where `wg_manager` is a synthesized derived hat, not a role. Entries whose owner is in the anchored WG approver set skip stage 1 and land here directly. Rejection returns the record to an editable state; approval-history rows are immutable. |

### F-02 — Own the commercial record

| field | value |
|---|---|
| Actor | finance |
| Trigger | a contract is sold, adopted, repriced, or retired |
| Steps | 1. Create the contract (type `project`\|`support`, `sold_hours`, `sold_period`). 2. Adopt it so logged work can draw on it. 3. Recalculate mileage when distances or rates change. 4. Edit or delete as the commercial terms move. |
| Surfaces | Contracts (list + detail) |
| API interactions | `GET /contracts`, `POST /contracts`, `GET /contracts/{id}`, `PUT /contracts/{id}`, `DELETE /contracts/{id}`, `POST /contracts/{id}/adopt`, `POST /contracts/{id}/recalculate-mileage` |
| Current state | exists — `web/src/routes/_authenticated/contracts/` |
| Backend authority | Update, delete, and mileage recalculation are finance-gated; list, get, and adopt are auth-only. Contract creation carries **no role gate today** — that absence is the gap `G1`, `MUST NOT` be presented as a finance permission. Bucket balances drawn against a contract are the coverage read in `F-05`; allocation writes are the manager job `M-04`. |

### F-03 — Own customer records

| field | value |
|---|---|
| Actor | finance |
| Trigger | a customer is onboarded, changes hands, or is retired |
| Steps | 1. Create the customer (name, `is_internal` flag, commercial identifiers). 2. Edit the record as the relationship changes. 3. Delete it when the relationship ends. 4. Read it for reporting and contract adoption. |
| Surfaces | Customers (list + detail) |
| API interactions | `GET /customers`, `POST /customers`, `GET /customers/{id}`, `PUT /customers/{id}`, `DELETE /customers/{id}` |
| Current state | exists — `web/src/routes/_authenticated/customers/` |
| Backend authority | Create, update, and delete are finance-gated; list and get are auth-only for any role. `POST /organizations/invite-customer` is a stub that returns a canned message and calls no service — customer onboarding is not end-to-end (`G7`). The `customer` membership role has no app surface at all (`C-01`). |

### F-04 — Own activities

| field | value |
|---|---|
| Actor | finance |
| Trigger | work needs a funded container, or an employee proposes an activity |
| Steps | 1. Create the activity from a finance-owned origin. 2. Set its kind and budget. 3. Edit or delete as scope changes. 4. Approve employee proposals through the shared routing service. |
| Surfaces | Activities (list + detail) |
| API interactions | `POST /activities`, `GET /activities`, `GET /activities/{id}`, `PUT /activities/{id}`, `DELETE /activities/{id}`, `POST /activities/{id}/approve-proposal`, `GET /activity-kinds` |
| Current state | exists — `web/src/routes/_authenticated/activities/` |
| Backend authority | Creation is gated by origin: `manager_assignment` and `customer_ticket` require `manager`\|`finance`; `employee_proposal` is self-created and approved through the shared routing service (`E-06`). Update and delete are finance-gated; list, get, and kinds are auth-only. Governance models (`creator_controlled`, `unanimous`, `majority`) are validated and stored but **no voting or quorum semantics exist** — no surface may promise approval by vote (`G10`). |

### F-05 — Read the money labels

| field | value |
|---|---|
| Actor | finance |
| Trigger | I need to know how logged hours were labeled and what is still uncovered |
| Steps | 1. Open an entry's coverage. 2. Read its allocation proposal and allocation history. 3. Read the to-cover queue. 4. Read a contract's bucket balance and a period snapshot. |
| Surfaces | Coverage (read-only) · Contracts (balance) · Cutoffs (snapshot read) |
| API interactions | `GET /coverage/proposals/{entry_id}`, `GET /coverage/allocations/{entry_id}/history`, `GET /coverage/to-cover`, `GET /coverage/buckets/{contract_id}/balance`, `GET /coverage/snapshots/{close_id}` |
| Current state | partial — every read is wired server-side; no client route consumes them |
| Backend authority | Every read listed is gated `manager`\|`finance`. The **target** authorization is finance read plus manager write: allocation writes are the manager job `M-04`, and the entry owner is forbidden from writing them outright. `F-05` `MUST NOT` be presented as a finance write path. |

### F-06 — Govern membership

| field | value |
|---|---|
| Actor | finance |
| Trigger | someone joins, changes responsibility, or leaves the organization |
| Steps | 1. Read the member list. 2. Assign or change a member's roles. 3. Deactivate a member. 4. Maintain the legacy organization settings that remain finance-writable. |
| Surfaces | Membership (org-hierarchy surface) |
| API interactions | `GET /organizations/members`, `PUT /organizations/members/{member_id}/roles`, `DELETE /organizations/members/{member_id}`, `PUT /organizations/{id}/settings` |
| Current state | exists — `web/src/routes/_authenticated/org-hierarchy/` |
| Backend authority | Role assignment and deactivation are finance-gated and carry a last-finance guard; the `{id}/settings` PUT is the finance-writable legacy path (the planning keys are a manager write). `hr` cannot be assigned through this path because `models.Role.IsValid()` rejects it and every equality gate fails closed (`G5`). Invitations, the other way people enter, run unauthenticated today (`G4`). This is a finance job, **not** an Admin/Settings surface (SC2). |

### F-07 — Org-wide reporting

| field | value |
|---|---|
| Actor | finance |
| Trigger | I need the organization's timesheet and expense data outside the app |
| Steps | 1. Pick the report: timesheets, expenses, or combined. 2. Set the date range (731 days maximum). 3. Read the counts before exporting. 4. Download CSV or XLSX. |
| Surfaces | Exports |
| API interactions | `GET /exports/timesheets`, `GET /exports/expenses`, `GET /exports/combined` (+ `/count` variants) |
| Current state | partial — the exports surface exists; the per-unit non-billed cost report does not |
| Backend authority | The export routes carry no role gate; row scoping ends in the repository's `default` branch, which makes `finance` (and `hr` and `customer`) org-wide — org-wide reach is the gap `G6`, not an explicit finance scope. The 731-day maximum range is enforced server-side. The per-unit non-billed cost report is a **target** surface with no implementation. |

### F-08 — Cutoffs (read side)

| field | value |
|---|---|
| Actor | finance |
| Trigger | a period has been closed and I need its frozen numbers |
| Steps | 1. Open the period snapshot for a close. 2. Read its snapshot rows. 3. Fold the frozen numbers into the reporting cadence. |
| Surfaces | Cutoffs (target surface — no client route today) |
| API interactions | `GET /coverage/snapshots/{close_id}` |
| Current state | absent on the client — the read is wired server-side (`cmd/server/main.go:142`); no finance surface consumes it |
| Backend authority | The snapshot read is gated `manager`\|`finance`. The **close action is not finance's**: `POST /coverage/close` is manager-only and is the job `M-05`; snapshots are append-only and a duplicate close returns 409. Finance owns the read and the reporting cadence, the manager owns the close. This split is a division of work, not a gap. |

## Do / don't

- Don't present stage-2 as the whole approval: stage 1 is the `manager` membership role (`M-01`) and is not available to finance, hat or no hat (`G12`).
- Don't promote the WG hat into a role, and don't restate `wg_manager` as a membership role — it is a synthesized derived hat that grants queue visibility.
- Don't give finance the period close; name `M-05`. `F-08` is the read-only half.
- Don't promise coverage writes: allocation writes are `M-04`, and the entry owner is forbidden outright.
- Don't promise activity approval by vote or quorum: the governance models are stored but not implemented (`G10`).
- Don't treat contract creation or org-wide exports as finance permissions — they are enforcement gaps `G1` and `G6`.
- Don't offer HR membership assignment as available: `hr` is rejected by `Role.IsValid()` (`G5`).
- Don't restate nav groups, tokens, or component APIs.

## Pointers

- `docs/design/CHROME.md` — frame, four lifecycle nav groups, role-scoped chrome.
- `docs/design/LANGUAGE.md` — status vocabulary for entry/expense and coverage states.
- `web/src/lib/role-visibility.ts` — the predicate mechanism this contract's scope rows agree with.
- `web/src/components/approval/approval-buttons.tsx` — the existing role × status action gate behind `F-01`.
- `internal/core/services/routing/routing.go` — stage-1 resolution and the finance claim that closes stage 2.
- `docs/history/planning/REQUIREMENTS.md` — FIN-01 (this contract) and the archived leftovers (`SURF-04`, `SURF-05`) used as job hints.
- `docs/history/planning/phases/19-role-contracts/19-RESEARCH.md` — the verified fact base behind every `file:line` above.
- `manager.md` (`M-04`, `M-05`), `employee.md` (`E-01`, `E-02`), `hr.md` (`H-01`, `H-02`), `customer.md` (`C-01`) — the sibling contracts named in the scope rows.

## Not in this file

- No chrome, navigation, or page-anatomy model (that is `CHROME.md`); no design-language vocabulary or values (that is `LANGUAGE.md`).
- No page layouts, route definitions, component APIs, sketches, or implementation sequencing.
- No cross-role composition map (Phase 20, `COMPOSITION.md`).
- No Admin/Settings surface (out of v0.2.1, SC2).
- No value tables (colors, spacing, sizes).

## Gaps

- `G1` — `middleware.RequireRole` is defined and unit-tested but never wired; every gate is an inline equality check. Contract creation (`F-02`) has no gate at all, and the finance gates elsewhere must be satisfied by handler/service checks.
- `G4` — `POST /invitations` runs without Auth middleware and takes `organization_id` from the request body: the other way people enter the organization is unauthenticated, so membership governance (`F-06`) is not the only door.
- `G5` — `hr` exists in the `organization_memberships` CHECK but not in `models.Role`; `Role.IsValid()` rejects it, so `F-06` cannot assign HR and every equality gate fails closed.
- `G6` — export row scoping ends in a `default` branch that makes `finance` org-wide; `F-07`'s org-wide reach is that branch, not an explicit finance scope.
- `G7` — `POST /organizations/invite-customer` is a stub returning a canned message: customer onboarding (`F-03`) is not end-to-end.
- `G10` — governance models are validated and stored but no voting or quorum semantics exist: `F-04` must not promise approval by vote.
- `G12` — the WG manager/delegate hat sees its group's stage-1 queue but cannot act on it, and a finance membership cannot execute stage 1 either: the stage-1 transition accepts only the `manager` role (`internal/core/services/time_entry/time_entry.go:193`).
