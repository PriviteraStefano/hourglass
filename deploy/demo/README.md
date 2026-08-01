# Hourglass Demo Environment

Private demo: single machine, one Compose stack, Cloudflare Tunnel + Access.
Decision record: `hourglass-vault/decisions/backend/ADR-BE-015`.
Ops doc: `openwiki/operations/demo-deployment.md`.

## Layout

```
deploy/demo/
├── docker-compose.yml   # web (Caddy) · app (Go) · postgres · cloudflared · migrate · seed · recover-db
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

## Troubleshooting: postgres auth failure (28P01)

**Symptom:** the `app` container crash-loops; its log shows

```
Failed to initialize PostgreSQL pool: ... FATAL: password authentication failed for user "hourglass" (SQLSTATE 28P01)
```

**Root cause:** Postgres reads `POSTGRES_PASSWORD` **only on the first init of an
empty `pgdata` volume**. Changing `POSTGRES_PASSWORD` in `.env` afterwards never
touches the stored password, so the app's `DATABASE_URL` (composed from the same
`.env`) is rejected.

**Fix A — preserve data (preferred):** realign the stored password with `.env`:

```bash
docker compose run --rm recover-db   # idempotent — or `make demo-recover-db` from repo root
docker compose up -d app
```

**Fix B — full demo reset (wipes demo data):**

```bash
docker compose down -v
docker compose up -d
docker compose run --rm migrate
docker compose run --rm seed
```

Postgres re-initializes the empty volume with the current `.env` password.

## Properties

* **Zero published host ports** — only outbound 443/7844 (cloudflared). Host firewall stays closed.
* **Postgres is never proxied** — internal network only; only `app` can reach it.
* **No cert management** — TLS at Cloudflare's edge; the tunnel is encrypted to the origin.
* **Demo gate is Cloudflare Access** (edge OTP) — the app is unchanged.
* `/api` handling matches `web/vite.config.ts` dev proxy exactly (prefix stripped), so FE behaviour is identical dev ↔ demo. `VITE_API_URL=/api` (default) → no FE env juggling.
