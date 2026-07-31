# F08 — Customers

## Pillar & Purpose

| Field | Value |
|-------|-------|
| **Pillar** | Structure (commercial) |
| **Purpose** | The commercial counterparty — makes work attributable to *who pays*, so hours and costs resolve to an economy, not just a list. |
| **Answers** | "What does the work cost and earn?" (Q3) |
| **Vision ref** | [[VISION]] §3 (Customers ↔ contracted work edge), §5 |
| **Decision ref** | [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-007 — Activity Ontology]] D-3 (commercial chain) |
| **Surface** | Economics → Customers (`/customers`), per [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]] D-1 |

## Overview

Customers are the organizations Hourglass orgs do work for — including the **internal customer** (the org itself) that anchors non-commercial work without fake scaffolding. Customer records are lean: identity and contact, not billing machinery.

## User Stories

| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As a finance user, I can create/edit customers with contact details so that contracts have a counterparty | ✅ Implemented | |
| US-002 | As a user, I can search and filter the customer list so that large portfolios stay navigable | ✅ Implemented | |
| US-003 | As a finance user, I see the internal customer clearly marked and its locked fields protected so that internal work stays correctly attributed | ✅ Implemented | |
| US-004 | As a finance user, I cannot delete a customer with active contracts so that commercial history is preserved | ✅ Implemented | |

## Acceptance Criteria

- [x] Customer CRUD (name, contact, email, phone, VAT, address)
- [x] Internal customer flag with visual badge and non-editable fields
- [x] Delete returns 409 when active contracts exist
- [ ] Backfill: full AC list from legacy docs

## Boundaries

- Does not: issue invoices or compute billing — Hourglass produces *trusted data* for those systems ([[VISION]] §8).
- Does not: give customers a login surface — the customer *role* exists, but its read-only surface is unspecced (open ADR candidate, [[VISION]] §3).
- Does not: decompose work — customers link to work only through contracts ([[F09-Contracts]]) and the derived commercial chain ([[ADR-P-007 — Activity Ontology]] D-3).

## Related

- [[VISION]] · [[ADR-P-002 — Four Pillars & Feature Purposes]] · [[ADR-P-007 — Activity Ontology]] · [[ADR-P-011 — Information Architecture & Role-Scoped Surfaces]]
- Feature docs: [[F09-Contracts]] · Schema: [[S01-Database-ERD]] · [[S04-API-Contracts]]

## Status

- **Status:** Implemented (v0.1)
- **Last updated:** 2026-07-30
