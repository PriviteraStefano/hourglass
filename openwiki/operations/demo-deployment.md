# Demo Deployment

The private demo environment is a single machine running one Docker Compose stack,
exposed through a Cloudflare Tunnel and gated by Cloudflare Access. Decision record:
ADR-BE-015 in the vault (`hourglass-vault/decisions/backend/`).

## Topology

```
Prospect browser
      │  HTTPS
      ▼
Cloudflare edge ── Access policy (email OTP, allowlisted prospect addresses)
      │  named tunnel (outbound-only, QUIC)
      ▼
cloudflared (container) ──► http://web:80
                                 │
                          web (Caddy)
                          ├── /*      → SPA statics (Vite build) + index.html fallback
                          └── /api/*  → app:8080 (prefix stripped, mirrors dev proxy)
                                 │
                          app (Go binary, distroless, :8080)
                                 │
                          postgres (PG17, internal network ONLY — never proxied)
```

**Zero published host ports.** The host needs only outbound 443/7844 for `cloudflared`.
TLS terminates at Cloudflare's edge; the tunnel itself is encrypted to the origin, so
no certificate management exists anywhere in the stack.

## Services (`deploy/demo/docker-compose.yml`)

| Service | Image | Role |
|---------|-------|------|
| `cloudflared` | `cloudflare/cloudflared` (pinned) | Named tunnel, token-managed (`TUNNEL_TOKEN`) |
| `web` | built from `deploy/demo/Dockerfile.web` | Caddy serving the SPA + `/api/*` reverse proxy |
| `app` | built from `deploy/demo/Dockerfile` | Static Go binary (CGO_ENABLED=0, pgx), distroless nonroot |
| `postgres` | `postgres:17-alpine` | Demo database, named volume, **no published ports** |
| `migrate` | same image as `app` (profile) | One-shot: applies `migrations/` via `cmd/migrate` |
| `seed` | `postgres:17-alpine` (profile) | One-shot: applies `scripts/seed_demo.sql` via `psql` |

The `/api` handling is deliberately identical to the dev proxy in `web/vite.config.ts`
(strip the `/api` prefix before forwarding), so frontend behaviour matches between
`vite dev` and the demo. `VITE_API_URL` defaults to `/api` — no FE env changes needed.

## One-time setup

1. Zero Trust dashboard → **Networks → Tunnels → Add tunnel** → name `hourglass-demo`.
   Copy the token into `deploy/demo/.env` (`TUNNEL_TOKEN`).
2. Tunnel → **Public hostnames**: `demo.<your-domain>` → service type `HTTP`, URL `web:80`.
   DNS is created automatically.
3. **Access → Applications → Add** → self-hosted, same hostname → policy: Allow,
   specific prospect emails, one-time PIN login. This is the demo gate.
4. `cd deploy/demo && cp .env.example .env` (fill secrets) `&& docker compose up -d --build`

## Daily ops

| Task | Command |
|------|---------|
| Redeploy | `make demo-redeploy` (git pull → rebuild → prune) |
| Migrate | `cd deploy/demo && docker compose run --rm migrate` |
| Seed now | `make demo-seed` (or `docker compose run --rm seed` in `deploy/demo/`) |
| Nightly reset | host cron: `15 6 * * * cd deploy/demo && docker compose run --rm seed` |
| Pre-demo check | `curl -s https://demo.<your-domain>/api/health` → 200, then log in via Access once |

### Troubleshooting: postgres auth failure (SQLSTATE 28P01)

**Symptom:** `app` crash-loops with `Failed to initialize PostgreSQL pool: ... password
authentication failed for user "hourglass" (SQLSTATE 28P01)`.

**Root cause:** postgres reads `POSTGRES_PASSWORD` only on first init of an empty
`pgdata` volume; changing `.env` later never touches the stored password.

**Fix (preserves data):** `make demo-recover-db` (realigns the stored password with
`.env`), then `make demo-up` or `docker compose up -d app`.

**Reset fallback (wipes demo data):** `docker compose down -v` → `up -d` →
`run --rm migrate` → `run --rm seed`.

See `deploy/demo/README.md` → "Troubleshooting: postgres auth failure (28P01)" for the
full write-up.

## Security notes

* `postgres` has no published ports and no ingress path — only `app` can reach it.
* `TUNNEL_TOKEN`, `POSTGRES_PASSWORD`, `JWT_SECRET` live in `.env` (gitignored);
  `.env.example` is the committed template.
* Access (edge OTP) is the only auth boundary in front of the demo — the app itself
  is unchanged.
* Escalation path: to move to a VPS for always-on self-serve access, run the same
  compose stack there and move the tunnel token. Nothing else changes.
