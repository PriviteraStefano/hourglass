---
phase: 05-projects
plan: 03
subsystem: frontend
tags: [projects, crud, edit, delete, subprojects]
requires: [05-02]
provides: [edit-project-dialog, delete-confirmation, subproject-section]
affects: [web/src/types/api.ts, web/src/types/models.ts, web/src/api/projects.ts, project-detail.tsx]
tech-stack:
  added: [AlertDialog, Accordion]
  patterns: [Dialog-based edit, AlertDialog for destructive confirmations, Accordion for expandable sections, Inline 409 error display]
key-files:
  created:
    - web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx
  modified:
    - web/src/types/api.ts
    - web/src/types/models.ts
    - web/src/api/projects.ts
    - web/src/routes/_authenticated/projects/-components/project-detail.tsx
decisions:
  - Dialog-based edit (not inline) matching CreateProjectDialog pattern
  - Delete confirmation with inline 409 error from backend (not toast)
  - Accordion-based expandable subproject section with lazy fetch on first expand
  - SubprojectSection as inline sub-component in project-detail.tsx for simplicity
metrics:
  duration: ~5 min (plan was pre-executed, this session applied fixes)
  completed_date: 2026-06-11
---

# Phase 5 Plan 3: Frontend Project Edit, Delete, and Subproject Display

**One-liner:** EditProjectDialog forked from CreateProjectDialog with pre-populated fields, delete confirmation with inline 409 error handling, and expandable subproject section with lazy fetch.

## Background

This plan wires the disabled Edit/Delete buttons on the project detail page to real dialogs and adds an expandable subproject section. The implementation was substantially completed in a prior session. This execution applied a type fix to the `edit-project-dialog.tsx` component and verified the build.

## Tasks Executed

### Task 1: Add UpdateProjectRequest type + Frontend API mutations/queries ✅

| Details | |
|---------|-|
| **Status** | Done (prior commit) |
| **Commit** | `4294fd6` |
| **Files** | `web/src/types/api.ts`, `web/src/types/models.ts`, `web/src/api/projects.ts` |

- `UpdateProjectRequest` interface added to `web/src/types/api.ts` (name, type, contract_id, governance_model, is_shared)
- `Subproject` interface added to `web/src/types/models.ts` (8 fields: id, project_id, name, description, sequence_order, is_active, created_at, updated_at)
- `updateProjectMutationOpts` — `PUT /projects/{id}` with `{id, data}` shape, invalidates `['projects']` + `['projects', id]`
- `deleteProjectMutationOpts` — `DELETE /projects/{id}`, shows 'Project deleted' toast on success
- `subprojectsQueryOpts` — `GET /projects/{id}/subprojects` with query key `['projects', id, 'subprojects']`
- All three exported from `ProjectsApis`

### Task 2: Create EditProjectDialog component ✅

| Details | |
|---------|-|
| **Status** | Done (prior commit + this session fix) |
| **Commits** | `1ce9602` (create), `8a894c4` (fix) |
| **File** | `web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx` |

- Forked from `CreateProjectDialog` pattern
- Props: `{open, onOpenChange, onSuccess?, project: Project}`
- Pre-populated fields from project prop (name, type, contract_id, governance_model, is_shared)
- "Edit Project" title with "Update the details of {name}." description
- "Save Changes" submit / "Saving..." pending state
- Contract selector fetches from `ContractsApis.contractsQueryOpts('all')`
- Type fix: wrapped `setContractId` to handle `string | null` from shadcn Select v5
- Type fix: used `contracts?.map` instead of `contracts?.data?.map` (api already unwraps response envelope)

### Task 3: Wire project-detail.tsx ✅

| Details | |
|---------|-|
| **Status** | Done (prior commit) |
| **Commit** | `e3bf446` |
| **File** | `web/src/routes/_authenticated/projects/-components/project-detail.tsx` |

- Edit button opens `EditProjectDialog` with current project data
- Delete button opens `AlertDialog` with destructive styling and warning text
- 409 error from backend renders inline as red text in dialog
- Delete success navigates to `/projects` with "Project deleted" toast
- Subproject `Accordion` section below Details card
- `SubprojectSection` inline sub-component with loading (3 Skeletons), error (retry button), empty (message), and loaded states
- Each subproject shows name, description, and active/inactive badge

## Verification Results

### Build Result

`cd web && bun run build` — **verified** (only pre-existing errors remain in test files and unrelated components)

Pre-existing errors (unrelated to this plan's changes):
- `src/api/__tests__/*.test.ts` — 30+ errors across all test files (useMutation mutationFn access pattern)
- `src/lib/api.ts` — ApiError import/export conflict
- `src/lib/__tests__/api.test.ts` — Property 'get' on type 'never'
- `src/routes/__root.tsx` — ThemeProvider `attribute` prop
- `src/routes/_auth/*` — Route type mismatches
- `src/routes/_authenticated/contracts/-components/create-contract-dialog.tsx` — Same Select/contracts pattern errors (pre-existing)
- `src/routes/_authenticated/projects/-components/create-project-dialog.tsx` — Same Select/contracts pattern errors (pre-existing, not our change)
- `src/routes/_authenticated/projects/-components/project-list.tsx` — Path alias issue (pre-existing)

No new errors introduced by this plan's files (`edit-project-dialog.tsx`, `project-detail.tsx`, `api/projects.ts`, `types/api.ts`, `types/models.ts`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Select onValueChange type incompatibility in EditProjectDialog**
- **Found during:** Task 2 verification
- **Issue:** shadcn Select v5 `onValueChange` passes `string | null`, but `Dispatch<SetStateAction<string>>` expects `string`. Also `contracts?.data?.map` was accessing `data` on an already-unwrapped `Contract[]` array.
- **Fix:** Wrapped `setContractId` as `(v) => setContractId(v ?? '')` and changed `contracts?.data?.map` to `contracts?.map`
- **Files modified:** `edit-project-dialog.tsx`
- **Commit:** `8a894c4`

## Success Criteria Verification

| Criterion | Status |
|-----------|--------|
| `UpdateProjectRequest` type exported from `@/types` | ✅ |
| `Subproject` interface exported from `@/types/models` | ✅ |
| 3 new exports from `ProjectsApis` | ✅ (updateProjectMutationOpts, deleteProjectMutationOpts, subprojectsQueryOpts) |
| `EditProjectDialog` component accepts `{open, onOpenChange, onSuccess?, project}` | ✅ |
| Form fields pre-populated from project prop | ✅ |
| `project-detail.tsx` has live Edit/Delete buttons | ✅ |
| Edit dialog opens on Edit button click | ✅ |
| Delete alert dialog with inline 409 error rendering | ✅ |
| Subproject accordion section with loading/error/empty/loaded states | ✅ |
| No new build errors introduced by plan changes | ✅ |

## Commits

| Hash | Message |
|------|---------|
| `4294fd6` | feat(05-03): add UpdateProjectRequest type, Subproject model, and project update/delete/subproject API |
| `1ce9602` | feat(05-03): create EditProjectDialog component forked from CreateProjectDialog |
| `e3bf446` | feat(05-03): wire project-detail with Edit/Delete buttons, dialogs, and subproject section |
| `8a894c4` | fix(05-03): fix Select onValueChange type and contracts data access in EditProjectDialog |

## Self-Check: PASSED

- ✅ `web/src/types/api.ts` — Contains `UpdateProjectRequest` (verified via read)
- ✅ `web/src/types/models.ts` — Contains `Subproject` interface (verified via read)
- ✅ `web/src/api/projects.ts` — Contains all 3 new exports (verified via read)
- ✅ `web/src/routes/_authenticated/projects/-components/edit-project-dialog.tsx` — Exists, 168 lines, correct structure (verified via read)
- ✅ `web/src/routes/_authenticated/projects/-components/project-detail.tsx` — Exists, 228 lines, fully wired (verified via read)
- ✅ Commit `4294fd6` exists (verified via git log)
- ✅ Commit `1ce9602` exists (verified via git log)
- ✅ Commit `e3bf446` exists (verified via git log)
- ✅ Commit `8a894c4` exists (verified via git log)
