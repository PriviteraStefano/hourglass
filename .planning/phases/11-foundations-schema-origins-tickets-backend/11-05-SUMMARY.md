---
phase: 11-foundations-schema-origins-tickets-backend
plan: 05
subsystem: api
tags: [go, hexagonal, origins, proposals, tickets, audit-logs, routing, adr-p-013, adr-be-016]

# Dependency graph
requires:
  - phase: 11-foundations-schema-origins-tickets-backend
    provides: migrations 014-017 (tickets/origins/audit_logs) + shared routing package (11-03, BE-014) + activity service (10)
provides:
  - origin axis end-to-end: OriginType + 5 ref fields on activities, per-type role gates (D-04), same-org ref validation (D-02), set-once immutability (D-03), customer_ticket open|triage state precondition (OQ5/ADR-BE-016)
  - employee proposal approval: POST /activities/{id}/approve-proposal via shared routing (D-G), is_active flip, synchronous proposal_approved audit row (D-12, T-11-08)
  - ticket domain package (entity, 4 kinds, 7 statuses, locked transition matrix incl. open→dismissed, terminal states, sentinels + JSON name map)
  - general audit domain package + GeneralAuditLogRepository (synchronous INSERT INTO audit_logs, payload JSONB round-trip proven)
  - base org-scoped TicketRepository (Get) for origin validation; plan 06 extends the port
affects: [11-foundations-schema-origins-tickets-backend plan 06 (ticket lifecycle service builds on ticket domain + audit repo), 12-funding-sources, 13-direction]

# Tech tracking
tech-stack:
  added: [none — pure Go on the existing pgx/testify/testcontainers stack]
  patterns:
    - "Origin discriminator + typed refs set once at creation; immutability enforced in the service (ErrOriginImmutable) AND by the repo (origin columns absent from UPDATE SET) — double-enforcement (D-03, T-11-10)"
    - "Proposal approval consumes the same routing.Service as entry approval — single shared instance in cmd/server wiring (D-G parity)"
    - "General audit naming: GeneralAuditLogRepository distinguishes the D-05 audit_logs port from the BE-012 entry-scoped AuditLogRepository (time_entry_approvals) that predates it"

key-files:
  created:
    - internal/core/domain/ticket/ticket.go
    - internal/core/domain/audit/audit.go
    - internal/core/ports/ticket_repository.go
    - internal/core/ports/audit_log_repository.go
    - internal/adapters/secondary/postgres/ticket_repository.go
    - internal/adapters/secondary/postgres/audit_log_repository.go
    - internal/core/services/testdata/mock_ticket_repo.go
    - internal/core/services/testdata/mock_audit_log_repo.go
    - internal/adapters/secondary/postgres/ticket_repository_test.go
    - internal/core/services/activity/activity_origin_test.go
    - internal/core/services/activity/activity_proposal_test.go
  modified:
    - internal/core/domain/activity/activity.go
    - internal/core/services/activity/activity.go
    - internal/adapters/secondary/postgres/activity_repository.go
    - internal/adapters/primary/http/activity_handler.go
    - internal/core/services/testdata/mocks.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main.go
    - cmd/server/main_test.go
    - internal/core/services/activity/activity_test.go
    - internal/adapters/secondary/postgres/activity_repository_test.go

key-decisions:
  - "General audit types named GeneralAuditLogRepository (port + postgres): ports.AuditLogRepository and postgres.AuditLogRepository already exist for the entry-scoped BE-012 audit (time_entry_approvals, zero consumers on the port but real legacy repo behavior); renaming the new D-05 types is the minimal collision-free path preserving both behaviors"
  - "Dead entry-scoped MockAuditLogRepo in testdata renamed MockTimeEntryAuditLogRepo; the MockAuditLogRepo name now serves the general audit.AuditLog port (it had zero usages before)"
  - "ApproveProposal flips is_active via the REPO Update directly (bypassing the service Update finance gate) — the routing approver check IS the gate, per plan"
  - "Actor self-approval check (actorID == ProposedBy) precedes routing: proposals by a WG-manager proposer would otherwise hit the D-11 skipToFinance rejection; the explicit order makes the no-self-approval error deterministic"
  - "Proposer primary-unit lookup: invalid unit IDs degrade to uuid.Nil (no MustParse panic on malformed data); routing short-circuits to the terminal role-gated resolution"

patterns-established:
  - "Per-type origin gate + same-org membership validation in the service, DB CHECK backstop in 015 (double enforcement, T-11-01/T-11-06)"
  - "Synchronous audit write pattern: auditRepo.Create called inline, error propagates (T-11-08) — no detached goroutine"

requirements-completed: [FND-01, FND-02, FND-04, TICK-01, TICK-05]

# Metrics
duration: 10min
completed: 2026-08-07
---

# Phase 11 Plan 5: Origins End-to-End + Proposal Approval + Ticket/Audit Foundation Summary

**Origin axis lands end-to-end (FND-01/02/04): activities carry a set-once origin (manager_assignment / employee_proposal / customer_ticket) with per-type role gates (D-04) and same-org ref validation (D-02); employee proposals are created is_active=false and approved via the shared BE-014 routing with a synchronous audit_logs write (D-12); the ticket domain (entity + 7-status vocabulary + locked transition matrix incl. open→dismissed) and the general audit_logs repository lay the foundation plan 06's ticket lifecycle builds on**

## Performance

- **Duration:** 10 min
- **Started:** 2026-08-07T10:28:28Z (first commit c6c478e)
- **Completed:** 2026-08-07T10:38:08Z
- **Tasks:** 3
- **Files modified:** 21 (11 created, 10 modified)

## Accomplishments

- **Ticket + audit foundation (Task 1):** new `ticket` domain package — `Ticket`/`TicketComment` entities, 4 kind constants, 7 status constants, the locked transition matrix (`CanTransition`, open→dismissed superset pinned in ADR-BE-016), `IsTerminalStatus` (closed/dismissed), `IsOwner`/`IsAssignee` predicates, 6 sentinels + `JSONNames` map. New `audit` domain package — general `AuditLog` (EntityType/EntityID/Action/ActorID/Comment/Payload) per D-05. New ports (`TicketRepository{Get}` org-scoped — extended by plan 06; `GeneralAuditLogRepository{Create}` synchronous), postgres repos (parameterized Get with nullable locals; INSERT INTO audit_logs with JSONB payload), two testdata mocks, and integration tests proving org-scoped Get + nullable handling + payload JSONB round-trip + NULL actor path.
- **Origin axis end-to-end (Task 2):** `OriginType` + `AssignedBy/AssignedTo/ProposedBy/ReviewedBy/TicketID` on Activity/ActivityResponse/both requests; `IsActive *bool` on create (COALESCE→true, legacy default unchanged); `ErrOriginImmutable` sentinel; three OriginType constants mirroring the 015 CHECK. Repo: SELECT/scan/GetAncestry carry the six columns; Create INSERT uses `COALESCE($19, true)`; UPDATE SET never touches origin columns. Service `Create(ctx, role, orgID, userID, req)` applies per-type gates: manager_assignment & customer_ticket → manager|finance; employee_proposal → proposed_by==actor (spoofing guard) + forced is_active=false; same-org membership via `orgRepo.GetMembership` (D-02); customer_ticket state precondition open|triage after `ticketRepo.Get` (OQ5/ADR-BE-016); reviewed_by on any create rejected (ADR-P-013); unknown type rejected. Update rejects origin fields with `ErrOriginImmutable` (T-11-10). Handler parses origin payloads (empty→nil), passes role/userID, maps ErrOriginImmutable→409.
- **Proposal approval + wiring (Task 3):** `ApproveProposal(ctx, role, orgID, actorID, activityID)` — employee_proposal-only gate, already-approved reject, no self-approval, proposer's primary-unit lookup, `routing.ResolveManagerStage` resolution (D-G parity), role-gated→manager|finance, skipToFinance→ErrForbidden (proposer-only approver), actor∈approver set, repo Update flips is_active directly (the approver check IS the gate), synchronous `proposal_approved` audit row. `POST /activities/{id}/approve-proposal` registered; main.go + main_test.go + handler_test_helper wire auditRepo/ticketRepo/routingSvc (single routing instance shared with time_entry).
- **Tests:** 25 TestActivityOrigin* service subtests (role gates, spoofing guard, same-org, state precondition, immutability per field, legacy create, D-12 force/reject) + 3 repo integration tests (create/read-back, update-keeps-origin, proposal persistence + legacy default) + 11 TestProposal* subtests (approver set, outsider, self-approval, wrong origin, already-approved, missing, skipToFinance, roleGated pass/fail, R-2 unit-manager fallback, end-to-end via the Task 2 create path). Full `make test` green (838 ok lines, 0 failures), `go vet ./...` clean.

## Task Commits

Each task was committed atomically:

1. **Task 1: Ticket + audit domain foundation, base ticket repo, audit_logs repo** - `c6c478e` (feat)
2. **Task 2: Origin fields end-to-end (domain, repo, service, handler) with role gates + immutability** - `53c27ac` (feat)
3. **Task 3: Proposal approval (ApproveProposal) + main.go wiring** - `4e1a9d2` (feat)

**Plan metadata:** (docs commit below)

## Files Created/Modified

- `internal/core/domain/ticket/ticket.go` - NEW: Ticket/TicketComment, 4 kinds + 7 statuses, locked transition matrix (open→dismissed), terminal states, sentinels + JSON name map
- `internal/core/domain/audit/audit.go` - NEW: general append-only AuditLog (D-05)
- `internal/core/ports/ticket_repository.go` - NEW: org-scoped Get (plan 06 extends)
- `internal/core/ports/audit_log_repository.go` - NEW: GeneralAuditLogRepository{Create} synchronous
- `internal/adapters/secondary/postgres/ticket_repository.go` - NEW: org-scoped Get, nullable locals, ErrTicketNotFound
- `internal/adapters/secondary/postgres/audit_log_repository.go` - NEW: GeneralAuditLogRepository.Create — parameterized INSERT INTO audit_logs, JSONB payload, NULL actor/comment
- `internal/core/services/testdata/mock_ticket_repo.go` + `mock_audit_log_repo.go` - NEW mocks (MockAuditLogRepo now general; entry-scoped one renamed MockTimeEntryAuditLogRepo in mocks.go)
- `internal/core/domain/activity/activity.go` - origin fields on entity + both requests, IsActive on create, ErrOriginImmutable, origin-type constants
- `internal/core/services/activity/activity.go` - DI + NewService grow orgRepo/ticketRepo/auditRepo/routing; Create role/userID + validateOrigin gates; Update immutability guard; ApproveProposal + contains
- `internal/adapters/secondary/postgres/activity_repository.go` - origin columns in baseActivityQuery/GetAncestry/scans; Create INSERT COALESCE(req.IsActive, true)
- `internal/adapters/primary/http/activity_handler.go` - origin payload DTOs + parsing, role/userID pass-through, ErrOriginImmutable→409, ApproveProposal handler
- `cmd/server/main.go` + `main_test.go` + `handler_test_helper.go` - auditRepo/ticketRepo construction, activity service DI, approve-proposal route
- Tests: `ticket_repository_test.go`, `activity_repository_test.go` (+3 origin integration), `activity_origin_test.go` (NEW), `activity_proposal_test.go` (NEW), `activity_test.go` (fixture + signature)

## Decisions Made

- **General audit naming:** the plan named the new port `AuditLogRepository`, but `ports.AuditLogRepository` (entry-scoped, time_entry.AuditLog → time_entry_approvals) already exists with zero consumers, and `postgres.AuditLogRepository`/`NewAuditLogRepository` back real legacy BE-012 behavior used by its own test. The new D-05 types are named `GeneralAuditLogRepository`/`NewGeneralAuditLogRepository` — the minimal collision-free path that keeps both behaviors intact. Documented in both port and repo files.
- **Mock name freed:** the dead entry-scoped `MockAuditLogRepo` (defined, never referenced) was renamed `MockTimeEntryAuditLogRepo`; the acceptance-criterion name `MockAuditLogRepo` now serves the general audit port.
- **ApproveProposal persists via the repo directly** — bypassing the service Update's finance gate, per the plan: the routing approver check IS the gate.
- **Self-approval checked before routing** so the no-self-approval error is deterministic even when the proposer would otherwise hit the D-11 skipToFinance path.
- **Primary-unit lookup degrades gracefully:** invalid unit IDs parse to uuid.Nil instead of panicking (routing then falls to the terminal role-gated resolution).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] General audit port/repo renamed GeneralAuditLogRepository**
- **Found during:** Task 1 (first compile of the new ports file)
- **Issue:** The plan's `ports.AuditLogRepository { Create(*audit.AuditLog) }` collides with the existing `ports.AuditLogRepository { Create(*time_entry.AuditLog) }` (time_entry_repository.go, entry-scoped BE-012 audit). Same for `postgres.AuditLogRepository`/`NewAuditLogRepository` (writes time_entry_approvals, used by its own test). Go forbids two types of the same name in a package.
- **Fix:** New D-05 types named `GeneralAuditLogRepository` (port + postgres) + `NewGeneralAuditLogRepository`. Entry-scoped behavior untouched. Rationale comments in both files.
- **Files modified:** internal/core/ports/audit_log_repository.go, internal/adapters/secondary/postgres/audit_log_repository.go
- **Verification:** go build clean; TestTicketAudit_AuditLogRoundTrip green; legacy TestAuditLogRepository_Create still green
- **Committed in:** c6c478e (Task 1 commit)

**2. [Rule 3 - Blocking] Dead entry-scoped MockAuditLogRepo renamed MockTimeEntryAuditLogRepo**
- **Found during:** Task 1 (planning the mock file)
- **Issue:** testdata already defines `MockAuditLogRepo` (Create with *time_entry.AuditLog, zero usages repo-wide). The plan's acceptance criterion requires `MockAuditLogRepo` in testdata for the general audit port.
- **Fix:** Renamed the dead mock to `MockTimeEntryAuditLogRepo` (semantics preserved); the new `mock_audit_log_repo.go` defines `MockAuditLogRepo` with the general audit.AuditLog signature.
- **Files modified:** internal/core/services/testdata/mocks.go, internal/core/services/testdata/mock_audit_log_repo.go
- **Verification:** go test ./internal/core/services/activity/ green (mocks compile into the fixture)
- **Committed in:** c6c478e (Task 1 commit)

**3. [Rule 3 - Blocking] Third activity-service construction site wired (handler_test_helper)**
- **Found during:** Task 2 (go build ./internal/...)
- **Issue:** `internal/adapters/primary/http/handler_test_helper.go` constructs `activitysvc.NewService(activityRepo, contractRepo, unitRepo)` — the plan's files list for Task 2 covered only the service/repo/handler + cmd/server.
- **Fix:** Wired `postgres.NewTicketRepository(pool)` + `postgres.NewGeneralAuditLogRepository(pool)` + `routingSvc` into the fixture, matching main.go's wiring.
- **Files modified:** internal/adapters/primary/http/handler_test_helper.go
- **Verification:** go build ./... clean
- **Committed in:** 53c27ac (Task 2 commit)

**4. [Rule 1 - Bug] Undefined `err` in Create handler after parse refactor**
- **Found during:** Task 2 (first compile of handler changes)
- **Issue:** The origin parsing used `svcReq.X, err = parseOptionalUUID(...)` but `err` only existed inside the `if err := json.NewDecoder(...)` scope.
- **Fix:** Declared `var err error` before building svcReq; simplified the five origin refs to direct parse calls (dropped an intermediate map-based helper).
- **Files modified:** internal/adapters/primary/http/activity_handler.go
- **Verification:** go build clean; handler tests green
- **Committed in:** 53c27ac (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (3 Rule 3 blocking, 1 Rule 1 bug)
**Impact on plan:** All fixes were compile/correctness requirements — naming collisions with pre-existing entry-scoped audit types, a construction site the plan's file list missed, and a scoping error in the new parsing code. No scope creep: all four preserve the plan's semantics exactly (the General* name is documented as the deliberate disambiguation).

## Issues Encountered

- None beyond the deviations above. The routing-based ApproveProposal tests needed real routing.Service instances over the testdata mocks (matching 11-03's routing tests) — the fixture in activity_origin_test.go exposes all DI slots, and the R-2 unit-manager fallback case seeds a manager membership on the proposer's primary unit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **Plan 06 (ticket lifecycle service) can build directly on:** the ticket domain (status vocabulary, locked transition matrix, sentinels + JSON name map), the general audit domain + repository (synchronous proposal_approved pattern proven end-to-end), and the base org-scoped `TicketRepository.Get` (the port plan 06 extends with state mutators, triage, comments, history, LoggedHours, HasNonTerminalActivities).
- **Origins land (FND-01/02/04):** POST /activities accepts origin payloads with D-04 role gates + D-02 same-org validation; origin refs immutable (D-03); proposals created is_active=false and approvable via POST /activities/{id}/approve-proposal with the shared BE-014 routing + synchronous audit (FND-02, D-12); activity responses expose stored origin refs (FND-04 read path; the Phase-13 derivation fallback is documented, not built).
- **Full suite green:** `make test` 0 failures across all packages; `go vet ./...` clean — wave 3 gate satisfied.

## Self-Check: PASSED

- Created files verified on disk: ticket.go, audit.go, both new port files, both new postgres repos, both new mocks, ticket_repository_test.go, activity_origin_test.go, activity_proposal_test.go, 11-05-SUMMARY.md
- Commits verified in git log: c6c478e (Task 1), 53c27ac (Task 2), 4e1a9d2 (Task 3)
- Verification commands: `go test ./internal/adapters/secondary/postgres/ -run 'TestTicket' -count=1` ok; `go test ./internal/core/services/activity/ -run 'TestProposal|TestActivityOrigin' -count=1` ok; `go build ./...` ok; `go test ./internal/core/services/... ./internal/adapters/secondary/postgres/ -count=1` all ok; `make test` 0 failures; `go vet ./...` exit 0

---
*Phase: 11-foundations-schema-origins-tickets-backend*
*Completed: 2026-08-07*
