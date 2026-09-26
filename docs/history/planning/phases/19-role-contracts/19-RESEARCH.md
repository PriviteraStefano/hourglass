# Phase 19: Role contracts - Research

**Researched:** 2026-09-26
**Method:** three read-only codebase scans (backend route/gate inventory, frontend route/nav/predicate inventory, role-vocabulary + phase-17/18 lock inventory) plus direct reads of the vault ADRs. Every claim carries a `file:line` or `doc:line` citation. Nothing here is inferred from memory.

---

## 1. Role vocabulary — what is real today

| Fact | Evidence |
|---|---|
| Frontend role union has five values: `employee \| manager \| finance \| hr \| customer` | `web/src/types/models.ts:1` |
| Go role constants have four: employee, manager, finance, customer — **no HR** | `internal/models/models.go:11-16` |
| `models.Role.IsValid()` rejects `hr` | `internal/models/models.go:19-27` |
| The DB CHECK accepts `hr` on `organization_memberships` | `migrations/012_staffing_schema.up.sql:48-50` (down restores four: `012_staffing_schema.down.sql:20-23`) |
| The JWT `role` claim is minted from `organization_memberships.role` | `internal/auth/auth.go:27,59`; `internal/core/services/auth/auth.go:558-559` |
| An out-of-band `hr` membership reaches auth-only routes; every equality gate fails closed (treated as unprivileged) | backend scan §2 (hr row); `export_repository.go:29-37` `default` branch still yields org-wide export rows |
| HR is a curator/consumer, never an approver | `ADR-P-008` D-4 (lines 58-65); `ADR-P-011` D-3 (line 51), matrix line 66 |
| Frontend already strips HR from approval stages | `web/src/lib/role-visibility.ts:33`; pinned at `role-visibility.test.ts:125-128`, `sidebar-groups.test.tsx:175-187` |

**Consequence for contracts:** `hr.md` is first-class as a *contract*, but every HR job must carry the "not implemented in Go" gap (D-19-06/D-19-09).

## 2. Route surface — 118 registrations, gating is inline

`middleware.RequireRole` (`internal/middleware/middleware.go:46-61`) is defined, unit-tested, and **never referenced by route wiring**. Every gate is an equality check inside a handler or service. Contracts therefore state *target authorization*, and the gaps list says where the server does not enforce it yet.

| Domain | Routes | Gate reality |
|---|---|---|
| auth | 12 | public except `/auth/me`, `/auth/memberships`, `/auth/switch-organization` (self + target membership) |
| invitations | 4 | **no Auth middleware at all**; `POST /invitations` takes `organization_id` + email from the body (`invitation.go:32-49`) |
| units | 12 | auth only; **no role gate, no owner check, path ids not org-scoped** (`unit.go:51-56` …) |
| working groups | 8 | auth only; **no role gate** (`working_group.go:94-100` …) |
| customers | 5 | create/update/delete = **finance** (`customer.go:30,67,110`); list/get = auth only (any role, incl. customer) |
| organizations | 9 | create = caller becomes finance; `PUT .../settings` = finance; `PUT .../members/{id}/roles` + `DELETE .../members/{id}` = finance (+last-finance guard); rest auth only; `invite-customer` is a **stub** |
| activities | 8 | create gated **by origin** (assignment/customer-ticket = manager\|finance; employee_proposal = self); update/delete = finance; `approve-proposal` reuses routing; list/get/kinds auth only |
| tickets | 9 | **customer forbidden** on create/list/get/history; triage/dismiss/update = manager\|finance; transition/comment = owner\|assignee or manager\|finance; dismissal guard on logged hours |
| coverage | 9 | read (proposals/history/buckets/snapshots/to-cover) = manager\|finance; allocation write = owner forbidden + routing approver; `own` = self; **`POST /coverage/close` = manager only** (`coverage.go:529`) |
| direction | 7 | create/activate/cancel = creator or routing reach; `self_planned` requires actor == target; claims = WG membership; `GET /direction` org-wide = manager; coverage scope unit/wg = manager |
| contracts | 7 | create = **no role gate** (`contract.go:24-27`); update/mileage/delete = finance; list/get/adopt auth only |
| exports | 6 | **no role gate**; row scoping only — employee self, manager own+WG, **default (finance/hr/customer) org-wide** (`export_repository.go:29-37`); 731-day cap |
| time entries | 8 | owner for write/submit; approve/reject = manager\|finance + self-approval forbidden; `GET /time-entries` **org-wide for every role** (Role/RequestUserID never consumed, `time_entry_repository.go:97-152`); `pending` = manager\|finance\|wg_manager |
| expenses | 10 | same shape as time entries; receipt URL = owner or manager\|finance |

## 3. Capabilities per role (what the server actually enforces)

- **employee** — create/update/delete/submit own entries and expenses; read own detail; read own coverage (`coverage.go:196`); propose activities; ticket create/transition/comment when owner or assignee; self-planning when org mode is `self_planned`; claim WG rows as a member; exports scoped to own rows.
- **manager** — stage-1 approve/reject; pending queues; coverage read + allocation write; **close period**; org settings write (`planning_daily_hours`, `planning_deadline`, `planning_horizon`, `planning_mode`); ticket triage/dismiss/update; org-wide direction plan + unit/WG coverage; exports own+WG.
- **finance** — stage-2 approve; contracts update/mileage/delete; customers CRUD; activities create (assignment/customer-ticket origins)/update/delete; org settings (legacy) + member roles/deactivate; ticket triage/dismiss/update; coverage read; org-wide exports.
- **hr** — **no capability anywhere**: not a valid Go role, so invite/role-update reject it; reaches only auth-only routes; org-wide exports via the default branch.
- **customer** — no role-based grant. Tickets reject it. Everything auth-only (contracts read, customers read, activities read, entry/expense list reads, exports) is technically reachable — a gap, not an intent.

## 4. Approval routing (the two-stage chain contracts must describe)

`internal/core/services/routing/routing.go` (shared by time entries, coverage allocation writes, direction, proposals; the expense service holds its own copy at `svc/expense/expense.go:165-237`):

1. **Stage 1** resolves from the anchored working group: approver set = WG `manager_id` + `delegate_ids` (`routing.go:70-77`).
2. Skip rule: if the owner is in that set, submit goes straight to `pending_finance` with `current_approver_role='finance'` (`routing.go:77`).
3. R-2: commercial activity with no WG → `ErrActivityNotLoggable` (409); personal activity → nearest unit manager walking `parent_unit_id` upward (`routing.go:84-131`).
4. Terminal fallback: no unit manager anywhere → `RoleGated=true`, satisfied by org role `manager` (activity proposals also accept finance).
5. **Stage 2**: finance claim on `pending_finance` → `approved`.
6. Self-approval is structurally impossible for entries/expenses; the coverage owner is forbidden outright.
7. Pending queues admit WG manager/delegate as a synthesized `wg_manager` actor — a **derived hat**, not a membership role (`time_entry.go:448-460`).

Governance models (`creator_controlled`, `unanimous`, `majority`) are validated and stored (`models.go:60-71`) but **no voting semantics are implemented** — contracts must not promise quorum behavior.

## 5. Domain vocabulary the contracts may cite

- **coverage allocation** — `coverage_allocations`, source tagged union `contract|absorption|transfer`, reasons WarrantyBug/UnderEstimate/Goodwill, Σ = entry hours enforced in-tx; replace-set write. `replace-set` = one call replaces the whole allocation set for an entry.
- **to-cover queue** — approved, non-deleted `time` entries with uncovered hours; no implicit gap.
- **period close snapshot** — `coverage_period_closes` + `coverage_snapshot_rows`, append-only, duplicate close → 409; owner `ClosePeriod` (manager-only).
- **direction** — mode derived (`planned_date` set = scheduled, NULL = queued); XOR `directed_to`/`wg_id`; WG rows queued-only; claims are WG-member rows carrying `origin_direction_id` with a Σ-estimated-hours budget guard; supersede = create-with-`supersedes_id` (no endpoint); statuses draft/active/superseded/cancelled.
- **org policy** — `org_settings` keys `planning_daily_hours`, `planning_deadline`, `planning_horizon`, `planning_mode` (manager_planned | self_planned), plus a per-member `planning_mode` override.
- **availability windows** — kinds holiday|permit|medical|unavailable, status declared|confirmed, `hours` NULL = full absence, `certificate_ref` medical-only; **read-only today** (no write route).
- **capacity / workload read models** — capacity = daily hours − absence hours; coverage rows capacity/planned/gap; warnings `away|partial|over-capacity|invalid`.
- **tickets** — lifecycle open→triage→planned→in_progress→resolved→closed plus dismissed; kinds question|bug|change|evolution; dismissal blocked while linked activities carry logged hours.
- **org units / working groups / memberships** — units + `unit_memberships`, WGs + `wg_members`, `organization_memberships` (validity dates, planning_mode).
- **contracts / customers / activities** — `contract_type project|support`, `sold_hours`, `sold_period`; customers with `is_internal`; activities with origins (manager_assignment|employee_proposal|customer_ticket), `budget_amount`, beneficiary unit; funding sources are derived from the three row-level draws.
- **exports** — timesheet/expense/combined (+counts), CSV streamed / XLSX via excelize, 731-day max range.
- **audit log** — append-only `audit_logs`; observed `entity_type` vocabulary: `ticket`, `activity`, `coverage_allocation`, `direction`, `org_settings`. Writes are synchronous and same-transaction as the state change.

## 6. Existing surfaces (current-state evidence for jobs)

13 functional route files under `web/src/routes/_authenticated/` — no stubs, no placeholders: `/` (Today), `/time-entries`, `/expenses`, `/approvals`, `/activities`, `/working-groups`, `/customers`, `/contracts`, `/org-hierarchy`, `/exports`, plus the detail/auth-adjacent leaves. No `/tickets`, `/availability`, or `/settings` route files exist — for tickets and availability the nav item is a disabled placeholder.

Nav today is still ADR-P-011's eight groups + ungrouped Today (`sidebar.tsx:65-128`), filtered by four predicates (`role-visibility.ts`): `deriveApprovalStages` (WG manager/delegate + org manager/finance; hr stripped), `isReviewVisible` (stages and not hr), `isEconomicsVisible`, `isAdminVisible` (always false). Behavior is pinned by `role-visibility.test.ts` and `sidebar-groups.test.tsx`. Phase 18's four lifecycle groups are the **target** chrome, not the live tree.

## 7. IA input to confirm or revise (ADR-P-011)

- **D-1** job-language, pillar-mapped groups (Today/Track/Work/People/Economics/Review/Reports/Admin) — **revised by CHROME.md** into Record · Organize · Approve · Report + Today landing, Admin dropped.
- **D-2** Today is the landing, read-only composition, never blank — **kept**.
- **D-3** Review is its own group, role-gated, HR never sees it — **kept in substance** (now the `Approve` lifecycle group + the role-visibility mechanism).
- **D-4** Working Groups get a top-level surface — still live as a nav item.
- **D-5** the role×surface matrix (lines 61-70) — **input, not adopted**; Phase 19 declares per-role rows, Phase 20 assembles the matrix (D-19-08).
- **D-6** route naming follows the ontology — out of scope for contracts (no route work).
- Rejected list (lines 81-85): pillar names as user-facing labels; dashboards/KPIs on Today; a customer-facing surface; HR near Review.

## 8. Gap register (verified, to be mirrored in `workflows/README.md`)

| # | Gap | Evidence |
|---|---|---|
| G1 | `middleware.RequireRole` defined + tested, never wired | `internal/middleware/middleware.go:46-61` |
| G2 | `GET /time-entries` and `GET /expenses` are org-wide for every authenticated role (RequestUserID/Role filters dead) | `internal/adapters/secondary/postgres/time_entry_repository.go:97-152`; `ports/time_entry_repository.go:29-31` |
| G3 | All 12 unit routes and all 8 working-group routes have no role gate | backend scan §1 (units, working groups) |
| G4 | `POST /invitations` unauthenticated; org id from body | `invitation.go:32-49`; `cmd/server/main.go:64` |
| G5 | `hr` invalid in Go (`Role.IsValid`), valid in the DB CHECK — HR jobs cannot be executed end-to-end | `models.go:19-27`; `012_staffing_schema.up.sql:48-50` |
| G6 | Export row scoping's `default` branch makes finance **and hr and customer** org-wide | `export_repository.go:29-37` |
| G7 | `POST /organizations/invite-customer` is a stub returning a canned message | `organization.go:84-98` |
| G8 | Availability windows are read-only — no write route exists for declarations or curation | `direction_repository.go:763-817` |
| G9 | `work_permit_expires_at` column has no Go reader | `migrations/012_staffing_schema.up.sql:41-42` |
| G10 | Governance models stored but no voting/quorum semantics implemented | `models.go:60-71`; scan §3 |
| G11 | D-E's deciding note is not on disk (only secondhand citations in ADR-P-003) | scan §1 (role/gaps) |

## 9. Archived v0.2 leftovers — job-shaped hints (verbatim wording)

- **TICK-06** — "view tickets and their status (Track pillar + Today 'my open tickets'); tickets are tracked work, auto-approved with permission control" → employee/manager ticket jobs.
- **AVAIL-03** — absence calendar (personal + team/org) · **AVAIL-04** — manager capacity per activity/WG (weekly hours − confirmed absences, workload from submitted+approved entries) · **AVAIL-05** — availability surfaces in People with role-scoped visibility → employee absence job; manager/HR capacity job.
- **SURF-01** week-1 allocation screen with on-read proposals + mandatory reasons (Review group) · **SURF-02** to-cover queue (explicit uncovered state, soft target) · **SURF-03** employee own coverage read-only · **SURF-04** bucket setup + balance under Economics → Contracts · **SURF-05** per-unit non-billed cost report (resoconto) in Reports → manager coverage/allocation jobs; employee own-coverage job; finance money-labeling + reporting jobs.
- **SURF-06** Today in both shapes — composition, not two pages · **SURF-07** direction scheduler (calendar, drag & drop, P-008 warnings) · **SURF-08** direction queue + direction-coverage read-model → employee Today job; manager/self direction jobs.
- **POLS-01..11** — polish targets for existing surfaces (Today, entries, expenses, approvals, WGs, activities, customers, contracts, exports, People/org tree, auth) → **quality bar for JOB-01, not frozen into contracts** (D-19-11).
- **UXFD-02** — sketch-loop process → SKETCH-01 (Phase 20).

## 10. Research conclusion for the planner

Authoring rules that follow from the facts above:

1. Contracts describe **jobs and surfaces**; existing routes appear only as `current state` evidence (D-19-11).
2. Every job's API-interaction column must cite a real method+path or be marked `absent`.
3. Anything the server does not enforce yet goes in the job's backend-authority note and the gap register (D-19-09).
4. HR and customer contracts must be honest about a role with no working backend path (G5) and a role with no surface at all (D-19-07).
5. Governance/voting promises (G10), write routes that do not exist (G8), and permit-expiry reads (G9) must not be implied as available.
