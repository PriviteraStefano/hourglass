---
status: complete
quick_id: 260825-body-size-limit
date: 2026-08-25
---

# Summary — 260825-body-size-limit

**Concern:** CONCERNS.md #8 No request-body size limit on JSON endpoints.

**Change:** New `middleware.MaxBody(maxBytes)` applied as the outermost middleware in
`cmd/server/main.go`. Non-multipart (i.e. JSON) requests over the cap are rejected with `413
Request Entity Too Large` before the body is read; the reader is also wrapped in
`http.MaxBytesReader` so chunked bodies are capped at read time. Multipart requests are exempt so
the existing 10 MB receipt-upload limit is preserved. Default 1 MB, overridable via
`MAX_BODY_BYTES`.

**Result:** Closes the highest-impact unauthenticated memory-exhaustion DoS vector (previously any
client could POST an arbitrarily large body to `/auth/register`, `/auth/login`, `/time-entries`,
etc.).

**Verification:** `go build ./...`, `go vet`, and `go test ./internal/middleware/...` pass.
