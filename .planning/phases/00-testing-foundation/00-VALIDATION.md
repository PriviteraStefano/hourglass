---
phase: "00"
slug: testing-foundation
status: draft
nyquist_compliant: false
created: 2026-06-08
---

# Phase 00 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.26.1) + testcontainers-go v1.35+ |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test -v -run 'Test[^I]' ./internal/core/services/auth/...` |
| **Full suite command** | `go test -v ./...` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v -run 'Test[^I]' ./{affected_package}/...`
- **After every plan wave:** Run `go test -v ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 00-01-01 | 01 | 1 | TEST-01 | integration | `go test -v -run TestAuthBugs ./internal/...` | ⬜ pending |
| 00-02-01 | 02 | 1 | TEST-02 | integration | `go test -v -run TestContainerSetup ./internal/...` | ⬜ pending |
| 00-03-01 | 03 | 2 | TEST-03 | integration | `go test -v ./internal/core/services/...` | ⬜ pending |
| 00-04-01 | 04 | 2 | TEST-04 | integration | `go test -v -run 'TestHandler' ./internal/adapters/primary/http/...` | ⬜ pending |
| 00-05-01 | 05 | 3 | TEST-05 | integration | `go test -v ./...` | ⬜ pending |
| 00-06-01 | 06 | 3 | TEST-06 | e2e | `cd web && bunx playwright test` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements:

- `go test` is already installed (Go 1.26.1)
- `testcontainers-go` will be added as Go module dependency
- `testify/require` and `testify/assert` already available
- `playwright` already configured in `web/playwright.config.ts`
- No new test framework installation needed

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
