# Phase 1: Authorization - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-10
**Phase:** 01-Authorization
**Areas discussed:** Register cookies, OrgSwitcher integration, Landing page, Password reset code, Route generation

---

## Register → no cookies

| Option | Description | Selected |
|--------|-------------|----------|
| Fix: set cookies on register | Add cookie setting to Register handler. User auto-logs in after registration. | ✓ |
| Keep as-is | Two-step flow: register then login separately. | |

**User's choice:** Fix — set cookies on register
**Notes:** Align Register handler behavior with Login and Bootstrap handlers

## OrgSwitcher hardcoded `[]`

| Option | Description | Selected |
|--------|-------------|----------|
| Use memberships API + switching | Fetch memberships, wire switch mutation, invalidate queries on switch | ✓ |
| Simple fetch-only | Show org list, no token re-issue | |

**User's choice:** Full integration — memberships API + switching + query invalidation

## No landing page for `/`

| Option | Description | Selected |
|--------|-------------|----------|
| Redirect to /time-entries | Simple router redirect, no new page | ✓ |
| Dashboard page | Full overview page with stats, recent entries | |
| Minimal redirect | '/' → '/time-entries' redirect | ✓ |

**User's choice:** Minimal redirect for now (dashboard deferred)

## Password reset code leak

| Option | Description | Selected |
|--------|-------------|----------|
| Fix in this phase | Stop returning code in response, increase entropy, rate-limit | ✓ |
| Defer to later phase | Acceptable for v0.1 with existing rate limiting | |

**User's choice:** Fix in this phase

## Route tree generation

| Option | Description | Selected |
|--------|-------------|----------|
| bun run dev handles it | @tanstack/router-plugin auto-generates routeTree.gen | ✓ |
| Add explicit script | Separate generate-routes command | |

**User's choice:** Handled by dev server

---

## Deferred Ideas

- Dashboard/overview page at `/` — user wanted it but deferred for v0.1. Redirect to `/time-entries` as placeholder.
