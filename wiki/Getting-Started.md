# Hourglass — Getting Started

Hourglass is self-hosted software: you run it in your own environment, and the repository includes Docker and docker-compose to make that easy. There is no hosted trial yet — to try it, follow the local quickstart on this page or the developer reference. Pick the path that fits what you are here to do.

_Back to [Home](Home.md) · [Vision](Vision.md) · [FAQ](FAQ.md) · [Adopters](Adopters.md) · [Employees](Employees.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

## I'm an employee joining my organization

Your admin invites you — by code or by link. Accept the invitation and you arrive as an **employee** by default; invitations expire after 7 days, so accept yours while it is fresh.

Once you are in, the [Employees](Employees.md) page walks you through your daily loop: capture time and expenses, submit, and watch the status move until your entries are approved.

## I'm an admin setting up an organization

An organization and its admin account are created in one atomic step — no orphaned users, no half-created orgs. From there you invite members by code or link; those invitations expire after 7 days, and invitees arrive as employees by default, so membership stays deliberate.

The [Adopters](Adopters.md) page covers roles, the approval chain, governance models, and the security posture in depth.

## I'm evaluating or building with Hourglass

There is no hosted trial — the repository is how you run it. The [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md) quickstart gets the whole stack running in a few commands, and the [Developer](Developer.md) reference documents every environment variable, Makefile target, and testing command.

### Run the stack locally

```bash
git clone https://github.com/PriviteraStefano/hourglass.git
cd hourglass

make docker-up      # starts PostgreSQL and the app on :8080
make migrate-up     # apply migrations — or make db-init to initialize a fresh database
make run            # backend at http://localhost:8080

# Frontend, in a second terminal:
cd web
bun install
bun run dev         # http://localhost:3000, proxies /api to :8080
```

> **Note:** to initialize a fresh database, use `make migrate-up` or `make db-init`. The `make setup` and `make migrate-all` shortcuts behind the `-all` flag are not wired up yet — the [Developer](Developer.md) page documents the drift.

---

_Back to [Home](Home.md) · [Vision](Vision.md) · [FAQ](FAQ.md) · [Adopters](Adopters.md) · [Employees](Employees.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

_Source: README.md, the feature specifications (F05–F06), the developer reference (wiki/Developer.md), and the claim trace (docs/README-claim-trace.md), 2026-08-01._
