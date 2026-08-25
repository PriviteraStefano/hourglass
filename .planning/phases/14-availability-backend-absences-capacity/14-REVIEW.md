---
phase: 14-availability-backend-absences-capacity
reviewed: 2026-08-11T16:16:00Z
depth: standard
files_reviewed: 50
files_reviewed_list:
  - cmd/server/main.go
  - cmd/server/main_test.go
  - internal/adapters/primary/http/availability_handler.go
  - internal/adapters/primary/http/availability_handler_test.go
  - internal/adapters/primary/http/contract_types_handler.go
  - internal/adapters/primary/http/contract_types_handler_test.go
  - internal/adapters/primary/http/direction_handler_test.go
  - internal/adapters/primary/http/handler_test_helper.go
  - internal/adapters/primary/http/organization.go
  - internal/adapters/secondary/postgres/availability_ontology_migrations_test.go
  - internal/adapters/secondary/postgres/availability_read_models.go
  - internal/adapters/secondary/postgres/availability_repository.go
  - internal/adapters/secondary/postgres/availability_repository_test.go
  - internal/adapters/secondary/postgres/contract_type_repository_test.go
  - internal/adapters/secondary/postgres/direction_repository.go
  - internal/adapters/secondary/postgres/direction_repository_test.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/organization_management_repo.go
  - internal/adapters/secondary/postgres/organization_membership_repo.go
  - internal/adapters/secondary/postgres/organization_repo.go
  - internal/adapters/secondary/postgres/staffing_schema_migration_test.go
  - internal/core/domain/auth/membership.go
  - internal/core/domain/availability/availability.go
  - internal/core/domain/availability/availability_test.go
  - internal/core/domain/availability/errors.go
  - internal/core/domain/direction/direction.go
  - internal/core/domain/orgsettings/orgsettings.go
  - internal/core/ports/availability_repository.go
  - internal/core/ports/direction_repository.go
  - internal/core/ports/organization_management_repository.go
  - internal/core/services/availability/availability.go
  - internal/core/services/availability/availability_test.go
  - internal/core/services/availability/contract_types.go
  - internal/core/services/availability/contract_types_test.go
  - internal/core/services/availability/read_models.go
  - internal/core/services/availability/read_models_test.go
  - internal/core/services/direction/direction.go
  - internal/core/services/organization/organization.go
  - internal/core/services/organization/organization_integration_test.go
  - internal/core/services/organization/organization_test.go
  - internal/core/services/testdata/mock_availability_repo.go
  - internal/core/services/testdata/mocks.go
  - internal/models/models.go
  - internal/models/models_test.go
  - migrations/023_availability_status.down.sql
  - migrations/023_availability_status.up.sql
  - migrations/024_work_schedules.down.sql
  - migrations/024_work_schedules.up.sql
  - migrations/025_certificate_attachments.down.sql
  - migrations/025_certificate_attachments.up.sql
findings:
  critical: 1
  warning: 6
  info: 4
  total: 11
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-08-11T16:16:00Z
**Depth:** standard
**Files Reviewed:** 50
**Status:** issues_found

## Summary

Reviewed the Phase 14 availability surface: absence lifecycle (declare/confirm/reject/withdraw, HR medical edit, certificate attachments), work-schedule model (contract-types CRUD + ResolveSchedule fallback chain), org-wide windows read with D-14-24 medical-field filtering, the capacity read-model, the direction confirmed-only closure, migrations 023–025, and the main.go route wiring.

The architecture is disciplined and consistent with the plan (in-tx audit rows, FOR UPDATE matrix re-checks, sentinel mapping, D-G parity with shared services). The overlap guard, the confirmed-only closure, and the privacy carve-out are implemented correctly. However, one **data-loss bug** on the HR medical edit path was found (date-only edit NULLs `certificate_ref`), plus several validation/read-model defects: a float-precision bug in `WindowHoursValid` that rejects legitimate cent values, a workload CTE that ignores the requested period, a client-input 500 path on `hours_per_period`, and a dangling-org-default hole in `DeleteContractType`.

## Critical Issues

### CR-01: Date-only HR medical edit wipes certificate_ref (data loss)

**File:** `internal/core/services/availability/availability.go:331-344` (and `internal/adapters/secondary/postgres/availability_repository.go:421-424`)
**Issue:** `UpdateMedical` builds the update window without copying the current `CertificateRef`:

```go
updated := &availabilitydomain.Window{
    StartsOn: w.StartsOn,
    EndsOn:   w.EndsOn,
}
// CertificateRef is NOT copied from w
if req.CertificateRef != nil {
    updated.CertificateRef = req.CertificateRef
}
```

When the request carries only dates (the documented contract: "absent fields keep the current values"), `updated.CertificateRef` stays nil and the repo's unconditional `UPDATE ... SET starts_on = $1, ends_on = $2, certificate_ref = $3` writes SQL NULL. Migration 023 adds no CHECK on `certificate_ref`, so the NULL is persisted silently. This violates D-14-05 (medical windows require a certificate ref), corrupts the medical record on every date-only correction, and the `{before, after}` audit payload itself records the loss. Neither the service test (both dates + cert always sent) nor the repo test covers the date-only case. Additionally, `{"certificate_ref": ""}` is accepted here while Declare rejects empty refs — the invariant only holds on one path.

**Fix:**
```go
updated := &availabilitydomain.Window{
    StartsOn:       w.StartsOn,
    EndsOn:         w.EndsOn,
    CertificateRef: w.CertificateRef, // absent field keeps the current value
}
if req.CertificateRef != nil {
    if *req.CertificateRef == "" {
        return nil, availabilitydomain.ErrCertificateRequired
    }
    updated.CertificateRef = req.CertificateRef
}
```

## Warnings

### WR-01: WindowHoursValid rejects legitimate cent values (float-precision bug)

**File:** `internal/core/domain/availability/availability.go:207-209`
**Issue:** `math.Round(h*100) == h*100` is false for any value whose binary representation isn't exact after scaling. Verified by execution: `0.29` → `28.99999999999999644729` vs round `29` → invalid; `1.15` → invalid; `2.30` → invalid; while `0.25`, `0.10`, `1.99` pass. A partial-day window of 0.29h or 2.30h is therefore rejected with 400 even though it is a legal DECIMAL(4,2) value. This is exactly the Pitfall-6 class the helper was written to prevent — but inverted.

**Fix:**
```go
return h > 0 && math.Abs(math.Round(h*100)-h*100) < 1e-9 && h <= 99.99
// or equivalently: h == math.Round(h*100)/100
```

### WR-02: Capacity workload CTE ignores the requested period (lifetime sum)

**File:** `internal/adapters/secondary/postgres/availability_read_models.go:135-142`
**Issue:** The `workload` CTE sums all submitted+approved entries on the org's activity subtree with no `entry_date` predicate, while every other component of the same read-model (partial_abs, full_abs, declared) is period-bounded by `$3/$4`. The service then derives `available_hours = period_capacity_hours − workload_hours` — subtracting a lifetime workload from one week's capacity makes the metric meaningless (old entries keep dragging capacity down forever). The repo test seeds entries inside the period only, so it cannot catch this. D-14-19 defines the status pin but not the window; every sibling column is period-scoped, so the workload should be too.

**Fix:** bound the CTE with the same period args:
```sql
workload AS (
    SELECT te.user_id AS employee_id, SUM(te.hours) AS hours
    FROM time_entries te
    WHERE te.is_deleted = false
      AND te.status IN ('submitted','approved')
      AND te.activity_id IN (SELECT id FROM subtree)
      AND te.entry_date >= $3::date AND te.entry_date < $4::date + INTERVAL '1 day'
    GROUP BY te.user_id
)
```

### WR-03: hours_per_period has no ceiling — client input ≥ 1000 → 500

**File:** `internal/core/services/availability/contract_types.go:41-57` (+ migration 024 `DECIMAL(5,2)`)
**Issue:** `validateContractType` checks only `HoursPerPeriod > 0`. The column is `DECIMAL(5,2)` (max 999.99); a request with `hours_per_period: 1000` fails in PG with numeric overflow (SQLSTATE 22003), which `wrapPGError` does not map → 500 internal server error on pure client input. This violates the project's own "no 500 path for client input" rule (T-14g-03/18) — the same pitfall `WindowHoursValid` was created to avoid (Pitfall 6). The capacity derivation also divides this value by 5, so unbounded input reaches the read-model.

**Fix:** in `validateContractType`, after the `<= 0` check:
```go
if req.HoursPerPeriod > 999.99 {
    return availabilitydomain.ErrInvalidRequest
}
```

### WR-04: DeleteContractType can delete the org's default schedule → org-wide capacity 400s

**File:** `internal/adapters/secondary/postgres/availability_repository.go:726-769` (and `internal/core/services/availability/contract_types.go:126-139`)
**Issue:** The FK from `organization_memberships.contract_type_id` blocks deletion only for in-use types. The org default (`org_settings.default_contract_type_id`, a JSONB string with no FK) is never checked. An HR user can delete the type the manager set as default; every subsequent `ResolveSchedule` then hits the Link-2 guard and returns `ErrInvalidValue` → all `/availability/capacity` calls for the org fail with 400 ("corruption is surfaced, never silently defaulted" — the T-14g-19 precedent works as designed, but the delete path creates that corruption). The contract-type repo test covers only the membership FK.

**Fix:** in `DeleteContractType`, before deleting, read the org's `default_contract_type_id` setting and refuse (or clear it first):
```go
// in-tx, before DELETE:
var defaultID *string
err = tx.QueryRow(ctx,
    `SELECT value #>> '{}' FROM org_settings WHERE org_id = $1 AND key = 'default_contract_type_id'`,
    orgID).Scan(&defaultID)
if err == nil && defaultID != nil && *defaultID == id.String() {
    return availabilitydomain.ErrContractTypeInUse // or clear the key in the same tx
}
```
(Alternatively surface a dedicated sentinel → 409.)

### WR-05: Capacity scope=unit/wg accepts cross-org IDs — silent empty instead of explicit rejection

**File:** `internal/core/services/availability/read_models.go:352-392`
**Issue:** `unitRepo.ListMembers(ctx, unitID.String())` and `wgRepo.ListMembers(ctx, wgID)` are not org-scoped (verified: `unit_member_repository.go:25-37` filters by `unit_id` only; `wg_member_repository.go:22-24` by `wg_id` only). A caller from org A can pass a unit/WG id belonging to org B. Today the resulting foreign employee list is silently discarded by the D-14-22 validity split (`GetMembership(empID, orgA)` → nil → excluded), so no cross-org data leaks — but the response is a misleading empty capacity with 200, and the only thing standing between this and a cross-org data exposure is the validity split's behavior. This is inconsistent with the activity scope, which is org-guarded in SQL (`ActivityWorkloadEmployees` seeds the CTE with `org_id = $1`).

**Fix:** validate the unit/WG belongs to the org before resolving members (e.g., resolve the unit's org via `GetByID`/an org-scoped read and 400/404 when it doesn't match `orgID`), or add `org_id = $1` to the member queries.

### WR-06: Certificate upload content-type is client-controlled and served verbatim (stored-XSS vector)

**File:** `internal/adapters/primary/http/availability_handler.go:255-264, 310-312`
**Issue:** The upload gate validates only the file *extension*; `content_type` is taken from the client's part header and stored. `GetCertificate` serves the bytes with that stored header, no `Content-Disposition`, no `X-Content-Type-Options: nosniff`. A certificate named `cert.pdf` uploaded with `Content-Type: text/html` containing HTML/JS is rendered by the browser at the same origin for any privileged viewer (hr or the owner's unit manager) — a stored-XSS path. The privilege needed is hr, but defense-in-depth requires not trusting a client-supplied content type for a stored document.

**Fix:** validate `contentType` against the PDF/JPEG/PNG allowlist at upload (400 otherwise), and serve the document with:
```go
w.Header().Set("Content-Type", att.ContentType)
w.Header().Set("Content-Disposition", "attachment; filename=\"certificate\"")
w.Header().Set("X-Content-Type-Options", "nosniff")
```

## Info

### IN-01: ResolveSchedule returns the package-level fallback map by reference

**File:** `internal/core/services/availability/contract_types.go:277-281`
**Issue:** The Link-3 branch returns `DayHours: fallbackDayHours` — a direct alias of the package-level map. Current callers (`expandScheduleHours`, `mergeDayHours`) only read/copy it, so nothing mutates shared state today, but the exported `ScheduleResolution` hands the shared map to any future caller. Copy before returning:
```go
days := make(map[string]float64, len(fallbackDayHours))
for k, v := range fallbackDayHours {
    days[k] = v
}
return &availabilitydomain.ScheduleResolution{DayHours: days, SourceLevel: availabilitydomain.SourceFallback}, nil
```

### IN-02: parsePeriod accepts start > end → silent empty capacity

**File:** `internal/adapters/primary/http/direction_handler.go:283-298` (shared by the capacity handler)
**Issue:** `period_start > period_end` passes parsing; `generate_series` then yields zero days and the response is an empty 200 instead of a 400. Pre-existing behavior on the direction endpoint, but the new capacity route inherits it. A `start.After(end) → error` check would surface the client mistake.

### IN-03: UpdateMedical silently ignores a partially-specified date update

**File:** `internal/core/services/availability/availability.go:335-341`
**Issue:** When only one of `starts_on`/`ends_on` is present, the `&&` condition skips both and the request succeeds without applying anything (or applying only the certificate_ref) — a silent no-op on dates. A 400 ("both dates or neither") would be clearer than silently ignoring one field.

### IN-04: getWindow resolves windows via an unbounded org-wide read

**File:** `internal/core/services/availability/availability.go:166-177`
**Issue:** Every lifecycle mutator and every authority resolution loads the full org-wide window list (`Windows` with a nil filter) to find one row. This is O(org windows) per call and would break silently if the read ever gained a mandatory limit. The documented "no existence oracle" is a deliberate trade-off, but a single-row org-scoped read on the port would remove the coupling to the read-model's pagination.

---

_Reviewed: 2026-08-11T16:16:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
