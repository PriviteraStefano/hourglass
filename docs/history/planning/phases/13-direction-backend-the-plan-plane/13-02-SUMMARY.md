---
phase: 13-direction-backend-the-plan-plane
plan: 02
subsystem: decisions
tags: [adr, direction, org-settings, plan-plane, audit, vocabulary, vault]

# Dependency graph
requires:
  - phase: 13-direction-backend-the-plan-plane (13-CONTEXT, 13-RESEARCH, 13-PATTERNS — the D-13-01..34 + A1-A10 records)
    provides: locked decisions and assumption pins the ADRs encode
provides:
  - ADR-P-015 (project): the direction plane — one entity, derived mode, per-day rows, supersede chain, derived states, claim model, org policy, warning overlay, coverage read-model, origin fallback
  - ADR-BE-018 (backend): status/derived/claim-spectrum/audit/settings vocabularies, claim lock, supersede-of-claim-row pin, est_hours scale, settings CRUD, 8 assumption pins
affects: [13-03 domain constants, 13-04 orgsettings vertical, 13-05 audit rows, 13-06 coverage read-model, 13-07 direction service, 13-08 fallback, Phase 19 surfaces, Phase 14 availability]

# Actuals (#2632) — pairs with the plan's estimate (14000 est / 8000 raw)
actuals:
  tokens: 8150    # chars/4 over the realized diff (32550 chars of new ADR files + index edits)
  tasks: 2        # tasks completed
  commits: 3      # commits made

# Tech tracking
tech-stack:
  added: []       # docs-only plan — no dependencies added
  patterns:
    - "Project ADR drafted from the Part 14/15 record of truth (D-R..D-AA) with D-13-01..34 encoded (P-012 template)"
    - "BE encoding ADR from the ADR-BE-017 template: Context / Decisions / Schema encoding / Security / Consequences"
    - "Assumption pins recorded as decisions with alternatives documented (A1/A2/A4-A10)"

key-files:
  created:
    - hourglass-vault/decisions/project/ADR-P-015 — Direction, The Plan Plane.md
    - hourglass-vault/decisions/backend/ADR-BE-018 — Direction & Org Settings Encoding.md
  modified:
    - hourglass-vault/decisions/project/_index.md
    - hourglass-vault/decisions/backend/_index.md

key-decisions:
  - "ADR-P-015 records the three assumption-delta decisions: identity no-change (row id is the noun, no UNIQUE on day-sharing), origin fallback add-alongside (stored refs authoritative, never written back), planning policy promoted to first-class data (org_settings key/value + membership override)"
  - "ADR-BE-018 pins supersede-of-claim-row semantics as a cross-plan contract: superseding a claim row inherits origin_direction_id and MUST stay user-targeted (WG-shaped superseding row -> ErrInvalidTarget); cancel releases hours"
  - "Claim over-subscription closure pinned: FOR UPDATE on WG row, in-tx Σ in cents over status IN ('draft','active') claim rows, 409 ErrClaimOverBudget, uncapped when est_hours NULL"
  - "Wikilink filename consistency: ADR-P-015 uses the plan-declared comma filename 'Direction, The Plan Plane' — aligned index + BE-018 links to it"

requirements-completed: [DIR-01, DIR-02, DIR-03, DIR-04, DIR-05, DIR-06]

# Coverage metadata (#1602) — one entry per shipped deliverable
coverage:
  - id: D1
    description: "ADR-P-015 — Direction, The Plan Plane (project ADR): direction plane, derived mode, per-day rows, supersede chain, derived states, claim model with Σ-consumption, org policy, warning overlay, coverage read-model, origin fallback + assumption-delta decisions"
    requirement: DIR-01
    verification:
      - kind: other
        ref: "ls hourglass-vault/decisions/project/ADR-P-015 — Direction, The Plan Plane.md + section/decision coverage grep"
        status: pass
    human_judgment: false
  - id: D2
    description: "ADR-BE-018 — Direction & Org Settings Encoding (backend ADR): status/derived/claim-spectrum/audit/settings vocabularies, claim lock, supersede-of-claim-row pin, est_hours DECIMAL(8,2), settings CRUD, 8 assumption pins"
    requirement: DIR-02
    verification:
      - kind: other
        ref: "ls hourglass-vault/decisions/backend/ADR-BE-018 — Direction & Org Settings Encoding.md + vocabulary grep matrix (status/audit/keys/modes/claim spectrum all present)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Vault index entries for both new ADRs following the existing per-folder entry format"
    verification:
      - kind: other
        ref: "grep -c 'ADR-BE-018' hourglass-vault/decisions/backend/_index.md (1) + grep -c 'ADR-P-015' hourglass-vault/decisions/project/_index.md (1)"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-08-08
status: complete
---

# Phase 13 Plan 02: Direction & Org Settings ADRs Summary

**ADR-P-015 (direction plane) + ADR-BE-018 (schema/audit/settings encoding) drafted into the vault from the Part 14/15 record of truth, with all D-13-01..34 decisions, A1-A10 assumptions, and three assumption-delta decisions recorded; both ADRs indexed in the per-folder vault indexes.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-08-08T12:21:01Z
- **Completed:** 2026-08-08T12:24:48Z
- **Tasks:** 2 completed
- **Files modified:** 4 (2 created, 2 index files)

## Accomplishments

- **ADR-P-015 — Direction, The Plan Plane** (project ADR, Status: Proposed): the direction plane as the third plane of direction → facts → coverage (P-012 convention); one direction entity with mode derived from `planned_date` (D-R); per-day rows always, partial days first-class, multiple rows may share a day, no intra-day ordering (D-W, D-AA); est_hours semantics per mode (required on scheduled, optional queued budget) with hard per-row validation and soft per-day warnings (D-13-02/03); immutable rows + supersede-only writes (D-13-04/08); lifecycle draft → active → superseded/cancelled with derived done/lapsed/claimed, never stored, no nightly jobs (D-V, D-13-07/09); WG-direction queued-only + hours-based split claims with Σ-consumption (D-T, D-13-11..16); org policy org-configurable stored not enforced (D-X, D-13-18..21); scheduler warning overlay from P-008 windows + employment validity, never blocks (D-Y, D-13-28..31); direction-coverage read-model (D-Z, D-13-24..27); origin fallback from first direction record (R4, D-13-32..34). Cross-references ADR-P-008, ADR-P-012, ADR-P-013.
- **Three assumption-delta decisions recorded in ADR-P-015's Decision section**: identity no-change (row id is the noun; day-sharing is a grouping convention, no UNIQUE constraint); origin fallback add-alongside (stored refs authoritative, drift mitigated by never writing back + origin_type discriminator); planning policy promoted to first-class data (org_settings key/value + membership planning_mode override).
- **ADR-BE-018 — Direction & Org Settings Encoding** (backend ADR, Status: Proposed) from the ADR-BE-017 template (Context; Decisions; Schema encoding; Security; Consequences), pinning exactly: (1) status vocabulary draft/active/superseded/cancelled with the matrix (draft→active, draft|active→cancelled, superseded ONLY via create-with-supersedes_id — no transition endpoint); (2) derived-state vocabulary done (terminal-activity CTE re-anchored at activities.id) / lapsed (past planned_date/due_date AND no non-deleted entries on subtree — any status, OQ2/A3) / claimed spectrum not_claimed/partially_claimed/fully_claimed (fully only when budget set and Σ == budget); (3) audit vocabulary entity_type='direction' actions created/activated/cancelled/superseded/claimed/unclaimed + entity_type='org_settings' action settings-updated with {key, before, after}, entity_id = org id (017:18 UUID NOT NULL); (4) settings key vocabulary planning_daily_hours (default 8.0)/planning_deadline/planning_horizon (day|week|month, stored not enforced)/planning_mode (manager_planned|self_planned, org default + membership override); (5) claim over-subscription closure (FOR UPDATE on WG row, in-tx Σ in cents over draft|active claim rows, 409 ErrClaimOverBudget, uncapped when est_hours NULL) + supersede-of-claim-row semantics (origin_direction_id inherited, user-targeted only — WG-shaped superseding row → ErrInvalidTarget; cancel releases hours); (6) est_hours DECIMAL(8,2) mirroring time_entries.hours (A2); (7) settings CRUD GET/PUT /organizations/settings literal routes, JWT-resolved org, manager+ gate, unknown key → 400; (8) assumption pins A1/A2/A4/A5/A6/A8/A9/A10 as recorded decisions with alternatives.
- **Vault index updated**: ADR-P-015 entry in `decisions/project/_index.md`, ADR-BE-018 entry in `decisions/backend/_index.md`, following the existing per-folder entry format.

## Task Commits

Each task was committed atomically:

1. **Task 1: Draft ADR-P-015 (Direction — The Plan Plane)** - `99a0313` (docs)
2. **Task 2: Draft ADR-BE-018 (Direction & Org Settings Encoding) + vault index update** - `2cb8d35` (docs)
3. **Task 2 follow-up: align ADR-P-015 wikilinks with comma filename** - `f67b0a5` (docs)

**Plan metadata:** `pending` (committed after SUMMARY)

## Files Created/Modified

- `hourglass-vault/decisions/project/ADR-P-015 — Direction, The Plan Plane.md` - Project ADR: the direction plane (one entity, derived mode, per-day rows, supersede chain, derived states, WG claim model, org policy, warnings, coverage read-model, origin fallback) + three assumption-delta decisions
- `hourglass-vault/decisions/backend/ADR-BE-018 — Direction & Org Settings Encoding.md` - BE encoding ADR: status/derived/claim-spectrum/audit/settings vocabularies, claim lock, supersede-of-claim-row pin, est_hours scale, settings CRUD, 8 assumption pins
- `hourglass-vault/decisions/project/_index.md` - Added ADR-P-015 row to the project ADR table
- `hourglass-vault/decisions/backend/_index.md` - Added ADR-BE-018 row to the backend ADR table

## Decisions Made

- **ADR-P-015 uses the plan-declared comma filename** ("Direction, The Plan Plane") — the P-012/P-013 pre-existing wikilinks use the colon form ("Direction: The Plan Plane"); aligned the new index entry + BE-018 links to the comma filename so the vault's wikilinks resolve to the created file.
- **Vault index path adaptation**: the plan's `hourglass-vault/decisions/_index.md` doesn't exist — the vault uses per-folder indexes; entries went to `decisions/project/_index.md` (P-015) and `decisions/backend/_index.md` (BE-018), preserving the plan's intent (both ADRs indexed following the existing entry format).
- **Three assumption-delta decisions** (identity no-change / fallback add-alongside / policy promoted) recorded in ADR-P-015's Decision section per the plan's must_haves, and operationalized in ADR-BE-018 §9.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Vault index path doesn't exist — per-folder indexes used instead**
- **Found during:** Task 2 (vault index update)
- **Issue:** The plan's `<verify>` and `files_modified` reference `hourglass-vault/decisions/_index.md`, but the vault has no single index file — it uses per-folder indexes (`decisions/project/_index.md`, `decisions/backend/_index.md`) per the established entry format the plan itself references.
- **Fix:** Added ADR-P-015 to the project index and ADR-BE-018 to the backend index, following the existing table format (wikilink | decision | pillar/resolves | status).
- **Files modified:** hourglass-vault/decisions/project/_index.md, hourglass-vault/decisions/backend/_index.md
- **Verification:** `grep -c "ADR-P-015" project/_index.md` → 1; `grep -c "ADR-BE-018" backend/_index.md` → 1 (both acceptance criteria satisfied on the real index files).
- **Committed in:** 2cb8d35

---

**Total deviations:** 1 auto-fixed (1 blocking path adaptation)
**Impact on plan:** Minimal — plan's index intent fully preserved; only the target path adapted to the vault's actual structure. No scope creep.

## Issues Encountered

None - plan executed as documented.

## User Setup Required

None - no external service configuration required.

## Self-Check: PASSED

- [x] `hourglass-vault/decisions/project/ADR-P-015 — Direction, The Plan Plane.md` exists (verified `ls`)
- [x] `hourglass-vault/decisions/backend/ADR-BE-018 — Direction & Org Settings Encoding.md` exists (verified `ls`)
- [x] Task 1 acceptance: Status/Context/Decision/Consequences sections present; Decision covers derived mode, per-day multiplicity, supersede chain, derived states, claim model with Σ-consumption, org policy storage-not-enforcement, warning overlay, coverage read-model, origin fallback; three assumption-delta decisions recorded with one-line rationales (grep-verified)
- [x] Task 2 acceptance: exact vocabularies present (status values, audit actions + settings-updated, keys, modes manager_planned/self_planned); claim lock (FOR UPDATE + in-tx Σ in cents over draft|active claim rows) + supersede-of-claim-row pin recorded; 8 assumption pins (A1/A2/A4/A5/A6/A8/A9/A10) present (grep-verified)
- [x] Index files contain lines referencing both ADR-P-015 and ADR-BE-018 (grep-verified)
- [x] Commits exist: `99a0313`, `2cb8d35`, `f67b0a5` (git log-verified)
- [x] Plan-level verification: both ADRs self-consistent with 13-CONTEXT.md D-13-01..34; ADR-BE-018 vocabularies match the 13-03 domain constants plan (grep cross-checked status/derived/claim-spectrum/audit/settings/mode/horizon vocabularies)

## Next Phase Readiness

- Ready for plan 13-03 (domain constants): ADR-BE-018's vocabulary pins are the contract the exported Go constants compile against — status, derived states, claim spectrum, warning types, audit entity/action names, settings keys, modes, horizons all pinned verbatim.
- Plan 13-05 (audit inserts) compiles against the pinned audit vocabulary (entity_type='direction' + 6 actions, entity_type='org_settings' + settings-updated with {key, before, after}).
- Plan 13-04 (orgsettings vertical) builds on the settings key vocabulary + CRUD shape pinned in ADR-BE-018 §4/§7.
- No blockers.

---
*Phase: 13-direction-backend-the-plan-plane*
*Completed: 2026-08-08*
