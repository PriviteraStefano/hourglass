# Phase Pg-2: PostgreSQL adapters - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-07
**Phase:** Pg-2-Adapters
**Areas discussed:** Repository test strategy

---

## Repository Test Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Defer (as design doc says) | Don't write PG repo tests in Pg-2. Remove SurrealDB tests in Pg-3. Tests get their own phase post-migration. | |
| Write tests in Pg-2 | Write PostgreSQL repo tests alongside each new repo. More work in Pg-2 but tests aren't lost. | ✓ |

**User's choice:** Write tests in Pg-2
**Notes:** Write PostgreSQL repository tests alongside each new repository. 14 of 29 SurrealDB files are test files — these will be deleted in Pg-3. Writing PG tests in Pg-2 avoids losing coverage.

---

## Deferred Ideas

- None — discussion stayed within phase scope
