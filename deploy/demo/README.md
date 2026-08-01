# Hourglass Demo Environment

Private demo: single machine, one Compose stack, Cloudflare Tunnel + Access.
Decision record: `hourglass-vault/decisions/backend/ADR-BE-015`.
Ops doc: `openwiki/operations/demo-deployment.md`.

## Layout

```
deploy/demo/
├── docker-compose.yml   # web (Caddy) · app (Go) · postgres · cloudflared · migrate · seed
├── Dockerfile           # Go binary (distroless, static, pgx)
├── Dockerfile.web       # Vite build (bun) → Caddy image
├── Caddyfile            # SPA + /api reverse proxy (prefix strip, mirrors dev proxy)
├── .env.example         # commit this; .env stays local (gitignored)
└── README.md
```

## One-time setup

1. Zero Trust → **Networks → Tunnels → Add tunnel** → `hourglass-demo` → token into `.env` (`TUNNEL_TOKEN`).
2. Tunnel → **Public hostnames**: `demo.<your-domain>` → service `HTTP`, URL `web:80`. (DNS is automatic.)
3. **Access → Applications → Add** → self-hosted, same hostname → Allow specific prospect emails, OTP.
4. `cp .env.example .env` (fill `JWT_SECRET`, `POSTGRES_PASSWORD`), then:
   ```bash
   docker compose up -d --build
   docker compose run --rm migrate
   docker compose run --rm seed
   ```

## Daily ops

| Task | Command |
|------|---------|
| Redeploy | `git pull && docker compose up -d --build && docker image prune -f` (or `make demo-redeploy` from repo root) |
| Migrate | `docker compose run --rm migrate` |
| Seed now | `docker compose run --rm seed` (idempotent — see `scripts/seed_demo.sql`) |
| Nightly reset | host cron: `15 6 * * * cd deploy/demo && docker compose run --rm seed` |
| Pre-demo check | `curl -s https://demo.<your-domain>/api/health` → 200; log in through Access once |

## Properties

* **Zero published host ports** — only outbound 443/7844 (cloudflared). Host firewall stays closed.
* **Postgres is never proxied** — internal network only; only `app` can reach it.
* **No cert management** — TLS at Cloudflare's edge; the tunnel is encrypted to the origin.
* **Demo gate is Cloudflare Access** (edge OTP) — the app is unchanged.
* `/api` handling matches `web/vite.config.ts` dev proxy exactly (prefix stripped), so FE behaviour is identical dev ↔ demo. `VITE_API_URL=/api` (default) → no FE env juggling.
