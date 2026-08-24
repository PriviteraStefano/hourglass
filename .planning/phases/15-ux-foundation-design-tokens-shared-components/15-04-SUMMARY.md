---
phase: 15-ux-foundation-design-tokens-shared-components
plan: 04
subsystem: ui
tags: [sketch-loop, uxfd-02, ui-spec, sidebar, dark-mode]

requires:
  - phase: 15-ux-foundation-design-tokens-shared-components
    provides: approved 15-UI-SPEC.md (D-15-10) + StatusBadge role tokens (plan 01)
provides:
  - Standalone sketch-loop contract at .planning/sketches/SKETCH-LOOP-CONTRACT.md (D-15-09 / D-15-11)
  - Presence verification that 15-UI-SPEC.md is status: approved (not regenerated)
affects: [16-integrity-repair, v0.3-presentation]

actuals:
  tokens: 900
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Downstream presentation work inherits sketch rules from SKETCH-LOOP-CONTRACT.md, not CONTEXT.md"
    - "2026-08-23 amendment: no applies-to phases 16-26; no global UI-only/no-backend ban; contracts/composition precede sketching"

key-files:
  created:
    - .planning/sketches/SKETCH-LOOP-CONTRACT.md
  modified: []

key-decisions:
  - "Contract pins seven rules: sketch-first, 2-3 variants / one winner, MANIFEST updates, UI-only follow-up plans, 3-round cap with wrap-up excluded, docs(sketch-NNN) commits, inheritance from this file"
  - "15-UI-SPEC.md status: approved verified; file not regenerated (D-15-10)"
  - "2026-08-17 adversarial review: UI-only contract stands. Missing product claims are either cut or funded as Phase 12 leftover — not by amending this contract"
  - "2026-08-23 lock: contract AMENDED in place. UI-only ban removed. applies-to phases 16-26 removed. Sketch loop waits for v0.3 contracts + composition."

patterns-established:
  - "Pattern 1: process contract lives under .planning/sketches/ and is the source of truth for v0.3 presentation (amended 2026-08-23)"
  - "Pattern 2: wrap-up is not a sketch round (research A6 / D-15-11)"

requirements-completed: []

coverage:
  - id: D1
    description: "SKETCH-LOOP-CONTRACT.md exists with all seven pinned rules; 15-UI-SPEC.md status: approved verified without regeneration"
    requirement: UXFD-02
    verification:
      - kind: other
        ref: "git show 00a9cd6; test -f .planning/sketches/SKETCH-LOOP-CONTRACT.md; grep -E '^status: approved' 15-UI-SPEC.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "Human visual check: sidebar collapsed-mode hover/navigation + dark-mode StatusBadge smoke (ROADMAP SC4)"
    requirement: UXFD-02
    verification:
      - kind: checkpoint
        ref: "user approved 2026-08-24; static preconditions: 54f465a pointer-events-none present (ui/sidebar.tsx:415); index.css :root(80-88)+.dark(125-133) status tokens; 51/51 vitest pass; tsc clean"
        status: pass
    human_judgment: true
    rationale: "Color/token rendering and collapsed-sidebar hit targets are visual; jsdom cannot verify oklch/.dark swap or pointer-events dead zones. User ran the live check and approved 2026-08-24."

duration: 2026-08-17 → 2026-08-24
completed: 2026-08-24
status: complete
---

# Phase 15 Plan 04: Sketch Loop Contract + Human Visual Gate

**Sketch-loop contract committed (00a9cd6) then amended 2026-08-23; UI-SPEC presence verified; human sidebar/dark-mode check approved 2026-08-24**

## Performance

- **Duration:** pending (Task 1 only)
- **Started:** 2026-08-17
- **Completed:** —
- **Tasks:** 2 of 2 (Task 2 approved 2026-08-24)
- **Files modified:** 1 created (`.planning/sketches/SKETCH-LOOP-CONTRACT.md`)

## Accomplishments

- **Task 1 committed** as `00a9cd6` `docs(15-04): add sketch loop contract`. The contract pins the seven D-15-09 / D-15-11 rules: sketch-first, 2–3 variants with exactly one winner, MANIFEST updates, UI-only follow-up plans, 3-round cap with `--wrap-up` excluded, `docs(sketch-NNN): [winner] …` commits, inheritance from this file (not CONTEXT.md).
- **UI-SPEC presence verified** without regeneration: `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` frontmatter is `status: approved` (D-15-10 / research A5).
- **`.planning/sketches/` created.** `MANIFEST.md` is intentionally absent — the first sketching phase creates it (plan artifact note).
- **Task 2 approved (2026-08-24).** User ran the live check: collapsed-sidebar icon-mode hover/navigation (dead-zone fix `54f465a` holds) and dark-mode StatusBadge rendering (semantic `--status-*` tokens swap correctly under `.dark`). Static preconditions verified earlier: `pointer-events-none` present at `ui/sidebar.tsx:415`; `index.css` defines all five status tokens under both `:root` (80–88) and `.dark` (125–133); `status-badge.tsx` resolves exclusively via semantic tokens; 51/51 vitest pass; `tsc` clean. UXFD-02 / ROADMAP SC4 closed.

## Task Commits

1. **Task 1: Write SKETCH-LOOP-CONTRACT.md + verify UI-SPEC presence** — `00a9cd6` (docs)
2. **Task 2: Human visual verification** — approved 2026-08-24 (user); static preconditions verified (54f465a sidebar fix; index.css :root/.dark status tokens; 51/51 vitest; tsc clean)

**Plan metadata:** written 2026-08-24 — Task 2 approved; close-out committed.

## Files Created/Modified

- `.planning/sketches/SKETCH-LOOP-CONTRACT.md` — standalone process contract (amended 2026-08-23: v0.3 after composition; no phases 16–26; no global UI-only ban)

## Decisions Made

- Followed the plan for Task 1. No regeneration of `15-UI-SPEC.md`.
- **Adversarial review 2026-08-17 (this session, not plan 15-04 work):** the UI-only rule stays. Product claims without an API are cut or funded as Phase 12 leftover. See ROADMAP Phase 16–19 rewrite and STATE “v0.2 surface lock”.
- **Milestone lock 2026-08-23 (not plan 15-04 execution):** the contract was amended in place. Former Phases 16–26 are deleted. Phase 16 is Integrity Repair (no sketch). Sketch loop applies in v0.3 after contracts + composition. UI-only/no-backend ban removed. Task 2 and Window #1 remain open.

## Deviations from Plan

None for Task 1. Plan 15-04 is **not complete** — Task 2 is still the blocking human checkpoint.

## Issues Encountered

- Planning ledger was stale when this plan’s commit landed: `STATE.md` still said “stopped after 15-02”; ROADMAP progress table still said Phase 15 `0/4 Not started`; no `15-04-SUMMARY.md` existed. Reconciled in the same session as this summary.
- Window #1 (`bun run fmt:check` / 31 pre-existing oxfmt-drift files) remains **open**. It is not caused by 15-04. `/gsd-ship` stays blocked until waived or fixed. Recommendation: waive as pre-existing (base `4189c00`) or run a dedicated format sweep — do not pretend Phase 15 formatted the repo.

## User Setup Required

None.

## Next Phase Readiness

**Not ready to close Phase 15, and not ready to start Phase 16 Integrity Repair as if it were Availability Frontend.** Close this plan first:

1. ✅ Task 2 approved 2026-08-24 (user) — static preconditions + live visual confirmed.
2. Fix or explicitly waive Window #1. Do not mark it fixed here.
3. Flip this SUMMARY `status` to `complete`, mark D2 verification, tick UXFD-02, tick ROADMAP 15-04.
4. Then Phase 16 Integrity Repair (own-coverage read, two race smokes, expense `unit_id`, receipt auth, WR-05, rate-limiter bugs). **No sketches. No UI.**
5. Do not create v0.3 phases until v0.2 is archived.

---

*Phase: 15-ux-foundation-design-tokens-shared-components*
*Written: 2026-08-17 — Task 1 only; awaiting human*
*Amended: 2026-08-23 — contract lock; historical Task 1 evidence unchanged*
