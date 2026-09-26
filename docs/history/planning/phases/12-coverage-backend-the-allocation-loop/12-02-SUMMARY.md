---
phase: 12-coverage-backend-the-allocation-loop
plan: 02
subsystem: adr
tags: ["adr", "coverage", "allocations", "schema", "snapshots", "vault"]

# Dependency graph
requires:
  - phase: 12-coverage-backend-the-allocation-loop (plan 01)
    provides: migrations 018-020 shapes + constraint names ADR-BE-017 documents
provides:
  - ADR-BE-017 (Accepted) — the backend encoding record for the coverage plane: schema 018-020, tagged-union source_type, derived balance, proposal function, replace-set write, manager gate, snapshots, audit vocabulary, D-K cost, OQ1-7 resolutions
  - ADR-P-012 status flipped Proposed → Accepted with a dated Acceptance section and Implemented-by link to ADR-BE-017
affects: [Phase 12 plans 03-07 (repo/service/handler executors consult the ADR), Phase 17 surfaces (allocation screen, to-cover queue, bucket balance, per-unit report)]

actuals:
  tokens: 6300    # chars/4 over realized diff: ADR-BE-017 file (22.4KB) + ADR-P-012 Acceptance section + index edits
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Vault ADR append-only flip: status cell + dated Acceptance section, no content rewrite"
    - "Encoding ADR mirrors ADR-BE-016 structure: Code header, Operationalizes/Basis/Extends/Gates-on, section-per-semantic, costs stated honestly"

key-files:
  created:
    - "hourglass-vault/decisions/backend/ADR-BE-017 — Coverage Encoding.md"
  modified:
    - "hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md"
    - hourglass-vault/decisions/backend/_index.md
    - hourglass-vault/decisions/project/_index.md

key-decisions:
  - "ADR-P-012 accepted 2026-08-07; D-1..D-6 operationalized via ADR-BE-017; snapshot-not-lock implemented as the frozen period-close snapshot (D-10/D-11/D-12)"
  - "ADR-BE-017 pins the zero-value predicate contract_type='project' AND sold_hours IS NOT DISTINCT FROM 0 (A3), raw bucket balance without period scaling (A8), duplicate close → 409 (A6), audit vocabulary entity_type='coverage_allocation' + actions allocations-set/coverage-closed (A7)"
  - "D-K polymorphic validation cost stated honestly: one service branch rejecting entry_type != 'time' + the entry_type CHECK; COV-06 needs an additive ALTER + service rule change, not a redesign"

requirements-completed: [COV-01, COV-02, COV-03, COV-04, COV-05]

coverage:
  - id: D1
    description: "ADR-P-012 flips Proposed → Accepted with a dated Acceptance section (2026-08-07) recording D-1..D-6 operationalization and the frozen-snapshot implementation of the snapshot-not-lock choice; Implemented-by header links ADR-BE-017"
    requirement: COV-01
    verification:
      - kind: other
        ref: "grep -c 'Accepted' ADR-P-012 (3) && grep -c 'ADR-BE-017' ADR-P-012 (3) && grep '**Status:** Accepted' ADR-P-012"
        status: pass
    human_judgment: false
  - id: D2
    description: "ADR-BE-017 exists and documents all ten sections: migrations 018-020 shapes, tagged-union source_type with 3VL guard, derived bucket balance, chain-driven proposal function + extension seam, replace-set with in-tx Σ under FOR UPDATE, manager gate via routing.ResolveManagerStage, no corrections, frozen period-close snapshots with 409 duplicate rejection, audit vocabulary, honest D-K cost, OQ1-7 resolutions"
    requirement: COV-02
    verification:
      - kind: other
        ref: "grep markers in ADR-BE-017: coverage_allocations(14) coverage_allocations_source_check(2) 'source_type IS NULL OR'(1) allocations-set(1) coverage-closed(1) 'IS NOT DISTINCT FROM 0'(3) 'FOR UPDATE'(4) ResolveManagerStage(2) financial_cutoff_periods(3) entry_type(6) 'entry_type != 'time''(1) 409(4) sold_hours(10)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Vault indexes consistent: backend _index.md gains the ADR-BE-017 row (Accepted), project _index.md ADR-P-012 status cell → Accepted; git diff shows only the intended row/cell changes"
    requirement: COV-03
    verification:
      - kind: other
        ref: "grep -c 'ADR-BE-017' backend/_index.md (1) && grep -c 'Accepted' project/_index.md (3); git diff --stat = 2 files, 2 insertions 1 deletion"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-08-08
status: complete
---

# Phase 12 Plan 02: Coverage Encoding ADRs Summary

**ADR-P-012 accepted (status flip + dated Acceptance section + Implemented-by link) and ADR-BE-017 drafted — the backend encoding record pinning every coverage semantic (tagged-union source_type with 3VL guard, derived bucket balance, chain-driven proposals, replace-set with in-tx Σ validation, manager gate, frozen snapshots) and every planner-resolved open question, with the D-K polymorphic validation cost stated honestly.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-08-08T09:01:01Z
- **Completed:** 2026-08-08T09:03:33Z
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- ADR-P-012 status flipped **Proposed → Accepted** with a dated (2026-08-07) Acceptance section recording that Phase 12 operationalizes D-1..D-6 via ADR-BE-017 and that the snapshot-not-lock choice (Q10 amendment) is implemented as the frozen period-close snapshot (D-10/D-11/D-12); `Implemented by:` header now resolves to `[[ADR-BE-017 — Coverage Encoding]]` instead of the ADR-BE-0xx placeholder
- **ADR-BE-017 — Coverage Encoding** created (Status: Accepted, Date: 2026-08-07) following ADR-BE-016's structure, with all ten required sections: migrations 018-020 shapes with exact constraint names (`coverage_allocations_source_check` 3VL guard, source_type/reason_vocab/entry_type vocabularies, mandatory-field CHECKs, snapshot tables with CASCADE), tagged-union `source_type` (3 row-level values → five derived funding sources), derived bucket balance (raw `sold_hours − Σ drawn`, overdraw allowed, no period scaling), chain-driven proposal decision function with the zero-value predicate `contract_type='project' AND sold_hours IS NOT DISTINCT FROM 0` and the ticket-kind extension seam, replace-set write (`PUT /time-entries/{id}/allocations`) with in-tx Σ validation under the FOR UPDATE entry-row lock and cents arithmetic, manager gate via `routing.ResolveManagerStage` (RoleGated branch requires the org manager claim; structural self-barrier; finance read-only), no correction handling, frozen period-close snapshots (`POST /coverage/close`, inclusive range, 409 duplicate rejection, `financial_cutoff_periods` stays facts-only, close returns snapshot rows), audit vocabulary (`entity_type='coverage_allocation'`, actions `allocations-set`/`coverage-closed`, in-tx synchronous), and the honest D-K cost (one service branch + the CHECK; COV-06 = additive ALTER + service rule change, not a redesign)
- OQ1-7 resolutions pinned in ADR-BE-017 §10 (3-value vocabulary, both proposal exposures, zero-value predicate, close returns rows, raw balance, 409 duplicate close, fully separate from `financial_cutoff_periods`) — no ambiguity left for Phase 17 to re-litigate (T-12-04 closed)
- Both vault indexes updated: backend `_index.md` gains the ADR-BE-017 row (Accepted, COV-01..05, D-01..D-12); project `_index.md` ADR-P-012 status cell → Accepted; no table reflow beyond the intended changes

## Task Commits

Each task was committed atomically:

1. **Task 1: Flip ADR-P-012 to Accepted + draft ADR-BE-017** - `7236d5a` (docs)
2. **Task 2: Update both vault indexes for ADR-BE-017 + ADR-P-012 status** - `4d1f938` (docs)

## Files Created/Modified

- `hourglass-vault/decisions/backend/ADR-BE-017 — Coverage Encoding.md` (created) - Accepted encoding ADR: schema 018-020, tagged-union source_type, derived balance, proposal function + seam, replace-set write, manager gate, snapshots, audit vocabulary, D-K cost, OQ1-7 resolution table
- `hourglass-vault/decisions/project/ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md` (modified) - Status → Accepted; header gains `**Accepted:** 2026-08-07`; `Implemented by:` → ADR-BE-017; append-only Acceptance section added
- `hourglass-vault/decisions/backend/_index.md` (modified) - one new ADR-BE-017 row (Accepted)
- `hourglass-vault/decisions/project/_index.md` (modified) - ADR-P-012 status cell Proposed → Accepted

## Decisions Made

- ADR-P-012 accepted per the vault append-only rule (status cell + Acceptance section; no content rewrite) — the T-12-05 mitigation disposition (accept) holds
- ADR-BE-017 pins the research-note resolutions the code plans rely on: zero-value predicate (A3), raw balance (A8), 409 duplicate close (A6), audit vocabulary (A7), both proposal read-path exposures (OQ2), fully separate close vs `financial_cutoff_periods` (OQ7)
- D-K cost documented honestly per ROADMAP: one service branch + one CHECK, with the COV-06 extension path (ALTER + service rule change, not redesign)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- ADR-P-012 and the 2026-08-01 research note were **untracked in git** (vault draft artifacts never committed). Task 1's commit `git add`'d ADR-P-012, so it now enters tracking with the status flip already applied (status flip + Acceptance section in one commit — the diff history for the file starts at Accepted). The research note remains untracked (not part of this plan's files).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 12-03..12-07 (domain/ports/service/handler/repo) can consult ADR-BE-017 as the record of truth for the semantics they encode
- Phase 17 surfaces (allocation screen, to-cover queue, bucket balance, per-unit report) read the pinned resolutions — no open questions left to re-litigate
- ADR-P-012 is now Accepted in both the file header and the project index

---
*Phase: 12-coverage-backend-the-allocation-loop*
*Completed: 2026-08-08*
