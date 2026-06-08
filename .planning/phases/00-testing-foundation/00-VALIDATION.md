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
| **Quick run command** | `go test -v -run 'Test[^I]' ./{affected_package}/...` |
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
| 00-02-T1 | 00-02 | 1 | TEST-02 | unit | `go build ./internal/...` | ⬜ pending |
| 00-02-T2 | 00-02 | 1 | TEST-02 | integration | `go test -v -count=1 -run TestSmoke ./cmd/server/...` | ⬜ pending |
| 00-02-T3 | 00-02 | 1 | TEST-02 | integration | `go test -v -count=1 -run TestSetupPackageContainer ./internal/...` | ⬜ pending |
| 00-01-T1 | 00-01 | 2 | TEST-01 | unit | `go build ./internal/...` | ⬜ pending |
| 00-01-T2 | 00-01 | 2 | TEST-01 | integration | `go build ./internal/...` | ⬜ pending |
| 00-01-T3 | 00-01 | 2 | TEST-01 | integration | `go test -v -count=1 -run 'TestAuthIntegration' ./internal/adapters/primary/http/...` | ⬜ pending |
| 00-03-T1 | 00-03 | 3 | TEST-03 | integration | `go test -v -count=1 ./internal/core/services/...` | ⬜ pending |
| 00-03-T2 | 00-03 | 3 | TEST-03 | integration | `go test -v -count=1 ./internal/core/services/...` | ⬜ pending |
| 00-03-T3 | 00-03 | 3 | TEST-03 | integration | `go test -v -count=1 ./internal/core/services/...` | ⬜ pending |
| 00-04-T1 | 00-04 | 4 | TEST-04 | integration | `go test -v -count=1 -run 'TestAuth|TestPasswordReset' ./internal/adapters/primary/http/...` | ⬜ pending |
| 00-04-T2 | 00-04 | 4 | TEST-04 | integration | `go test -v -count=1 -timeout 180s ./internal/adapters/primary/http/...` | ⬜ pending |
| 00-04-T3 | 00-04 | 4 | TEST-04 | integration | `go test -v -count=1 -timeout 180s ./internal/adapters/primary/http/...` | ⬜ pending |
| 00-05-T1 | 00-05 | 5 | TEST-05 | integration | `go test -v -count=1 ./...` | ⬜ pending |
| 00-05-T2 | 00-05 | 5 | TEST-05 | manual | (human checkpoint) | ⬜ pending |
| 00-05-T3 | 00-05 | 5 | TEST-05 | integration | `go test -v -count=1 ./...` | ⬜ pending |
| 00-06-T1 | 00-06 | 6 | TEST-06 | e2e | `cd web && bunx playwright test` | ⬜ pending |
| 00-06-T2 | 00-06 | 6 | TEST-06 | e2e | `cd web && bunx playwright test --reporter=list` | ⬜ pending |

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

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Bug buffer human-in-the-middle review | TEST-05 | D-15 requires human review of major/complex bugs before batch fix | `00-05 Task 2` — inspect each t.Skip'd test, triage severity, confirm fix approach |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
