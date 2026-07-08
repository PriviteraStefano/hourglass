---
phase: 07
slug: exports
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-08
---

# Phase 07 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard testing + testify (backend) / Vitest (frontend) |
| **Config file** | none — Go convention + web/vite.config.ts |
| **Quick run command** | `cd /Users/stefanoprivitera/Projects/hourglass && go test -count=1 ./internal/... -run 'Export'` |
| **Full suite command** | `cd /Users/stefanoprivitera/Projects/hourglass && go test -count=1 -timeout 120s ./internal/...` |
| **Estimated runtime** | ~15 seconds (export-specific) |

---

## Sampling Rate

- **After every task commit:** Run `go test -count=1 ./internal/... -run 'Export'`
- **After every plan wave:** Run `go test -count=1 -timeout 120s ./internal/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | EXPT-01, EXPT-04 | T-07-01 / T-07-03 | Auth-guarded timesheet CSV/XLSX with date range + Content-Disposition | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | EXPT-02, EXPT-04 | T-07-01 / T-07-03 | Auth-guarded expense CSV/XLSX with date range + Content-Disposition | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | EXPT-03, EXPT-04 | T-07-01 / T-07-03 | Auth-guarded combined CSV/XLSX with date range + Content-Disposition | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ W0 | ⬜ pending |
| 07-01-04 | 01 | 1 | EXPT-05 | — | Count endpoint returns 0 for empty range → no download triggered | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ❌ W0 | ⬜ pending |
| 07-01-05 | 01 | 1 | EXPT-06 | T-07-01 | Missing auth returns 401 on all endpoints | handler integration | `go test -count=1 ./internal/adapters/primary/http/... -run 'Export' -v` | ✅ Existing | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/adapters/primary/http/export_test.go` — rewrite to cover format param, count endpoints, project/user filters
- [ ] `internal/core/services/export/export_test.go` — update to test new count methods

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| XLSX file renders correctly in Excel | EXPT-01, EXPT-02, EXPT-03 | Visual format verification | Download XLSX export, open in Excel/LibreOffice, verify headers bold, columns auto-sized, data correct |
| Empty date range toast message | EXPT-05 | UI behavior | Set date range with no data, click Download, verify toast "No data to export" appears |
| Download button states | EXPT-04 | Visual state verification | Verify loading spinner, disabled state during download, re-enabled after completion |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
