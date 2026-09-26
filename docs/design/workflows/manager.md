# Hourglass Manager Role Contract

## Purpose

This contract names the jobs a user acting as a **manager** performs in Hourglass, and the surfaces those jobs need. Jobs are workflows, not routes (D-19-02/D-19-11).

- **Authority split:** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary; `docs/design/CHROME.md` wins on the frame, sidebar, and page anatomy. This file adds jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **Not authorization:** every visibility statement here is UX scoping. The server's routing and role checks are authoritative; where enforcement is missing it is listed as a gap, never as a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added manager role contract

## Actor

- **Role string:** `manager`, carried by the current membership (`GET /auth/me` → memberships) and minted into the JWT claim from `organization_memberships.role`. Switching organizations switches the active membership and therefore the role.
- **Independent dimensions, not roles:** three separate facts decide what a manager may do, and none of them substitutes for another — the org membership role (`organization_memberships.role = 'manager'`); the **working-group manager/delegate hat** (WG `manager_id` / `delegate_ids`); and the **unit membership role** (`unit_memberships.role = 'manager'`), which is the row the routing walk finds when it climbs `parent_unit_id` from an entry's unit. Only the first is a membership role. The pending-queue endpoints synthesize a `wg_manager` actor label for a WG manager or delegate whose membership role is not `manager` — that label is a derived hat, never a role. The hat makes that group's stage-1 queue visible and puts the holder inside the routing-resolved approver set; the approve/reject action itself requires the `manager` membership role today (`G12`).
- **What a manager is not:** not the stage-2 approver (`F-01`), not the owner of commercial records (contracts, customers — `F-02`, `F-03`), not the governor of membership (role assignment, deactivation — `F-06`), not the owner of the cutoff bookkeeping (finance reads snapshots and owns the reporting cadence — `F-08`), not HR curating availability or employment validity (`H-01`, `H-02`), not an employee capturing time or expenses through this contract (`E-01`, `E-02`), and `MUST NOT` be presented as able to approve their own submissions — self-approval is structurally impossible for entries and expenses, and the coverage owner is forbidden outright.
- **No Admin:** the org tree, unit membership, and staffing below are **manager jobs**. There is no Admin/Settings role or surface in v0.2.1 (SC2), and `isAdminVisible` returns false for every role.

## Scope

Per-role rows (the assembled role×surface matrix is Phase 20, D-19-08):

| Surface | Manager sees | Evidence today |
|---|---|---|
| Approvals | the stage-1 queue for entries and expenses, scoped to the working groups the actor manages or delegates — queue visibility only: the WG hat alone does not admit the action (`G12`) | `web/src/routes/_authenticated/approvals/` |
| Direction (the plan) | the organization's plan rows and the capacity/coverage read-model; the `unit` and `wg` scopes are manager-only | `M-02`, `M-03` (no client route yet) |
| Coverage | proposals, allocation writes, the to-cover queue, bucket balances, allocation history | `M-04` (backend only, no client route) |
| Cutoffs | the close action for a period | `M-05` (no client route yet) |
| Organization (units) | the unit tree, unit membership, descendant and batch member reads | `web/src/routes/_authenticated/org-hierarchy/` |
| Working groups | group records and group membership | `web/src/routes/_authenticated/working-groups/` |
| Tickets | triage, dismissal, assignment and update of a ticket | `M-08` (no client route yet) |
| Planning policy | the planning keys: daily hours, deadline, horizon, planning mode | `M-09` (no client surface yet) |
| Exports | own rows plus the rows of activities anchored in the groups the actor manages or delegates | `web/src/routes/_authenticated/exports/` |
| Commercial record writes (contracts, customers) | **never** — finance owns them | `F-02`, `F-03` |
| Membership governance (role assignment, deactivation) | **never** — finance owns it | `F-06` |
| Period-close bookkeeping (snapshot reads, reporting cadence) | **never** — the manager only closes; finance reads snapshots and owns the cadence | `F-08` |
| Stage-2 finance approval | **never** — `pending_finance` is finance's queue | `F-01` |
| Employee capture (own time, own expenses) | **never** — `E-01`, `E-02` | `employee.md` |
| HR curation (availability, employment validity) | **never** — `H-01`, `H-02` | `hr.md` |
| Admin / Settings | no such surface exists (SC2) | — |

The manager `MUST NOT` be offered stage-2 approval, commercial-record writes, membership governance, or the period-close bookkeeping. **Caveat:** these rows are UX scoping only. Hiding a surface or a control is never authorization; the server's role checks decide, and where they do not exist the gap is named below.

## Jobs

### M-01 — Approve the first stage

| field | value |
|---|---|
| Actor | manager — the membership role. The WG manager/delegate hat and the `unit_memberships` manager row decide *whether this manager* is the resolved approver; neither admits the action on its own (`G12`). |
| Trigger | entries and expenses have been submitted and wait on the first approval decision |
| Steps | 1. Open the approvals surface. 2. Read the pending queue. 3. Inspect the record and its approval history. 4. Approve to advance it to finance, or reject with a reason. |
| Surfaces | Approvals (stage 1) · entry/expense detail |
| API interactions | `POST /time-entries/{id}/approve`, `POST /time-entries/{id}/reject`, `POST /expenses/{id}/approve`, `POST /expenses/{id}/reject`, `GET /time-entries/pending`, `GET /expenses/pending` |
| Current state | exists — `web/src/routes/_authenticated/approvals/`; the role × status gate is `web/src/components/approval/approval-buttons.tsx:32` |
| Backend authority | Stage 1 is executed by the membership role `manager` on a `submitted` record: the handler returns 403 for every other JWT role and the service accepts only `manager` for that transition (`internal/adapters/primary/http/time_entry.go:351`, `internal/core/services/time_entry/time_entry.go:193`). The actor must also sit inside the routing-resolved approver set — the anchored group's `manager_id` + `delegate_ids`, or the nearest `unit_memberships` row with role `manager` walking `parent_unit_id` upward — unless no unit manager exists up to the org root, where the terminal role-gated branch accepts the org `manager` claim. Approving advances the record to `pending_finance` and stamps `current_approver_role='finance'`. Self-approval is structurally impossible (`internal/core/services/time_entry/time_entry.go:188`), and when the owner is inside the anchored approver set submission skips stage 1 and lands straight in finance. The queue reads are scoped to the groups the actor manages or delegates, and admit the synthesized `wg_manager` derived hat; the org-wide `pending_finance` queue belongs to `finance`. |

### M-02 — Set the plan

| field | value |
|---|---|
| Actor | manager, when the organization's planning mode is `manager_planned` (or through manager reach on the anchored activity) |
| Trigger | a planning horizon opens and people or groups need direction |
| Steps | 1. Create a row for a person (scheduled date + estimate) or for a working group (queued). 2. Activate it. 3. Cancel it with a reason when it lapses. 4. Supersede it by creating a replacement that names the row it replaces. |
| Surfaces | Direction (the plan — target surface, no client route today) |
| API interactions | `POST /direction`, `POST /direction/{id}/activate`, `POST /direction/{id}/cancel`, `GET /direction` |
| Current state | absent on the client; the backend is complete |
| Backend authority | Creation fast-fails the XOR `directed_to` / `wg_id` (exactly one target) and requires a same-org, active member target; working-group rows are **queued-only** and must sit inside the group's anchored activity or its ancestry. Activation and cancellation are creator-or-manager-reach; cancellation requires a reason and writes an audit row. Statuses are draft/active/superseded/cancelled. Supersede is create-with-`supersedes_id` — it has **no endpoint of its own**, and the replaced row must be draft or active. `GET /direction` is org-wide only for the org role `manager`; anyone else must ask for their own `employee_id`. |

### M-03 — Read the plan and capacity

| field | value |
|---|---|
| Actor | manager |
| Trigger | I need to see what is planned and where the capacity gaps are before I direct more work |
| Steps | 1. Read the plan rows for the period. 2. Read the capacity/coverage read-model for a scope (own, a person, a unit, a group). 3. Read the warnings. |
| Surfaces | Direction (plan + capacity read-model — target surface, no client route today) |
| API interactions | `GET /direction`, `GET /direction/coverage` |
| Current state | absent on the client — both reads are wired server-side (`cmd/server/main.go:151`, `cmd/server/main.go:152`); no client surface consumes them |
| Backend authority | Capacity is computed as daily hours − absence hours, and coverage rows carry capacity / planned / gap. Warnings are a pure, never-blocking overlay with the types `away`, `partial`, `over-capacity`, `invalid` (validity-outside employees are dropped from the rows and surface only as an `invalid` warning). The `unit` and `wg` scopes are manager-only; the `employee` scope requires manager reach or the actor's own id. Suppressing a warning is not possible. |

### M-04 — Allocate coverage

| field | value |
|---|---|
| Actor | manager (inside the routing-resolved approver set for the entry) |
| Trigger | an approved entry's hours are still uncovered, or have to be re-labeled |
| Steps | 1. Open the to-cover queue. 2. Read the computed proposal for an entry. 3. Read the contract bucket balance before drawing on it. 4. Write the replacement allocation set with a reason on every absorption row. 5. Read the allocation history. |
| Surfaces | Coverage (proposals, to-cover queue, bucket balance, history — backend only, no client route today) |
| API interactions | `GET /coverage/proposals/{entry_id}`, `PUT /time-entries/{id}/allocations`, `GET /coverage/to-cover`, `GET /coverage/buckets/{contract_id}/balance`, `GET /coverage/allocations/{entry_id}/history` |
| Current state | partial — every route is wired server-side (`cmd/server/main.go:135`, `cmd/server/main.go:137`, `cmd/server/main.go:139`–`140`, `cmd/server/main.go:143`); no client route consumes them |
| Backend authority | Proposals are **computed on read**, never stored. The write is a **replace-set**: one call carries the whole allocation set for the entry, and the summed allocations must equal the entry's hours in the same transaction (the service fast-fails in cents before the repo call and the repository re-checks under the write). Absorption rows require a reason from the fixed vocabulary and transfer rows require a justification; contract references must be same-org or adopted. The entry **owner is forbidden outright** and a non-approver is forbidden, so no manager reaches this job for a submission they own. The to-cover queue is an explicit state — approved, non-deleted entries with uncovered hours — not an implicit gap. |

### M-05 — Close the period and read the snapshot

| field | value |
|---|---|
| Actor | manager |
| Trigger | a period ends and its coverage must be frozen |
| Steps | 1. Confirm the period bounds. 2. Close the period. 3. Read back the frozen snapshot for that close. |
| Surfaces | Cutoffs (close + snapshot read — target surface, no client route today) |
| API interactions | `POST /coverage/close`, `GET /coverage/snapshots/{close_id}` |
| Current state | absent on the client — both routes are wired server-side (`cmd/server/main.go:141`, `cmd/server/main.go:142`); no client surface consumes them |
| Backend authority | The close action is **manager-only** and nothing else reaches it (`internal/core/services/coverage/coverage.go:529`). The snapshot is append-only and carries no aggregates — a duplicate close returns 409 and an inverted period is rejected before the write, because the append-only overlap predicate would otherwise lock that range forever. The snapshot read is `manager`\|`finance`. Finance **reads** snapshots and does **not** close (`F-08`). |

### M-06 — Shape the org

| field | value |
|---|---|
| Actor | manager (this job is shared with HR, `H-03`) |
| Trigger | the organization grows, splits, or a unit's membership changes |
| Steps | 1. Read the unit tree. 2. Create or edit a unit. 3. Place members in a unit and set their unit role. 4. Remove a member from a unit. 5. Read descendants and batch member rows. |
| Surfaces | Organization (unit tree + unit membership) |
| API interactions | `GET /units`, `POST /units`, `GET /units/{id}`, `PUT /units/{id}`, `DELETE /units/{id}`, `GET /units/tree`, `GET /units/{id}/descendants`, `GET /units/{id}/members`, `POST /units/{id}/members`, `PUT /units/{id}/members/{membership_id}`, `DELETE /units/{id}/members/{membership_id}`, `GET /units/members/batch` |
| Current state | exists — `web/src/routes/_authenticated/org-hierarchy/`, but with **no role gate** (`G3`) |
| Backend authority | No role gate exists today: every unit route is auth-only and unit ids are not org-scoped on the path (`G3`) — `MUST NOT` be read as a manager permission. The **target authorization is `manager`, shared with HR** (`H-03`). This job is where the stage-1 approver set is staffed: the `PUT /units/{id}/members/{membership_id}` unit role of `manager` is the row the routing walk finds when it climbs `parent_unit_id`, so `M-01`'s reach follows from `M-06`. This is a manager job, explicitly **not** Admin/Settings (SC2, MGR-01). |

### M-07 — Staff the work

| field | value |
|---|---|
| Actor | manager |
| Trigger | work needs a standing group with its own manager, delegates, and members |
| Steps | 1. Create a working group anchored to its activity. 2. Set its manager and delegates. 3. Add and remove members. 4. Edit or retire the group. |
| Surfaces | Working groups (list + detail) |
| API interactions | `GET /working-groups`, `POST /working-groups`, `GET /working-groups/{id}`, `PUT /working-groups/{id}`, `DELETE /working-groups/{id}`, `GET /working-groups/{id}/members`, `POST /working-groups/{id}/members`, `DELETE /working-groups/{id}/members/{member_id}` |
| Current state | exists — `web/src/routes/_authenticated/working-groups/`, with **no role gate** (`G3`) |
| Backend authority | No role gate exists on any of the eight routes today (`G3`) — `MUST NOT` be read as a manager permission. The **target authorization is `manager`**. The `manager_id` / `delegate_ids` columns set here are the first half of the stage-1 approver set (`M-01`) and of the manager export scope (`M-10`); membership of a group is what makes a row claimable from the group queue (`E-09`). |

### M-08 — Triage tickets

| field | value |
|---|---|
| Actor | manager |
| Trigger | an internal ticket needs a decision, an owner, a plan, or closure |
| Steps | 1. Read the ticket and its history. 2. Triage it into planned work or dismiss it with a reason. 3. Assign or update it. 4. Transition it along its lifecycle. 5. Comment as decisions land. |
| Surfaces | Tickets (target surface — no client route today) |
| API interactions | `POST /tickets/{id}/triage`, `POST /tickets/{id}/dismiss`, `PUT /tickets/{id}`, `POST /tickets/{id}/transition`, `POST /tickets/{id}/comments` |
| Current state | partial — the backend is complete; no client route consumes it, and the nav item is a disabled placeholder |
| Backend authority | Triage, dismissal, and update are gated `manager`\|`finance`; transition and comment admit the owner/assignee or a `manager`\|`finance` actor. **Dismissal is blocked while any activity linked to the ticket carries logged hours**, so a ticket that has produced work cannot be silently discarded. The `customer` role is forbidden outright on tickets (`C-01`). Lifecycle states are open→triage→planned→in_progress→resolved→closed plus dismissed; kinds are question/bug/change/evolution. |

### M-09 — Set planning policy

| field | value |
|---|---|
| Actor | manager |
| Trigger | the organization's planning cadence changes, or the daily-hours budget moves |
| Steps | 1. Read the current settings. 2. Write the planning keys: daily hours, deadline, horizon, planning mode. 3. Rely on the per-member mode override where it exists. |
| Surfaces | Planning policy (target surface — no client surface today) |
| API interactions | `GET /organizations/settings`, `PUT /organizations/settings` |
| Current state | partial — both routes are wired (`cmd/server/main.go:107`, `cmd/server/main.go:108`); no client surface consumes them |
| Backend authority | The `PUT` is manager-gated, validates each key against the known-key vocabulary (unknown key or invalid value → 400), and audits every write in the same transaction. The planning mode (`manager_planned` / `self_planned`) is resolved server-side and per member — a member's override beats the org value. The read is auth-only. The legacy `PUT /organizations/{id}/settings` remains a finance-writable path (`F-06`); the planning keys are this job's. |

### M-10 — Team reporting

| field | value |
|---|---|
| Actor | manager |
| Trigger | I need my team's timesheet or expense data outside the app |
| Steps | 1. Pick the report: timesheets, expenses, or combined. 2. Set the date range. 3. Read the counts before exporting. 4. Download CSV or XLSX. |
| Surfaces | Exports |
| API interactions | `GET /exports/timesheets`, `GET /exports/expenses`, `GET /exports/combined` (+ `/count` variants) |
| Current state | exists — `web/src/routes/_authenticated/exports/` |
| Backend authority | Row scoping for the `manager` role is own rows plus rows whose activity is anchored in a working group the actor manages or delegates (the same hat chain as `M-01`). The export routes themselves carry **no role gate** — scoping is the only mechanism, and the repository's `default` branch makes `finance`, `hr`, and `customer` org-wide (`G6`), which `MUST NOT` be read as a manager scope. The 731-day maximum range is enforced server-side. |

## Do / don't

- Don't promote a hat into a role: the WG manager/delegate hat, the `wg_manager` queue label, and the `unit_memberships` manager row are three separate dimensions, not membership roles.
- Don't present the pending queues as org-wide for the manager role — they are scoped to the groups the actor manages or delegates, and the org-wide `pending_finance` queue is finance's (`F-01`).
- Don't open Admin/Settings: the org tree and staffing are manager jobs (SC2, MGR-01).
- Don't promise direction, coverage, cutoff, ticket, or planning-policy surfaces as available: they have no client route or surface yet.
- Don't give the manager stage-2 approval (`F-01`), commercial record writes (`F-02`, `F-03`), membership governance (`F-06`), or HR curation (`H-01`, `H-02`).
- Don't present any approval as a vote or a quorum: the governance models are stored but never executed (`G10`).
- Don't treat the ungated unit and working-group routes as manager permissions — they are `G3`.
- Don't restate nav groups, tokens, or component APIs.

## Pointers

- `docs/design/CHROME.md` — frame, four lifecycle nav groups, role-scoped chrome.
- `docs/design/LANGUAGE.md` — status vocabulary for entry/expense, coverage, direction, and ticket states.
- `web/src/lib/role-visibility.ts` — the predicate mechanism this contract's scope rows agree with (`deriveApprovalStages`, `isReviewVisible`, `isEconomicsVisible`).
- `web/src/components/approval/approval-buttons.tsx` — the existing role × status action gate behind `M-01`.
- `internal/core/services/routing/routing.go` — the stage-1 approver set, the owner-in-set skip to finance, the unit-manager walk, and the role-gated terminal case.
- `internal/core/services/coverage/coverage.go` — the manager-only close and the allocation write gates behind `M-04`/`M-05`.
- `docs/history/planning/REQUIREMENTS.md` — MGR-01 (this contract) and the archived leftovers `SURF-01`, `SURF-02`, `SURF-04`, `SURF-05`, `SURF-07`, `AVAIL-04`, `TICK-06` used as job hints.
- `docs/history/planning/phases/19-role-contracts/19-RESEARCH.md` — the verified fact base behind every claim above.
- `employee.md` (`E-01`, `E-02`, `E-05`, `E-09`), `finance.md` (`F-01`, `F-02`, `F-03`, `F-05`, `F-06`, `F-08`), `hr.md` (`H-01`, `H-02`, `H-03`), `customer.md` (`C-01`) — the sibling contracts named in the scope rows.

## Not in this file

- No chrome, navigation, or page-anatomy model (that is `CHROME.md`); no design-language vocabulary or values (that is `LANGUAGE.md`).
- No page layouts, route definitions, component APIs, sketches, or implementation sequencing.
- No cross-role composition map (Phase 20, `COMPOSITION.md`).
- No Admin/Settings surface (out of v0.2.1, SC2).
- No value tables (colors, spacing, sizes), and no claims that `PageHeader`, `FilterBar`, `DataTable`, `EmptyState`, or `ConfirmDialog` exist — they are absent inputs.

## Gaps

- `G1` — `middleware.RequireRole` is defined and unit-tested but never wired: every gate is an inline equality check, and queue/approver scoping is a hand-rolled per-repository switch. One consequence: the pending queue is scoped to the actor's managed groups, so a manager whose only reach is the unit-manager walk may approve rows the queue does not list (`M-01`).
- `G3` — all 12 unit routes and all 8 working-group routes carry no role gate and no org check on the path ids: the target authorization for `M-06` and `M-07` is not enforced today.
- `G10` — governance models (`creator_controlled`, `unanimous`, `majority`) are validated and stored but no voting or quorum semantics exist: no manager approval (`M-01`, `M-08`) may be presented as a vote.
- `G2` — `GET /time-entries` and `GET /expenses` are org-wide for every authenticated role; a manager's roster-sized list views are client-scoped, and the pending queues are **not** covered by this gap.
- `G6` — export row scoping ends in a `default` branch that makes `finance`, `hr`, and `customer` org-wide; the manager's own+group scope (`M-10`) is the explicit branch, and the default branch is the gap.
- `G5` — `hr` exists in the `organization_memberships` CHECK but not in `models.Role`, so the shared org-tree job (`M-06` / `H-03`) cannot grant HR membership through the API today.
- `G12` — the WG manager/delegate hat sees its group's stage-1 queue but cannot act on it: the approve/reject routes admit only the `manager`/`finance` JWT role and the stage-1 transition accepts only `manager` (`internal/adapters/primary/http/time_entry.go:351`, `internal/core/services/time_entry/time_entry.go:193`). Target: either admit the hat to the action or stop rendering the queue for it (`M-01`).
