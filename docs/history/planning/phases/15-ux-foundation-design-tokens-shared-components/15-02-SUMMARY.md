---
phase: 15-ux-foundation-design-tokens-shared-components
plan: 02
subsystem: ui
tags: [data-table, filter-bar, tanstack-table, react-table-v9, vitest, react, typescript]

# Dependency graph
requires:
  - phase: 13-ui-spec
    provides: frozen DataTable/FilterBar API contracts + a11y label strings (15-UI-SPEC lines 159-179)
  - phase: 15-ux-foundation-design-tokens-shared-components
    provides: EmptyState wrapper (plan 01) consumed by the DataTable empty state
provides:
  - Generic sortable, client-paginated DataTable<T> on @tanstack/react-table v9 with the UI-SPEC a11y suite (sort/aria-sort labels, labeled pager buttons, page-size select)
  - Stateless FilterBar (search input + select/date-range filters + active-count chip + conditional Reset) fully wired via consumer-owned values/onChange
  - @tanstack/react-table@^9.1.2 dependency with an exported DataTableFeatures type so consumers type ColumnDefs against the registered feature set
affects: [16-tickets-ui, 18-today, 19-scheduler-ui, 20-approvals-polish, 21-time-entries-migration, phase-verification]

# Actuals (#2632) — pairs with the plan's estimate (10000 tokens, low confidence)
actuals:
  tokens: 604      # chars/4 over the realized diff (git diff numstat *3 added, excluding the mechanical bun.lock delta)
  tasks: 3
  commits: 6       # 2 RED + 2 GREEN + 1 test-align + 1 style

# Tech tracking
tech-stack:
  added:
    - "@tanstack/react-table@^9.1.2 (v9 major — user override of the plan's ^8.21.3 pin)"
  patterns:
    - "v9 feature registration: tableFeatures({ rowSortingFeature, rowPaginationFeature, sortedRowModel, paginatedRowModel, sortFns }) declared statically, exported as DataTableFeatures for column typing"
    - "v9 row-model pipeline: core (automatic) → sortedRowModel → paginatedRowModel feature slots; pagination auto-resets page index on data change (page-clamp preserved)"
    - "v9 accessor discrimination via cell.column.accessorFn (resolved) — the auto-injected default cell renderer hides missing-value cases"
    - "Base-UI Select items activate on the pointer sequence (pointerdown/pointerup/click) in jsdom tests — never bare fireEvent.click on options"

key-files:
  created:
    - web/src/components/shared/data-table.tsx
    - web/src/components/shared/filter-bar.tsx
    - web/src/components/shared/__tests__/data-table.test.tsx
    - web/src/components/shared/__tests__/filter-bar.test.tsx
  modified:
    - web/package.json
    - web/bun.lock

key-decisions:
  - "User-override of the plan's version pin: @tanstack/react-table is installed at ^9.1.2 (the 2026-08-09 v9 major), NOT ^8.21.3 — the v9 line is the intended API going forward (approved at the Task 1 package gate); the DataTable implementation follows the installed v9 API verbatim"
  - "DataTable uses the v9 surface only: useTable(options) instead of useReactTable, row models as tableFeatures({...}) slots instead of get*RowModel options, and the table.FlexRender component for header/cell markup — no v8 APIs (the plan's prohibition grep for v9 leakage is inverted by the version override)"
  - "Em-dash contract discriminated by cell.column.accessorFn (resolved accessor presence), not the cell-renderer type — v9 auto-injects a default cell function on accessor columns that returned empty for undefined values"
  - "Header-cell a11y: the sort button owns the aria-label contract ('Sort by {label}' → 'Sorted by {label} ({direction})'); the columnheader's accessible name stays the visible header text with aria-sort on the th — matches how testing-library computes names"
  - "FilterBar select triggers render the matched option LABEL via SelectValue children — base-ui renders the raw value string by default; the frozen contract shows human labels ('Draft', not 'draft')"
  - "FilterBar date-range placeholder shows the filter's own label (e.g. 'Period') instead of the generic 'Date range' from entries-filters — filters are self-describing in the frozen design"
  - "Base-UI Select items require the pointer sequence (pointerDown/pointerUp/click) to activate in jsdom — tests mirror the real interaction instead of bare click"

patterns-established:
  - "Pattern 1: v9 feature set as a statically-defined exported constant (features/DataTableFeatures) — consumers type columns as ColumnDef<DataTableFeatures, T>[], keeping per-column option keys pinned to the registered registries"
  - "Pattern 2: frozen list-surface contract held entirely in tests — sort a11y, pagination math, skeleton/empty/em-dash states asserted via roles and DOM facts, not component internals"
  - "Pattern 3: stateless filter surface with consumer-owned values — the only component state anywhere in the frozen set is the date popover open flag"

requirements-completed: [UXFD-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "DataTable<T> — sortable client-paginated table on @tanstack/react-table v9 with the UI-SPEC a11y suite (sort toggle asc→desc→none with aria-label flip + aria-sort, 'Showing {first}–{last} of {total}' footer, labeled prev/next buttons, 10/20/50 page-size select), 5-row skeleton loading state, EmptyState defaults overridable via empty prop, em-dash for missing accessor values, truncate + title cells, onRowClick row affordance"
    requirement: UXFD-01
    verification:
      - kind: unit
        ref: "web/src/components/shared/__tests__/data-table.test.tsx (6 tests)"
        status: pass
      - kind: automated_ui
        ref: "bun run typecheck + bun run build exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "FilterBar — stateless search + select/date-range filters with the '{N} active' chip (search excluded) and a conditional Reset ghost button; select filters show matched option labels; date-range filters open a 2-month range calendar with 'dd MMM' labels and a Clear control"
    requirement: UXFD-01
    verification:
      - kind: unit
        ref: "web/src/components/shared/__tests__/filter-bar.test.tsx (6 tests)"
        status: pass
      - kind: automated_ui
        ref: "bun run typecheck + bun run build exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "Dependency install @tanstack/react-table@^9.1.2 (v9, human-approved at the Task 1 blocking gate) with bun.lock updated; entries-table.tsx / entries-filters.tsx untouched (D-15-12); no data-lifecycle imports in either frozen component"
    requirement: UXFD-01
    verification:
      - kind: other
        ref: "grep '^9.1.2' web/package.json; grep reuse of useQuery/zustand/useSearchParams in both components returns 0 hits; git diff empty for entries-* paths"
        status: pass
    human_judgment: false

# Metrics
duration: 31 min
completed: 2026-08-17
status: complete
---

# Phase 15 Plan 02: DataTable + FilterBar Summary

**Frozen list-surface components on @tanstack/react-table v9: sortable client-paginated DataTable with the pinned a11y suite (sort labels + aria-sort, showing-label footer, labeled pager, 10/20/50 page-size select) and a stateless FilterBar (search + select/date-range filters, '{N} active' chip, conditional Reset) — both with full state-coverage unit tests, delivered on the user-overridden v9 API**

## Performance

- **Duration:** 31 min (this continuation session — Tasks 2-3 + close-out; Task 1 gate ran in the prior session)
- **Started:** 2026-08-17T13:27:14Z
- **Completed:** 2026-08-17T13:58:29Z
- **Tasks:** 3 (Task 1 = human-approved package gate in the prior session; Tasks 2-3 here)
- **Files modified:** 6 (2 components + 2 test files created; package.json + bun.lock modified)

## Accomplishments

- **Dependency (human-approved at Task 1):** `@tanstack/react-table@^9.1.2` — the v9 major (published 2026-08-09), overriding the plan's ^8.21.3 v8 pin per the user's decision at the package gate. The DataTable implementation was built from the installed 9.1.2 type surface and the package's bundled migration skills (v9 renamed `useReactTable` → `useTable`, moved row models into `tableFeatures({...})` feature slots, removed `getState()`, and auto-injects a default cell renderer). No v8 APIs are present.
- **Task 2 — DataTable** (RED `f60d8a0` → GREEN `8e27843`): 6 unit tests driving the frozen contract — sort cycle asc→desc→none with the exact 'Sort by {label}' / 'Sorted by {label} ({direction})' aria-label flip and `aria-sort` on the active header cell (T-15-03); 'Showing 1–10 of 25' footer with labeled prev/next icon buttons and a 10/20/50 page-size select; 5 skeleton rows while loading; EmptyState defaults ('No matching records' / 'Try adjusting your search or filters.') overridable via `empty`; em-dash for missing accessor values; truncate-with-title cells; `onRowClick` affordance (cursor-pointer + hover bg). Component is presentational-only — no data-lifecycle imports; the v9 feature set (`rowSortingFeature` + `rowPaginationFeature` + sortFns registry) is exported as `DataTableFeatures` so consumers type `ColumnDef<DataTableFeatures, T>[]`.
- **Task 3 — FilterBar** (RED `d1d37bb` → GREEN `329c21b`): 6 unit tests — search input first (InputGroup + SearchIcon, `min-w-40 grow`) bound to `values.search`; '{N} active' chip derived from non-empty filter values EXCLUDING the search term; chip + Reset hidden at zero active; Reset fires `onReset`; select filters (ui/select single-select) render options and report `onChange(id, value)`; date-range filters open a 2-month `mode="range"` calendar via Popover with 'dd MMM' range labels and a Clear button that collapses the value to undefined. Fully stateless (D-15-08) — only local state is the date popover open flag.
- **TDD discipline:** both task pairs show `test(15-02)` before `feat(15-02)` in git history; RED failures were genuine (module-not-found at import for both components). One test was aligned mid-GREEN when the v9 rendering surface differed from the v8-shaped assumption (columnheader accessible name — see deviations).
- **Wave gate green:** full suite 22 files / 191 tests pass; `bun run lint` clean on the 4 plan-owned files (remaining repo warnings are pre-existing in untouched files); `bun run build` (tsc -b + vite) exit 0; `bunx oxfmt --check` passes on all 4 plan-owned files; graphify knowledge graph rebuilt (2715 nodes, 4232 edges).

## Task Commits

Each task was committed atomically with the RED-before-GREEN TDD gate:

1. **Task 1: Package gate (@tanstack/react-table legitimacy)** - approved by human in prior session (SUS verdict cleared; v9 selected)
2. **Task 2: DataTable** - RED `f60d8a0` (test) → GREEN `8e27843` (feat)
3. **Task 3: FilterBar** - RED `d1d37bb` (test) → GREEN `329c21b` (feat)

Supporting commits in this session:
- `cdb326c` (test) - aligned data-table tests with the v9-rendered a11y surface (columnheader name is the visible header text; em-dash via accessor discrimination)
- `7270050` (style) - oxfmt formatting of data-table.tsx

**Plan metadata:** `docs(15-02): complete DataTable + FilterBar plan` (this commit)

## Files Created/Modified

- `web/package.json` - `"@tanstack/react-table": "^9.1.2"` added (v9 — user override of the plan's ^8.21.3)
- `web/bun.lock` - lockfile updated with @tanstack/react-table 9.1.2 + @tanstack/table-core 9.1.2
- `web/src/components/shared/data-table.tsx` - `DataTable<T>` + `DataTableProps<T>` + exported `features`/`DataTableFeatures` (D-15-05)
- `web/src/components/shared/filter-bar.tsx` - `FilterBar` + exported `Filter`/`FilterValue` types (D-15-08)
- `web/src/components/shared/__tests__/data-table.test.tsx` - 6 tests: sort a11y cycle, pagination footer/select, skeleton + empty defaults, empty override, em-dash + truncation, row affordance + onRowClick
- `web/src/components/shared/__tests__/filter-bar.test.tsx` - 6 tests: search binding, active-count semantics, chip/reset visibility, reset handler, select options + change reporting, date-range labels + popover + clear

## Decisions Made

- **v9 API surface (user-override):** the plan assumed the v8 pin (`useReactTable`, `getCoreRowModel()`-style options, standalone `flexRender`); the installed 9.1.2 major requires `useTable`, explicit `tableFeatures({...})` registration, and the `table.FlexRender` component. This implementation is the v9 surface verbatim — verified against the installed package's own type declarations and migration skills rather than v8 memory.
- **Exported feature set for typing:** `DataTableFeatures = typeof features` lets consumers write `ColumnDef<DataTableFeatures, T>[]` — column-level `sortFn`/`filterFn` options resolve against the registered registries, and the v9 generics stay sound without exposing the internal feature object.
- **Em-dash via resolved accessor:** v9 auto-injects a default `cell` renderer on accessor columns, so checking the renderer type can't distinguish display columns from data columns; `cell.column.accessorFn` (resolved on accessor columns only) is the discriminator.
- **Header a11y split:** sort-button aria-label carries the full state contract; the `th` carries `aria-sort` and its accessible name remains the visible header text — the split matches how AT/testing-library derive names.
- **Select trigger shows labels:** base-ui's `Select.Value` renders the raw value by default; the frozen FilterBar renders the matched option label (falling back to the raw value) so snake_case statuses never leak into the trigger.
- **Pagination page-clamp preserved:** v9's `autoResetPageIndex` (default on for client pagination) resets to page 0 when data shrinks — matching the entries-table clamp behavior without custom logic.

## Deviations from Plan

### User-Override (explicit, human-approved)

**1. [User decision — version policy] @tanstack/react-table pinned at v9 (^9.1.2) instead of the plan's ^8.21.3 v8 pin**
- **Found during:** Task 1 package gate (prior session)
- **Decision:** the human reviewed the SUS verdict and chose the LATEST major (9.1.2, published 2026-08-09) as the intended dependency going forward — the plan's v8 pin is superseded.
- **Impact on implementation:** the DataTable was written entirely against the installed 9.1.2 API (`useTable` + `tableFeatures` slots + `table.FlexRender`); the plan's prohibitions were adjusted accordingly — the "no v9 API leak" grep (`useTable(`/`table.FlexRender`) is inverted to REQUIRE the v9 surface, while the data-lifecycle prohibition (`useQuery`/stores/`useSearchParams`) still holds and passes with 0 hits.
- **Files modified:** web/package.json, web/bun.lock, web/src/components/shared/data-table.tsx

### Auto-fixed Issues

**1. [Rule 1 - Bug] Em-dash never rendered for missing accessor values**
- **Found during:** Task 2 GREEN (test 5 failed: em-dash text missing from the rendered DOM)
- **Issue:** v9 auto-injects a default cell renderer (`props.renderValue()?.toString() ?? null`) on accessor columns, so the renderer-type check treated every column as a custom cell and skipped the em-dash branch — undefined values rendered as empty spans.
- **Fix:** discriminate on `cell.column.accessorFn` (present only on accessor columns); missing values on accessor columns render the frozen em-dash while display/derived columns keep their own markup.
- **Files modified:** web/src/components/shared/data-table.tsx
- **Verification:** data-table test 5 passes; typecheck clean.
- **Committed in:** 8e27843 (Task 2 GREEN)

**2. [Rule 1 - Bug] Sort-test queried the columnheader by the button's aria-label**
- **Found during:** Task 2 GREEN (test 1 failed: no columnheader named "Sort by Date")
- **Issue:** the `th`'s accessible name is the visible header text ("Date"), not the inner button's aria-label — the test's query matched nothing.
- **Fix:** query the columnheader by the visible header text and assert `aria-sort` on it; the button's aria-label assertions stay as the contract carrier.
- **Files modified:** web/src/components/shared/__tests__/data-table.test.tsx
- **Verification:** test 1 passes; the a11y contract strings are still asserted verbatim against the button.
- **Committed in:** cdb326c (test align)

**3. [Rule 1 - Bug] FilterBar select trigger displayed the raw value instead of the option label**
- **Found during:** Task 3 GREEN (test 5 failed: trigger text was "approved" not "Approved")
- **Issue:** base-ui's `Select.Value` renders the selected item's raw value by default.
- **Fix:** render the matched option label via `SelectValue` children (falling back to the raw value when unmatched).
- **Files modified:** web/src/components/shared/filter-bar.tsx
- **Verification:** test 5 passes; typecheck clean.
- **Committed in:** 329c21b (Task 3 GREEN)

**4. [Rule 3 - Blocking] base-ui Select items do not activate on bare fireEvent.click inside the FilterBar tree**
- **Found during:** Task 3 GREEN (test 5 failed: 0 onChange calls after clicking an option)
- **Issue:** inside the FilterBar's composed tree the base-ui option ignored a bare click event (the isolated probe passed, the composed tree did not).
- **Fix:** tests fire the pointer sequence (pointerDown + pointerUp + click) — the same interaction a physical click produces and the sequence base-ui's activation listens for.
- **Files modified:** web/src/components/shared/__tests__/filter-bar.test.tsx
- **Verification:** test 5 passes; the pointer sequence matches a real browser click.
- **Committed in:** 329c21b (Task 3 GREEN)

**5. [Rule 3 - Blocking] graphify module not importable from the default python3**
- **Found during:** plan verification (AGENTS.md graphify step after Task 2)
- **Issue:** `python3` (3.11, framework Python) lacks the `graphify` module; the tool is installed via pipx under Python 3.14.
- **Fix:** invoked the graphify rebuild through the pipx venv interpreter (`~/.local/pipx/venvs/graphifyy/bin/python`); graph rebuilt after both code commits (2715 nodes / 4232 edges).
- **Files modified:** none (toolchain invocation only)
- **Committed in:** n/a (no source change)

---

**Total deviations:** 5 handled (1 user decision + 3 Rule 1 bugs + 1 Rule 3 blocker) + 1 test alignment
**Impact on plan:** the version override is the plan's own gate outcome — it redirected the API surface but not the frozen component contracts (props, a11y strings, layout, state coverage are all per the UI-SPEC). All auto-fixes were required for the tests to express the frozen contract against the installed v9 renderer. No scope creep; no architectural changes beyond the approved version choice.

## TDD Gate Compliance

Both task pairs satisfy the RED→GREEN gate:

| Task | RED commit | GREEN commit | Status |
|------|-----------|-------------|--------|
| DataTable | `f60d8a0` test(15-02): add failing data-table tests | `8e27843` feat(15-02): implement DataTable | PASS |
| FilterBar | `d1d37bb` test(15-02): add failing filter-bar tests | `329c21b` feat(15-02): implement FilterBar | PASS |

Both RED commits failed for the right reason (module not found for the not-yet-existent component). No REFACTOR commits (no cleanup warranted — implementations were minimal against the real API).

## Issues Encountered

- **v9 API discovery was the main friction point:** 9.1.2 is 8 days old; the implementation was built from the installed package's type declarations and its bundled migration/sorting/pagination skills after the user approved the v9 line. The v8-shaped plan text was treated as intent, not API reference, and every call was verified against the installed types before use.
- **Ambient working-tree noise:** the Obsidian vault files (`hourglass-vault/.obsidian/workspace.json`, `SG-08-Setup-and-Usage.md`) remain modified by the running app — out of scope, left unstaged (same situation as 15-01).
- **Graphify invocation:** the AGENTS.md python3 one-liner fails on this machine (module installable only via the pipx venv); the rebuild was run through the pipx interpreter manually after each code commit.

## Known Stubs

None — DataTable renders real v9-driven rows/skeletons/empty states with no placeholder copy; FilterBar renders live controls bound to consumer values; no hardcoded empty values or TODO markers.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The two frozen list-surface components are ready for all Phase 16–26 consumers: DataTable (sorting/pagination/loading/empty/partial states) and FilterBar (search/select/date-range filters + active-count/reset). Consumers should type columns as `ColumnDef<DataTableFeatures, T>[]` and pass `initialSorting`/`onRowClick`/`empty` per the exported props contract.
- Phase 21's time-entries migration (D-15-12) can now replace entries-table/entries-filters with these components — both remain untouched in this phase per the plan.
- **Phase-gate note:** the pre-existing repo-wide `fmt:check` drift (31 files, logged in 15-01's deferred-items.md) is unchanged — the 4 plan-owned files are all fmt-clean.
- The v9 registration pattern (static `tableFeatures` + exported `DataTableFeatures`) is the convention later data surfaces should follow so column-level option keys stay type-checked against the registered features.

---

*Phase: 15-ux-foundation-design-tokens-shared-components*
*Completed: 2026-08-17*

## Self-Check: PASSED

- All 4 created source/test files + SUMMARY.md exist on disk (verified `[ -f ]`)
- All 6 plan commits present in git history (f60d8a0, 8e27843, d1d37bb, 329c21b, cdb326c, 7270050)
- Full test suite 22/22 files green (191 tests, incl. 12 new), lint clean on plan-owned files, build exit 0, typecheck clean, oxfmt --check clean on all 4 plan-owned files
- Wave-gate grep gates: 0 data-lifecycle imports in both components; entries-table/entries-filters untouched; package.json pins ^9.1.2