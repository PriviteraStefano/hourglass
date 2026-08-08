---
phase: 13-direction-backend-the-plan-plane
reviewed: 2026-08-08T00:00:00Z
depth: standard
files_reviewed: 38
files_reviewed_list:
  - cmd/server/main.go
  - cmd/server/main_test.go
  - hourglass-vault/decisions/backend/_index.md
  - hourglass-vault/decisions/project/_index.md
  - internal/adapters/primary/http/direction_handler.go
  - internal/adapters/primary/http/direction_handler_test.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/org_settings_handler.go
  - internal/adapters/primary/http/org_settings_handler_test.go
  - internal/adapters/secondary/postgres/direction_ontology_migrations_test.go
  - internal/adapters/secondary/postgres/direction_repository.go
  - internal/adapters/secondary/postgres/direction_repository_test.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/org_settings_repository.go
  - internal/adapters/secondary/postgres/org_settings_repository_test.go
  - internal/adapters/secondary/postgres/organization_repo.go
  - internal/core/domain/auth/membership.go
  - internal/core/domain/direction/direction.go
  - internal/core/domain/direction/errors.go
  - internal/core/domain/orgsettings/errors.go
  - internal/core/domain/orgsettings/orgsettings.go
  - internal/core/ports/direction_repository.go
  - internal/core/ports/org_settings_repository.go
  - internal/core/services/activity/activity.go
  - internal/core/services/activity/activity_origin_fallback_test.go
  - internal/core/services/activity/activity_origin_test.go
  - internal/core/services/activity/activity_test.go
  - internal/core/services/direction/direction.go
  - internal/core/services/direction/direction_test.go
  - internal/core/services/orgsettings/orgsettings.go
  - internal/core/services/orgsettings/orgsettings_test.go
  - internal/core/services/testdata/mock_direction_repo.go
  - internal/core/services/testdata/mock_org_settings_repo.go
  - internal/core/services/testdata/mocks.go
  - migrations/021_direction_rows.down.sql
  - migrations/021_direction_rows.up.sql
  - migrations/022_org_settings.down.sql
  - migrations/022_org_settings.up.sql
findings:
  critical: 0
  warning: 5
  info: 4
  total: 9
status: issues_found
---

# Phase 13: Code Review Report

**Reviewed:** 2026-08-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 38
**Status:** issues_found

## Summary

Reviewed the full direction plan-plane stack (ADR-P-015 / ADR-BE-018): migrations 021/022, the postgres `DirectionRepository` + `OrgSettingsRepository`, the direction/orgsettings domain + services, the two HTTP handlers, the shared routing/orgsettings seams, the activity-service origin-fallback integration, the `cmd/server` wiring, and the complete test surface (unit, integration, up/down/up migration cycles, handler e2e).

**Overall assessment:** strong implementation. The 13-09/13-10 gap-closure hardening is real and correctly covered: supersede-on-create writes both audit rows in-tx (CR-02, asserted at service, repo and e2e level), the nil-WgID claim guard mirrors the repo's `wg_id IS NOT NULL` lock predicate (CR-01, trap-tested), `wholeCent` caps at the DECIMAL(8,2) ceiling (WR-02, boundary-tested at 999999.99/1000000), the directed_to active-membership gate runs before any mode/routing decision (WR-01, 400-tested for nil/inactive membership), and `AuditActionUnclaimed` is aligned across domain/service/tests. The FOR UPDATE + in-tx Σ cents guard with the 5-way concurrent claim battery is the right concurrency closure. All hermetic service tests pass (`go test` on direction/orgsettings/activity packages), `go build ./...` and `go vet` are clean, and the handler fixture mirrors the production wiring.

Findings below are edge/robustness defects the current tests do not catch — no critical issues found, but five warnings deserve attention before Phase 19 consumes these read-models.

## Warnings

### WR-01: Coverage `scope=unit` accepts a non-UUID `scope_id` → 500 (client input path)

**File:** `internal/core/services/direction/direction.go:678-701` (case "unit" in `resolveScopeEmployees`)
**Issue:** The `employee` and `wg` scope branches validate `scope_id` with `uuid.Parse` → `ErrInvalidRequest` (400), but the `unit` branch passes the raw string straight into `s.unitRepo.ListMembers(ctx, scopeID)` and `s.unitRepo.GetDescendants(ctx, scopeID)`. The postgres adapter parses it (`unit_member_repository.go:25-29` — `uuid.Parse(unitID)` → raw `fmt.Errorf`), so `GET /direction/coverage?scope=unit&scope_id=garbage` surfaces a raw parse error → handler default → **500**, violating the phase's own T-13-32 contract ("no 500 path for client input"). The unit test at `direction_test.go:1052-1054` only exercises the mock (which returns empty without parsing), so the 500 path is untested. Secondary: `scope_id` is never org-validated for unit/wg scopes, so a manager can probe another org's unit/wg membership ids — the resulting "Outside validity period" warnings leak cross-org membership existence (low impact, but inconsistent with the same-org discipline applied everywhere else in this phase).
**Fix:**
```go
case "unit":
	if role != string(models.RoleManager) {
		return nil, directiondomain.ErrForbidden
	}
	unitID, err := uuid.Parse(scopeID)
	if err != nil {
		return nil, directiondomain.ErrInvalidRequest
	}
	members, err := s.unitRepo.ListMembers(ctx, unitID.String())
	if err != nil {
		return nil, err
	}
	// ... descendants with the parsed id
```

### WR-02: WG-row lifecycle permission is asymmetric with create (A10 not applied in `lifecycleAllowed`)

**File:** `internal/core/services/direction/direction.go:372-386` vs `:227-253,290`
**Issue:** Create routes WG rows on the WG's **anchored** activity (A10 — `wgActivityID = g.SubprojectID`, `managerReach(... wgActivityID ...)`), so a WG delegate in the approver set can create a WG row whose `activity_id` is a *descendant* of the anchor. But `lifecycleAllowed` resolves manager reach on **`d.ActivityID`** (the row's own activity, possibly the descendant): `ResolveManagerStage` then finds no WG anchored on the descendant, and for a commercial descendant returns `ErrActivityNotLoggable` → `ErrForbidden`, or degrades to the role-gated terminal stage. A delegate who legitimately created the row (and is in the anchored WG's approver set) gets **403 on Activate/Cancel** of that same row. The doc comment ("the routing degrades to the anchored-WG approver set") is only true when the row's activity is the anchor itself. Claim rows (origin inherited, activity = WG row's activity) inherit the same asymmetry on the Unclaim/lifecycle path.
**Fix:** resolve the lifecycle gate on the anchored activity for WG rows — fetch the WG by `*d.WgID` and use its `SubprojectID` in `managerReach`, mirroring the create path (A10).

### WR-03: Reversed period bounds (`period_start` > `period_end`) silently return empty data

**File:** `internal/adapters/primary/http/direction_handler.go:283-298` (`parsePeriod`), impact in `direction_repository.go:744-803` and `direction.go:860`
**Issue:** `parsePeriod` validates presence and format but not ordering. A reversed period (`period_start=2026-08-20&period_end=2026-08-10`) passes the boundary, `generate_series($3::date, $4::date)` produces zero days, `ListPlan`'s range predicate matches nothing, and `computeWarnings`' day loop never iterates — the API answers **200 with an empty plan/coverage**, which reads as "no work planned / no uncovered days" rather than a client error. A UI typo silently looks like a real (empty) plan. (An unbounded multi-year period also produces a very large `generate_series` + per-day loop — the ordering check plus a reasonable horizon guard would close both.)
**Fix:**
```go
if start.After(end) {
	return time.Time{}, time.Time{}, errors.New("period_start must be <= period_end")
}
```

### WR-04: Create commits the row before the warnings overlay — a warnings-read failure returns 500 after commit (duplicate-on-retry risk)

**File:** `internal/core/services/direction/direction.go:351-359`
**Issue:** `repo.Create` (with audit rows) commits first; only then does step 8 compute the warnings overlay via `computeWarnings` (which calls `AbsenceWindows` + `Coverage` + per-day `GetMembership`). If any of those reads fail (DB hiccup, connection drop), `Create` returns an error **after the row is durable** and the handler maps it to 500. An API client retrying the create will insert a duplicate direction row (supersede chains, claim budgets and the plan view all treat rows as distinct facts — D-W/D-AA multiplicity makes duplicates invisible to uniqueness checks). Either compute warnings from the pre-write pool reads, return the row with warnings best-effort on read failure, or document that 500-on-create is a may-have-committed response.
**Fix (minimal):** degrade warnings-read failures to an empty/partial overlay with the row still returned:
```go
if req.PlannedDate != nil && req.DirectedTo != nil {
	warnings, _ = s.computeWarnings(ctx, orgID, []uuid.UUID{*req.DirectedTo}, *req.PlannedDate, *req.PlannedDate)
}
```
(audit the failure; the write already committed — a 500 would trigger retries)

### WR-05: Org-settings multi-key PUT is not atomic across keys

**File:** `internal/core/services/orgsettings/orgsettings.go:140-170`
**Issue:** The service validates the full key set up front ("an invalid batch never partially commits" — true for validation), but each key is then upserted in its **own** transaction (`org_settings_repository.go:84-110`). A repo-level failure on the second key (connection loss, constraint edge) leaves the first key committed — a partial batch write, each half with its own audit row. The doc comment promises batch atomicity that the implementation does not deliver.
**Fix:** either loop the writes through a single shared tx (extend the port with a batch upsert), or downgrade the comment to per-key atomicity and document the partial-commit semantics.

## Info

### IN-01: Non-manager coverage self-view is a case-sensitive string compare

**File:** `internal/core/services/direction/direction.go:670`
**Issue:** `scopeID != actorID.String()` compares the raw query string before `uuid.Parse`. `uuid.Parse` accepts uppercase/mixed-case hex, so a non-manager passing their own UUID in a different case is 403'd for a legitimate self-view. Compare after parsing, or lowercase both sides.

### IN-02: Duplicate membership fetch in the create gate chain

**File:** `internal/core/services/direction/direction.go:265-272` + `internal/core/services/orgsettings/orgsettings.go:62-64`
**Issue:** `Create` calls `orgRepo.GetMembership` for the active-membership gate and `ResolvePlanningMode` immediately fetches the same membership again. Two pool round-trips per create for one row. Harmless but redundant — pass the already-fetched membership (or a mode resolver taking it) into the mode resolution.

### IN-03: Cancel/unclaim reason is not trimmed — whitespace-only passes

**File:** `internal/core/services/direction/direction.go:419,459` and `direction_repository.go:340`
**Issue:** `reason == ""` accepts `"   "` — a whitespace-only reason is persisted and audited as a "reason". Trim before the emptiness check (both service and repo boundary).

### IN-04: `SetupTestSchema` swallows migration failures

**File:** `internal/adapters/secondary/postgres/exported_test_helpers.go:60-75`
**Issue:** Migration errors are `t.Logf`'d, not `require.NoError`'d. A broken migration silently leaves a partial schema and every subsequent test fails with cryptic "relation does not exist" errors instead of pointing at the migration. Making this fatal converts whole-suite confusion into one actionable failure. (Pre-existing pattern, but this file is part of the phase's test surface.)

---

_Reviewed: 2026-08-08T00:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
