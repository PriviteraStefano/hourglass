---
phase: 13-direction-backend-the-plan-plane
verified: 2026-08-08T16:19:33Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 7/8
  gaps_closed:
    - "CR-02: supersede-on-create now writes BOTH audit rows ('created' + 'superseded') in the same tx (Service.Create audits slice -> repo in-tx loop)"
    - "CR-01: Service.Claim nil-WgID guard returns ErrDirectionNotFound (404) on user-targeted rows — no panic; mock mirrors the repo wg_id IS NOT NULL predicate; ListMembersFn trap seam"
    - "WR-02: wholeCent upper bound (maxEstHours = 999999.99) maps absurd est_hours to ErrInvalidHours -> 400 at create AND claim; no DECIMAL(8,2) 22003 path from client input"
    - "WR-01: Service.Create validates *req.DirectedTo is an ACTIVE member of orgID (orgRepo.GetMembership) BEFORE the mode gate — cross-org/inactive refs -> ErrInvalidRequest -> 400"
    - "WR-03: Service.Unclaim writes AuditActionUnclaimed ('unclaimed') with {reason}; port doc + repo test aligned to ADR-BE-018 §3; constant is live, no vocabulary drift"
  gaps_remaining: []
  regressions: []
unverified_prohibitions:
  - statement: "Supersede-on-create must never drop the 'superseded' audit event for the flipped target — audit-first via BE-012 is the DIR-02 contract"
    llm_judge_verdict: "HONORED — Service.Create appends the 'superseded' row when SupersedesID != nil (direction.go:336-345); CR-02 unit test asserts 2-row slice with created-then-superseded ordering (direction_test.go:418-449); HTTP test asserts both audit_logs rows via pool queries (direction_handler_test.go:351-375)"
    flag: "unverified-prohibition — human review recommended"
  - statement: "Client input (row ids, est_hours) must never cause a server panic or a 500 response — invalid input maps to 404/400"
    llm_judge_verdict: "HONORED — nil-WgID guard -> 404 (direction.go:505-507; handler test decodes the 404 envelope, connection stays up); maxEstHours ceiling -> 400 (direction.go:116,123; handler test asserts 400 not 500). No panic/500 path remains reachable from client input"
    flag: "unverified-prohibition — human review recommended"
  - statement: "Direction rows must never reference users outside the org — cross-org directed_to must be rejected with 400, never silently inserted (FK users(id) alone does not enforce org containment)"
    llm_judge_verdict: "HONORED — orgRepo.GetMembership gate at the top of the DirectedTo branch before ResolvePlanningMode (direction.go:265-271); nil/inactive membership -> ErrInvalidRequest; unit subtests (no-membership/inactive/active) + handler cross-org 400 subtest all pass"
    flag: "unverified-prohibition — human review recommended"
  - statement: "The pinned audit vocabulary (ADR-BE-018 §3) must be honored by live code paths — no exported constant left dead and no vocabulary drift in port docs"
    llm_judge_verdict: "HONORED — Service.Unclaim writes AuditActionUnclaimed (direction.go:472); port doc pins 'One unclaimed audit row in the same tx (ADR-BE-018 §3)' (direction_repository.go:66); repo test asserts the pinned action (direction_repository_test.go:853,858); ADR-BE-018 unchanged (already pinned 'unclaimed'); no dead exported constant remains"
    flag: "unverified-prohibition — human review recommended"
---

# Phase 13: Direction Backend — The Plan Plane — Verification Report (Final, All 10 Plans)

**Phase Goal:** Plan plane: direction entity, lifecycle, claim model, org policy, coverage read-model; ADR-P-015 + BE encoding. Direction backend for the plan plane — direction rows (entity, matrix, vocabularies), lifecycle (draft→active→superseded/cancelled with derived done/lapsed/claimed states), claim model (WG-direction queued-only + claim via origin_direction_id), org policy (deadline/horizon/mode via org_settings), coverage read-model (planned vs capacity per employee/day), and the ADR-P-015 + ADR-BE-018 records. Includes gap-closure plans 13-09 (CR-02/CR-01/WR-02) and 13-10 (WR-01/WR-03).
**Verified:** 2026-08-08T16:19:33Z
**Status:** passed
**Re-verification:** Yes — all 5 gaps from the 13-01..08 verification closed by 13-09 + 13-10; full 10-plan phase re-verified

## Goal Achievement

### Observable Truths

Truths from ROADMAP.md Phase 13 Success Criteria (8 criteria — the contract). The 5 previously-failed truths received full 3-level verification (exists, substantive, wired + behavioral tests); the previously-passed truths received quick regression (existence + full-suite green).

| #   | Truth | Status     | Evidence       |
| --- | ----- | ---------- | -------------- |
| 1   | Direction rows exist per (employee, activity, day, est_hours) — per-day storage always, multiple rows may share a day, no intra-day ordering; mode derived (planned_date set → scheduled, null → queued with priority + due_date) (DIR-01) | ✓ VERIFIED | Migration 021 table + six named CHECKs, no UNIQUE on day tuple; migration cycle test PASSES; WR-02 ceiling now bounds est_hours at the service (999999.99 accepted / 1000000 → 400 — see truth 8) |
| 2   | Self-direction is first-class (`directed_by == directed_to`, no approval); managers direct within subtree/WG reach via BE-014 machinery (DIR-01) | ✓ VERIFIED | Create gate chain incl. the new WR-01 membership gate (direction.go:257-293); handler battery self-direction 200 / colleague-in-manager-org 403 PASS; cross-org directed_to → 400 now enforced |
| 3   | Lifecycle draft → active → superseded/cancelled works; done/lapsed/claimed are derived, never stored; supersedes_id chains replanning with **audit-first via BE-012** (DIR-02) | ✓ VERIFIED (gap CR-02 closed) | `Service.Create` builds a named audits slice — 'created' on the new row always, 'superseded' on the flipped target when `SupersedesID != nil` (direction.go:328-345) — handed to the repo in-tx loop (port contract restored, ADR-BE-018 §3). Unit: 2-row slice with ordering/entity-ids/actor/nil-payloads (direction_test.go:418-449). HTTP: both `audit_logs` rows proven via pool queries (direction_handler_test.go:351-375). `go test ./internal/adapters/primary/http/ -run TestDirectionHandler` PASS |
| 4   | WG-direction rows are queued-only; a member's claim creates a user-targeted row via origin_direction_id; claimed is derived (DIR-03) | ✓ VERIFIED (gap CR-01 closed) | Repo claim tx Σ-guard in cents under FOR UPDATE unchanged. `Service.Claim` now guards `wg.WgID == nil → ErrDirectionNotFound` between the status fast-fail and `ListMembers` (direction.go:505-507), mirroring the repo's lock predicate. Unit: user-row claim returns ErrDirectionNotFound and the ListMembers trap never fires (direction_test.go:738-756). HTTP: claims on an active user-targeted row → 404 envelope, connection stays up (direction_handler_test.go:305-321). Mock mirrors the predicate (mock_direction_repo.go:175-178) |
| 5   | Org policy is configurable: deadline date, horizon (day/week/month), per-employee mode (manager-planned vs self-planned); soft-policy (block vs nag) configurable for UI (DIR-04) | ✓ VERIFIED | org_settings vertical + ResolvePlanningMode precedence + literal GET/PUT routes; regression: TestOrgSettingsHandler_* + route coexistence PASS (full suite green) |
| 6   | Scheduler read path consumes P-008 absence windows + employment validity and returns plan-time warnings; never blocks (DIR-05) | ✓ VERIFIED | AbsenceWindows + computeWarnings + validity warnings, never block writes; regression: repo + service warning tests PASS |
| 7   | Direction-coverage read-model returns planned hours vs capacity per employee/period with uncovered days surfaced, per employee / unit / WG (DIR-06) | ✓ VERIFIED | Repo Coverage query + service scope resolution + handler; regression: TestDirectionRepository_Coverage_* + handler coverage tests PASS |
| 8   | Client input (row ids, est_hours, directed_to) is hardened at the service boundary: no panic, no 500, no cross-org references (WR-01/02/03 + CR-01/02 closure contract) | ✓ VERIFIED | WR-02: `maxEstHours = 999999.99` const + wholeCent ceiling (direction.go:113-123); create/claim reject 1000000 → ErrInvalidHours, ceiling accepted (direction_test.go:308-333, 728-730; handler 400 subtest direction_handler_test.go:297-303). WR-01: membership gate → 400 (direction_test.go:231-264; handler cross-org 400 direction_handler_test.go:189-205). WR-03: unclaim writes 'unclaimed' (direction.go:472). All dual-layer regression tests PASS (see Behavioral Spot-Checks) |
| 9   | Origin fallback active: activities with empty origin refs resolve refs from the first direction record (FND-04 read path, additive) | ✓ VERIFIED | FirstDirectionRefs + OriginType == nil predicate; regression: activity origin fallback tests PASS |

**Score:** 9/9 truths verified (8 roadmap SCs + the gap-closure hardening contract; 0 present, behavior-unverified; 5 previously-failed truths all re-verified as closed)

### Required Artifacts

All artifacts from the previous 27-file pass remain present and substantive; the 13-09/13-10 modifications are verified in full. Representative table (all three levels: exists, substantive, wired):

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `internal/core/services/direction/direction.go` | audits slice on supersede + Claim nil guard + wholeCent ceiling + membership gate + unclaim action | ✓ VERIFIED | audits slice 2 rows on supersede (328-345); `maxEstHours` const (116); wholeCent ceiling (123); GetMembership gate (265-271); Claim guard (505-507); Unclaim → AuditActionUnclaimed (472). All changes read in full |
| `internal/core/services/direction/direction_test.go` | regression subtests for all 5 fixes | ✓ VERIFIED | CR-02 2-row audit subtest (418-449); WR-01 no-membership/inactive/active (231-264); WR-02 absurd + ceiling (308-333, 728-730); CR-01 ListMembers trap (738-756); WR-03 unclaim assertion (807-824). `go test ./internal/core/services/direction/` PASS |
| `internal/core/services/testdata/mock_direction_repo.go` | Claim user-row guard | ✓ VERIFIED | `wg.WgID == nil → ErrDirectionNotFound` after the org check (175-178), mirroring the repo predicate |
| `internal/core/services/testdata/mocks.go` | ListMembersFn per-method override | ✓ VERIFIED | Field declared (939), consulted before the default WGMembers lookup (1007-1008), nil = backward compatible |
| `internal/adapters/primary/http/direction_handler_test.go` | 4 new HTTP subtests | ✓ VERIFIED | cross-org 400 (189-205); est_hours ceiling 400 (297-303); claims-on-user-row 404 (305-321); supersede BOTH audit rows via pool queries (351-375). `go test ./internal/adapters/primary/http/ -run TestDirectionHandler` PASS |
| `internal/core/ports/direction_repository.go` | Unclaim doc aligned | ✓ VERIFIED | "One 'unclaimed' audit row in the same tx (ADR-BE-018 §3)" (line 66) |
| `internal/adapters/secondary/postgres/direction_repository_test.go` | Unclaim test pinned action | ✓ VERIFIED | directionAudit + countDirectionAudits on AuditActionUnclaimed (853, 858); repo is a pass-through — no logic change. `go test ... -run TestDirectionRepository_Unclaim` PASS |
| Migrations 021/022, domain direction/orgsettings, postgres repo, handler, main.go wiring, ADR-P-015/BE-018 | unchanged phase artifacts | ✓ VERIFIED | Quick regression: files present, full suite green (24 packages), ADR-BE-018 NOT amended (still pins 'unclaimed' verbatim, §3) |

**Artifacts:** 29/29 verified (27 prior + 2 modified-set re-checks; manual verification of every declared path in all 10 plans)

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| Service.Create audits slice | repo.Create in-tx audit loop | audits slice → insertDirectionAudit | ✓ WIRED | Both rows on supersede now reach the tx (direction.go:346; direction_repository.go:242-246); DB-level test proves both rows exist |
| Service.Claim | wgRepo.ListMembers | nil-guard before deref | ✓ WIRED | ListMembers reached only when WgID != nil (direction.go:505-508); trap proves it is never called on user rows |
| wholeCent | DECIMAL(8,2) column | maxEstHours ceiling | ✓ WIRED | Every accepted value insertable — no 22003 path from client input |
| Service.Create DirectedTo branch | orgRepo.GetMembership | precedes ResolvePlanningMode | ✓ WIRED | Cross-org refs die before any mode/routing decision (direction.go:265 vs 272) |
| Service.Unclaim | repo.Unclaim audit | pinned 'unclaimed' action | ✓ WIRED | Port doc + repo test aligned; ADR-BE-018 §3 unchanged (was already correct) |
| Service mode gate | orgsettings.Service.ResolvePlanningMode | constructor dep | ✓ WIRED | Unchanged from prior pass |
| directionRepo | activityService | constructor arg (fallback seam) | ✓ WIRED | Unchanged from prior pass |
| 7 direction routes | directionHandler | middleware.Auth | ✓ WIRED | Unchanged from prior pass |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Coverage repo | capacity/planned/gap | SQL over direction + org_settings + availability_windows | ✓ FLOWING | Real queries; testcontainers integration tests prove non-empty rows |
| ListPlan repo | derived done/lapsed/claimed | terminal-activity CTE + hasAnyEntries + claimSpectrum | ✓ FLOWING | Tests assert derived values |
| Handler create | row + warnings | service → repo tx | ✓ FLOWING | CR-02 subtest proves persisted audit rows via pool queries |
| Org settings GET | known keys + defaults | org_settings table + code-level defaults | ✓ FLOWING | No hardcoded empty returns |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full test suite | `go test ./...` | exit 0 — all 24 packages ok, zero failures | ✓ PASS |
| Service unit (all 5 fixes) | `go test ./internal/core/services/direction/ -run "TestService_Create\|TestService_Claim\|TestService_Unclaim" -count=1` | ok (0.326s) | ✓ PASS |
| Service package full | `go test ./internal/core/services/direction/ -count=1` | ok (0.215s) | ✓ PASS |
| Handler integration (testcontainers) | `go test ./internal/adapters/primary/http/ -run "TestDirectionHandler\|TestOrgSettingsHandler" -count=1` | ok (18.7s) — includes CR-02 DB audit check, CR-01 404, WR-02 400, WR-01 cross-org 400 | ✓ PASS |
| Repo mutator + migrations | `go test ./internal/adapters/secondary/postgres/ -run "Direction\|OrgSettings\|Migration021\|Migration022" -count=1` | ok (9.2s) | ✓ PASS |
| Repo Unclaim contract | `go test ./internal/adapters/secondary/postgres/ -run "TestDirectionRepository_Unclaim" -count=1` | ok (4.2s) | ✓ PASS |
| Build + vet | `go build ./...` + `go vet` (direction/orgsettings/http/postgres) | exit 0 | ✓ PASS |

### Probe Execution

No probe scripts declared for this phase (no `scripts/*/tests/probe-*.sh`, no probe references in PLAN/SUMMARY). Behavior verified through the Go test suites above. N/A.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DIR-01 | 13-01, 13-03, 13-05, 13-07, 13-08, 13-09, 13-10 | Direction rows per-day, derived mode, self-direction + est_hours boundary | ✓ SATISFIED | Migration 021 + Create gate chain (now incl. WR-01 membership + WR-02 ceiling); all tests PASS |
| DIR-02 | 13-01, 13-02, 13-03, 13-05, 13-06, 13-07, 13-08, 13-09, 13-10 | Lifecycle + derived states + supersede chain, audit-first via BE-012 | ✓ SATISFIED | CR-02 closed: both audit rows in one tx, DB-proven; WR-03 closed: unclaim writes pinned action |
| DIR-03 | 13-01, 13-02, 13-03, 13-05, 13-07, 13-08, 13-09, 13-10 | WG claim model | ✓ SATISFIED | Claim tx + concurrent battery PASS; CR-01 closed: user-targeted rows → 404, no panic |
| DIR-04 | 13-01, 13-02, 13-03, 13-04 | Org policy configurable | ✓ SATISFIED | org_settings vertical + ResolvePlanningMode; tests PASS |
| DIR-05 | 13-02, 13-06, 13-07, 13-08 | Scheduler reads absences + validity, warnings never block | ✓ SATISFIED | AbsenceWindows + computeWarnings + validity; tests PASS |
| DIR-06 | 13-02, 13-03, 13-06, 13-07, 13-08 | Coverage read-model | ✓ SATISFIED | Repo math + service scope resolution + handler; tests PASS |
| FND-04 (phase 11 read-path) | 13-08 | Origin fallback from first direction record | ✓ SATISFIED | Activity read path + contract tests PASS |

**Coverage:** 7/7 requirements satisfied — DIR-02 and DIR-03 (previously blocked/partial) are now fully satisfied.

Requirement cross-reference check: plans 13-09/13-10 declare `[DIR-01, DIR-02, DIR-03]` — all within the phase's DIR-01..06 scope and consistent with the 13-01..08 requirement map (which covered DIR-01..06 + FND-04). REQUIREMENTS.md maps exactly DIR-01..06 to Phase 13 — all six present in at least one plan, all evaluated. **No orphaned requirements, no unaccounted IDs.**

### Decision Coverage (non-blocking gate, #2492)

All 34 D-13 decisions traced to shipped artifacts in the prior pass. D-13-03's "absurd values" clause — previously unmet (WR-02) — is now honored via the `maxEstHours` ceiling. The 13-10 `decision_record` (WR-03: align code to ADR, reverse the 13-05 note) is implemented and recorded in STATE.md's learnings. All decisions honored.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | - | TBD/FIXME/XXX/HACK/PLACEHOLDER/empty-return scan across all 7 files modified by 13-09/13-10 | - | No debt markers, no placeholders, no empty implementations found |

**Anti-patterns:** 0 found. No debt-marker blockers.

### Unverified Prohibitions (ADR-550 D4 — judgment-tier, LLM-judge verdicts)

The 4 prohibitions declared in 13-09/13-10 frontmatter (`status: unverified, flagged: true`, judgment-tier). Autonomous-mode handling: NON-AUTHORITATIVE LLM-judge verdicts recorded with direct code + test evidence; each flagged `unverified-prohibition — human review recommended` (never a silent pass; no AFK hard halt).

| # | Prohibition | LLM-Judge Verdict | Evidence |
| - | ----------- | ----------------- | -------- |
| 1 | Supersede-on-create must never drop the 'superseded' audit event (BE-012 / DIR-02) | **HONORED** | direction.go:336-345 appends the row; unit 2-row assertion + HTTP DB-level count (both pass) |
| 2 | Client input must never cause a panic or 500 — maps to 404/400 | **HONORED** | nil-guard → 404 (handler decodes envelope); maxEstHours → 400 at create and claim (both pass) |
| 3 | Direction rows must never reference users outside the org | **HONORED** | GetMembership gate precedes mode gate; nil/inactive → 400; cross-org handler test 400 (pass) |
| 4 | Pinned audit vocabulary (ADR-BE-018 §3) honored by live code paths | **HONORED** | Unclaim writes 'unclaimed'; port doc aligned; repo test pinned; ADR unchanged; no dead constant |

**Human review recommended (non-blocking):** confirm the 4 verdicts above against the cited lines/tests at the next human checkpoint.

### Human Verification Required

N/A for UI/UX — infrastructure/foundation phase (backend-only: migrations, repos, services, HTTP handlers, ADRs). All acceptance criteria verified programmatically via the Go test suites (real-Postgres testcontainers integration).

The 4 unverified-prohibition flags above are the only human-review-recommended items (ADR-550 D4 soft-gate — autonomous completion reads "complete with 4 flagged prohibitions").

### Gaps Summary

**None.** All 5 verification gaps from the 13-01..08 pass are closed by plans 13-09 (CR-02, CR-01, WR-02) and 13-10 (WR-01, WR-03), each with TDD regression tests at the service-unit AND HTTP/repo boundaries:

1. **CR-02** — `Service.Create` hands the repo a 2-row audits slice on supersede (direction.go:328-345); DB-level HTTP test proves both `audit_logs` rows; unit test pins ordering/entity-ids.
2. **CR-01** — nil-WgID guard before `ListMembers` (direction.go:505-507); mock mirrors the repo predicate; ListMembers trap proves no deref path remains; HTTP test proves 404 with the connection up.
3. **WR-02** — `maxEstHours = 999999.99` ceiling in `wholeCent` (direction.go:116,123); 1000000 → 400 at create AND claim; ceiling value accepted; no 22003 path.
4. **WR-01** — `orgRepo.GetMembership` active-membership gate before the mode gate (direction.go:265-271); cross-org/inactive → 400; no existence oracle (house style).
5. **WR-03** — `Service.Unclaim` writes `AuditActionUnclaimed` (direction.go:472); port doc, repo test, service test all aligned; ADR-BE-018 §3 untouched (already pinned).

Full suite green (24 packages, exit 0), `go build ./...` + `go vet` clean, no debt markers in any phase file. **Phase 13 goal achieved: 9/9 truths verified (8/8 roadmap success criteria + gap-closure hardening), 7/7 requirements satisfied.**

---

_Verified: 2026-08-08T16:19:33Z_
_Verifier: the agent (gsd-verifier)_
