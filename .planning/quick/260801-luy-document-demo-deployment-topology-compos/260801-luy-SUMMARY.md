---
quick_id: 260801-luy
status: complete
date: 2026-08-01
---

# Quick Task 260801-luy — Summary

**Task:** Document demo deployment topology (Compose + Caddy + cloudflared tunnel) — ADR, openwiki operations, vault index.

## What was done

### Decision record (wiki)
* **ADR-BE-015 — Demo Environment Topology** created (`hourglass-vault/decisions/backend/`), Status: Accepted, 2026-08-01. Captures context, decision, alternatives (Workers/Wasm, VPS+tunnel, k8s deferred, embedded-SPA), consequences, runbook pointer.
* Registered in `hourglass-vault/decisions/backend/_index.md`.

### Ops documentation (wiki)
* `openwiki/operations/demo-deployment.md` — topology diagram, service table, one-time setup, daily ops, security notes, VPS escalation path.
* Linked from `openwiki/operations/README.md` under a new "Environments" section.

### Runnable artifacts (`deploy/demo/`)
* `docker-compose.yml` — `web` (Caddy), `app` (Go), `postgres` (PG17, internal-only), `cloudflared` (named tunnel), plus `migrate`/`seed` one-shot services under the `ops` profile. Zero published host ports.
* `Dockerfile` — Go 1.26.1 build, static binaries (server + migrate), distroless nonroot, migrations baked in.
* `Dockerfile.web` — bun build (`tsc -b && vite build`) → Caddy image.
* `Caddyfile` — SPA + `index.html` fallback, `/api/*` reverse proxy with `uri strip_prefix /api` (mirrors dev proxy).
* `.env.example`, `README.md`.
* Makefile `demo-up` / `demo-migrate` / `demo-seed` / `demo-redeploy` targets.

## Verification
* `docker compose config` → OK (only expected unset-env warnings; `.env` not yet created).
* `caddy validate` → **Valid configuration**.

## Notes / follow-ups
* `.env` must be created from `.env.example` (TUNNEL_TOKEN, JWT_SECRET, POSTGRES_PASSWORD) before `demo-up`.
* One-time Cloudflare setup (named tunnel, public hostname → `http://web:80`, Access application) is manual — documented in the ops doc.
* Not yet smoke-tested end-to-end (`demo-up` + migrate + seed + tunnel) — requires the Cloudflare tunnel token.

## Addendum (2026-08-01)
* Also published to the **GitHub wiki**: page [Demo-Deployment](https://github.com/PriviteraStefano/hourglass/wiki/Demo-Deployment) (live, HTTP 200), linked from `Home` and `Developer`. Wiki commit `a1cb0df` on `PriviteraStefano/hourglass.wiki` (master). Note: GitHub wiki is a separate repo from the main codebase.
