---
phase: "00-testing-foundation"
plan: "00-01"
name: "Infrastructure Setup — Quick-scan, testify, testdata package"
subsystem: "testing-infrastructure"
tags: ["go", "testing", "infrastructure", "testdata"]
key-files:
  created:
    - "scripts/quick-scan.sh"
    - ".planning/phases/00-testing-foundation/BUGS.md"
    - "internal/core/services/testdata/factories.go"
    - "internal/core/services/testdata/mocks.go"
    - "internal/core/services/testdata/mocks_test.go"
  modified:
    - "go.mod"
    - "go.sum"
metrics:
  go_files_created: 3
  scripts_created: 1
  doc_files_created: 1
  lines_of_code: ~750
---

# Summary — Plan 00-01: Infrastructure Setup

**Objective:** Create shared testing infrastructure — quick-scan probe script, make testify a direct dependency, build the shared testdata/mocks package, and scaffold BUGS.md.

## Commits

| Task | Description | Files |
|------|-------------|-------|
| Task 1 | Quick-scan probe script + BUGS.md scaffold | `scripts/quick-scan.sh`, `BUGS.md` |
| Task 2 | testify direct dependency | `go.mod`, `go.sum` |
| Task 3 | Testdata factories, mocks, and instantiation test | `testdata/factories.go`, `testdata/mocks.go`, `testdata/mocks_test.go` |

## Deviations from Plan

- **TimeEntryItem factory removed:** `TimeEntryItem` exists only in `models` package (not domain `time_entry`), so factory was removed to avoid confusion. Not needed for service-layer tests.
- **MockAuditLogRepo as separate struct:** `AuditLogRepository.Create` has a different signature than `TimeEntryRepository.Create`, requiring a separate mock struct instead of embedding both interfaces on one type.
- **MockOrgRepo split into two:** `OrganizationRepository` (`auth.Organization`) and `OrganizationManagementRepository` (`orgdomain.Organization`) have completely different method sets. Implemented as `MockOrgRepo` and `MockOrgMgmtRepo`.
- **Mocks use actual port interface signatures:** The plan's suggested mock method signatures were simplified; actual implementations match the real port interfaces exactly.

## Self-Check

- `go build ./...` — PASSED
- `go test ./internal/core/services/testdata/... -count=1 -v` — PASSED
- `scripts/quick-scan.sh` exists and is executable — PASSED
- `BUGS.md` exists with table format — PASSED
- testify v1.11.1 in go.mod — PASSED

## Notes

- Pre-existing build errors in `internal/adapters/secondary/surrealdb/*_test.go` are not related to Plan 01. These will be addressed in downstream plans (00-06: repository tests).
- The testdata package is importable by all service test files. Downstream plans (00-03, 00-04) can `import "github.com/stefanoprivitera/hourglass/internal/core/services/testdata"` for factories and mocks.
