# Hourglass — Context

Shared language for this repo. Read before naming anything in code, issues, tests, or docs. This file holds vocabulary and constraints; depth lives in the pointers at the bottom.

## What Hourglass is

Time entry and expense tracking with role-based approval workflows for organizations. Go (hexagonal) + PostgreSQL + React 19. Approval routing resolves through activity → working group → manager/delegate and is enforced at approval time.

## The three planes (the cardinal rule)

- **Direction** — the plan: per-day rows owned by a manager or by the person themselves. Mutable.
- **Facts** — captured effort: time entries and expenses. Immutable after approval.
- **Coverage** — the money label: allocations saying who pays. Mutable, snapshot-protected at cutoffs.

**The plan/decision never rewrites the fact.** Coverage allocations stay editable indefinitely; a cutoff is a reporting snapshot, not a lock. Gap between sold and actual is data, not a violation. Σ allocations over an entry's items equals the entry's hours.

## Vocabulary

| Term | Meaning |
|---|---|
| **Activity** | Recursive work entity (replaced projects/subprojects). Commercial context and billability derive through a CTE. |
| **Origin** | Where demand came from, recorded on an activity. |
| **Ticket** | Internal-only demand record — not task execution. `ticket → activity → entries` keeps a single-FK capture path. No customer-facing ticket portal. |
| **Contract** | Commercial agreement carrying `sold_hours`. |
| **Working group (WG)** | Unit of approval routing and staffing; carries a governance model. |
| **Governance model** | `creator_controlled` \| `unanimous` \| `majority` — how a contract/WG approves. |
| **Coverage allocation** | Money-labeling row splitting an entry's hours across funding sources. |
| **To-cover queue** | Entries whose coverage is missing or incomplete. |
| **Cutoff** | Finance reporting snapshot closing a period — not a lock. |
| **Absence / capacity** | Availability: declared absences, HR medical curation, work schedules. |
| **Employment validity** | HR-curated window during which a person's employment holds. |
| **Job** | A workflow a role performs, named in `docs/design/workflows/*` with a stable id (`E-01…`, `M-01…`, `F-01…`, `H-01…`, `C-01…`). Jobs are workflows, not routes. |
| **Job cluster** | A group of jobs implemented together in v0.2.1 — the unit of presentation work, not a route or a page. |

## Roles and states

- **Org membership roles** (DB CHECK, extended by `migrations/012_staffing_schema.up.sql`): `employee`, `manager`, `finance`, `customer`, `hr`. `internal/models` declares only the first four — check the consumer before relying on an `hr` value in Go.
- **Entry status**: `draft` → `submitted` → `pending_manager` → `pending_finance` → `approved` \| `rejected`.
- **Approval actions**: `submit`, `approve`, `reject`, `edit_approve`, `edit_return`, `partial_approve`, `delegate`.
- **Expense categories**: `mileage`, `meal`, `accommodation`, `other`.
- **Approval history is immutable** (`*_approvals` tables).

## Hard constraints

- PostgreSQL only (pgx/v5, hand-written SQL, no ORM); migrations append-only (ADR-BE-004).
- Hexagonal: business logic in `internal/core/services/*`, thin handlers in `internal/adapters/primary/http/*`, wiring in `cmd/server/main.go`.
- Auth: JWT in HttpOnly cookies, refresh-token rotation with reuse detection and family revocation. The server is authoritative for authorization; visibility rules in design contracts are UX scoping, not permission. Where the server does not enforce a rule today, that is a gap, not a permission.
- Presentation is contract-first: design language → chrome → role contracts → composition map → sketch → implement **by job cluster**, not by the current route tree.

## Pointers

- `docs/design/INDEX.md` — design documentation map (first stop for any `web/src` UI work).
- `docs/design/LANGUAGE.md` — authority on type, color, density, motion, status vocabulary.
- `docs/design/CHROME.md` — frame, sidebar, nav groups (`Record` · `Organize` · `Approve` · `Report`), page anatomy.
- `docs/design/workflows/README.md` — role contracts, job ids, gap register.
- `docs/codebase/` — architecture, structure, stack, conventions, testing, concerns.
- `docs/agents/` — issue tracker, triage labels, domain-doc rules for the engineering skills.
- `hourglass-vault/` — ADRs (`decisions/`), feature specs (`01-Features/`), schema docs (`03-Schema/`), research (`research/`).
- `docs/history/planning/` — frozen pre-migration planning records (GSD era).
