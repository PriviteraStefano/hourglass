# Deferred Items — Phase 04-contracts

## Pre-existing Build Issues (Discovered during 04-02 execution)

The following TypeScript build errors are pre-existing in the project and not caused by any Phase 4 changes. They are tracked here for resolution in a future phase.

### 1. `queryFn` / `mutationFn` possibly undefined in test files

**Files:** `auth.test.ts`, `contracts.test.ts`, `customers.test.ts`, `projects.test.ts`, `time-entries.test.ts`

**Issue:** TanStack Query v5 types mark `queryFn` and `mutationFn` as optional. Test files invoke them directly (e.g., `opts.queryFn()`, `mutationFn(data)`), causing TS2722 / TS18048 / TS2554 errors.

**Root cause:** Tests were written assuming `queryFn` / `mutationFn` would be set, but the types disagree.

### 2. `Select` `onValueChange` type mismatch

**Files:** `create-contract-dialog.tsx` (line 137), `create-project-dialog.tsx` (line 129), `edit-project-dialog.tsx` (line 111)

**Issue:** `<Select value={currency} onValueChange={setCurrency}>` passes `Dispatch<SetStateAction<string>>` but `onValueChange` expects `(value: string | null, eventDetails: SelectRootChangeEventDetails) => void`.

### 3. Route type mismatches

**Files:** `bootstrap-form.tsx`, `invitation-accept-form.tsx`, `login-form.tsx`, `password-reset-request-form.tsx`, `project-list.tsx`

**Issue:** Various route parameter type mismatches — navigate targets, search params, etc.

### 4. Environment / theme types

**Files:** `api.ts` (line 1, 4), `__root.tsx`

**Issue:** `ImportMeta` missing `env`, `ThemeProviderProps` missing `attribute` prop
