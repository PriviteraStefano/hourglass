# ADR-BE-015 — Demo Environment Topology: Compose + Caddy + cloudflared

---
tags: ["adr", "backend", "deployment", "demo", "cloudflare", "docker", "caddy"]
---

# ADR-BE-015 — Demo Environment Topology: Compose + Caddy + cloudflared

**Status:** Accepted
**Date:** 2026-08-01
**Code:** `deploy/demo/` (docker-compose.yml, Dockerfile, Dockerfile.web, Caddyfile, .env.example, README.md)
**Phase:** 11 (demo readiness)

## Context

Phase 11 needs a **private demo environment**. Constraints:

* Solo maintainer → minimal ops burden; the box must be boring enough to ignore for two weeks.
* The demo must not be publicly crawlable — it's shown to selected prospects, not the internet.
* Hourglass's product identity is **on-prem self-hosting** — the demo topology should *be* the deployment story, not a PaaS detour.
* Demo data must be resettable (persona set: `manager_alex`, `hr_rachel`, … — see `scripts/seed_demo.sql`).
* Near-zero cost; no inbound exposure of the host.

Earlier exploration rejected compiling the Go backend to WebAssembly for Cloudflare Workers (experimental DB drivers, breaks the self-host narrative). Kubernetes/k3s was considered and deferred (fleet orchestration for a single node; learning curve under deadline — revisited when enterprise packaging needs Helm).

## Decision

A single always-on machine runs one Docker Compose stack — `deploy/demo/docker-compose.yml`:

```
Prospect ── Cloudflare edge (Access: email OTP) ── named tunnel (outbound-only)
                                                      │
                                              cloudflared (container)
                                                      │ http://web:80
                                              ┌───────┴────────┐
                                              │  web (Caddy)   │  SPA statics + index.html fallback
                                              │  /api/* ───────┼──► app (Go binary, distroless) ──► postgres
                                              └────────────────┘      (pgx, static, :8080)         (PG17, internal-only)
```

* **Zero published host ports.** The only network requirement is outbound 443/7844 for `cloudflared`. Host firewall stays fully closed.
* **TLS terminates at Cloudflare's edge; the tunnel is encrypted to the origin.** No cert management anywhere.
* **Cloudflare Access gates the hostname** (email OTP allowlist) — demo auth without touching app code. ASVS L1-aligned.
* **`web` = Caddy serving the Vite-built SPA** with `/api/*` reverse-proxied to the Go binary, stripping the `/api` prefix — this mirrors the dev proxy in `web/vite.config.ts` exactly, so frontend behaviour is identical in dev and demo. `VITE_API_URL=/api` (the default) means no FE env juggling.
* **`app` = static Go binary** (`CGO_ENABLED=0`, pgx) in a `distroless/static-debian12:nonroot` image. Migrations are baked in (`migrations/`) and applied by `cmd/migrate` as a one-shot compose service.
* **Seed is SQL, not a Go binary.** `cmd/` has no seed entrypoint — the demo seed is `scripts/seed_demo.sql` applied with `psql` inside the postgres container via a `seed` one-shot service (wraps the Makefile's `make seed` flow).
* **Caddy is retained** (rather than embedding the SPA in the Go binary) to decouple FE/BE builds and keep a direct-exposure fallback open: pointing Caddy at a real domain gives automatic Let's Encrypt TLS with a one-line change, dropping the tunnel without re-architecting.
* **The tunnel is token-managed** (remotely configured in the Zero Trust dashboard), so `deploy/demo/` doubles as the future shipped on-prem reference compose file.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Workers + Wasm backend | Rejected earlier: experimental Wasm DB drivers; breaks the on-prem/self-host story. |
| VPS + tunnel | Identical topology, adds cost/ops now. **Remains the escalation path** if always-on self-serve access is needed — the tunnel config is portable, migration is "install two services elsewhere, move the token." |
| Kubernetes / k3s | Fleet orchestration for a single node; learning curve under a demo deadline. Deferred to a dedicated spike when enterprise packaging (Helm chart) actually demands it. |
| SPA embedded in Go binary (`embed.FS`, no Caddy) | Viable simplification; rejected to keep FE/BE builds decoupled and preserve the direct-exposure TLS fallback. |

## Consequences

* `docker compose up -d --build` = entire demo incl. ingress. One command, reproducible.
* `deploy/demo/` is 90% of the future shipped on-prem `docker-compose.yml` — demo infra graduates into a product artifact.
* No open ports, no certs, auth-for-free via Access.
* ⚠️ Availability is tied to the host machine/ISP — acceptable for scheduled demos; VPS is the known upgrade path.
* ⚠️ `TUNNEL_TOKEN` is a managed secret (`.env`, already gitignored; `.env.example` committed).

## Runbook

See `deploy/demo/README.md`. One-time: create named tunnel, set public hostname → `http://web:80`, add Access application. Daily: `make demo-redeploy`, `make demo-seed` (nightly via cron).

## Related

* [[ADR-BE-004 — Database Migrations]] (seed separation: `*_seed.up.sql` excluded from migrations; demo seed lives in `scripts/`)
* [[ADR-BE-003 — Data Access pgxpool No ORM]] (pure-Go driver → static binary)
* [[ADR-BE-011 — CORS Policy]] (same-origin via Caddy means no CORS surface in the demo)
* `.planning/ROADMAP.md` Phase 11
