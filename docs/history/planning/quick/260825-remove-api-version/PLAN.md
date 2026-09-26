---
quick_id: 260825-remove-api-version
description: Remove decorative APIVersion middleware (CONCERNS — API versioning)
date: 2026-08-25
status: complete
---

# Quick Task 260825-remove-api-version

## Plan

Address the "API versioning middleware is decorative" concern — chosen resolution: REMOVE.

Task: delete `internal/middleware/version.go` and `version_test.go` (nothing in any handler reads
`VersionKey`/`GetAPIVersion`; it was a no-op that implied a versioning contract). Drop
`middleware.APIVersion(...)` from the `cmd/server/main.go` handler chain. Updated the stale
signature comment in `cors.go`.

## Verify

- `go build ./...` passes
- `go vet ./cmd/server ./internal/middleware` passes
- `go test ./internal/middleware/...` passes
