---
status: complete
quick_id: 260825-cookie-secure-flag
date: 2026-08-25
---

# Summary — 260825-cookie-secure-flag

**Concern:** CONCERNS.md #12 Refresh-token `Secure` flag trusts `X-Forwarded-Proto`.

**Change:** `internal/cookies/cookies.go` — `IsSecureRequest` no longer reads the attacker-
controllable `X-Forwarded-Proto` header. The `Secure` attribute is now set only when the
connection is actually TLS (`r.TLS != nil`) or the operator explicitly sets `SECURE_COOKIES=1|true`.
This removes the spoofing vector where a client could force an insecure (HTTP) cookie and expose
tokens to network sniffing behind a misconfigured proxy. Cookies remain `HttpOnly` +
`SameSite=Strict`.

**Result:** Cookie `Secure` is operator-controlled (correct for a TLS-terminating proxy) rather
than client-influenced.

**Verification:** `go build`, `go vet`, and both package test suites pass.
