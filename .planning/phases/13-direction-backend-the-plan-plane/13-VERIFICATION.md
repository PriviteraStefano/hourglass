---
phase: 13-direction-backend-the-plan-plane
verified: 2026-08-08T15:07:05Z
status: gaps_found
score: 7/8 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Lifecycle draft → active → superseded/cancelled works; done/lapsed/claimed are derived, never stored; supersedes_id chains replanning with audit-first via BE-012 (DIR-02)"
    status: failed
    reason: "CR-02: supersede-on-create writes no 'superseded' audit row. Service.Create passes ONE audit row ('created') to repo.Create even when supersedes_id is set (direction.go:303-310). The repo flips the target to superseded in the same tx (direction_repository.go:227-238) but no audit event records the transition — violating ADR-BE-018 §1/§3 (superseded action pinned 'so Phase 19 history reads filter deterministically'), the port doc contract ('two audit rows: created + superseded', direction_repository.go:27-29), and DIR-02's audit-first via BE-012. A Phase 19 history filter on 'superseded' returns nothing for real supersedes. The repo accepts both rows (TestDirectionRepository_Create_Supersede proves it); the service just never sends the second."
    artifacts:
      - path: "internal/core/services/direction/direction.go"
        issue: "Service.Create passes a single 'created' audit row; the 'superseded' audit for the flipped target is never built"
    missing:
      - "Service.Create: when req.SupersedesID != nil, append a second audit row {EntityType: direction, EntityID: targetID, Action: AuditActionSuperseded} to the audits slice"
  - truth: "Claim endpoint behaves correctly on invalid input — a user-targeted row id must not crash the server (DIR-03 claim model robustness)"
    status: failed
    reason: "CR-01: Service.Claim dereferences nil WgID on user-targeted rows (direction.go:462). After the status fast-fail, s.wgRepo.ListMembers(ctx, *wg.WgID) panics when wg.WgID == nil (user-targeted row, incl. claim rows and activated personal rows). Any authenticated user can trigger the panic via POST /direction/claims with a user row id; http.Server drops the connection — no 4xx, stack trace logged. The repo's lock query guards AND wg_id IS NOT NULL (returns ErrDirectionNotFound) but the service crashes first. MockDirectionRepo has the same shape, so unit tests never exercise the deref. Fix: after the Get + status fast-fail, return ErrDirectionNotFound when wg.WgID == nil (mirrors the repo predicate)."
    artifacts:
      - path: "internal/core/services/direction/direction.go"
        issue: "Service.Claim at line ~462 dereferences *wg.WgID without a nil guard"
      - path: "internal/core/services/testdata/mock_direction_repo.go"
        issue: "Mock Claim mirrors the deref shape — no test exercises the user-row path"
    missing:
      - "nil guard on wg.WgID in Service.Claim returning ErrDirectionNotFound"
      - "regression test: POST /direction/claims with a user-targeted row id → 404, no panic"
  - truth: "Service rejects est_hours <= 0 and absurd values at write; client input never produces a 500 (D-13-03 hard per-row validation, T-13-32 boundary contract)"
    status: failed
    reason: "WR-02: wholeCent (direction.go:116-118) has no upper bound — hours > 0 && round(h*100) == h*100 only. est_hours >= 1,000,000 passes the gate chain and fails inside the repo insert as PG error 22003 (DECIMAL(8,2) overflow); wrapPGError does not map 22003, so the handler's writeError default returns 500 — violating the 'never a 500 for client input' boundary contract the handler itself documents (T-13-32) and ADR-BE-018 §6 'absurd values' pin (D-13-03)."
    artifacts:
      - path: "internal/core/services/direction/direction.go"
        issue: "wholeCent accepts arbitrarily large hours; overflow surfaces as a 500"
    missing:
      - "upper bound in wholeCent (e.g. hours <= 999999.99) so absurd values map to ErrInvalidHours → 400"
  - truth: "directed_to / wg_id / activity_id refs validated same-org at the service (ADR-BE-018 §Security, house style)"
    status: failed
    reason: "WR-01: the Create gate chain validates the activity (same-org) and the WG (same-org + scope) but never checks that *req.DirectedTo is an active member of orgID. In manager-planned mode a manager whose routing passes (or a role-gated org manager) can create rows directed at users of other orgs — the FK on directed_to is users(id) only, so the insert succeeds. Reads stay org-contained, but the cross-org reference is exactly what the ADR pins against."
    artifacts:
      - path: "internal/core/services/direction/direction.go"
        issue: "no orgRepo.GetMembership(*req.DirectedTo, orgID) check in the DirectedTo != nil branch before the mode gate"
    missing:
      - "same-org active-membership validation for directed_to in Service.Create"
  - truth: "Direction audit action vocabulary pinned verbatim in ADR-BE-018 §3 is honored by the code paths (created/activated/cancelled/superseded/claimed/unclaimed)"
    status: failed
    reason: "WR-03: domain exports AuditActionUnclaimed = 'unclaimed' (domain/direction.go:168) but NO code path writes it — Service.Unclaim writes AuditActionCancelled (direction.go:431-439) and the port doc pins 'One cancelled audit row' (direction_repository.go:66-67). The code matches plan 13-05's contract but drifts from the ADR vocabulary; the exported constant is a trap for the next implementer, and a Phase 19 history filter on the pinned action set silently merges unclaims into cancels. Needs a decision: write 'unclaimed' from Unclaim, or remove the constant and amend ADR-BE-018 §3."
    artifacts:
      - path: "internal/core/services/direction/direction.go"
        issue: "Unclaim writes AuditActionCancelled; AuditActionUnclaimed never written"
      - path: "internal/core/domain/direction/direction.go"
        issue: "AuditActionUnclaimed exported but dead in the real path"
    missing:
      - "align ADR-BE-018 §3 with the actual unclaim action (either direction)"
---

# Phase 13: Direction Backend — The Plan Plane — Verification Report

**Phase Goal:** The third plane lands: direction entity with per-day storage and derived modes, lifecycle with supersede chaining, WG claim model, org-configurable planning policy, and the direction-coverage read-model (DIR-01..06). ADR-P-015 + BE encoding drafted.
**Verified:** 2026-08-08T15:07:05Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Truths derived from ROADMAP.md Phase 13 Success Criteria (the contract) — 8 criteria, each verified against actual code, not SUMMARY claims.

| #   | Truth | Status     | Evidence       |
| --- | ----- | ---------- | -------------- |
| 1   | Direction rows exist per (employee, activity, day, est_hours) — per-day storage always, multiple rows may share a day, no intra-day ordering; mode derived (planned_date set → scheduled, null → queued with priority + due_date) (DIR-01) | ✓ VERIFIED | `migrations/021_direction_rows.up.sql`: 16-column table, six named CHECKs (XOR `direction_target_check`, `direction_wg_queued_check`, `direction_est_hours_check`, `direction_scheduled_hours_check`, `direction_status_check`, `direction_cancel_reason_check`), four indexes, NO UNIQUE constraint on the day tuple. `TestMigration021_DirectionRows_UpDownUpCycle` PASSES (2.60s) — asserts both-side IS [NOT] NULL constraint rejection (23514 + named constraint) and per-day multiplicity (two rows same employee/activity/day both insert). |
| 2   | Self-direction is first-class (`directed_by == directed_to`, no approval); managers direct within subtree/WG reach via BE-014 machinery (DIR-01) | ✓ VERIFIED | `Service.Create` gate chain (direction.go:190-295): mode gate via `ResolvePlanningMode` (self_planned self-direction skips routing; manager_planned routes everyone incl. self), `managerReach` uses `routing.ResolveManagerStage` coverage-gate shape. Handler battery proves matrix: self-direction 200; employee creating for colleague in manager_planned org → 403 (`direction_handler_test.go:167,179` PASS). |
| 3   | Lifecycle draft → active → superseded/cancelled works; done/lapsed/claimed derived, never stored; supersedes_id chains replanning with **audit-first via BE-012** (DIR-02) | ✗ FAILED | Lifecycle + derived states work and are tested (repo Activate/Cancel with FOR UPDATE re-validation + status-precondition backstop; ListPlan derives done/lapsed/claim-spectrum on read, never stored; `TestDirectionRepository_*` suite PASSES). **But CR-02: the 'superseded' transition writes no audit row** — `Service.Create` hands the repo one audit row (created) while the port contract and ADR-BE-018 §1/§3 pin two. Audit-first (BE-012) is violated for the supersede transition. |
| 4   | WG-direction rows are queued-only; a member's claim creates a user-targeted row via origin_direction_id; claimed is derived (DIR-03) | ✓ VERIFIED | Schema CHECK `direction_wg_queued_check` + service fast-fail; repo Claim tx (direction_repository.go:418+): WG-row FOR UPDATE lock, in-tx status/membership re-checks, Σ guard in cents over draft|active claim rows, uncapped when budget NULL, claim row carries origin_direction_id + directed_by attribution. `TestDirectionClaim_Concurrent` (bounded set commits, no over-subscription) and `TestDirectionClaim_SupersedeCancelReclaim` PASS. **CR-01 defect on the invalid-input path recorded as a gap** (user-targeted row id → nil-deref panic instead of 404). |
| 5   | Org policy is configurable: deadline date, horizon (day/week/month), per-employee mode (manager-planned vs self-planned); soft-policy (block vs nag) configurable for UI (DIR-04) | ✓ VERIFIED | `migrations/022_org_settings.up.sql` (key/value JSONB, PK(org_id,key)) + `planning_mode` membership override; domain keys/validators (orgsettings.go); orgsettings service Get (code-level defaults, 8.0) / Put (manager+ gate, batch validation, in-tx audit `{key, before, after}`); literal GET/PUT `/organizations/settings` routes coexist with typed wildcard (main.go:238-241); `ResolvePlanningMode` precedence override → org default → manager_planned. `TestOrgSettingsHandler_*`, `TestOrgSettingsRepository_*`, `TestOrgSettingsHandler_RouteCoexistence` all PASS. |
| 6   | Scheduler read path consumes P-008 absence windows + employment validity and returns plan-time warnings; never blocks (DIR-05) | ✓ VERIFIED | `AbsenceWindows` (declared+confirmed, D-13-29), `computeWarnings` pure overlay (away/partial/over-capacity/invalid with pinned messages), validity warnings via `orgRepo.GetMembership` valid_from/valid_until, warnings ride create + read responses, never block writes (prohibition honored — no backend enforcement of deadlines). `TestService_Warnings` + `TestDirectionRepository_AbsenceWindows` PASS. |
| 7   | Direction-coverage read-model returns planned hours vs capacity per employee/period with uncovered days surfaced, per employee / unit / WG (DIR-06) | ✓ VERIFIED | Repo `Coverage` (one query: unnest × generate_series, capacity = COALESCE(planning_daily_hours, 8.0) − absence hours, GREATEST(…,0) floor, planned Σ draft|active, cents-rounded); service `resolveScopeEmployees` (employee/unit+descendants/WG, manager gates), validity split, fully-absent days excluded from uncovered surfacing, period totals over full row set. `TestDirectionRepository_Coverage_*` + handler coverage tests PASS. |
| 8   | Origin fallback active: activities with empty origin refs resolve refs from the first direction record (FND-04 read path, additive) | ✓ VERIFIED | `FirstDirectionRefs` (earliest created_at non-cancelled, org-scoped, nil when none); activity service GetByID + List apply `OriginType == nil` predicate, response-only derivation, never written back. `TestActivityOriginFallback_GetByID/List` PASS (all three branches: derives / stays empty / stored refs authoritative). |

**Score:** 7/8 truths verified (0 present, behavior-unverified)

### Required Artifacts

All 27 declared artifacts exist and were read in full (three-level check: exists ✓, substantive ✓, wired ✓). Representative table:

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `migrations/021_direction_rows.{up,down}.sql` | direction table + CHECKs + indexes | ✓ VERIFIED | 16 cols, 6 named CHECKs, 4 indexes, no UNIQUE on day tuple; down drops CASCADE |
| `migrations/022_org_settings.{up,down}.sql` | key/value JSONB + planning_mode | ✓ VERIFIED | PK(org_id,key), no JSONB CHECK (vocabulary in code); down column-before-table |
| `internal/core/domain/direction/direction.go` | entity + matrix + vocabularies | ✓ VERIFIED | 15 fields, status consts, transitionMatrix, CanTransition/IsTerminalStatus, derived-state + claim-spectrum consts, Warning/CoverageRow/PlanRow/DirectionRefs/AbsenceWindow shapes, audit vocabulary |
| `internal/core/domain/orgsettings/orgsettings.go` | keys/validators/errors | ✓ VERIFIED | 4 key consts, mode/horizon vocabularies, DefaultDailyHours 8.0, knownKeys validators, audit pins |
| `internal/core/ports/direction_repository.go` | 10-method port contract | ✓ VERIFIED | In-tx audit contract documented; compile-time contract for repo/service/mocks |
| `internal/adapters/secondary/postgres/direction_repository.go` | mutators + read-models | ✓ VERIFIED | Create supersede-tx, Activate/Cancel/Unclaim with FOR UPDATE + backstop, Claim Σ-guard in cents, ListPlan/Coverage/AbsenceWindows/FirstDirectionRefs; `var _ ports.DirectionRepository` assertion |
| `internal/core/services/direction/direction.go` | gate chain + warnings + read assembly | ✓ VERIFIED | 841 lines; XOR→hours→same-org→WG-scope→mode→routing gates, lifecycle orchestration, warning overlay, coverage assembly, read gates |
| `internal/adapters/primary/http/direction_handler.go` | 7-route handler + sentinel map | ✓ VERIFIED | writeError 404/400/403/409/500 map matches plan; warnings normalized to always-array |
| `cmd/server/main.go` | wiring | ✓ VERIFIED | directionRepo → activityService fallback seam + directionService (shared orgsettings/routing deps) + 7 routes middleware.Auth |
| `internal/core/services/activity/activity.go` | origin fallback | ✓ VERIFIED | GetByID + List apply OriginType == nil → FirstDirectionRefs, read-only |
| `hourglass-vault/decisions/project/ADR-P-015 — Direction, The Plan Plane.md` | project ADR | ✓ VERIFIED | 144 lines, D-R..D-AA + R4 + assumption-delta decisions |
| `hourglass-vault/decisions/backend/ADR-BE-018 — Direction & Org Settings Encoding.md` | BE encoding ADR | ✓ VERIFIED | 124 lines, vocabularies + claim lock + supersede-of-claim-row pin + 8 assumption pins; both indexed in `_index.md` |

**Artifacts:** 27/27 verified (gsd-tools `verify.artifacts` returned empty due to plan-format mismatch — manual verification of every declared path substituted)

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Service.Create | repo.Create | audits slice → same-tx insertDirectionAudit | ⚠️ PARTIAL | Wire exists but carries ONE audit row on supersede (CR-02) — repo iterates `audits` slice (would write both) |
| Service.Claim | repo.Claim | wgRepo.ListMembers fast-fail → FOR UPDATE tx | ⚠️ PARTIAL | Wire works for WG rows; nil-deref panic before the repo call on user-targeted rows (CR-01) |
| Service mode gate | orgsettings.Service.ResolvePlanningMode | constructor dep (13-07 pin) | ✓ WIRED | directionService built after orgsettings block with shared instance (main.go:186) |
| directionRepo | activityService | constructor arg (fallback seam) | ✓ WIRED | main.go:153-154 + handler_test_helper.go:87-88 |
| OrgSettings handler | literal /organizations/settings routes | ServeMux literal-beats-wildcard | ✓ WIRED | main.go:238-241; coexistence test PASSES; typed wildcard routes NOT removed |
| FirstDirectionRefs | activity read path | OriginType == nil predicate | ✓ WIRED | activity.go:73-74, 107-108; contract tests PASS |
| 7 direction routes | directionHandler | middleware.Auth | ✓ WIRED | main.go:284-290, all 7 registered |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Coverage repo | capacity/planned/gap | SQL over `direction` + `org_settings` + `availability_windows` | ✓ FLOWING | Real queries; testcontainers integration tests prove non-empty rows |
| ListPlan repo | derived done/lapsed/claimed | terminal-activity CTE + hasAnyEntries + claimSpectrum | ✓ FLOWING | Recursive CTE re-anchored at activities.id; tests assert derived values |
| Handler create | row + warnings | service → repo tx | ✓ FLOWING | Integration battery asserts persisted rows + warnings array |
| Org settings GET | known keys + defaults | org_settings table + code-level defaults | ✓ FLOWING | No hardcoded empty returns; absent daily-hours defaults to 8.0 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full test suite | `go test ./...` | exit 0 — all packages pass (testcontainers Postgres) | ✓ PASS |
| Migration 021/022 cycles | `go test -run "TestMigration021|TestMigration022"` | 2/2 PASS (23514 + constraint-name assertions) | ✓ PASS |
| Direction repo mutators | `go test -run "TestDirectionRepository_(Create|Activate|Cancel|Claim|Unclaim)"` | 19/19 PASS incl. supersede chain, cross-org, concurrent claims | ✓ PASS |
| Read-models | `go test -run "TestDirectionRepository_(ListPlan|Coverage|AbsenceWindows|FirstDirectionRefs|DerivedStates)"` | 9/9 PASS | ✓ PASS |
| Handler battery | `go test ./internal/adapters/primary/http/ -run "Direction|OrgSettings"` | 10/10 PASS (permission matrix, sentinels, route coexistence) | ✓ PASS |
| Service unit | `go test ./internal/core/services/direction/` | 8/8 PASS (Create/Activate/Cancel/Claim/Unclaim/Warnings/ListPlan/Coverage) | ✓ PASS |
| Origin fallback | `go test ./internal/core/services/activity/ -run "OriginFallback"` | 2/2 PASS (GetByID + List) | ✓ PASS |
| Build/vet | `go build ./...` + `go vet` (direction/orgsettings/http pkgs) | exit 0 | ✓ PASS |
| Deferred 011-cycle fix | `go test -run "TestMigration011"` | PASS (skip list extended, commit 4f7143a) — deferred-items.md item resolved in code | ✓ PASS |

### Probe Execution

No probe scripts declared for this phase (no `scripts/*/tests/probe-*.sh`, no probe references in PLAN/SUMMARY). Behavior verified through the Go test suites above. N/A.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DIR-01 | 13-01, 13-03, 13-05, 13-07, 13-08 | Direction rows per-day, derived mode, self-direction | ✓ SATISFIED | Migration 021 + Create gate chain + handler; tests PASS |
| DIR-02 | 13-01, 13-02, 13-03, 13-05, 13-06, 13-07, 13-08 | Lifecycle + derived states + supersede chain, audit-first | ✗ BLOCKED (partial) | Lifecycle/derived states work; **supersede transition audit row missing (CR-02)** |
| DIR-03 | 13-01, 13-02, 13-03, 13-05, 13-07, 13-08 | WG claim model | ✓ SATISFIED | Claim tx + concurrent battery PASS; CR-01 panic on invalid input recorded as gap |
| DIR-04 | 13-01, 13-02, 13-03, 13-04 | Org policy configurable | ✓ SATISFIED | org_settings vertical + ResolvePlanningMode; tests PASS |
| DIR-05 | 13-02, 13-06, 13-07, 13-08 | Scheduler reads absences + validity, warnings never block | ✓ SATISFIED | AbsenceWindows + computeWarnings + validity; tests PASS |
| DIR-06 | 13-02, 13-03, 13-06, 13-07, 13-08 | Coverage read-model | ✓ SATISFIED | Repo math + service scope resolution + handler; tests PASS |
| FND-04 (phase 11 read-path) | 13-08 | Origin fallback from first direction record | ✓ SATISFIED | Activity read path + contract tests PASS |

**Coverage:** 6/7 requirements satisfied; DIR-02 partially blocked by CR-02.

Requirement cross-reference check: every ID in every plan's `requirements` frontmatter (DIR-01..06, FND-04) is accounted for above. No orphaned requirements — REQUIREMENTS.md maps exactly DIR-01..06 to Phase 13, all present in at least one plan, all evaluated.

### Decision Coverage (non-blocking gate, #2492)

13-CONTEXT.md `<decisions>` block carries D-13-01..34 + the agent's-discretion list. All 34 D-13 decisions traced to shipped artifacts (ADRs P-015/BE-018, migrations, domain, service, repo, handler) — verified during reading. **One decision partially honored:** D-13-03 "Service rejects `est_hours <= 0` (and absurd values)" — the absurd-values clause is unmet (WR-02). All other decisions honored. Non-blocking warning.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | - | TBD/FIXME/XXX/TODO/HACK/placeholder/empty-return scan across all 27 phase files | - | No debt markers, no placeholders, no empty implementations found |

**Anti-patterns:** 0 found. No debt-marker blockers.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
| ---------- | ---------- | ------ | ------- | -------- | --------------- | ------- |
| direction_ontology_migrations_test.go | DIR-01..04 | 2 | 0 | No | Value (23514 + constraint name, per-day multiplicity) | OK |
| direction_repository_test.go | DIR-01..06 | 28+ | 0 | No | Behavioral (concurrent battery, chain invariants, Σ in cents) | OK |
| direction_test.go (service) | DIR-01..06 | 8 | 0 | No | Behavioral (gate chain, warnings) | OK |
| direction_handler_test.go | DIR-01..06 | 15+ | 0 | No | Behavioral (permission matrix, sentinels over real stack) | OK |
| activity_origin_fallback_test.go | FND-04 | 2 | 0 | No | Behavioral (derives / empty / authoritative branches) | OK |
| org_settings suite (3 files) | DIR-04 | 24+ | 0 | No | Behavioral (rollback leaves no row AND no audit) | OK |

- **Disabled tests on requirements:** 0
- **Circular patterns detected:** 0 — integration tests run real migrations against a real Postgres (testcontainers); expected values are asserted, not captured from the SUT
- **Insufficient assertions:** 0 — value- and behavioral-level assertions throughout
- **Coverage quantity:** matches plan-declared batteries (handler 7-matrix cases, repo concurrent claims, migration cycles)

### Human Verification Required

N/A — Infrastructure/foundation phase (backend-only: migrations, repos, services, HTTP handlers, ADRs). No user-facing elements; all acceptance criteria verified programmatically via the Go test suites (which include real-Postgres integration tests).

Note: CR-01/CR-02/WR-02 are code-level defects with concrete fixes — they are recorded as gaps (above), not as human-verification items. The claim-panic path is currently UNTESTED (no test claims a user-targeted row) — a regression test for the fix is listed in the gap.

### Gaps Summary

The phase is ~90% delivered: all eight roadmap success criteria are functionally implemented, all tests pass (full suite green, including the concurrent-claim battery and migration cycle tests), and all 27 artifacts exist, are substantive, and are wired. **However, the phase's own code review (13-REVIEW.md) found 2 critical and 3 warning defects, all confirmed against the code during this verification, and CR-02 directly invalidates roadmap SC #3 (audit-first via BE-012 for the supersede chain):**

**Critical gaps (block progress):**

1. **CR-02 — Missing 'superseded' audit row on supersede-on-create** (`internal/core/services/direction/direction.go:303-310`). Violates ADR-BE-018 §1/§3, the port contract ("two audit rows: created + superseded"), and DIR-02's audit-first pin. The repo writes whatever audits it is handed; the service passes only the 'created' row. A Phase 19 history filter on 'superseded' returns nothing for actual supersedes. Fix: append the second audit row in Service.Create when `req.SupersedesID != nil`.

2. **CR-01 — Nil-pointer panic in Service.Claim on user-targeted rows** (`internal/core/services/direction/direction.go:462`). Any authenticated user can crash the request via `POST /direction/claims` with a user row id — no 4xx, connection dropped, stack trace logged. Fix: `if wg.WgID == nil { return ErrDirectionNotFound }` before `ListMembers(ctx, *wg.WgID)` + regression test.

**Warning gaps (should fix before Phase 19 consumes the API):**

3. **WR-02 — Absurd est_hours → DB overflow → 500** (`direction.go:116-118`). `wholeCent` lacks an upper bound; est_hours ≥ 1,000,000 violates the "never a 500 for client input" boundary contract (T-13-32) and D-13-03's absurd-values pin. Fix: cap at 999999.99.

4. **WR-01 — directed_to never validated same-org** (`direction.go:251-268`). Cross-org user refs insertable per ADR-BE-018 §Security. Fix: `orgRepo.GetMembership(*req.DirectedTo, orgID)` active-membership check in the `DirectedTo != nil` branch.

5. **WR-03 — Audit vocabulary drift: 'unclaimed' never written** (domain `direction.go:168` vs service `Unclaim`). Either write `AuditActionUnclaimed` from Unclaim or delete the constant and amend ADR-BE-018 §3 — pick one and pin it.

**Deferred:** none — no later milestone phase (14 Availability, 19 Direction Surfaces, etc.) addresses these backend defects; Phase 19 consumes the audit vocabulary and would inherit the broken filter.

---

_Verified: 2026-08-08T15:07:05Z_
_Verifier: the agent (gsd-verifier)_
