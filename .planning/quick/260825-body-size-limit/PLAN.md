---
quick_id: 260825-body-size-limit
description: Add MaxBody middleware capping JSON request bodies (CONCERNS #8)
date: 2026-08-25
status: complete
---

# Quick Task 260825-body-size-limit

## Plan

Address CONCERNS.md #8 "No request-body size limit on JSON endpoints".

Task: add `internal/middleware/maxbody.go` `MaxBody(maxBytes)` middleware. It rejects
non-multipart requests whose `Content-Length` exceeds the cap with 413 before reading, and caps
the reader via `http.MaxBytesReader`. Multipart requests (receipt upload, which enforces its own
10 MB) are exempt. Default 1 MB, tunable via `MAX_BODY_BYTES`. Wire it as the outermost middleware
in `cmd/server/main.go`.

## Verify

- `go build ./...` passes
- `go vet ./cmd/server ./internal/middleware` passes
- `go test ./internal/middleware/...` passes
