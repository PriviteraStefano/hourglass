# Deferred Items — 10-01 Foundation (Role type + activities API + route rename)

Out-of-scope discoveries logged during plan execution (scope boundary: do not
auto-fix issues not caused by the current task's changes).

## 1. Pre-existing typecheck rot (10 errors, unrelated files)

- **Discovered:** 2026-07-31, Task 1 typecheck verification
- **Symptom:** `cd web && bun run typecheck` reports 10 errors in files
  unrelated to the projects→activities rename. Baseline (git stash of this
  plan's changes) shows **87 typecheck errors**; this plan eliminates the
  ~77 rename-related + TanStack v5.101 `queryFn`/`mutationFn` test-rot errors,
  leaving these 10.
- **Errors (all pre-existing, none caused by 10-01):**
  - `src/lib/__tests__/api.test.ts(84)` — TS2339 `get` on type `never`
  - `src/routes/__root.tsx(16)` — TS2322 ThemeProviderProps `attribute`
  - `src/routes/_auth/bootstrap/-components/bootstrap-form.tsx(57)` — TS2741
    `org_name` missing (BootstrapRequest shape drift)
  - `src/routes/_auth/invite/-components/invitation-accept-form.tsx(13)` —
    TS2724 `useSearchParams` removed from @tanstack/react-router v1.170
  - `src/routes/_auth/password-reset/-components/password-reset-request-form.tsx(45)` —
    TS2345 navigate argument mismatch
  - `src/routes/_authenticated/contracts/$id/-components/contract-detail.tsx(220,280)` —
    TS2322 string|null vs string|undefined; TS2339 `Customer.name`
  - `src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx(101,151)` —
    TS2345 navigate options; TS2322 Select onChange signature
  - `src/routes/_authenticated/org-hierarchy/-components/dialogs/unit-detail-panel.tsx(197)` —
    TS2345 string|undefined vs string
- **Impact:** The plan's `typecheck exits 0` acceptance criterion is
  technically unattainable at baseline; this plan's changes introduce zero new
  errors (all project/activity references typecheck clean). The 10 remaining
  errors touch files owned by later Phase 10 plans (10-03 contract wrap) and
  out-of-phase surfaces — fixing them here would collide with those plans.
