# Hourglass Role Contracts

## Purpose

The five role contracts name the **jobs** each role performs in Hourglass and the **surfaces** those jobs need. Jobs are workflows, not routes: an existing route appears only as `current state` evidence.

- **Authority stack (D-17-09):** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary. `docs/design/CHROME.md` wins on the frame, the sidebar, the four lifecycle nav groups (`Record` · `Organize` · `Approve` · `Report`), and page anatomy. These contracts add jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **No invented values:** these contracts `MUST NOT` copy CSS/px/oklch tables. Point at the live value stores by repo-root path.
- **Not authorization:** visibility and action gating described here are **UX scoping**. The server stays authoritative (D-19-12); where it does not enforce something today, that is a gap, not a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added role contracts (employee, manager, finance, hr, customer) and this index

## Contracts

| Contract | Role string | Job ids | Jobs |
|----------|-------------|---------|------|
| [employee.md](./employee.md) | `employee` | `E-01`…`E-11` | Capture time · capture expenses · Today · follow submissions · raise internal demand · propose an activity · own coverage · declare absence · claim queued work · plan own work · export own rows |
| [manager.md](./manager.md) | `manager` | `M-01`…`M-10` | Stage-1 approval · set the plan · read plan + capacity · allocate coverage · close period · shape the org · staff the work · triage tickets · planning policy · team exports |
| [finance.md](./finance.md) | `finance` | `F-01`…`F-08` | Stage-2 approval · commercial record · customer records · activities · money labels (read) · membership governance · org-wide reporting · cutoffs (read) |
| [hr.md](./hr.md) | `hr` | `H-01`…`H-05` | Curate availability · curate employment validity · people composition (shared with manager) · capacity + payroll view · never an approver |
| [customer.md](./customer.md) | `customer` | `C-01`…`C-03` | No app surface · the internal jobs that serve customers · what would reopen the decision |

## Job ids

- One namespace per contract: `E-` employee · `M-` manager · `F-` finance · `H-` hr · `C-` customer, each `NN` two-digit.
- Ids are **stable**. `docs/design/COMPOSITION.md` (Phase 20) and the job-cluster phases inserted after it cite jobs by id, never by title text.
- A job may be split, merged, or re-scoped inside its contract; the id namespace `MUST NOT` be renumbered.

## Scope rows

Each contract declares its own per-role visibility rows (what the role sees, what it never sees). The assembled role×surface matrix — ADR-P-011 D-5 as an input to confirm or revise — is **Phase 20** (`COMPOSITION.md`). A contract `MUST NOT` present its rows as the cross-role matrix.

## Gap register

Verified gaps between the contracts' target authorization and today's server behavior. Each contract's `Gaps` section repeats the ids that touch it.

| Id | Gap | Evidence |
|----|-----|----------|
| G1 | `middleware.RequireRole` is defined and unit-tested but never wired to a route; every gate is an inline equality check. | `internal/middleware/middleware.go:46` |
| G2 | `GET /time-entries` and `GET /expenses` are org-wide for every authenticated role — `ListFilters.Role` / `RequestUserID` are never consumed. | `internal/adapters/secondary/postgres/time_entry_repository.go:97` |
| G3 | All 12 unit routes and all 8 working-group routes carry no role gate. | `internal/adapters/primary/http/unit.go`, `working_group.go` |
| G4 | `POST /invitations` runs without Auth middleware and takes `organization_id` from the request body. | `internal/adapters/primary/http/invitation.go:32` |
| G5 | `hr` exists in the `organization_memberships` CHECK but not in `models.Role`; `Role.IsValid()` rejects it, so HR cannot be invited or assigned through the API and every equality gate fails closed. | `internal/models/models.go:19`, `migrations/012_staffing_schema.up.sql:49-50` |
| G6 | Export row scoping ends in a `default` branch that makes `finance`, `hr`, and `customer` org-wide. | `internal/adapters/secondary/postgres/export_repository.go:29` |
| G7 | `POST /organizations/invite-customer` is a stub that returns a canned message and calls no service. | `internal/adapters/primary/http/organization.go:84` |
| G8 | Availability windows are read-only: no write route exists for employee declarations or HR curation. | `internal/adapters/secondary/postgres/direction_repository.go:763` |
| G9 | `organization_memberships.work_permit_expires_at` has no Go reader; only `valid_from` / `valid_until` are consumed. | `migrations/012_staffing_schema.up.sql:42` |
| G10 | Governance models (`creator_controlled`, `unanimous`, `majority`) are validated and stored, but no voting or quorum semantics are implemented. | `internal/models/models.go:60` |
| G11 | The note that decided D-E ("no customer app surface") is not on disk; D-E survives only as secondhand citations. | `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md:11` |
| G12 | The WG manager/delegate hat reads its group's stage-1 queue but cannot act on it: the approve/reject routes admit only the `manager`/`finance` JWT role, and the stage-1 transition accepts only `manager`. | `internal/adapters/primary/http/time_entry.go:351`, `internal/core/services/time_entry/time_entry.go:193` |

Gaps `MUST NOT` be closed by editing these contracts; they are the delta list for the job-cluster phases inserted after Phase 20.

## Not in this file

- No chrome/layout contract (that is `CHROME.md`).
- No design-language vocabulary (that is `LANGUAGE.md`).
- No cross-role composition map (that is Phase 20, `COMPOSITION.md`).
- No route architecture, page layouts, component APIs, or implementation plans.
- No Admin/Settings surface (out of v0.2.1, SC2).
- No sketches or worked visual examples.

## Pointers

- `docs/design/LANGUAGE.md` — language authority.
- `docs/design/CHROME.md` — shell/navigation authority.
- `docs/history/planning/REQUIREMENTS.md` — EMP-01, MGR-01, FIN-01, HR-01, CUST-01 and the archived-leftover hint table.
- `cmd/server/main.go` — the route surface the `API interactions` columns cite.

## Gaps

- The assembled role×surface matrix does not exist yet (Phase 20, `COMPOSITION.md`).
- The page-action predicate (CHROME.md GAP A) is not built; contracts state which actions a role may take, not how the UI disables them.
- `docs/history/planning/phases/19-role-contracts/19-RESEARCH.md` carries the full verified fact base and the per-role capability inventory behind these contracts.
