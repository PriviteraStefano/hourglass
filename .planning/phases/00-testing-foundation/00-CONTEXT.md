# Phase 0: testing-foundation - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish a testing foundation across the entire codebase — backend (Go) and frontend (React). This is a prerequisite for all future phases. Work proceeds in two waves: first write comprehensive tests across all domains, then batch-fix bugs discovered during testing. Includes a quick-scan probe pass before structured test writing begins.

</domain>

<decisions>
## Implementation Decisions

### Bug Discovery Strategy
- **D-01:** **Tests-that-fail-first** — Write tests for expected behavior first; failing tests naturally reveal bugs. Do NOT manually explore before tests.
- **D-02:** **Quick scan first** — Run a lightweight probe script (curl/endpoint walkthrough) before structured test writing to catch obvious 500s, CORS issues, missing auth checks.
- **D-03:** **Adversarial/break-it approach** — Actively probe edge cases: empty payloads, missing auth, boundary values, concurrent requests. Not just happy-path.
- **D-04:** **Log bugs to BUGS.md** — Accumulate findings in `.planning/phases/00-testing-foundation/BUGS.md` with severity, location, and description. Then batch-fix in wave 2.
- **D-05:** **Two-wave phase structure** — Wave 1: write all tests. Wave 2: fix all logged bugs from BUGS.md. Same phase, two major steps.
- **D-06:** **Cover all domains equally** — No priority bias; auth, users, orgs, units, time entries, projects, contracts, and customers all get thorough coverage.

### Backend Coverage Scope
- **D-07:** **All domains** — Every Go domain gets tests. No skip.
- **D-08:** **All three layers** — Handler integration tests (existing pattern) + service unit tests (new) + repository tests (new or extended). Full stack.
- **D-09:** **Test containers preferred for service tests** — Use Testcontainers for Go to spin up isolated SurrealDB instances per test run. Fall back to existing `GetTestDBWithNamespace` if test containers isn't feasible.
- **D-10:** **No coverage target** — Write thorough tests per domain without gating on a percentage. Pragmatic given near-zero baseline.

### Frontend Test Approach
- **D-11:** **Playwright E2E + Vitest/RTL** — Deepen Playwright E2E for all major user flows AND add Vitest + React Testing Library for component/hook/unit tests. Both.
- **D-12:** **Vitest/RTL scope: API client + hooks + forms** — Test the non-UI logic: `api.ts` HTTP client, React Query hooks (`useProjects`, `useTimeEntries`, etc.), and Zod form validation schemas. Skip pure UI primitives.
- **D-13:** **Playwright covers all CRUD flows** — Create, read, update, delete for each entity: time entries, projects, contracts, customers, org hierarchy.

### Service-Layer Testing
- **D-14:** **State transitions first** — Approval workflow state machines (draft→submitted→pending_manager→pending_finance→approved/rejected) get the deepest coverage. Most complex logic in the app.
- **D-15:** **Full role×action×state matrix** — Test every role×action×state combination for approval workflows, including unauthorized transitions (employee self-approval, out-of-order approvals, etc.).
- **D-16:** **Cover domain-level validation rules** — Business rules (time entry overlap, contract deletion with active entries, etc.) are tested at the service layer, not just via handlers.
- **D-17:** **Table-driven tests** — Go-idiomatic table-driven test pattern: one test function with slice of test cases covering happy path, validation errors, auth failures, edge cases.
- **D-18:** **Shared testdata package** — `internal/core/services/testdata/` with factory functions for reusable test entities. Reduces duplication.
- **D-19:** **Co-located service tests** — `_test.go` files next to source in existing service packages. Go convention, consistent with handler pattern.

### the agent's Discretion
- Table-driven test case selection — agent decides which specific cases per domain
- Shared testdata package shape — agent decides entity factory structure
- Vitest configuration details — agent sets up as needed
- Test containers setup — agent determines feasibility during planning/execution

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Testing Infrastructure & Patterns
- `.planning/codebase/TESTING.md` — Full testing patterns doc: frameworks, run commands, existing test structure, missing coverage
- `.planning/codebase/CONVENTIONS.md` — Code style, error handling, patterns for both Go and TypeScript
- `.planning/codebase/STRUCTURE.md` — Directory layout, where tests live, where to add new test files
- `.planning/codebase/ARCHITECTURE.md` — Hexagonal architecture, how layers connect
- `.planning/codebase/STACK.md` — Tech stack, Go version, available tools

### Existing Test References
- `internal/adapters/primary/http/auth_test.go` — Reference: largest handler test file (749 lines, 24 tests). Pattern for handler integration tests.
- `internal/adapters/primary/http/time_entry_test.go` — Reference: handler test pattern (64 lines, 3 tests).
- `internal/adapters/secondary/surrealdb/helpers.go` — `GetTestDBWithNamespace`, schema bootstrap for test DB setup
- `web/e2e/auth.spec.ts` — Reference: Playwright E2E test pattern (current frontend testing approach)
- `web/playwright.config.ts` — Playwright configuration

### Domain Models (for test data factories)
- `internal/core/domain/timeentry/timeentry.go` — TimeEntry domain model, states, transitions
- `internal/core/domain/expense/expense.go` — Expense domain model
- `internal/core/domain/user/user.go` — User/Organization domain
- `internal/core/domain/project/project.go` — Project domain
- `internal/core/domain/contract/contract.go` — Contract domain
- `internal/core/ports/*_repository.go` — Repository interfaces (needed for service test setup)
- `internal/models/models.go` — Shared Role, Status, Governance constants
- `web/src/types/api.ts` — Frontend API types for Vitest setup reference

### Prior Phase Context
- `.planning/phases/01-org-hierarchy-edge-driven/01-CONTEXT.md` — Prior context with existing patterns and decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/adapters/secondary/surrealdb/helpers.go:GetTestDBWithNamespace` — Creates isolated test SurrealDB with schema. Core test infrastructure for all backend tests.
- `auth_test.go:setupTestServer` — Test server wiring pattern with httptest.Server. Adaptable for all handler tests.
- `uniqueID()`, `uniqueEmail()`, `uniqueOrgName()` — Inline test data helpers. Pattern to centralize in testdata package.
- `web/src/lib/api.ts` — `api<T>()` HTTP client. Primary target for Vitest API client tests.
- `web/src/lib/query-client.ts` — Shared QueryClient. Testable in Vitest for query invalidation patterns.
- `web/playwright.config.ts` — Playwright config with webServer auto-start. Extend for additional E2E tests.

### Established Patterns
- **Backend test style:** Integration-test style with real SurrealDB, httptest.Server, no mocking framework. "Test what you ship" philosophy.
- **Handler test setup:** `setupTestServer(t)` returns `*testServer` with wired handler, server, client, db. t.Cleanup for teardown.
- **Middleware injection:** `middleware.SetUserID(req.Context(), uuid)` for auth context in handler tests.
- **Response parsing:** `json.NewDecoder(resp.Body).Decode(&result)` with `result["data"]` envelope unwrap.
- **Playwright E2E:** Page-object style with `test.describe`, `async/await`, `expect().toHaveURL()`, `expect().toBeVisible()`.

### Integration Points
- Handler tests wired through http.ServeMux in cmd/server/main.go patterns
- Service tests need repository interfaces from ports package
- Vitest needs jsdom/ happy-dom for React component tests
- Playwright E2E tests run against real dev server (auto-started by config)
- BUGS.md discovered during testing flows back to fixes in wave 2

</code_context>

<specifics>
## Specific Ideas

- Two-wave phase: (1) write all tests across all domains, (2) batch-fix all bugs logged in BUGS.md
- Quick-scan probe: lightweight curl/script that exercises all API endpoints to find obvious issues before deep test writing
- Approval workflow state machine as primary test target for service-layer tests
- Test containers preferred for DB isolation in service tests, falling back to current GetTestDBWithNamespace pattern
- Shared testdata package to centralize entity factories (currently duplicated as inline helpers per file)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 0-testing-foundation*
*Context gathered: 2026-05-18*
