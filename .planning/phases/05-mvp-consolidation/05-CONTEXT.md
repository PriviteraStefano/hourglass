# Phase 5: MVP Consolidation - Context

**Gathered:** 2026-05-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Consolidate the project structure and create initial MVP demo seeding. Replace the existing seed data (`002_seed_tcg.surql`) with a clean, idempotent `003_seed_demo.surql` that populates all core entities so the app is immediately demonstrable. Structural cleanup is limited to deprecating the old seed file — no deep restructuring.

This phase is seed-focused. The Go CLI seed command is deferred to a future end-of-milestone phase.

</domain>

<decisions>
## Implementation Decisions

### Seed Entity Scope
- **D-01:** **Org + projects level** — Seed org, users, units, memberships, projects, contracts, and one customer. Time entries and expenses included as sample data (3-5 per employee).
- **D-02:** **Medium setup** — 3 contracts with 6 projects total. Realistic but not overwhelming.
- **D-03:** **One demo customer** — Required for projects to show customer relationships.
- **D-04:** **Full role spectrum** — 6 users: 2 managers, 1 finance, 3 employees. Covers all permission levels.
- **D-05:** **Fresh seed file** — `003_seed_demo.surql` with clean, consistent UUIDs. Not extending `002_seed_tcg.surql`.
- **D-06:** **Keep TCG theme** — Tech Consulting Group as the demo company.

### Seed Mechanism
- **D-07:** **Hybrid approach** — SurQL file first (immediately usable). Go CLI seed command deferred to a future end-of-milestone phase after Phase 0 (testing) verifies all APIs work.
- **D-08:** **Single monolithic seed file** — `003_seed_demo.surql`. Not split by domain.
- **D-09:** **All UUIDs throughout** — Consistent ID format, no mixed short-string IDs like `users:u108`.
- **D-10:** **Idempotent** — Use `IF NOT EXISTS` / `OR REPLACE` patterns so re-running `cmd/schema` is safe.
- **D-11:** **Old seed deprecated** — `002_seed_tcg.surql` renamed to `002_seed_tcg.deprecated.surql` so `cmd/schema` skips it. Kept for reference.
- **D-12:** **Bootstrap ≠ seed** — The existing `POST /auth/bootstrap` flow stays separate. Bootstrap creates the admin org+user. Seed populates demo data. They do not conflict.
- **D-13:** **Fixed demo credentials** — Pre-hashed bcrypt passwords in seed data (not plaintext). Demo login documented for easy access.

### Demo Scenario Design
- **D-14:** **Manager as primary demo persona** — Manager sees org hierarchy, all pages, and approval workflows. Best for showing the full system.
- **D-15:** **6 users total** — 2 managers (department leads), 1 finance user, 3 employees (time loggers). Each assigned to specific units.
- **D-16:** **Sample time entries** — 3-5 per employee from the past week. Shows time tracking pages with real data.
- **D-17:** **Sample expenses** — 1-2 per employee (mileage, meal). Shows expense pages with real data.

### Consolidation Scope
- **D-18:** **Seed-focused** — Consolidation means replacing the seed infrastructure. No deep structural changes, no API fixes.
- **D-19:** **Minimal deprecation** — Rename old seed file only. No cleanup of hardcoded UUIDs in test data or codebase.
- **D-20:** **Manual verification pass** — Run `cmd/schema`, start the server, log in as demo manager, check each page renders with data. Document any issues.

### the agent's Discretion
- Exact bcrypt hashes for demo passwords — agent picks the implementation approach
- Seed data structure and file formatting — agent decides SurQL style
- Time entry and expense sample data specifics — agent picks reasonable demo values
- Verification checklist format — agent documents findings as appropriate

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Schema & Seed
- `schema/001_schema.surql` — Database schema (all tables, fields, indexes, permissions)
- `schema/002_seed_tcg.surql` — Old seed file (source of truth for existing data patterns, will be deprecated)
- `cmd/schema/main.go` — Schema/seed loader (reads `*.surql` files alphabetically)

### Auth & Bootstrap
- `internal/adapters/primary/http/auth.go:Bootstrap` — Bootstrap handler (POST /auth/bootstrap)
- `internal/adapters/primary/http/auth.go:BootstrapCheck` — Bootstrap check (GET /auth/bootstrap-check)
- `internal/core/services/auth/service.go:Bootstrap` — Bootstrap service implementation
- `web/src/api/auth.ts:bootstrapCheckQueryOpts` — Frontend bootstrap check query

### Domain Models (for seed data structure)
- `internal/core/domain/user/user.go` — User domain model
- `internal/core/domain/project/project.go` — Project domain model
- `internal/core/domain/contract/contract.go` — Contract domain model
- `internal/core/domain/timeentry/timeentry.go` — TimeEntry domain model
- `internal/core/domain/expense/expense.go` — Expense domain model
- `internal/models/models.go` — Role, Status, Governance constants

### Frontend Routes (to verify seed data renders)
- `web/src/routes/_authenticated/contracts/` — Contracts page
- `web/src/routes/_authenticated/customers/` — Customers page
- `web/src/routes/_authenticated/projects/` — Projects page
- `web/src/routes/_authenticated/time-entries/` — Time entries page
- `web/src/routes/_authenticated/org-hierarchy/` — Org hierarchy page

### Prior Phase Context
- `.planning/phases/00-testing-foundation/00-CONTEXT.md` — Testing approach, patterns, infrastructure
- `.planning/phases/01-org-hierarchy-edge-driven/01-CONTEXT.md` — Org hierarchy patterns

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `schema/001_schema.surql` — Full SurrealDB schema with all table definitions, fields, indexes, permissions. Seed data must match these field types and constraints.
- `schema/002_seed_tcg.surql` — Existing seed as reference for SurQL syntax and patterns (will be deprecated).
- `cmd/schema/main.go` — Loads `*.surql` files alphabetically from `schema/` dir. New `003_seed_demo.surql` will be loaded after schema, before any future seeds.
- `internal/models/models.go` — Role constants (`employee`, `manager`, `finance`, `customer`), status constants — ensure seed matches allowed values.

### Established Patterns
- **SurQL seeding:** `CREATE table:id SET field = value` pattern in `.surql` files
- **UUID format:** `uuid` type fields use string UUIDs (e.g., `019df6f5-ea95-735d-888b-158583ae4516`)
- **Record links:** SurrealDB record links use `table:id` format (e.g., `organizations:u'8d152bac-...'`)
- **Time values:** `time::now()` for timestamps in seed data
- **Schema loading:** Files loaded alphabetically — `001_`, `002_`, `003_` ordering matters

### Integration Points
- Seed data connects to existing `cmd/schema/main.go` — create `003_seed_demo.surql`, deprecate `002_seed_tcg.surql`
- Demo credentials must be bcrypt-hashed to match the auth service's password verification
- Seed org UUID must be known for users to reference it with `RELATE` / record links
- Manual verification requires running `go run ./cmd/schema` then `go run ./cmd/server`, then logging in from the web UI

</code_context>

<specifics>
## Specific Ideas

- Replace the hardcoded/unwieldy UUID mix in `002_seed_tcg.surql` with clean, consistent UUIDs throughout
- Demo manager logs in and can see everything across all pages — best showcase
- The Go CLI seed command (`cmd/seed`) is a future improvement after Phase 0 verifies all APIs work
- Manual verification should check: login, org hierarchy page, projects list, contracts list, customers, time entries, expenses

</specifics>

<deferred>
## Deferred Ideas

### Go CLI Seed Command
Build a `cmd/seed` Go CLI that seeds data by calling API endpoints. Deferred because many APIs may have bugs — Phase 0 testing will reveal these. Best tackled at end of milestone once APIs are verified.

### Deep Consolidation / API Fixes
Known broken endpoints (unit members, org members) are out of scope. These belong in their own phase or can be fixed as bugs discovered by Phase 0 testing.

### Major Codebase Restructuring
Reorganizing handler/service layers, renaming files, aligning patterns — out of scope. This phase is seed-focused with minimal structural cleanup.

</deferred>

---

*Phase: 5-mvp-consolidation*
*Context gathered: 2026-05-19*
