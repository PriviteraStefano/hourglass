---
status: complete
quick_id: 260825-remove-api-version
date: 2026-08-25
---

# Summary — 260825-remove-api-version

**Concern:** "API versioning middleware is decorative."

**Resolution (user choice):** Remove the dead middleware.

**Change:** Deleted `internal/middleware/version.go` (and `version_test.go`). No handler ever read
the version from context, so it was pure overhead implying a versioning contract that doesn't
exist. Removed `middleware.APIVersion(...)` from the chain in `cmd/server/main.go`. Fixed the
stale signature comment in `cors.go`.

**Result:** One less no-op middleware per request; no false promise of API versioning. If real
versioning is ever needed, it should be designed deliberately rather than implied by dead code.

**Verification:** `go build ./...`, `go vet`, and `go test ./internal/middleware/...` pass.
