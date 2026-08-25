---
status: complete
quick_id: 260825-no-error-leak
date: 2026-08-25
---

# Summary — 260825-no-error-leak

**Concern:** CONCERNS.md #6 Internal error messages leaked to clients.

**Change:** `internal/adapters/primary/http/auth.go` — the two `err.Error()` leak sites
(Register default case, Bootstrap error) now return stable generic messages ("registration
failed" / "bootstrap failed") and log the detailed error server-side with `log.Printf`. The
`log` import was added. A grep confirms these were the only two `err.Error()` calls passed to
`api.RespondWithError` in the handler package.

**Convention:** handlers must never return `err.Error()` to clients; services should surface
typed sentinel errors that handlers map to safe messages.

**Verification:** `go build` + `go vet` pass for the http adapter package.
