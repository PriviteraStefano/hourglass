# Deferred Items — 260801-got (collapsed-sidebar pointer-events fix)

## Pre-existing frontend build breakage (out of scope, NOT caused by this plan)

**Found during:** Task 1 verification (`cd web && bun run build`)

`bun run build` (`tsc -b && vite build`) fails with 6 pre-existing TypeScript
errors and 1 rolldown missing-export error, **all in files unrelated to this
plan** (`web/src/components/ui/sidebar.tsx` is not among them). A/B verified:
reverting this plan's change reproduces the identical error set, so none of
these are regressions from the `pointer-events-none` edit.

This plan's change itself is verified correct via grep
(`pointer-events-none flex h-8` present, `-mt-8`/`opacity-0` untouched) and by
the A/B tsc comparison (zero new errors introduced).

**Errors (fix in a future plan):**

1. `src/lib/__tests__/api.test.ts(84,29)` — TS2339: `Property 'get' does not exist on type 'never'`
2. `src/routes/__root.tsx(16,7)` — TS2322: `attribute` does not exist on `ThemeProviderProps`
3. `src/routes/_auth/bootstrap/-components/bootstrap-form.tsx(57,34)` — TS2741: `org_name` missing in type, required in `BootstrapRequest`
4. `src/routes/_auth/invite/-components/invitation-accept-form.tsx(13,10)` — TS2724: `useSearchParams` is not exported by `@tanstack/react-router` (also fails `vite build` with `[MISSING_EXPORT]` — rolldown hard failure, not just type-check)
5. `src/routes/_auth/password-reset/-components/password-reset-request-form.tsx(62,17)` — TS2820: `"/password-reset/verify"` not assignable to route union
6. `src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx(197,57)` — TS2345: `string | undefined` not assignable to `string`

**Likely root causes:** dependency drift (`@tanstack/react-router` export
removed/changed, `next-themes` ThemeProviderProps changed) and stale types
(`BootstrapRequest.org_name` vs `organization_name`).

**Impact on this plan:** verification criterion "`bun run build` exits 0"
cannot be satisfied for any commit on this branch until the above are fixed —
this is a repo-wide pre-existing condition, not a regression. The
`pointer-events-none` change is complete and correct on its own.
