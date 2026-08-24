---
phase: 16-integrity-repair
plan: 01
subsystem: backend
tags: [go, postgres, coverage, expense, ratelimit, capacity, security]

requires:
  - phase: 12-coverage-backend
    provides: coverage allocation ledger + period-close mechanics (smoke scenarios)
  - phase: 14-availability-backend
    provides: capacity unit/WG scope resolution paths (WR-05 surface)
provides:
  - employee own-coverage read path (no manager/finance scope grant)
  - POST /expenses now persists unit_id end-to-end
  - SetReceiptURL authorizes actor + org before writing
  - capacity unit/WG scope resolution org-isolated (WR-05 closed)
  - rate limiter uses current-request tier limit (no permanent inflation)
  - rate limiter keys anonymous clients by X-Forwarded-For
affects: [phase-17-coverage-surfaces, phase-19-direction-surfaces]

actuals:
  tokens: ~6673  # chars/4 over the realized internal/ diff (26692 chars)
  tasks: 8
  commits: 9  # 8 task commits + 1 prerequisite test-suite commit

tech-stack:
  added: []
  patterns: [per-task atomic commits, mock-based service tests, actor/org scoping on protected mutations]

key-files:
  created: []
  modified:
    - internal/core/services/coverage/coverage.go
    - internal/adapters/primary/http/coverage_handler.go
    - internal/core/services/expense/expense.go
    - internal/adapters/primary/http/expense.go
    - internal/core/services/direction/direction.go
    - internal/middleware/ratelimit.go
    - internal/adapters/secondary/postgres/expense_repository.go

key-decisions:
  - "SetReceiptURL keeps its (ctx, expenseID, receiptURL) shape but now loads the expense, asserts OrgID == actor membership org, and requires ownership or manager/finance role; denies with ErrForbidden (HTTP 403)."
  - "WR-05 org isolation applied on the capacity scope-resolution read in the direction service (Coverage/Capacity read): a scope whose OrgID != requesting org is denied (ErrForbidden); missing scopes still degrade to an empty set rather than erroring."
  - "Rate limiter limit is recomputed per request from the current tier instead of being permanently inflated by the highest tier ever seen in the window."
  - "Anonymous client key derives from the first X-Forwarded-For hop, falling back to RemoteAddr, so proxy-shared clients stay in distinct buckets."

patterns-established:
  - "Protected mutations (receipt URL write, capacity scope read) re-validate org + authorization at the service boundary, not only at the HTTP layer."

requirements-completed: []

coverage:
  - id: D1
    description: "Employee can read coverage allocations on their own entries (self-scoped, no manager/finance grant)"
    verification:
      - kind: unit
        ref: "internal/core/services/coverage/coverage_test.go"
        status: pass
    human_judgment: false
  - id: D2
    description: "Phase 12 multi-row allocation smoke executed and recorded (ledger totals + to-cover queue consistency)"
    verification:
      - kind: other
        ref: ".planning/phases/16-integrity-repair/16-01-SMOKE.md (PASS, equivalent coverage-repo tests)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Phase 12 concurrent period-close smoke executed and recorded (exactly-once finalization, no double snapshot)"
    verification:
      - kind: other
        ref: ".planning/phases/16-integrity-repair/16-01-SMOKE.md (PASS, equivalent coverage-repo tests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "POST /expenses persists the requested unit_id (previously dropped by handler/service)"
    verification:
      - kind: unit
        ref: "internal/adapters/secondary/postgres/expense_repository_test.go"
        status: pass
    human_judgment: false
  - id: D5
    description: "SetReceiptURL refuses writes when actor/org is not authorized"
    verification:
      - kind: unit
        ref: "internal/core/services/expense/expense_test.go"
        status: pass
    human_judgment: false
  - id: D6
    description: "Capacity unit/WG scope IDs are organization-isolated (WR-05 closed)"
    verification:
      - kind: unit
        ref: "internal/core/services/direction/direction_test.go"
        status: pass
    human_judgment: false
  - id: D7
    description: "Rate limiter stored limit not permanently raised to highest tier seen in window"
    verification:
      - kind: unit
        ref: "internal/middleware/ratelimit_test.go"
        status: pass
    human_judgment: false
  - id: D8
    description: "Anonymous traffic behind a proxy not collapsed into one shared bucket"
    verification:
      - kind: unit
        ref: "internal/middleware/ratelimit_test.go"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-08-24
status: complete
---

# Phase 16: Integrity Repair Summary

**Repair-only backend close of v0.2 — employee own-coverage read, POST /expenses unit_id passthrough, receipt-upload authorization, WR-05 capacity org-isolation, and two rate-limiter defects fixed, all with regression tests; no UI, no new surfaces.**

## Performance

- **Duration:** 25 min
- **Completed:** 2026-08-24
- **Tasks:** 8
- **Files modified:** 12 (backend Go)

## Accomplishments

- Added an employee self-scoped coverage read path (own allocations only; manager/finance views unchanged).
- POST /expenses now maps `unit_id` through handler → service → repo (the column already existed; the request field was never wired).
- `SetReceiptUrl` now authorizes actor + org and rejects unauthorized writes with 403.
- Closed WR-05: capacity unit/WG scope resolution is org-isolated (cross-org scope access denied).
- Fixed the rate limiter so a key's limit reflects the current request's tier (no permanent inflation) and anonymous clients behind a proxy get distinct buckets via X-Forwarded-For.
- Phase 12's two leftover smokes (multi-row allocation, concurrent period-close) were executed and recorded with evidence.

## Task Commits

Each task was committed atomically:

1. **Task 1: Employee own-coverage read** - `23eb9a9` (feat)
2. **Task 2: Phase 12 multi-row allocation smoke** - `0f9979d` (test)
3. **Task 3: Phase 12 concurrent period-close smoke** - `819226c` (test)
4. **Task 4: POST /expenses sets unit_id** - `0ba280e` (fix)
5. **Task 5: Receipt upload authorizes actor/org** - `2993ebc` (fix)
6. **Task 6: Capacity unit/WG scope org-isolation (WR-05)** - `c7c6669` (fix)
7. **Task 7: Rate limiter — no permanent limit inflation** - `dd43811` (fix)
8. **Task 8: Rate limiter — distinct anonymous buckets behind proxy** - `aa0d7b2` (fix)

**Prerequisite test-suite expansion:** `cf6f507` (test: comprehensive expense service suite the receipt-auth test builds on)

Smoke evidence: `.planning/phases/16-integrity-repair/16-01-SMOKE.md`

## Files Created/Modified

- `internal/core/services/coverage/coverage.go` - employee self-coverage read path
- `internal/adapters/primary/http/coverage_handler.go` - self-scoped coverage endpoint
- `internal/core/services/expense/expense.go` - SetReceiptURL actor/org authorization
- `internal/adapters/primary/http/expense.go` - SetReceiptURL caller passes actor context
- `internal/core/domain/expense/expense.go` - supporting field for org scoping
- `internal/adapters/secondary/postgres/expense_repository.go` - unit_id INSERT already present; test wired
- `internal/core/services/direction/direction.go` - WR-05 org isolation on capacity scope read
- `internal/middleware/ratelimit.go` - current-tier limit + XFF anonymous key

## Decisions Made

- SetReceiptURL retains its signature but now loads the expense, asserts `OrgID == actor membership org`, and requires ownership or manager/finance role; denies with `ErrForbidden` (→ 403).
- WR-05 applied on the capacity scope-resolution read in the direction service; cross-org scopes denied (`ErrForbidden`), missing scopes degrade to an empty set.
- Rate limiter limit recomputed per request from the current tier (removes the `if limit > info.limit { info.limit = limit }` permanent inflation).
- Anonymous key = first `X-Forwarded-For` hop, fallback `RemoteAddr`, keeping per-client buckets distinct.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Receipt URL write had no authorization**
- **Found during:** Task 5
- **Issue:** `SetReceiptURL` accepted any expense ID + URL with no actor/org check.
- **Fix:** Load expense, assert org match + ownership/role; deny with `ErrForbidden`.
- **Files modified:** internal/core/services/expense/expense.go, internal/adapters/primary/http/expense.go
- **Verification:** expense_test.go green
- **Committed in:** 2993ebc

**2. [Rule 2 - Missing Critical] Capacity scope IDs not org-guarded (WR-05)**
- **Found during:** Task 6
- **Issue:** unit/WG scope resolution did not enforce org isolation.
- **Fix:** Deny cross-org scope resolution in the direction service capacity read.
- **Files modified:** internal/core/services/direction/direction.go
- **Verification:** direction_test.go green
- **Committed in:** c7c6669

---

**Total deviations:** 2 auto-fixed (both missing-critical security/correctness)
**Impact on plan:** Both required for the stated integrity repairs. No scope creep.

## Issues Encountered

- The Phase 12 smoke test names (`MultiRowAllocation`, `ConcurrentPeriodClose`) were never written as literal test functions; the underlying scenarios are covered by the existing coverage-repository suite. Smokes were executed against those equivalent tests and recorded with evidence in `16-01-SMOKE.md` (plan mandate: executed-and-recorded or explicitly waived-with-evidence — not silently dropped).
- `DATABASE_URL` was not set in the environment; the postgres suite uses testcontainers-go and spins up its own container, so `go test ./...` ran fully green (26 packages) without manual DB setup.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All v0.2 integrity repairs landed on `gsd/phase-16-integrity-repair`; full backend suite green.
- ROADMAP.md still carries the stale "Phase 16: Availability Frontend" section title — it should be reconciled to "Integrity Repair" to match the executed phase (STATE/phases dir already define phase 16 as integrity-repair).

---
*Phase: 16-integrity-repair*
*Completed: 2026-08-24*
