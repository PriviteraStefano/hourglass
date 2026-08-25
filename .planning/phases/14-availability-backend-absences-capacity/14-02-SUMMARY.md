---
phase: 14-availability-backend-absences-capacity
plan: 02
subsystem: backend
tags: [go, domain, ports, adr, availability, roles]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane
    provides: audit.AuditLog in-tx write contract (BE-012/BE-016), ADR-BE-018 vault template, mock_direction_repo.go shape
  - phase: 14-availability-backend-absences-capacity (plan 01)
    provides: migration 023 status vocabulary ('declared','confirmed','rejected','withdrawn') the domain constants must match exactly
provides:
  - ADR-P-008 revision recording the D-1a simplification (unit-manager-only confirmation) and the D-5 document-storage boundary change (DB-backed medical certificates, GDPR special-category flag)
  - ADR-BE-019 availability schema encoding ADR (vocabularies, transition matrix, 2VL rejection_reason CHECK, audit vocabulary, overlap-guard encoding, contract_types + override model, attachment table, role gates, assumption pins)
  - internal/core/domain/availability package: Window entity, kind/status constants, transitionMatrix + CanTransition/IsTerminalStatus, audit constants, ContractType/Attachment/EmployeeSchedule/ScheduleResolution, windowHoursValid + DayHours validators
  - ports.AvailabilityRepository (mutators with in-tx audit, read-models, contract-type CRUD) + testdata.MockAvailabilityRepo with compile-time assertion
  - models.RoleHR = "hr" — first backend consumer of the hr role
affects: [14-03 declare path, 14-05 lifecycle mutators, 14-06 contract types, 14-07 read models, 14-08 HTTP surface]

actuals:
  tokens: 24857   # chars/4 over realized diff (~99430 chars, 8 files)
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Domain as single source of truth for status/audit vocabularies — DB CHECK (023) and callers read from it (anti-drift)"
    - "Transition matrix const + CanTransition/IsTerminalStatus predicate methods (direction domain precedent)"
    - "Port doc contract: every mutator takes its *audit.AuditLog and writes it in the same transaction"
    - "MockAvailabilityRepo with mutex + maps + per-method Fn overrides + Audits capture + var _ compile assertion (mock_direction_repo shape)"

key-files:
  created:
    - internal/core/domain/availability/availability.go
    - internal/core/domain/availability/errors.go
    - internal/core/ports/availability_repository.go
    - internal/core/services/testdata/mock_availability_repo.go
    - hourglass-vault/decisions/backend/ADR-BE-019 — Availability Schema Encoding.md
  modified:
    - hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md
    - internal/models/models.go
    - internal/models/models_test.go

key-decisions:
  - "Medical windows never enter the transition matrix — auto-confirmed at declare (D-14-02); documented on the domain package and the ADR"
  - "Overlap guard encoded as in-tx SELECT ... FOR UPDATE on ACTIVE declared+confirmed windows (D-14-15) — no btree_gist, no EXCLUDE (first-extension-free house style)"
  - "RoleHR as compile-time constant + IsValid() case (research Pitfall 8) — JWT claims carry the role verbatim, no auth change"
  - "Task 3 deliberately not tdd-tagged (checker INFO accepted): port/mock portion has no RED-able test; tests ship in-task"

patterns-established:
  - "Per-phase BE encoding ADR mirroring ADR-BE-018 structure with 14-PATTERNS 'No Analog Found' pins"
  - "Domain constants ↔ DB CHECK vocabulary parity (023) — pinned in ADR-BE-019"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "ADR-P-008 revised in place (D-1a unit-manager-only confirmation, D-5 DB-backed medical certificate boundary with GDPR flag) and ADR-BE-019 drafted pinning the window status vocab + matrix, 2VL rejection_reason CHECK, audit vocabulary, overlap-guard encoding, contract_types + override model, and assumption pins"
    requirement: AVAIL-01
    verification:
      - kind: other
        ref: "test -f ADR-P-008 && test -f ADR-BE-019 && grep -c availability_windows_reject_reason_check ADR-BE-019 (exit 0, count 2)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Availability domain package — Window entity, closed kind/status vocabulary matching the 023 CHECK, transition matrix (declared → confirmed|rejected|withdrawn, terminal rejected/withdrawn), six-action audit vocabulary, schedule/attachment types, hours + day-hours validators, 11 sentinels with JSONNames; no 'cancelled'/'superseded' literal"
    requirement: AVAIL-01
    verification:
      - kind: unit
        ref: "internal/core/domain/availability/availability_test.go (matrix, vocabulary, hours, day-hours probes)"
        status: pass
    human_judgment: false
  - id: D3
    description: "ports.AvailabilityRepository full mutator (audit-in-tx) + read-model + contract-type CRUD surface; testdata.MockAvailabilityRepo implements it with the var _ compile-time assertion"
    requirement: AVAIL-02
    verification:
      - kind: other
        ref: "go vet ./internal/core/ports/ ./internal/core/services/testdata/ ./internal/models/ (exit 0)"
        status: pass
    human_judgment: false
  - id: D4
    description: "models.RoleHR = 'hr' added to the constants block, accepted by IsValid(), covered in models_test.go validCases"
    verification:
      - kind: unit
        ref: "internal/models/models_test.go#validCases (RoleHR case)"
        status: pass
    human_judgment: false

# Metrics
duration: interrupted-session
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 02: Compile-Time Contract Layer Summary

**The availability domain package (Window entity, status/audit vocabularies, transition matrix, schedule types, sentinels), ports.AvailabilityRepository + MockAvailabilityRepo compile-asserted mock, models.RoleHR — plus the ADR-P-008 revision and ADR-BE-019 encoding ADR that pin the D-14-01 routing simplification and the D-14-06/07 DB-backed medical-certificate boundary — all landed; every later availability plan compiles against these contracts.**

> **Close-out note:** This plan's four commits were landed in a prior session that ended before SUMMARY.md was written (safe-resume gate). Work was verified post-hoc: `go build ./...` exit 0, domain + models tests green, `go vet` clean, ADR grep checks pass — then this summary was authored from the committed state.

## Performance

- **Duration:** unknown (interrupted session — close-out authored 2026-08-11)
- **Completed:** 2026-08-11
- **Tasks:** 3 (Task 2 full TDD cycle: RED + GREEN)
- **Files modified:** 8 (5 created, 3 modified)

## Accomplishments

- **ADR-P-008 revision (Task 1):** D-1a — two-line confirmation collapsed to unit-manager-only (WG line dropped, D-14-01); D-4 HR-curator-never-approver kept verbatim (D-14-03); self-confirm rule (D-14-04) and certificate_ref-required-at-declare (D-14-05) added; D-5 — medical certificate documents ARE stored in Hourglass via the DB-backed attachment table (D-14-06/07) with an explicit GDPR special-category flag and the one-way reversibility note. Status/date header updated.
- **ADR-BE-019 (Task 1):** Availability schema encoding ADR mirroring ADR-BE-018 — window status vocabulary + transition matrix (declared → confirmed|rejected|withdrawn; rejected/withdrawn terminal; medical auto-confirms at declare), rejection_reason 2VL CHECK (`availability_windows_reject_reason_check`), audit vocabulary (entity_type `availability_window`; actions declared/confirmed/rejected/withdrawn/edited/certificate_attached; reject/withdraw payloads {reason}; HR edits {before, after}), overlap guard (in-tx SELECT ... FOR UPDATE, active-only statuses, inclusive date-range intersection — no btree_gist/EXCLUDE), contract_types + membership override model (cadence week|month, hours_per_period > 0, day_hours JSONB code-side validated, override merges over the type matrix), org default via org_settings key `default_contract_type_id`, attachment table + BYTEA + 5 MB cap + PDF/JPEG/PNG allowlist, role gates (hr first backend consumer), and the assumption pins (workload literals 'submitted'/'approved', capacity response shape, D-13-29 closure confirmed-only).
- **Availability domain package (Task 2):** `Window` entity (ID, OrgID, UserID, Kind, StartsOn, EndsOn, Hours *float64, CertificateRef *string, Note *string, Status, RejectionReason *string, CreatedBy, CreatedAt, UpdatedAt) with json tags (NULL hours = full day); kind constants holiday/permit/medical/unavailable mirroring the 012 CHECK; status constants StatusDeclared/StatusConfirmed/StatusRejected/StatusWithdrawn with `transitionMatrix` + CanTransition + IsTerminalStatus; doc comment stating medical skips the matrix (auto-confirmed at declare, D-14-02); audit constants AuditEntityWindow + the exact six actions (D-14-12); ContractType/Attachment/EmployeeSchedule/ScheduleResolution types (resolution source levels override|type|org_default|fallback, D-14-18); windowHoursValid (whole-cent via math.Round, 99.99 DECIMAL(4,2) ceiling, research Pitfall 6) + DayHours validation (keys "1".."7", 0 < h <= 24, non-empty for week cadence); 11 sentinels + JSONNames (ErrWindowNotFound, ErrInvalidRequest, ErrForbidden, ErrInvalidTransition, ErrOverlap, ErrRejectReasonRequired, ErrInvalidHours, ErrInvalidKind, ErrCertificateRequired, ErrNotMedical, ErrContractTypeNotFound, ErrContractTypeInUse).
- **Port + mock + RoleHR (Task 3):** `ports.AvailabilityRepository` — doc header pinning the in-tx audit write shape (BE-012/BE-016); Declare (with overlap-guard doc comment: lock ACTIVE declared+confirmed overlapping windows FOR UPDATE → ErrOverlap; medical → status confirmed immediately with second audit row), Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate (each takes its audit row), read-models Windows/Capacity/Attachment/AttachmentsByWindowIDs/ActivityWorkloadEmployees, contract-type CRUD List/Create/Update/Delete. `MockAvailabilityRepo` mirrors mock_direction_repo (maps, per-method Fn overrides, Audits capture, `var _ ports.AvailabilityRepository = (*MockAvailabilityRepo)(nil)`). `models.RoleHR = "hr"` + IsValid() case + models_test validCases (D-14-26).

## Task Commits

Each task committed atomically:

1. **Task 1: ADR-P-008 revision + ADR-BE-019** - `09e9333` (docs)
2. **Task 2: Domain package (TDD)** - `713587d` (test: failing domain tests), `7c250f1` (feat: domain implementation)
3. **Task 3: Port + mock + RoleHR** - `87cfec2` (feat)

## Files Created/Modified

- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` - D-1a simplification + D-5 boundary change + D-14-04/05 rules (modified)
- `hourglass-vault/decisions/backend/ADR-BE-019 — Availability Schema Encoding.md` - full encoding pins (created)
- `internal/core/domain/availability/availability.go` - entity, vocabularies, matrix, audit constants, schedule types, validators (created)
- `internal/core/domain/availability/errors.go` - 11 sentinels + JSONNames (created)
- `internal/core/ports/availability_repository.go` - mutator/read-model/CRUD surface (created)
- `internal/core/services/testdata/mock_availability_repo.go` - MockAvailabilityRepo (created)
- `internal/models/models.go` - RoleHR constant + IsValid case (modified)
- `internal/models/models_test.go` - RoleHR validCase (modified)

## Decisions Made

- **Medical auto-confirm at declare** — medical windows skip the transition matrix entirely (D-14-02); the repo writes the confirmed audit row in the same declare transaction.
- **Overlap guard in-tx, not btree_gist** — SELECT ... FOR UPDATE on active overlapping windows (D-14-15), consistent with the first-extension-free house style; no EXCLUDE constraint.
- **RoleHR compile-time constant** — no "hr" literal in service code paths; JWT claims pass the role verbatim (research Pitfall 8).
- **Task 3 not tdd-tagged** — checker INFO accepted: the port/mock portion has no RED-able test; tests ship in-task.

## Deviations from Plan

None - plan executed exactly as written (per committed state).

## Issues Encountered

- **Interrupted session before SUMMARY creation** — all four commits landed, but the session ended before the summary/state writes. Recovered via the safe-resume gate: verified build + tests + vet, then closed out manually.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **14-03 (declare path):** repo mutator implementation compiles against the port contract; the in-tx audit shape and overlap-guard encoding are pinned in ADR-BE-019; MockAvailabilityRepo is the test fixture for the service layer.
- **14-05 (lifecycle):** UpdateMedical/AttachCertificate/Withdraw/Confirm signatures ready; the six-action audit vocabulary is the service-side constant source.
- **14-06 (contract types):** ContractType entity + CRUD port + ScheduleResolution levels ready for the fallback chain.
- **14-07/14-08 (read models + HTTP):** Windows/Capacity/Attachment read-model signatures and RoleHR gate ready for handlers.

---
*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*
## Self-Check: PASSED
