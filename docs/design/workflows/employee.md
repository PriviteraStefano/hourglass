# Hourglass Employee Role Contract

## Purpose

This contract names the jobs a user acting as an **employee** performs in Hourglass, and the surfaces those jobs need. Jobs are workflows, not routes (D-19-02/D-19-11).

- **Authority split:** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary; `docs/design/CHROME.md` wins on the frame, sidebar, and page anatomy. This file adds jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **Not authorization:** every visibility statement here is UX scoping. The server's routing and role checks are authoritative; where enforcement is missing it is listed as a gap, never as a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added employee role contract

## Actor

- **Role string:** `employee`, carried by the current membership (`GET /auth/me` → memberships) and minted into the JWT claim. Switching organizations switches the active membership and therefore the role.
- **Derived hats:** an employee may also hold the **working-group manager or delegate** hat (WG `manager_id` / `delegate_ids`). That hat is not a membership role and is not `manager`; it makes that group's stage-1 queue visible and puts the holder inside the routing-resolved approver set, while the approve/reject action itself needs the `manager` membership role today — the hat-only path is a gap (`G12`), and the client already hides the action for it (`manager.md` `M-01`). Everywhere else in this contract, "employee" means the membership role alone.
- **What an employee is not:** not an approver beyond the derived WG hat, not an allocator of coverage, not a closer of periods, not a reader of org-wide reports, not the owner of commercial records (contracts, customers — that is `finance`, `F-02`/`F-03`), and `MUST NOT` be presented as able to approve their own submissions — self-approval is structurally impossible for entries and expenses, and the coverage owner is forbidden outright.
- A user whose membership role is `customer` has no app surface at all (`customer.md`, `C-01`); the `employee` jobs below `MUST NOT` be offered to them.

## Scope

Per-role rows (the assembled role×surface matrix is Phase 20, D-19-08):

| Surface | Employee sees | Evidence today |
|---|---|---|
| Today | own composition + "what waits on me" when an approver hat applies | `web/src/routes/_authenticated/-components/today-page.tsx` |
| `Record` surfaces (time, expenses, tickets) | own records and drafts | `web/src/routes/_authenticated/time-entries/`, `expenses/` |
| Tickets | own/assigned tickets only; tickets are internal | `E-05` (no route yet) |
| Coverage | own coverage, read-only | `web/src/routes/_authenticated/` (Phase 16 own-coverage read) |
| Direction | own plan + WG queue claims | `E-09`, `E-10` (no route yet) |
| Availability | own declared/confirmed windows | `E-08` (no write route — G8) |
| `Approve` | only via the WG manager/delegate hat → `M-01` | `web/src/lib/role-visibility.ts:25` |
| Org-wide entry/expense lists | **never intended** — currently reachable, see G2 | `G2` |
| Contracts, customers, org settings, org-wide exports | never | — |

## Jobs

### E-01 — Capture time

| field | value |
|---|---|
| Actor | employee |
| Trigger | I worked; I need the hours recorded before they can be approved |
| Steps | 1. Open the time surface. 2. Add a dated entry with one or more line items (hours per activity). 3. Save as draft. 4. Submit when the day/week is complete. |
| Surfaces | Time (list + editor) · Today (CTA) |
| API interactions | `POST /time-entries`, `PUT /time-entries/{id}`, `DELETE /time-entries/{id}`, `POST /time-entries/{id}/submit` |
| Current state | exists — `web/src/routes/_authenticated/time-entries/` |
| Backend authority | Server scopes ownership: an entry is editable while draft, submitted, or rejected, and deletable while draft; submit resolves the approval route (WG approver set, unit-manager fallback, commercial-without-WG rejected). A dismissed ticket blocks submission via the logged-hours guard. |

### E-02 — Capture expenses

| field | value |
|---|---|
| Actor | employee |
| Trigger | I paid for something the organization covers |
| Steps | 1. Open the expense surface. 2. Pick a category (mileage, meal, accommodation, other) and amount. 3. Attach the receipt. 4. Submit. |
| Surfaces | Expenses (list + editor) |
| API interactions | `POST /expenses`, `PUT /expenses/{id}`, `DELETE /expenses/{id}`, `POST /expenses/{id}/submit`, `POST /expenses/{id}/receipt` |
| Current state | exists — `web/src/routes/_authenticated/expenses/` |
| Backend authority | Owner-scoped: editable while draft, submitted, or rejected; deletable while draft. Submit resolves the same two-stage route; the receipt URL is writable by the owner or a manager/finance reviewer. |

### E-03 — Today: what is mine now

| field | value |
|---|---|
| Actor | employee |
| Trigger | I open the app at the start of a day |
| Steps | 1. Land on Today. 2. Read the composed answer (my week's work; what waits on me when I hold an approver hat). 3. Follow one CTA into the surface that resolves it. |
| Surfaces | Today (landing, read-only composition) |
| API interactions | `GET /time-entries`, `GET /expenses`, `GET /time-entries/pending` (approver hats only) |
| Current state | exists — `today-page.tsx`; approver section gated on `isApprover` |
| Backend authority | Read-only composition: Today `MUST NOT` create state (ADR-P-004 D-2 via `CHROME.md`), and `MUST NOT` become a dashboard of KPIs. The approver section is UX scoping over the pending queues. |

### E-04 — Follow my submissions

| field | value |
|---|---|
| Actor | employee |
| Trigger | I submitted something and need to know where it stands |
| Steps | 1. Open my entries/expenses. 2. Read status (draft → submitted → pending_manager → pending_finance → approved/rejected). 3. Open the detail for the approval history. 4. After a rejection, edit the draft and resubmit. |
| Surfaces | Time / Expenses (list + detail) · status vocabulary from `LANGUAGE.md` |
| API interactions | `GET /time-entries`, `GET /time-entries/{id}`, `GET /expenses`, `GET /expenses/{id}` |
| Current state | exists |
| Backend authority | Rejection returns the entry to an editable state; the approval history rows are immutable. Status *vocabulary* authority is `LANGUAGE.md`; the status set itself is the domain enum and the wording lives in the client surfaces. |

### E-05 — Raise internal demand

| field | value |
|---|---|
| Actor | employee (also as ticket owner or assignee) |
| Trigger | something needs tracking as work: a question, a bug, a change, an evolution |
| Steps | 1. Create a ticket with a kind. 2. Comment as work progresses. 3. Transition it along its lifecycle. 4. Close when the linked work is done. |
| Surfaces | Tickets (target surface — no route exists today) |
| API interactions | `POST /tickets`, `GET /tickets`, `GET /tickets/{id}`, `POST /tickets/{id}/comments`, `POST /tickets/{id}/transition`, `GET /tickets/{id}/history` |
| Current state | partial — backend complete, no frontend route; the nav item is a disabled placeholder |
| Backend authority | Customer role is forbidden outright; triage and dismissal are manager/finance jobs (`M-08`); transitions are limited to the owner/assignee or a manager/finance actor; dismissal is blocked while linked activities carry logged hours. |

### E-06 — Propose an activity

| field | value |
|---|---|
| Actor | employee |
| Trigger | I see work that should exist as an activity but was not assigned to me |
| Steps | 1. Create the activity as a proposal. 2. Wait for the approver resolved by routing. 3. Read the outcome. |
| Surfaces | Activities (list + detail) |
| API interactions | `POST /activities` (employee-proposal origin), `GET /activities`, `GET /activities/{id}` |
| Current state | partial — the activities surface exists; the proposal path is the employee-visible subset |
| Backend authority | Origin decides the gate: employee proposals are self-created and approved through the shared routing service; manager-assignment and customer-ticket origins are finance/manager jobs (`F-04`). |

### E-07 — See my own coverage

| field | value |
|---|---|
| Actor | employee |
| Trigger | I want to know how my logged hours were labeled (billed vs absorbed) |
| Steps | 1. Open my entry. 2. Read its coverage. |
| Surfaces | Coverage (own, read-only) |
| API interactions | `GET /coverage/own/{entry_id}` |
| Current state | exists — own-coverage read shipped with the integrity-repair phase |
| Backend authority | Read-only for the employee: allocation writes are a manager job (`M-04`) and the owner is forbidden from writing them. |

### E-08 — Declare absence or unavailability

| field | value |
|---|---|
| Actor | employee |
| Trigger | I will be away, on holiday, on permit, or otherwise unavailable |
| Steps | 1. Declare the window (kind + dates; full or partial hours). 2. Attach a certificate reference when the kind is medical. 3. Track whether it is declared or confirmed. |
| Surfaces | Availability (target surface — no route exists today) |
| API interactions | `absent` (no write route; reads feed the direction capacity read-model) — `G8` |
| Current state | absent — schema exists (`availability_windows`, kinds holiday/permit/medical/unavailable), reads are wired, writes are not |
| Backend authority | Nothing is enforced because nothing is exposed. The target rule: the employee declares; HR curates and confirms (`H-01`); the scheduler warns about the window and never blocks. |

### E-09 — Claim queued work

| field | value |
|---|---|
| Actor | employee (as a working-group member) |
| Trigger | my group's queue holds work nobody has taken |
| Steps | 1. Open the group queue. 2. Read the queued rows and their estimates. 3. Claim a row. 4. Release it if priorities change. |
| Surfaces | Direction queue (target surface — no route exists today) |
| API interactions | `POST /direction/claims`, `POST /direction/claims/{id}/cancel`, `GET /direction` |
| Current state | partial — backend complete; no frontend route |
| Backend authority | Membership of the working group is required (no role gate); the claim budget guard is enforced in-transaction against the summed claimed hours; claims are WG-queue rows only. |

### E-10 — Plan my own work

| field | value |
|---|---|
| Actor | employee, when the organization's planning mode is `self_planned` |
| Trigger | I plan my own days instead of receiving a plan |
| Steps | 1. Create a direction row for myself (scheduled date or queued). 2. Activate it. 3. Cancel with a reason when it lapses. |
| Surfaces | Direction (own plan — target surface, no route exists today) |
| API interactions | `POST /direction`, `POST /direction/{id}/activate`, `POST /direction/{id}/cancel`, `GET /direction` |
| Current state | absent on the client; backend complete |
| Backend authority | `self_planned` mode requires the actor to be the target; cancellation requires a reason; the org mode and any per-member override resolve server-side. Suppressing absence warnings is not possible: they are advisory only. |

### E-11 — Export my own rows

| field | value |
|---|---|
| Actor | employee |
| Trigger | I need my own timesheet/expense data outside the app |
| Steps | 1. Pick the export type and date range. 2. Download CSV or XLSX. |
| Surfaces | Exports |
| API interactions | `GET /exports/timesheets`, `GET /exports/expenses`, `GET /exports/combined` (+`/count` variants) |
| Current state | exists — `web/src/routes/_authenticated/exports/` |
| Backend authority | Row scoping is by role: an employee receives only their own rows. The 731-day maximum range is enforced server-side. |

## Do / don't

- Don't promote the WG manager/delegate hat into a role: it is a derived dimension that unlocks `M-01` for that group only.
- Don't treat the org-wide entry/expense list reads as employee capability — they are an enforcement gap (`G2`).
- Don't promise absence declarations, ticket surfaces, or direction surfaces as available: they have no route yet.
- Don't give the employee coverage writes, period closes, or org-wide reporting; name the owning job instead (`M-04`, `M-05`, `F-07`).
- Don't restate nav groups, tokens, or component APIs.

## Pointers

- `docs/design/CHROME.md` — frame, four lifecycle nav groups, role-scoped chrome.
- `docs/design/LANGUAGE.md` — status vocabulary for entry/expense states.
- `web/src/lib/role-visibility.ts` — the predicate mechanism this contract's scope rows agree with.
- `.planning/REQUIREMENTS.md` — EMP-01 (this contract) and the archived leftovers `TICK-06`, `AVAIL-03`, `SURF-03`, `SURF-06` used as job hints.
- `internal/core/services/routing/routing.go` — approval-stage resolution behind `E-01`/`E-02`.

## Not in this file

- No manager, finance, HR, or customer jobs (see the sibling contracts).
- No cross-role composition map (Phase 20, `COMPOSITION.md`).
- No page layouts, route definitions, component APIs, or implementation sequencing.
- No value tables (colors, spacing, sizes) — those live in the CSS and `LANGUAGE.md`.
- No Admin/Settings surface.

## Gaps

- `G2` — org-wide entry/expense list reads: an employee's "my entries" view is scoped by the client, not the server.
- `G8` — no absence/unavailability write route: `E-08` cannot be performed end-to-end today.
- `G12` — the WG manager/delegate hat can read its group's stage-1 queue but cannot approve or reject: those routes admit only the `manager`/`finance` JWT role and the stage-1 transition accepts only `manager`.
- `G1` — no route-level role enforcement exists to rely on; every statement above must be satisfied by service/handler checks.
- Ticket, direction, and availability surfaces have no frontend route yet (`E-05`, `E-09`, `E-10`, `E-08`) — their nav items are disabled placeholders.
