---
phase: 15-ux-foundation-design-tokens-shared-components
plan: 03
subsystem: ui
tags: [react, alert-dialog, tailwind, vitest, zustand, react-query, design-tokens]

# Dependency graph
requires:
  - phase: 15-ux-foundation-design-tokens-shared-components
    provides: UI-SPEC ConfirmDialog contract (D-15-07), PATTERNS confirm-dialog
      assignment, ui/alert-dialog + ui/textarea primitives, D-13-10/D-13-16
      server-reason invariant (backend)
provides:
  - Frozen ConfirmDialog (presentational destructive confirmation) with the
    required-reason gate mirroring the server 400 invariant
  - Three consumer pages (customers, org-hierarchy, working-groups) rendering
    ConfirmDialog with their existing zustand/props state wiring, three
    per-page delete dialogs deleted
affects: [Phase 16, absence-reject, claim-unclaim, future destructive actions]

# Actuals (#2632) — pairs with plan estimate (9000 tokens)
actuals:
  tokens: 7611
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: [lucide Loader2Icon spinner pattern in shared component]
  patterns:
    - Presentational shared dialog: controlled props (open/onOpenChange/onConfirm)
      with zero store/API/toast imports; sites own all wiring
    - Client-side required-reason gate mirroring the server 400 invariant
      (D-13-10/D-13-16): disabled confirm until non-empty, 'A reason is
      required.' on touched-and-empty
    - Structural null-guard at call sites: open/description/onConfirm all gated
      on the selected entity so no unguarded dereference survives tsconfig
      strict:true (mirrors old `if (!entity) return null` semantics)

key-files:
  created:
    - web/src/components/shared/confirm-dialog.tsx
    - web/src/components/shared/__tests__/confirm-dialog.test.tsx
  modified:
    - web/src/routes/_authenticated/customers/-components/customers-page.tsx
    - web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx
    - web/src/routes/_authenticated/working-groups/-components/working-groups-page.tsx
  deleted:
    - web/src/routes/_authenticated/customers/-components/dialogs/delete-confirm-dialog.tsx
    - web/src/routes/_authenticated/org-hierarchy/-components/dialogs/delete-confirm-dialog.tsx
    - web/src/routes/_authenticated/working-groups/-components/delete-working-group-dialog.tsx

key-decisions:
  - "Frozen copy strings are component-level constants: 'A reason is required.', 'Could not complete the action. Try again.', 'Cancel' — verbatim UI-SPEC strings"
  - "error prop semantics: error !== undefined shows the inline block; empty/absent-content falls back to the default copy, a non-empty value wins"
  - "org-hierarchy invalidateQueries marked void: TanStack v5 returns a Promise; the moved fire-and-forget cache sync triggers no-floating-promises (lint's own suggested fix)"

patterns-established:
  - "ConfirmDialog freezes the destructive confirmation pattern (D-15-07): every future destructive action renders ConfirmDialog with its requiredReason/error/isSubmitting props wired by the owning site"

requirements-completed: [UXFD-01]

coverage:
  - id: D1
    description: "ConfirmDialog component with required-reason gate — presentational, controlled (open/onOpenChange/title/description/confirmLabel/variant/requiredReason/reasonLabel/reasonPlaceholder/isSubmitting/error/onConfirm(reason?))"
    requirement: UXFD-01
    verification:
      - kind: unit
        ref: "web/src/components/shared/__tests__/confirm-dialog.test.tsx#gates the confirm button on a non-empty reason when requiredReason is set"
        status: pass
      - kind: unit
        ref: "web/src/components/shared/__tests__/confirm-dialog.test.tsx#passes the entered reason to onConfirm only when requiredReason is set"
        status: pass
      - kind: unit
        ref: "web/src/components/shared/__tests__/confirm-dialog.test.tsx#disables the confirm button and shows a spinner while submitting"
        status: pass
      - kind: unit
        ref: "web/src/components/shared/__tests__/confirm-dialog.test.tsx#renders inline error copy, defaulting to the UI-SPEC message"
        status: pass
      - kind: unit
        ref: "web/src/components/shared/__tests__/confirm-dialog.test.tsx#clears the local reason state when the dialog closes"
        status: pass
      - kind: integration
        ref: "bun run build && bun run lint && bun run typecheck"
        status: pass
    human_judgment: false
  - id: D2
    description: "Customers + org-hierarchy + working-groups delete flows render ConfirmDialog with existing store/props wiring; three per-page dialogs deleted; reparent dialog untouched"
    requirement: UXFD-01
    verification:
      - kind: integration
        ref: "bun run build && bun run lint"
        status: pass
      - kind: other
        ref: "file-absence greps: delete-confirm-dialog.tsx x2 + delete-working-group-dialog.tsx absent; git diff reparent-confirm-dialog.tsx empty"
        status: pass
      - kind: other
        ref: "behavior greps: 'Cannot delete customer with linked contracts', 'queryKey: [\"units\"]', 'Failed to delete working group' preserved at sites"
        status: pass
    human_judgment: false
  - id: D3
    description: "ConfirmDialog content never overflows the alert-dialog viewport — long reasons/descriptions scroll or clip within dialog bounds (UI-SPEC backstop row)"
    requirement: UXFD-01
    verification: []
    human_judgment: true
    rationale: "Layout-overflow behavior under long content is a visual backstop that unit tests cannot assert in jsdom; it is covered by the phase-gate human visual check planned for plan 04 Task 2."

duration: 25min
completed: 2026-08-17
status: complete
---

# Phase 15 Plan 3: ConfirmDialog Consolidation Summary

**Frozen presentational ConfirmDialog with the required-reason gate (D-15-07), absorbing all three per-page delete dialogs (customers, org-hierarchy, working-groups) with their state wiring preserved at the sites**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-17T09:35:16Z
- **Completed:** 2026-08-17T12:00:00Z
- **Tasks:** 3
- **Files modified:** 8 (2 created, 3 modified, 3 deleted)

## Accomplishments
- `ConfirmDialog` landed in `web/src/components/shared/` — controlled, strictly presentational (no store/query/mutation/sonner imports — prohibition grep clean), composing `ui/alert-dialog` + `ui/textarea` + `ui/label` with the frozen copy strings
- Required-reason gate mirrors the server 400 invariant (D-13-10/D-13-16): textarea + label render only when `requiredReason`, confirm disabled until non-empty, "A reason is required." on touched-and-empty, local reason state resets on close
- `isSubmitting` disables the confirm button and shows a `Loader2Icon` spinner; `error` renders inline muted-danger text defaulting to "Could not complete the action. Try again." with custom-message override
- All three delete flows consolidated: customers-page and org-hierarchy-page render ConfirmDialog from their zustand stores (409 toast / units invalidate preserved), working-groups-page from its wg/onClose state pair — three per-page dialogs deleted, `reparent-confirm-dialog.tsx` untouched (D-15-12)
- TDD discipline: RED test commit (`test(15-03)`) precedes GREEN implementation (`feat(15-03)`)

## Task Commits

Each task was committed atomically:

1. **Task 1: ConfirmDialog component with required-reason gate** - `861e14c` (test, RED) + `6792643` (feat, GREEN)
2. **Task 2: Absorb customers + org-hierarchy delete dialogs** - `80fcafc` (refactor)
3. **Task 3: Absorb working-groups delete dialog** - `013c520` (refactor)
4. **Formatting pass (my files only)** - `28e7c31` (style)

**Plan metadata:** `*pending*` (docs: complete plan)

_Note: Task 1 is a TDD task with multiple commits (test → feat)._

## Files Created/Modified
- `web/src/components/shared/confirm-dialog.tsx` - Frozen presentational ConfirmDialog (D-15-07); props open/onOpenChange/title/description?/confirmLabel/variant?/requiredReason?/reasonLabel?/reasonPlaceholder?/isSubmitting?/error?/onConfirm(reason?); freezes the three UI-SPEC copy strings as exported constants
- `web/src/components/shared/__tests__/confirm-dialog.test.tsx` - 5-test Vitest suite covering the required-reason gate, reason pass-through, isSubmitting spinner/disabled, error copy (default + custom override), and reason-state reset on close
- `web/src/routes/_authenticated/customers/-components/customers-page.tsx` - Renders ConfirmDialog with `deleteOpen && !!selectedCustomer` structural gate; deleteCustomer mutation + 409 toast moved into the page
- `web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx` - Renders ConfirmDialog in OrgHierarchyDialogs with the deleteUnit mutation (success toast + `['units']` invalidate + failure toast), structural `!!selectedUnit` gate
- `web/src/routes/_authenticated/working-groups/-components/working-groups-page.tsx` - Renders ConfirmDialog driven by the page's deleteWg state pair; `deleteWgMutation` named to avoid the duplicate identifier with the state pair
- Deleted: customers + org-hierarchy `delete-confirm-dialog.tsx`, working-groups `delete-working-group-dialog.tsx`

## Decisions Made
- Frozen copy strings are component-level constants (`REASON_REQUIRED_COPY`, `DEFAULT_ERROR_COPY`, "Cancel") matching the UI-SPEC Copywriting Contract verbatim
- `error` semantics: the inline block renders when `error !== undefined`; an empty/absent-content value falls back to the default copy, a non-empty custom message wins
- org-hierarchy `invalidateQueries({ queryKey: ["units"] })` marked with `void` — TanStack v5 returns a Promise and the moved fire-and-forget cache sync otherwise trips `no-floating-promises` (lint's own recommended fix)
- ConfirmDialog uses `variant="destructive"` on the confirm Button (cleaner than the raw class string the old dialogs used, per PATTERNS note)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `AlertDialogAction` spinner/portal quirks in unit tests**
- **Found during:** Task 1 (ConfirmDialog GREEN)
- **Issue:** The base-ui alert-dialog portal renders to `document.body`, so `container.querySelector("svg.animate-spin")` returned null; and `fireEvent.change` to the same value (`""` → `""`) does not fire React's controlled onChange, leaving the touched flag false
- **Fix:** Scoped the spinner assertion to the confirm button element (`confirmButton.querySelector`); reordered the touched-and-empty test to type-then-clear so every change event carries a different value
- **Files modified:** web/src/components/shared/__tests__/confirm-dialog.test.tsx
- **Verification:** 5/5 tests pass; `bun run build` + `bun run lint` + `bun run typecheck` green
- **Committed in:** 6792643 (part of Task 1 commit)

**2. [Rule 3 - Blocking] Harness `vi.fn()` typing failed `tsc -b` (TS2739/assignability)**
- **Found during:** Task 1 (ConfirmDialog GREEN verification)
- **Issue:** `Partial<ConfirmDialogProps>` intersected with `onConfirm?: ReturnType<typeof vi.fn>` produced an unassignable intersection, and title/confirmLabel became required when the props were omitted
- **Fix:** Reworked the Harness props to `Omit<..., "open" | "onOpenChange" | "onConfirm" | "title" | "confirmLabel">` with plain `(reason?: string) => void` + optional title/confirmLabel defaults declared in the destructure
- **Files modified:** web/src/components/shared/__tests__/confirm-dialog.test.tsx
- **Verification:** `bun run typecheck` passes
- **Committed in:** 6792643 (part of Task 1 commit)

**3. [Rule 1 - Bug] Floating `invalidateQueries` promise flagged in the moved org-hierarchy code**
- **Found during:** Task 2 (org-hierarchy consolidation)
- **Issue:** `queryClient.invalidateQueries(...)` in TanStack v5 returns a Promise; the line moved from the deleted dialog into the page surfaced as `no-floating-promises`
- **Fix:** Marked it `void` per the lint's own suggested remediation (fire-and-forget cache sync is intentional)
- **Files modified:** web/src/routes/_authenticated/org-hierarchy/-components/org-hierarchy-page.tsx
- **Verification:** `bun run lint` exit 0 with no mentions of the file
- **Committed in:** 80fcafc (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 bug)
**Impact on plan:** All auto-fixes were test/type/lint hygiene required for the frozen component and its moved code to land cleanly. No scope creep; the component contract and consumer wiring are exactly as planned.

## Issues Encountered
- `bun run fmt:check` fails phase-wide on ~29 pre-existing files (activities/auth/contracts tests, sidebar, etc.) — pre-existing debt, out of scope; my five touched files were formatted with oxfmt in a dedicated `style(15-03)` commit so this plan adds no formatting debt
- `graphify.watch._rebuild_code` is not importable in the default `python3`; the graphify pipx venv at `~/.local/pipx/venvs/graphifyy/bin/python` runs it successfully (AGENTS.md graphify step executed after each code-modifying task)

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The frozen ConfirmDialog is proven at three in-repo consumer sites — ready for every future destructive action (absence reject, claim unclaim, Phase 16-26) to render it with `requiredReason` where the backend enforces the D-13-10/D-13-16 400 invariant
- UI-SPEC backstop (long-content overflow inside the dialog viewport) awaits the phase-gate human visual check planned in plan 04 Task 2
- Phase wave gate green: `bun run test` 179/179 pass, `bun run lint` exit 0, `bun run build` + `bun run typecheck` pass

---
*Phase: 15-ux-foundation-design-tokens-shared-components*
*Completed: 2026-08-17*
## Self-Check: PASSED

- ConfirmDialog + test file exist; all 5 task commits verified in git log (861e14c test, 6792643 feat, 80fcafc refactor, 013c520 refactor, 28e7c31 style)
- Wave gate: full suite 179/179 pass, lint exit 0, build + typecheck green
