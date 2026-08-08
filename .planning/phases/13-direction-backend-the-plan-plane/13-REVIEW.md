---
phase: 13-direction-backend-the-plan-plane
reviewed: 2026-08-08T12:00:00Z
depth: standard
files_reviewed: 36
files_reviewed_list:
  - cmd/server/main.go
  - cmd/server/main_test.go
  - internal/adapters/primary/http/direction_handler.go
  - internal/adapters/primary/http/direction_handler_test.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/org_settings_handler.go
  - internal/adapters/primary/http/org_settings_handler_test.go
  - internal/adapters/secondary/postgres/activity_ontology_migration_test.go
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
  - migrations/021_direction_rows.up.sql
  - migrations/021_direction_rows.down.sql
  - migrations/022_org_settings.up.sql
  - migrations/022_org_settings.down.sql
findings:
  critical: 2
  warning: 3
  info: 4
  total: 9
status: issues_found
---

# Phase 13: Code Review Report

**Reviewed:** 2026-08-08T12:00:00Z
**Depth:** standard
**Files Reviewed:** 36
**Status:** issues_found

## Summary

The phase implements the plan plane per ADR-P-015/ADR-BE-018: migrations 021/022 (direction + org_settings), the direction repository (mutator txs with FOR UPDATE re-validation, Σ-cents claim guard, read-models), the direction service (gate chain, warning overlay, read gates), the org_settings vertical slice, the activity origin fallback, and full wiring. The codebase is exceptionally well-documented and the test suite is deep (concurrent claim battery, supersede-chain Σ invariants, migration up/down/up cycles). The core design — in-tx audit writes, CR-01 lock closures, cents arithmetic, org-scoped reads — is sound and consistently applied.

Two critical defects were found: a nil-pointer panic in the Claim fast-fail path (triggerable by any authenticated user with a user-targeted row id), and the missing `superseded` audit row on supersede-on-create (the service passes one audit row where the port contract and ADR pin two). Three warnings cover the missing same-org check on `directed_to`, an unvalidated absurd-value path that turns client input into a 500 (DB numeric overflow), and audit-vocabulary drift (`unclaimed` never written). Four info items note minor robustness/consistency gaps.

## Critical Issues

### CR-01: Claim fast-fail dereferences nil `WgID` — panic on user-targeted row

**File:** `internal/core/services/direction/direction.go:462`
**Issue:** `Service.Claim` reads the row, checks only `wg.Status != StatusActive`, then calls `s.wgRepo.ListMembers(ctx, *wg.WgID)`. When the claimed row is a **user-targeted row** (`wg_id` NULL — including any claim row or any activated personal row), `wg.WgID` is nil and the dereference panics. The repo's lock query guards with `AND wg_id IS NOT NULL` (returns `ErrDirectionNotFound`), but the service-level fast-fail crashes first. Any authenticated user can trigger this by `POST /direction/claims` with a user row id. `http.Server` recovers the panic by dropping the connection — the client gets no 4xx and the request logs a stack trace. The mock repo's `Claim` has the same shape (mock_direction_repo.go:168), so unit tests never exercise the deref.
**Fix:**
```go
// direction.go, Service.Claim — after the Get + status fast-fail:
if wg.WgID == nil {
    return nil, directiondomain.ErrDirectionNotFound // mirrors the repo predicate (wg_id IS NOT NULL)
}
members, err := s.wgRepo.ListMembers(ctx, *wg.WgID)
```

### CR-02: Supersede-on-create writes no `superseded` audit row

**File:** `internal/core/services/direction/direction.go:303-310`
**Issue:** `Service.Create` hands the repo a single audit row (`created`). When `req.SupersedesID` is set, the repo flips the target to `superseded` in the same tx (direction_repository.go:227-238) but nothing writes the target's audit event. This violates three pinned contracts at once: ADR-BE-018 §1 ("Every transition writes an `audit_logs` row in the same tx"), §3 (the `superseded` action is pinned verbatim "so Phase 19 history reads filter deterministically — T-13-06"), and the port doc itself (`ports/direction_repository.go:27-29`: "two audit rows: created + superseded"). The repo test `TestDirectionRepository_Create_Supersede` proves the repo accepts both rows — the service just never sends the second. `directiondomain.AuditActionSuperseded` is dead code in the real path, and a Phase 19 history filter on `superseded` will return nothing for actual supersedes.
**Fix:**
```go
// direction.go, Service.Create — build the second row when superseding:
audits := []*audit.AuditLog{{
    OrgID: orgID, EntityType: directiondomain.AuditEntityDirection,
    EntityID: row.ID, Action: directiondomain.AuditActionCreated,
    ActorID: &actor, CreatedAt: time.Now().UTC(),
}}
if req.SupersedesID != nil {
    targetID := *req.SupersedesID
    audits = append(audits, &audit.AuditLog{
        OrgID: orgID, EntityType: directiondomain.AuditEntityDirection,
        EntityID: targetID, Action: directiondomain.AuditActionSuperseded,
        ActorID: &actor, CreatedAt: time.Now().UTC(),
    })
}
created, err := s.repo.Create(ctx, orgID, row, req.SupersedesID, audits)
```

## Warnings

### WR-01: `directed_to` ref is never validated same-org

**File:** `internal/core/services/direction/direction.go:251-268`
**Issue:** ADR-BE-018 §Security: "`directed_to`/`wg_id`/`activity_id` refs validated same-org at the service (house style)". The Create gate chain validates the activity (same-org) and the WG (same-org + scope), but never checks that `*req.DirectedTo` is an active member of `orgID`. In manager-planned mode a manager whose routing resolution passes (or a role-gated org manager) can create rows directed at users of other orgs — the FK on `directed_to` is to `users(id)` only, so the insert succeeds. The direction row is org-scoped so reads stay contained, but the cross-org reference is exactly what the ADR pins against.
**Fix:** in the `req.DirectedTo != nil` branch, before the mode gate:
```go
m, err := s.orgRepo.GetMembership(ctx, *req.DirectedTo, orgID)
if err != nil {
    return nil, nil, err
}
if m == nil || !m.IsActive {
    return nil, nil, directiondomain.ErrInvalidRequest
}
```

### WR-02: Absurd `est_hours` not rejected at the service → DB numeric overflow → 500

**File:** `internal/core/services/direction/direction.go:206-211`
**Issue:** ADR-BE-018 §6: "Service rejects `est_hours <= 0` (and absurd values) at write (D-13-03 hard per-row validation)". The service validates only positive whole-cent. `est_hours` ≥ 1,000,000 passes the gate chain and then fails inside the repo insert as PG error 22003 (`DECIMAL(8,2)` overflow); `wrapPGError` (postgres.go:16-30) does not map 22003, so the handler's `writeError` default returns a 500 — violating the "never a 500 for client input" boundary contract (T-13-29/32).
**Fix:**
```go
func wholeCent(hours float64) bool {
    return hours > 0 && hours <= 999999.99 && math.Round(hours*100) == hours*100
}
```

### WR-03: Audit vocabulary drift — `unclaimed` action pinned by ADR but never written

**File:** `internal/core/services/direction/direction.go:431-439`; `internal/core/domain/direction/direction.go:168`
**Issue:** ADR-BE-018 §3 pins the direction audit actions as `created / activated / cancelled / superseded / claimed / unclaimed` "pinned verbatim so Phase 19 history reads filter deterministically (T-13-06)". The domain exports `AuditActionUnclaimed = "unclaimed"` but no code path ever writes it — `Service.Unclaim` writes `AuditActionCancelled`, and the port doc (direction_repository.go:66-67) says "One 'cancelled' audit row". Either the ADR's vocabulary is wrong (unclaim should be `cancelled`, and the constant should be deleted) or the unclaim path should write `unclaimed` — as written, a Phase 19 filter on the pinned action set silently merges unclaims into cancels, and the exported constant is a trap for the next implementer.
**Fix:** pick one and pin it: write `directiondomain.AuditActionUnclaimed` from `Service.Unclaim` (and the repo tests), or remove the constant and amend ADR-BE-018 §3.

## Info

### IN-01: Create-response over-capacity warning silently misses non-midnight `planned_date`

**File:** `internal/core/services/direction/direction.go:318-323, 828-830`
**Issue:** `computeWarnings` iterates from `start = *req.PlannedDate` (arbitrary JSON time-of-day) and looks up `coverageByDay[dayKey{emp, d}]`, but repo coverage rows are normalized to UTC midnight (normalizeDay). A client sending `"planned_date":"2026-08-10T10:30:00Z"` gets a warning miss for that day's over-capacity check (away/partial still work — date-only). Harmless for the read-model paths (bounds parsed as `2006-01-02` → midnight), and warnings are advisory — but the create contract (D-13-03) claims the overlay rides every create. Fix: normalize `d` to UTC midnight before the map lookup.

### IN-02: Create commits before warning computation — a warning error yields 500 with a persisted row

**File:** `internal/core/services/direction/direction.go:315-324`
**Issue:** `repo.Create` commits, then `computeWarnings` runs; if it errors (e.g., a transient DB failure on the coverage/absence read), the handler returns 500 while the row exists. A client retrying on the 500 creates a duplicate row. Acceptable for advisory overlay, but consider degrading warning-computation errors to empty warnings on the create path.

### IN-03: Multi-key settings PUT is not atomic across keys

**File:** `internal/core/services/orgsettings/orgsettings.go:140-170`
**Issue:** Each key's `Upsert` runs in its own transaction. The service doc correctly scopes the claim to validation ("an invalid batch never partially commits" — true, validation precedes writes), but a runtime error on key 2 (e.g., transient pool failure) leaves key 1 committed and the handler 500s. If batch atomicity is desired, the port would need a multi-key tx; otherwise document the per-key behavior.

### IN-04: `membershipValid` ignores `IsActive`

**File:** `internal/core/services/direction/direction.go:731-745`
**Issue:** Deactivated members (is_active=false) with an open employment-validity window are treated as valid employees for coverage and remain eligible in unit/WG member lists feeding scope resolution, so they keep surfacing as planned/covered. The ADR's validity semantics are the employment window (D-13-31), but an inactive membership is arguably the stronger signal; worth a decision note for Phase 19.

---

_Reviewed: 2026-08-08T12:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
