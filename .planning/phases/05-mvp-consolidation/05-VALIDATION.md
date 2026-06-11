---
phase: 05
slug: projects
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-11
---

# Phase 05 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `stretchr/testify` (backend), Vitest + Playwright (frontend) |
| **Config file** | `go.mod` for Go; `web/package.json` for frontend |
| **Quick run command** | `go test -count=1 -timeout 120s ./internal/core/services/project/...` |
| **Full suite command** | `go test -count=1 -timeout 300s ./...` |
| **Estimated runtime** | ~120 seconds (service-only) / ~300 seconds (full) |

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 -timeout 120s ./internal/core/services/project/...`
- **After every plan wave:** Run `go test -count=1 -timeout 300s ./internal/...`
- **Before `/gsd-verify-work`:** Full suite must be green + `cd web && bun run build`
- **Max feedback latency:** 300 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | PROJ-03 | T-05-01 / V4 | Role gate finance-only | unit | `go test ./internal/core/services/project/... -run TestService_Update` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | PROJ-04 | T-05-01 / V4 | Role gate + active entries check | unit | `go test ./internal/core/services/project/... -run TestService_Delete` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 1 | PROJ-03 | T-05-01 / V5 | Input validation | unit | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_Update` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 1 | PROJ-04 | T-05-01 / V8 | 409 error format | unit | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_Delete` | ❌ W0 | ⬜ pending |
| 05-01-05 | 01 | 1 | PROJ-05 | — | N/A | unit | `go test ./internal/adapters/primary/http/... -run TestProjectHandler_ListSubprojects` | ❌ W0 | ⬜ pending |
| 05-01-06 | 01 | 1 | PROJ-04 | — | Cascade-clean adoptions | integration | `go test ./internal/adapters/secondary/postgres/... -run TestProjectRepository_Delete` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 2 | PROJ-03 | — | N/A | e2e/visual | Manual check | ❌ W0 | ⬜ pending |
| 05-02-02 | 02 | 2 | PROJ-04 | — | Delete confirmation UX | e2e/visual | Manual check | ❌ W0 | ⬜ pending |
| 05-02-03 | 02 | 2 | PROJ-05 | — | Subproject section renders | e2e/visual | Manual check | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/services/project/project_test.go` — add `TestService_Update`, `TestService_Delete`
- [ ] `internal/adapters/primary/http/project_test.go` — add `TestProjectHandler_Update`, `TestProjectHandler_Delete`, `TestProjectHandler_ListSubprojects`
- [ ] `internal/adapters/secondary/postgres/project_repository_test.go` — add `TestProjectRepository_Update`, `TestProjectRepository_Delete`, `TestProjectRepository_HasActiveTimeEntries`
- [ ] `internal/core/services/testdata/mocks.go` — add `Update`, `Delete`, `HasActiveTimeEntries` to `MockProjectRepo`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Edit dialog opens with pre-populated fields | PROJ-03 | Visual UI requires human verification | Open a project, click Edit, verify all fields show correct current values, save |
| Delete blocked shows 409 error in dialog | PROJ-04 | Error display rendering | Try to delete a project with active time entries, verify error message appears inline |
| Subproject expandable section renders | PROJ-05 | Visual layout check | Open project detail, click expand, verify subproject names and statuses display |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 300s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
