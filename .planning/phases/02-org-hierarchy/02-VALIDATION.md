---
phase: 2
slug: org-hierarchy
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-10
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify (backend), Playwright (frontend E2E) |
| **Config file** | none — standard Go test convention |
| **Quick run command** | `go test -count=1 -timeout 120s ./internal/core/services/unit/...` |
| **Full suite command** | `go test -count=1 -timeout 300s ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 -timeout 120s ./internal/core/services/unit/...`
- **After every plan wave:** Run `go test -count=1 -timeout 300s ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | ORG-03 | T-02-01 / — | Primary unit enforcement at service layer — org-scoped via JWT | integration | `go test -run TestUnitIntegration/UpdateMember -count=1 ./internal/core/services/unit/...` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | ORG-05 | T-02-02 / — | Delete protection — root unit, children, members checks | integration | `go test -run TestUnitIntegration/Delete -count=1 ./internal/core/services/unit/...` | ❌ W0 | ⬜ pending |
| 02-01-03 | 01 | 1 | ORG-03 | — | UpdateMember repository method | unit | `go test -run TestUnitMemberRepository/Update -count=1 ./internal/adapters/secondary/postgres/...` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | ORG-03 | — | Primary unit Make Primary button in side panel | manual | N/A — visual | N/A | ⬜ pending |
| 02-02-02 | 02 | 1 | ORG-03 | — | Reparent dialog uses reparentUnitMutationOpts | manual | N/A — visual | N/A | ⬜ pending |
| 02-03-01 | 03 | 2 | ORG-03 | — | Subtree member expandable groups in side panel | manual | N/A — visual | N/A | ⬜ pending |
| 02-03-02 | 03 | 2 | ORG-03 | — | Batch members endpoint GET /units/members?unit_ids=... | integration | `go test -run TestUnitIntegration/ListMembersByUnitIDs -count=1 ./internal/core/services/unit/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/core/services/unit/unit_integration_test.go` — subtests for `UpdateMember`, `Delete_RootUnit`, `Delete_HasChildren`, `Delete_HasMembers`
- [ ] `internal/adapters/secondary/postgres/unit_member_repository_test.go` — subtests for `Update` and `ListByUnitIDs`
- [ ] `internal/core/ports/unit_repository.go` — add `UpdateMember`, `HasChildren`, `ListMembersByUnitIDs` method signatures

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Primary unit "Make Primary" button visibility | ORG-03 | Visual UI interaction — button appears on hover, shows Badge when primary | Open side panel for a unit with members; verify "Primary" badge on primary member; hover non-primary member and click "Make Primary"; verify badge moves |
| Subtree member expandable groups | ORG-03 | Visual UI interaction — expandable groups, recursive nesting | Open side panel for a unit with child units; verify "Direct Members" section followed by expandable sub-unit groups; expand a group to see its members and recursively its children's members |
| Reparent dialog uses dedicated mutation | ORG-04 | Integration behavior — mutation call changed | Drag edge to reparent a unit; confirm dialog appears; verify reparent succeeds; check network tab shows PUT with only `parent_unit_id` |
| Edge-driven reparenting validation | ORG-04 | Visual interaction — cycle prevention | Attempt to drag edge to create a cycle (child to ancestor); verify edge is rejected (red indicator or no connection) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
