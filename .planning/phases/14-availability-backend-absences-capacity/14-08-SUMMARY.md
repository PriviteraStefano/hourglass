---
phase: 14-availability-backend-absences-capacity
plan: 08
subsystem: backend
tags: [go, http, availability, certificate-upload, permission-matrix, capacity, read-models, D-14-01..26]

# Dependency graph
requires:
  - phase: 14-availability-backend-absences-capacity (plan 03)
    provides: the declare route + AvailabilityHandler skeleton + writeError sentinel map
  - phase: 14-availability-backend-absences-capacity (plan 05)
    provides: the lifecycle service mutators (Confirm/Reject/Withdraw/UpdateMedical/AttachCertificate) with the unit-manager authority resolution
  - phase: 14-availability-backend-absences-capacity (plan 07)
    provides: ListWindows (D-14-24 privacy carve-out), Capacity (D-14-20..23), Attachment (privileged single-doc) service methods + response types
  - phase: 13-direction-backend-the-plan-plane
    provides: the thin-adapter handler skeleton (claims, writeError, parsePeriod, parseOptionalQueryUUID) and the fixture battery shape (registerAndLogin, seed chains, cookie discipline)
provides:
  - The complete /availability HTTP surface: 9 window/capacity/certificate routes + 4 contract-type routes, all middleware.Auth-gated, fixture-mirrored
  - The full permission battery over HTTP: withdraw owner-only, confirm/reject unit-manager-gated (hr-not-manager 403, self-confirm 200), HR medical edit + certificate attach, upload allowlist + 5MB cap, download gated to hr + unit manager
  - The D-14-24 privacy e2e: certificate_ref/documents absent from the JSON for non-privileged members (server-side drop proven at the boundary)
  - The capacity contract over HTTP: scope=activity|wg|unit|org + required period bounds, per-employee rows + totals + declared advisory + schedule resolution level
affects: [Phase 16 UI (AVAIL-03/04/05 — capacity grid + absences), Phase 19 (scheduler warnings + history filters), direction_handler test-battery consumers]

actuals:
  tokens: 11909    # chars/4 over the realized diff (47636 chars, 4 files, 5 commits) — estimate was 24000
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "The expense.go MIME/size gate shape for certificate uploads: MaxBytesReader 5 MB + ParseMultipartForm + FormFile + extension allowlist — but the bytes go to the service for BYTEA storage, NEVER the file-storage block (D-14-07 locked)"
    - "The D-14-24 privacy e2e asserts JSON field ABSENCE (certificate_ref/documents) server-side — the medical kind label stays, the ref never crosses for non-privileged members"
    - "ServeMux most-specific-wins discipline for the literal /availability/windows GET vs POST and the {id} wildcards — no duplicate registrations (Go 1.22 panics)"
    - "Cookie discipline in the shared-jar battery: loginUser re-issues the jar, switchToOrg re-establishes the fixture org after every fresh login"

key-files:
  created: []
  modified:
    - internal/adapters/primary/http/availability_handler.go
    - internal/adapters/primary/http/availability_handler_test.go
    - internal/adapters/primary/http/handler_test_helper.go
    - cmd/server/main.go

key-decisions:
  - "The six lifecycle + three read handler methods are thin adapters exactly per the direction_handler skeleton: claims → uuid.Parse PathValue (400 on malformed) → service call → sentinel map; writeError needed NO new sentinels (the 14-03 map already covered ErrNotMedical/ErrRejectReasonRequired/ErrCertificateRequired → 400 and ErrOverlap/ErrInvalidTransition → 409)"
  - "Certificate upload follows the expense.go gate shape but stores to BYTEA via the service — the file-storage block (os.MkdirAll/io.Copy) is deliberately NOT mirrored; the part header content-type + size are captured and stored (never inferred)"
  - "GET .../certificate serves the stored bytes with the stored content-type; a window without a document maps to 404 (not 204/empty) so the client distinguishes 'no doc' from 'forbidden'"
  - "Flagged assumption (AVAIL-02, specless probe) surfaced, never auto-resolved: the battery pins hr-not-manager confirm → 403 (D-14-03 reading), manager confirm + self-confirm → 200 (D-14-01/04) — exactly the D-14-03 lock"
  - "ListWindows parses period_start/period_end as 2006-01-02 (the parsePeriod shape) into the WindowsFilter Start/End — the repo's overlap semantics (ends_on >= start AND starts_on <= end)"

patterns-established:
  - "Thin-adapter route methods: claims extraction, PathValue uuid.Parse, JSON/multipart decode, service call, envelope — every malformed input 400, never 500 (T-14g-25)"
  - "Upload gates at the boundary, bytes storage in the DB: handler enforces MIME/size, the service enforces role/kind, the repo stores — three trust boundaries kept separate"
  - "Battery cast pattern: one unit chain (unit_memberships role='manager') + role set (manager/hr/employee) seeded once, every subtest re-establishes its actor via loginUser + switchToOrg"

requirements-completed: [AVAIL-01, AVAIL-02]

coverage:
  - id: D1
    description: "Lifecycle routes over HTTP (withdraw owner-only 200/403/409, confirm manager-gated 200/403/403/200/409 incl. hr-not-manager 403 D-14-03 and self-confirm 200 D-14-04, reject reason-required 400/200 with persisted rejection_reason, terminal-state 409s) with the audit rows asserted"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/withdraw"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/confirm"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/reject"
        status: pass
    human_judgment: false
  - id: D2
    description: "HR medical edit + certificate attach gates (200/400/403 matrix), the D-14-07 upload contract over multipart (PNG 200 + audit, non-hr 403, wrong extension 400, >5MB 400), and the download gate (hr + unit manager 200 with stored content-type + bytes, other member 403)"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/hr_medical_edit"
        status: pass
    human_judgment: false
  - id: D3
    description: "Windows read over HTTP: org-wide list with kind/status/period filters + limit/offset pagination + deterministic starts_on ordering; the D-14-24 privacy e2e (hr + resolved unit manager see certificate_ref/documents, plain employee's JSON lacks both fields, kind label stays)"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/GET_/availability/windows"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/D-14-24_privacy_e2e"
        status: pass
    human_judgment: false
  - id: D4
    description: "Capacity endpoint over HTTP: scope=activity (workload universe + 4h submitted Σ + fallback resolution level + declared advisory), scope=wg/unit/org row universes, totals; unknown scope / missing scope / malformed period → 400"
    requirement: AVAIL-02
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/GET_/availability/capacity"
        status: pass
    human_judgment: false
  - id: D5
    description: "Certificate download matrix incl. the 404 no-attachment case; cross-org window ids → 404 (no existence oracle); unauthenticated windows read → 401"
    requirement: AVAIL-01
    verification:
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/GET_certificate"
        status: pass
      - kind: integration
        ref: "internal/adapters/primary/http/availability_handler_test.go#TestAvailabilityHandler/cross-org_window_ids_404"
        status: pass
    human_judgment: false
  - id: D6
    description: "Production wiring: main.go registers the full 13-route availability set (9 window/capacity/certificate + 4 contract-type) behind middleware.Auth; the fixture registers the identical set (grep-verified); full suite green"
    verification:
      - kind: other
        ref: "make test (exit 0, 27 packages) + grep route-parity main.go vs handler_test_helper.go"
        status: pass
    human_judgment: false

# Metrics
duration: 11min
completed: 2026-08-11
status: complete
---

# Phase 14 Plan 08: Full /availability HTTP Surface + Integration Battery Summary

**The complete absence lifecycle is consumable over HTTP: all nine window/capacity/certificate routes live and auth-gated in cmd/server/main.go (fixture-mirrored), proven end-to-end by the 20-subtest integration battery — the D-14-03/04/26 permission matrix (hr-not-manager confirm 403, self-confirm 200), the D-14-24 privacy carve-out (certificate fields absent from the JSON server-side), the D-14-07 upload contract (5 MB cap + PDF/JPEG/PNG allowlist, bytes to BYTEA never disk), and the capacity scope/period contract with the declared advisory.**

## Performance

- **Duration:** ~11 min (first task commit 17:47:50Z → final commit 17:58:29Z)
- **Started:** 2026-08-11T17:47:50Z
- **Completed:** 2026-08-11T17:58:29Z
- **Tasks:** 3 (Tasks 1+2 full RED→GREEN TDD cycles; Task 3 wiring + full-suite gate)
- **Files modified:** 4 (all planned; main_test.go verified no-op per plan)

## Accomplishments

- **Lifecycle routes (Task 1):** `Withdraw`/`Confirm`/`Reject`/`UpdateMedical`/`AttachCertificate`/`GetCertificate` as thin adapters — claims → `uuid.Parse` PathValue (400 on malformed) → service call → the existing writeError sentinel map (it already covered every sentinel this plan needed — no new cases). The battery proves the D-14-03/04/26 gate matrix over HTTP: withdraw owner-only (the resolved unit manager is still 403), confirm by the resolved unit manager 200 + audit, plain employee 403, **hr-not-manager 403** (D-14-03 — the flagged assumption pinned, never auto-resolved), **self-confirm 200** (D-14-04 — the employee is the resolved manager of their own unit), reject-without-reason 400 + reason persisted in the response, terminal-state ops 409, cross-org ids 404.
- **Upload contract (Task 1):** `AttachCertificate` mirrors the expense.go gate shape — `http.MaxBytesReader(w, r.Body, 5<<20)`, `ParseMultipartForm`, `FormFile("file")`, the `.pdf/.jpg/.jpeg/.png` allowlist (400 on violation) — but the bytes + part-header content-type go to the service for **BYTEA storage in PostgreSQL, never to disk** (the expense.go file-storage block is deliberately not mirrored; no `uploads/` directory appeared during the battery). Oversized (>5 MB) uploads 400 via the MaxBytesReader cap.
- **Download gate (Task 1/2):** `GetCertificate` serves the stored bytes with the stored content-type to hr + the owner's resolved unit manager; other members 403 (T-14g-10), windows without a document 404.
- **Read routes (Task 2):** `ListWindows` (user_id/kind/status/period filters + limit/offset with 400s on malformed), `Capacity` (scope + scope_id + the direction_handler `parsePeriod` bounds verbatim), plus the read battery: the D-14-24 privacy e2e asserts **JSON field absence** — a plain employee's response lacks `certificate_ref` and `documents` while the medical kind label stays; the capacity contract per scope (activity workload universe with the 4h submitted Σ, wg/unit/org row universes, totals, the declared advisory never subtracted, the schedule resolution level documented); unknown scope / missing scope / malformed period → 400, never 500.
- **Production wiring (Task 3):** main.go registers the remaining 8 routes (all `middleware.Auth`-wrapped, T-14g-25) beside the declare + contract-type block — grep-verified byte-identical route parity with the fixture (13 availability routes each, no duplicates → no ServeMux panic). `cmd/server/main_test.go` needed no changes (the 14-06 constructor call sites already match). Full suite green: `go build ./... && go vet ./...` clean, `make test` exit 0 across 27 packages (direction + availability + cmd/server + everything else).

## Task Commits

Each task committed atomically (TDD: test commit then feat commit):

1. **Task 1 RED: lifecycle + upload battery** - `2e06058` (test)
2. **Task 1 GREEN: lifecycle + certificate routes** - `7a4f532` (feat)
3. **Task 2 RED: read battery** - `6f19201` (test)
4. **Task 2 GREEN: read routes — windows list, capacity, certificate download** - `7b3b0eb` (feat)
5. **Task 3: full /availability surface wired in main.go** - `0714fa7` (feat)

**Plan metadata:** committed after this file

## Files Created/Modified

- `internal/adapters/primary/http/availability_handler.go` - +Withdraw/Confirm/Reject/UpdateMedical/AttachCertificate/GetCertificate/ListWindows/Capacity (323 lines); writeError unchanged (already complete)
- `internal/adapters/primary/http/availability_handler_test.go` - full battery: lifecycle matrix, upload gates, D-14-24 privacy e2e, capacity per scope, download matrix, cross-org/unauth (557 lines + helpers)
- `internal/adapters/primary/http/handler_test_helper.go` - fixture registers the 9 window/capacity/certificate routes (declare + contract-types already present)
- `cmd/server/main.go` - the remaining 8 routes wired with middleware.Auth

## Decisions Made

- **writeError needed no extension** — the 14-03 sentinel map already mapped every error this plan's handlers surface (ErrNotMedical/ErrRejectReasonRequired/ErrCertificateRequired → 400; ErrOverlap/ErrInvalidTransition → 409); the plan's "writeError gains remaining sentinel cases" was already satisfied.
- **Bytes + part-header content-type to the service** — the handler captures `header.Header.Get("Content-Type")` and the read bytes; the service stores them on the `AttachCertificateRequest` (content_type/size_bytes/storage). No disk path exists in the handler.
- **404 for document-less windows** — `Attachment` returning nil maps to 404 (distinct from the 403 gate) so Phase 16 UI can show "no certificate yet".
- **Flagged assumption pinned, not resolved** — the battery encodes the D-14-03 reading (hr-not-manager confirm → 403) plus D-14-01/04 (manager + self confirm → 200). Unchanged from the plan's surfacing.
- **Capacity handler drops actorID** — the service signature is `Capacity(ctx, orgID, role, scope, scopeID, ...)` (no actor); the handler follows the pinned service surface rather than inventing an actor param.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `RejectRequest` name collision with time_entry.go**
- **Found during:** Task 1 GREEN (first compile)
- **Issue:** The handler's reject body type `RejectRequest` collides with the pre-existing `time_entry.go` type of the same name in the same package — build failure.
- **Fix:** Renamed the availability body type to `WindowRejectRequest` (scoped name, same shape).
- **Files modified:** availability_handler.go
- **Verification:** `go build ./...` clean.
- **Committed in:** 7a4f532

**2. [Rule 1 - Bug] Unit-scope battery assertion undercounted the cast**
- **Found during:** Task 2 GREEN (capacity scope=unit returned 3 rows, expected 2)
- **Issue:** The cast seeds privee as a unit member of the same unit (required for the D-14-24 manager-resolution e2e), so the unit universe is manager + emp + privee — the assertion said 2.
- **Fix:** Assertion corrected to 3 with the cast explanation.
- **Files modified:** availability_handler_test.go
- **Verification:** capacity battery green.
- **Committed in:** 7b3b0eb

**3. [Rule 3 - Blocking] Capacity service call signature mismatch**
- **Found during:** Task 2 GREEN (first compile)
- **Issue:** The pinned service `Capacity` takes `(ctx, orgID, role, scope, scopeID, start, end)` — no actorID; the first handler draft passed userID.
- **Fix:** Handler call aligned to the pinned signature (actor claims unused in this route).
- **Files modified:** availability_handler.go
- **Verification:** `go build ./...` clean; battery green.
- **Committed in:** 7b3b0eb

---

**Total deviations:** 3 auto-fixed (2 Rule 3 blocking-compile, 1 Rule 1 test correctness)
**Impact on plan:** All three were compile/test-correctness fixes within the plan's own files; no scope creep, no behavioral change beyond the plan.

## Issues Encountered

- **Test helper discipline:** the first Task-1 test draft called `h.loginUser` (a fixture method) — the established cookie discipline is `f.loginUser` + `h.switchToOrg`; fixed during RED authoring (not a deviation, part of the battery shape learning).
- **Testcontainers cost:** each battery run spins the postgres container (~14s per `TestAvailabilityHandler` run); the full suite stays ~42s of container time. No flakiness observed across the three battery runs.
- **RED nuance:** the unauthenticated-windows assertion read 405 at RED (only `POST /availability/windows` registered → ServeMux method-mismatch) and flipped to the correct 401 only after Task 2 registered the GET route — the RED failure mode was expected and documented in the test comments.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **The phase is consumable:** Phase 16 (AVAIL-03/04/05 UI) and Phase 19 (scheduler warnings) call the full endpoint set — windows CRUD + lifecycle, capacity grid, certificate download; contract-type CRUD rides the same handler.
- **Flagged assumption carried forward:** the "Manager/HR can confirm" specless probe stays surfaced (D-14-03 reading pinned by the battery) — any Phase 16/19 product decision that contradicts it must go through the decision log.
- Full suite green (`make test` exit 0, 27 packages); direction + coverage + availability suites all pass.
- No blockers. Phase 14 (availability-backend-absences-capacity) is complete: 8/8 plans with summaries.

---

*Phase: 14-availability-backend-absences-capacity*
*Completed: 2026-08-11*

## Self-Check: PASSED

- SUMMARY.md exists on disk; all 5 plan commits verified in git history (2e06058, 7a4f532, 6f19201, 7b3b0eb, 0714fa7).
- Plan-level verification re-run after the final task: `go test ./internal/adapters/primary/http/ -run TestAvailabilityHandler -count=1` ok (full battery), `go build ./... && go vet ./...` clean, `make test` exit 0 (27 packages).
- Route parity grep-verified: main.go and the fixture register the identical 13-route availability set.
- TDD gate compliance: Task 1 RED (2e06058) → GREEN (7a4f532); Task 2 RED (6f19201) → GREEN (7b3b0eb) — each RED immediately preceding its GREEN, each RED failing for the right reason (routes absent → 404/405, never a spurious pass).
