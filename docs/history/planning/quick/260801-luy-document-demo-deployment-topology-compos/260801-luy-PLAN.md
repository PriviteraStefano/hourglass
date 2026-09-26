---
quick_id: 260801-luy
description: Document demo deployment topology (Compose + Caddy + cloudflared tunnel) — ADR, openwiki operations, vault index
date: 2026-08-01
mode: quick (default — no discuss/research/validate)
---

# Quick Task 260801-luy: Document demo deployment topology

## Goal

Record the decided demo-environment topology in the project's documentation surfaces and scaffold the runnable artifacts, so the decision is traceable and the environment is one command up.

## Tasks

1. **ADR** — `hourglass-vault/decisions/backend/ADR-BE-015 — Demo Environment Topology.md` (Accepted, 2026-08-01): context, decision (Compose+Caddy+cloudflared), alternatives (Workers/Wasm, VPS+tunnel, k8s, embedded SPA), consequences, runbook pointer. Register row in `decisions/backend/_index.md`.
2. **Ops doc** — `openwiki/operations/demo-deployment.md`: topology diagram, service table, one-time setup, daily ops, security notes. Link from `openwiki/operations/README.md`.
3. **Artifacts** — `deploy/demo/`: docker-compose.yml, Dockerfile (Go distroless), Dockerfile.web (bun→Caddy), Caddyfile, .env.example, README.md. Makefile `demo-*` targets. Validate: `docker compose config`, `caddy validate`.

## Layout facts wired in (verified against codebase)

* Backend routes are unprefixed (`/auth/*`, `/units`, …); dev proxy strips `/api` → Caddy does `uri strip_prefix /api` to match.
* Seed is SQL (`scripts/seed_demo.sql` via psql), not a Go binary — `cmd/` has only `server` + `migrate`.
* Go 1.26.1 (go.mod), frontend built with bun (bun.lock), `bun run build` → `web/dist/`.
* Migrations are files (`migrations/`), applied by `cmd/migrate`; baked into the image.
