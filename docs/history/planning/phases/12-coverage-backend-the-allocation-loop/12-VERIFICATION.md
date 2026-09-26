---
phase: 12-coverage-backend-the-allocation-loop
verified: 2026-08-08T13:00:00Z
status: human_needed
score: 8/8 must-haves verified
behavior_unverified: 2
overrides_applied: 0
behavior_unverified_items:
  - truth: "CR-01 fix: multi-row/multi-entry allocation sets with nil IDs never collide on the table-wide PK (regenerated uuid.Nil → uuid.New() inside ReplaceAllocations)"
    test: "Manual smoke test against a live DB: perform two PUT /time-entries/{id}/allocations with ≥2 rows each (two different entries), then verify both sets read back with distinct non-nil ids"
    expected: "Both sets persist; every stored row has a unique id; no 23505 PK violation on the second insert anywhere in the ledger"
    why_human: "The fix branch (a.ID == uuid.Nil → uuid.New()) runs on every handler PUT, but no automated test drives a multi-row set through the nil-ID path — the repo test helper sets ID: uuid.New() explicitly and the handler test uses single-row PUTs, so the exact collision scenario is never exercised by a test"
  - truth: "WR-03 fix: concurrent overlapping closes for the same org can never both commit (pg_advisory_xact_lock serializes closes before the overlap check)"
    test: "Manual two-goroutine test against a live DB: fire two concurrent POST /coverage/close for the same period, expect exactly one 201 and one 409"
    expected: "Exactly one close commits; the loser observes the winner's committed header via the xact-scoped advisory lock and returns ErrPeriodAlreadyClosed (409); never two frozen snapshots for one period"
    why_human: "The advisory lock is present and wired before the overlap check, but the concurrent-close race is not covered by any test — the existing concurrency battery (TestCoverageReplace_Concurrent) covers ReplaceAllocations only, and the fix report itself flags this as requiring human verification"
human_verification:
  - test: "Manual smoke: two PUTs of ≥2-row allocation sets for two different entries against a live DB (CR-01 fix)"
    expected: "Both replace-sets commit; every stored allocation row has a unique non-nil id; the table-wide PK never collides"
    why_human: "No automated test drives the multi-row/multi-entry nil-ID path — repo test helper sets explicit IDs, handler test uses single-row PUTs"
  - test: "Manual concurrent-close test: two goroutines POST /coverage/close for the same period (WR-03 fix)"
    expected: "Exactly one 201 and one 409; never two snapshots for one period"
    why_human: "The advisory xact lock is wired but the concurrent-close race has no automated test"
---

# Phase 12: Coverage Backend — The Allocation Loop — Verification Report

**Phase Goal:** The coverage plane works server-side: funding sources, per-entry coverage allocations with the Σ invariant, to-cover queue, proposals computed on read, one-step manager confirmation, and period-close snapshots (COV-01..05). ADR-P-012 accepted; BE encoding ADR written (incl. D-K polymorphic validation cost).
**Verified:** 2026-08-08T13:00:00Z
**Status:** human_needed (2 behavior-unverified concurrency items from the review-fix round need manual smoke tests; all automated checks pass)
**Re-verification:** No — initial verification (post-fix state, HEAD d7b1bdc includes review-fix commits 2ce8f7e..7c98973)

## Goal Achievement

### Observable Truths

Scored against the ROADMAP success criteria (the roadmap contract — all 8 present in ROADMAP.md Phase 12):

| # | Truth (Roadmap SC) | Status | Evidence |
|---|--------------------|--------|----------|
| 1 | Approved time entries receive 1..N coverage allocations; API rejects any state where Σ allocations ≠ entry hours (COV-01) | ✓ VERIFIED | Service Σ fast-fail (coverage.go:340-346) + authoritative in-tx Σ re-check under FOR UPDATE (coverage_repository.go:116-136); multi-row sets tested (5+3 commit test); `TestCoverageRepository_ReplaceAllocations_SumMismatchLeavesNoRows`; `TestCoverageReplace_Concurrent` battery proves no violating state commits; handler Σ-mismatch → 400 test |
| 2 | All five funding sources work: contract budget, support bucket, service request (zero-value), internal absorption (WarrantyBug/UnderEstimate/Goodwill + unit), cross-project transfer (justification) (COV-02) | ✓ VERIFIED | `DefaultSource` D-04 chain (coverage.go:94-114) — all 6 table-driven cases pass incl. sold=0 and sold=nil → service-request draw (A3); per-row vocabulary + ref-pinning validation incl. absorption reasons and transfer justification (coverage.go:374-406); 019 CHECKs |
| 3 | Proposals computed on read from entry + activity chain; no proposal table exists; only confirmed allocations persist (COV-03) | ✓ VERIFIED | `DefaultSource` is pure (no repos); `Propose`/`ToCoverQueue` compute on read; grep of migrations 018-020 shows no proposal table; only `coverage_allocations` + snapshot tables exist |
| 4 | A single manager confirmation suffices; every allocation change is audit-logged (COV-03) | ✓ VERIFIED | D-08 gate via shared `routing.ResolveManagerStage` — ApproverIDs OR (RoleGated && role == "manager"), owner structurally forbidden (coverage.go:348-369); audit rows in-tx (BE-016) with `allocations-set`/`coverage-closed` actions; handler permission matrix: manager 200, owner/finance/employee/customer 403 |
| 5 | Uncovered entries queryable via to-cover queue; allocations remain editable indefinitely (COV-01, COV-04) | ✓ VERIFIED | `ToCoverQueue` repo query (LEFT JOIN + HAVING, includes no-source entries) + service enrichment; handler test asserts entry2 with uncovered_hours 4 appears, flagged proposal for no-source activity; no is_locked/closed flag in 019 (prohibition respected); allocations editable via replace-set on any approved entry |
| 6 | Period close generates a reporting snapshot; reported period never changes retroactively; no lock on allocations (COV-04) | ✓ VERIFIED | `ClosePeriod` single-tx freeze (coverage_repository.go:325-464): advisory xact lock + in-tx overlap check (409, A6) + FOR UPDATE entry lock + header/rows/audit; `TestCoverageRepository_ClosePeriod_FreezesSnapshot` proves later edits never alter the snapshot; `ClosePeriod_DuplicateRejected` covers identical/contained/partial/wider/later periods; handler close test 201+rows, overlap 409, bad dates 400 |
| 7 | Coverage references a polymorphic entry (entry_type + entry_id); validation rejects non-time in v0.2 (COV-05) | ✓ VERIFIED | `entry_id UUID NOT NULL` with NO FK (019, D-K); schema CHECK `entry_type IN ('time')`; service D-K branch rejects non-'time' (coverage.go:333-337); test asserts non-time → ErrInvalidRequest |
| 8 | Beneficiary unit nullable on activities, inherited downward like contract_id; absorption sources default from it (COV-05) | ✓ VERIFIED | Migration 018 (nullable column + index); `ResolveBeneficiaryUnit`/`ResolveFundingContext` CTE resolvers in activity_repository.go; `DefaultSource` absorption case uses resolved unit; `TestActivityBeneficiaryUnit*` integration tests (inheritance walk, nil-when-none) pass; handler/service same-org validation on Create+Update |

**Score:** 8/8 truths verified (0 present-but-behavior-unverified among roadmap SCs; 2 review-fix behaviors — CR-01 multi-row nil-ID, WR-03 concurrent close — are wired but not test-exercised, detailed below)

### Review-Fix Verification (post-fix state, commits 2ce8f7e..7c98973 on main)

All 7 review findings verified fixed in code + regression tests:

| Finding | Fix verified in code | Test evidence |
|---------|---------------------|---------------|
| CR-01 allocation-id collision | `if a.ID == uuid.Nil { a.ID = uuid.New() }` in `ReplaceAllocations` before INSERT (coverage_repository.go:152-155) | Every handler PUT drives the nil-ID branch through the real repo (handler DTO carries no id); stored rows read back with data. Multi-row/multi-entry collision scenario itself is NOT test-exercised → human verification item |
| WR-01 fractional-cent hours | `a.Hours <= 0 \|\| math.Round(a.Hours*100) != a.Hours*100` → ErrInvalidRequest (coverage.go:381-383) | `fractional-cent hours rejected (WR-01)` service test PASSES (run) |
| WR-02 inverted close period | `periodStart.After(periodEnd)` → ErrInvalidRequest before repo call (coverage.go:488-490) | `inverted period rejected (WR-02)` service test PASSES (run) |
| WR-03 close overlap race | `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))` at tx start, before overlap check (coverage_repository.go:333-336) | Wired; no concurrent-close automated test → human verification item (fix report agrees) |
| WR-04 activity Update kind/governance validation | `Update` mirrors Create: `KindExists` + `GovernanceModel.IsValid()` (activity.go:247-262) | `TestService_UpdateRefValidation` (kind not in catalog, invalid governance model) PASSES (run) |
| WR-05 sentinel normalization + Update contract ref validation | Create/Update normalize `ErrContractNotFound`/`ErrUnitNotFound` → `activitydomain.ErrInvalidRequest`; Update adds contractRepo.Get + org-visibility (activity.go:105-135, 273-305) | `TestService_UpdateRefValidation` (missing contract, cross-org contract rejected, same-org accepted) + `TestService_Create` PASS (run) |
| WR-06 session-timezone-dependent ::date casts | `db.NewPool` pins `timezone=UTC` (db.go:54-59); ClosePeriod predicates/inserts use `($n AT TIME ZONE 'UTC')::date` and `(entry_date AT TIME ZONE 'UTC')::date` (coverage_repository.go:348-395) | `TestCoverageRepository_ClosePeriod_Scope` (boundary-inclusive) + overlap/duplicate-close integration tests PASS (run) |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| migrations/018_activity_beneficiary_unit.{up,down}.sql | beneficiary_unit_id UUID REFERENCES units(id) + idx; down drops both | ✓ VERIFIED | Column + index present; down drops index then column (IF EXISTS) |
| migrations/019_coverage_allocations.{up,down}.sql | Tagged-union ledger: 6 constraints (source_check 3VL, source_type_check, reason_check, justification_check, reason_vocab_check, entry_type_check), 4 indexes, hours DECIMAL(8,2) CHECK > 0, entry_id no FK | ✓ VERIFIED | All constraints by exact name; nullable source_type (3VL legacy pass); entry_id NO REFERENCES (D-K); down CASCADE |
| migrations/020_coverage_snapshots.{up,down}.sql | coverage_period_closes + coverage_snapshot_rows (close_id ON DELETE CASCADE), 2 indexes, no aggregates, no UNIQUE on period (409 is repo-level) | ✓ VERIFIED | Exact shape; entry-level rows only; down rows-then-header CASCADE |
| internal/core/domain/coverage/coverage.go | 5 structs + 10 vocabulary constants | ✓ VERIFIED | CoverageAllocation, CoverageProposal, ToCoverQueueRow, PeriodClose, SnapshotRow; all constants match schema CHECKs |
| internal/core/domain/coverage/errors.go | 6 sentinels + JSONNames | ✓ VERIFIED | ErrEntryNotCoverable, ErrAllocationSumMismatch, ErrPeriodAlreadyClosed, ErrForbidden, ErrInvalidRequest, ErrNotFound |
| internal/core/ports/coverage_repository.go | 7-method replace-set-only port | ✓ VERIFIED | Exactly ReplaceAllocations/ListByEntry/ToCoverQueue/BucketBalance/ClosePeriod/GetSnapshot/ListHistory; no incremental CRUD |
| internal/core/services/coverage/coverage.go | Service + NewService + DefaultSource + 7 methods | ✓ VERIFIED | Full 7-step ReplaceAllocations, D-04 chain, read gates, ClosePeriod orchestration |
| internal/adapters/secondary/postgres/coverage_repository.go | 7 methods, in-tx audits, FOR UPDATE, advisory lock, UTC casts | ✓ VERIFIED | Compile-time port assertion; all tx semantics in place |
| internal/adapters/primary/http/coverage_handler.go | 8 handler methods + writeError + DTOs | ✓ VERIFIED | Sentinel map 404/400/403/409/500; parse errors → 400; org from claims only |
| cmd/server/main.go | coverage stack wiring + 8 routes | ✓ VERIFIED | 8 mux.HandleFunc registrations under middleware.Auth; single shared routingSvc (1 NewService instance) |
| ADR-P-012 | Status Accepted + Acceptance section + Implemented-by link | ✓ VERIFIED | `**Status:** Accepted`; links ADR-BE-017; dated 2026-08-07 acceptance note |
| ADR-BE-017 — Coverage Encoding | 10 sections incl. D-K cost, OQ resolutions | ✓ VERIFIED | All markers present (source_check, 3VL, allocations-set, coverage-closed, IS NOT DISTINCT FROM 0, FOR UPDATE, ResolveManagerStage, financial_cutoff_periods, entry_type, 409, raw balance); D-K cost stated honestly ("one extra validation branch") |
| Vault indexes | ADR-BE-017 row + ADR-P-012 Accepted | ✓ VERIFIED | backend/_index.md row with Accepted; project/_index.md status cell Accepted |
| Test suites (12-01..12-07) | Migration cycles, repo integration, service unit, handler matrix | ✓ VERIFIED | All run green (below) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| coverage_handler.go PutAllocations | coveragesvc.ReplaceAllocations | `h.service.ReplaceAllocations(ctx, orgID, entryID, allocs, userID.String(), role)` | ✓ WIRED | Handler thin; all invariants in service |
| coveragesvc.ReplaceAllocations | repo.ReplaceAllocations | audit log built in service, handed to repo for in-tx write | ✓ WIRED | audit.AuditLog{EntityType: coverage_allocation, Action: allocations-set} |
| repo.ReplaceAllocations | time_entries | `SELECT hours ... FOR UPDATE` + in-tx Σ re-check | ✓ WIRED | CR-01 closure; concurrent battery green |
| repo.ClosePeriod | coverage_period_closes / coverage_snapshot_rows | advisory lock → overlap check → FOR UPDATE entries → header+rows+audit → Commit | ✓ WIRED | Freeze semantics proven by integration tests |
| service.ClosePeriod | repo.ClosePeriod | closeID := uuid.New() + coverage-closed audit | ✓ WIRED | 409 propagates (test) |
| service (gate) | routing.ResolveManagerStage | shared *routing.Service from main.go wiring | ✓ WIRED | Single instance (grep: 1 NewService); RoleGated && role=="manager" mandatory |
| activity service | contractRepo/unitRepo | GetByID + org-visibility fetch-and-compare (Create + Update) | ✓ WIRED | WR-04/WR-05 fixes in place; tests pass |
| main.go | routes | 8 mux.HandleFunc + middleware.Auth | ✓ WIRED | Exact paths per plan; handler_test_helper mirrors wiring |
| ADR-P-012 | ADR-BE-017 | Implemented by: link | ✓ WIRED | Link resolves |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| ToCoverQueue (repo) | uncovered hours | time_entries LEFT JOIN coverage_allocations, HAVING | Real SQL aggregation | ✓ FLOWING |
| BucketBalance | balance | contracts.sold_hours − SUM(ca.hours), adoption-aware pre-check | Real SQL aggregation; negatives as-is | ✓ FLOWING |
| ClosePeriod snapshot rows | frozen refs | live coverage_allocations read in-tx under lock | Real data copied, never re-read later | ✓ FLOWING |
| GetSnapshot | PeriodClose+rows | coverage_period_closes + coverage_snapshot_rows | Real frozen rows | ✓ FLOWING |
| Propose | proposal + current allocations | ResolveFundingContext/ResolveBeneficiaryUnit + ListByEntry | Real chain data + ledger | ✓ FLOWING |
| ListHistory | audit stream | audit_logs WHERE entity_type='coverage_allocation' | Real JSONB payload unmarshaled | ✓ FLOWING |

### Behavioral Spot-Checks (run, not just enumerated)

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Build compiles | `go build ./...` | exit 0 | ✓ PASS |
| Coverage service suite (DefaultSource 6 cases, gate matrix, Σ, D-K, WR-01, WR-02) | `go test ./internal/core/services/coverage/ -count=1` | ok | ✓ PASS |
| Activity service suite (incl. TestService_UpdateRefValidation WR-04/WR-05) | `go test ./internal/core/services/activity/ -count=1` | ok | ✓ PASS |
| Coverage repo integration (replace/queue/balance/close/history/concurrency) | `go test ./internal/adapters/secondary/postgres/ -run 'TestCoverageRepository\|TestCoverageReplace_Concurrent' -count=1` | ok (5.9s, testcontainers) | ✓ PASS |
| Migration cycles 018/019/020 (3VL + 23514 assertions) | `go test ./internal/adapters/secondary/postgres/ -run 'TestMigration018\|TestMigration019\|TestMigration020' -count=1` | ok | ✓ PASS |
| Beneficiary unit + resolvers integration | `go test ./internal/adapters/secondary/postgres/ -run 'TestActivityBeneficiaryUnit\|TestResolveBeneficiaryUnit\|TestResolveFundingContext' -count=1` | ok | ✓ PASS |
| Handler permission matrix + sentinel battery + no-DELETE | `go test ./internal/adapters/primary/http/ -run 'TestCoverageHandler' -count=1` | ok (9.0s) | ✓ PASS |
| Full suite (phase gate) | `make test` | 22 packages ok, 0 FAIL | ✓ PASS |

### Probe Execution

N/A — no probes declared in PLANs or SUMMARYs (standard test-driven phase, no probe scripts).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
| ----------- | -------------- | ----------- | ------ | -------- |
| COV-01 | 12-01, 12-04, 12-05, 12-06, 12-07 | Approved entries coverable 1..N; Σ invariant; to-cover queue | ✓ SATISFIED | ReplaceAllocations (service+repo), ToCoverQueue, concurrent battery, handler tests |
| COV-02 | 12-01, 12-04, 12-05, 12-06 | Funding sources: budget/support/service-request/absorption/transfer | ✓ SATISFIED | DefaultSource D-04 + per-row validation + 019 CHECKs + BucketBalance |
| COV-03 | 12-02, 12-04, 12-05, 12-07 | One-step manager confirm; audit-logged; proposals on read | ✓ SATISFIED | D-08 gate, in-tx audits, pure DefaultSource, handler matrix |
| COV-04 | 12-01, 12-06, 12-07 | Allocations editable; period-close snapshot never a lock | ✓ SATISFIED | ClosePeriod freeze + no lock flag + GetSnapshot/ListHistory |
| COV-05 | 12-01, 12-03, 12-05 | Nullable beneficiary unit inherited; absorption default; polymorphic entries | ✓ SATISFIED | 018 + ResolveBeneficiaryUnit/ResolveFundingContext + D-K branch + CHECK |

All 5 requirement IDs are claimed by at least one plan and satisfied by verified code. No orphaned requirements (REQUIREMENTS.md maps COV-01..05 all to Phase 12, all Complete).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/placeholder markers in any phase file (scanned 12 files) | ℹ️ none | — |
| IN-01 (documented, out of scope) | coverage.go:153-169 | Propose checks entry existence before read gate (404 vs 403 ordering leaks entry existence) | ℹ️ Info | Accepted in fix report; low risk (UUIDs, no data leaked) |
| IN-02 (documented, out of scope) | activity_handler.go:109-112 | Handler scope default "owned" vs domain "own" — works via repo default branch | ℹ️ Info | Accepted; future repo refactor risk noted |
| IN-03 (documented, out of scope) | testdata/mocks.go, coverage_test.go | Test mocks set explicit IDs — the reason CR-01 was invisible at mock layers | ℹ️ Info | Accepted; CR-01 fix lives in postgres repo which integration tests exercise |

### Human Verification Required

Two review-fix behaviors are wired in code but have no automated test — both flagged "requires human verification" in 12-REVIEW-FIX.md and confirmed by this verifier:

### 1. CR-01 multi-row/multi-entry nil-ID collision avoidance

**Test:** Manual smoke against a live DB: perform two `PUT /time-entries/{id}/allocations` calls with ≥2 rows each, targeting two different entries (the handler DTO carries no ids, so all rows travel as uuid.Nil to the repo).
**Expected:** Both replace-sets commit; every stored allocation row reads back with a distinct non-nil id; no 23505 PK violation on the second insert anywhere in the ledger.
**Why human:** The fix branch (`a.ID == uuid.Nil → uuid.New()`, coverage_repository.go:152-155) executes on every handler PUT, but no automated test drives a multi-row set through the nil-ID path — the repo test helper (`contractAllocation`) sets `ID: uuid.New()` explicitly and the handler test uses single-row PUTs, so the exact collision scenario is never exercised.

### 2. WR-03 concurrent overlapping closes can never both commit

**Test:** Manual two-goroutine test against a live DB: fire two concurrent `POST /coverage/close` calls for the same period.
**Expected:** Exactly one 201 and one 409; never two frozen snapshots for one period (the advisory xact lock serializes closes; the loser's overlap check sees the winner's committed header).
**Why human:** The `pg_advisory_xact_lock(hashtextextended($1::text, 0))` (coverage_repository.go:333-336) is present and wired before the overlap check, but the concurrent-close race has no automated test — the existing concurrency battery covers `ReplaceAllocations` only.

### Gaps Summary

No blocking gaps found. All 8 roadmap success criteria are verified with passing behavioral tests; all 7 review-fix findings (CR-01, WR-01..WR-06) are confirmed fixed in the post-fix codebase (commits 2ce8f7e..7c98973 on main); the full suite is green. Two concurrency-adjacent fix behaviors (CR-01 multi-row nil-ID path, WR-03 concurrent-close serialization) are present and wired but not covered by automated tests — they are routed to human verification above rather than scored as verified.

---

_Verified: 2026-08-08T13:00:00Z_
_Verifier: the agent (gsd-verifier)_
