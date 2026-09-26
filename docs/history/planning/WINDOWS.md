---
schema_version: 1
open_count: 0
waived_count: 0
fixed_count: 1
total_count: 1
last_updated: 2026-08-24T00:00:00.000Z
---

# Broken Windows Ledger

> Cross-phase defect register. `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.
>
> v0.2 close-gate item 2 (Window #1) **fixed 2026-08-24** via a dedicated
> `oxfmt` sweep across the 29 repo-wide drift files (base `4189c00`). Not
> caused by and not credited to Phase 15. `bun run fmt:check` now passes (0
> issues / 205 files). `/gsd-ship` no longer blocked by this window.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 15 | unrun-verify | web/src/index.css |  | Phase gate bun run fmt:check cannot pass: 31 pre-existing oxfmt-drift files repo-wide (index.css, sidebar, working-groups dialogs, etc.) proven failing at base commit 4189c00; 8 plan-owned files formatted in 72f2e85; full sweep deferred | fixed | Dedicated oxfmt sweep on 2026-08-24 formatted all 29 remaining drift files; `bun run fmt:check` now passes (0/205). Not Phase 15's doing. | 2026-08-17T09:32:42.939Z | 2026-08-24T00:00:00.000Z |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "15",
    "file": "web/src/index.css",
    "line": null,
    "description": "Phase gate bun run fmt:check cannot pass: 31 pre-existing oxfmt-drift files repo-wide (index.css, sidebar, working-groups dialogs, etc.) proven failing at base commit 4189c00; 8 plan-owned files formatted in 72f2e85; full sweep deferred",
    "status": "fixed",
    "reason": "Dedicated oxfmt sweep on 2026-08-24 formatted all 29 remaining drift files; bun run fmt:check now passes (0/205). Not Phase 15's doing.",
    "recorded_at": "2026-08-17T09:32:42.939Z",
    "resolved_at": "2026-08-24T00:00:00.000Z"
  }
]
````
