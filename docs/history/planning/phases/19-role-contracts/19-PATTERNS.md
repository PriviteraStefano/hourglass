# Phase 19 - Authoring Pattern for Role Contracts

**Purpose:** the single authoring contract every `docs/design/workflows/*.md` file in this phase follows. Derived from `docs/design/LANGUAGE.md`, `docs/design/CHROME.md` (both verified in Phase 17/18), and decisions D-19-01..12.

---

## 1. Authority stack (do not invert)

```
LANGUAGE.md   wins: type, color, density, motion, status vocabulary
CHROME.md     wins: frame, sidebar, four lifecycle nav groups, page anatomy
workflows/*   add:  jobs, surfaces, copy, composition — nothing else
```

A workflow contract `MUST NOT` restate the nav model, restyle chrome, invent tokens or values, or claim frozen components (`PageHeader`, `FilterBar`, `DataTable`, `EmptyState`, `ConfirmDialog`) exist. They are absent inputs.

## 2. File skeleton (same order in every contract)

```markdown
# Hourglass <Role> Role Contract

## Purpose
- what this contract is, who its actor is, authority split line (LANGUAGE/CHROME win)
- RFC 2119 note (MUST/MUST NOT conformance, SHOULD defaults, rationale in prose)

## Changelog
- 2026-09-26 · Phase 19 · Added <role> role contract

## Actor
- membership role string, how it is obtained (/auth/me → memberships), org switching
- derived hats this role may also hold (WG manager/delegate) — named as hats, never roles
- what this role is NOT (the negative space, one line each)

## Scope
- per-role visibility rows (what this role sees, what it never sees)
- explicit "not authorization" caveat (UX scoping; server stays authoritative)

## Jobs
### <ID> — <Title>
| field | value |
|---|---|
| Actor | role / role + hat |
| Trigger | what starts the job |
| Steps | numbered, ≤6, workflow-shaped |
| Surfaces | named surfaces (target names, not routes) |
| API interactions | `METHOD /path` (+ `absent` where none exists) |
| Current state | exists | partial | absent — with one evidence pointer |
| Backend authority | what the server enforces today; note when enforcement is missing |

## Do / don't
## Pointers
## Not in this file
## Gaps
```

## 3. Job record rules

- **Ids are stable** and namespaced: `E-` employee · `M-` manager · `F-` finance · `H-` hr · `C-` customer. Phase 20 cites them; never renumber.
- **Jobs are workflows, not routes.** A job may span several surfaces; a route never becomes a job.
- **API interactions are real**: every `METHOD /path` must exist in `cmd/server/main.go` wiring; when nothing exists, write `absent` (with the gap id).
- **Current state** is `exists` / `partial` / `absent`, each with one evidence pointer (route path or file:line). This is the only place live routes appear.
- **Backend authority** distinguishes "server-enforced" from "UX scoping only" and names the gap (`G1`…`G11`) when enforcement is missing.
- **Negative space is mandatory**: for each role, name what the role cannot do and who does it instead.
- Copy may name surfaces; it `MUST NOT` specify URLs, page layouts, or component APIs.

## 4. Cross-file invariants (checked mechanically)

| Invariant | Check |
|---|---|
| Five contracts exist + README | `ls docs/design/workflows/{employee,manager,finance,hr,customer}.md README.md` |
| Stable ids present | `grep -c '^### [EMFHC]-[0-9][0-9]' per file` ≥ planned count |
| No route-as-job | no job heading contains a `/`-path |
| No Admin/Settings | `grep -i 'admin' docs/design/workflows/*.md` returns only explicit fence lines |
| No frozen-component existence claim | `grep -E '(PageHeader|FilterBar|DataTable|EmptyState|ConfirmDialog).*(exists|present|at `web/src)' ` empty |
| No value tables | `grep -E 'oklch\(|#[0-9a-f]{6}|[0-9]+px' docs/design/workflows/*.md` empty |
| Authority line present | every file contains "LANGUAGE.md" and "CHROME.md" |
| Changelog line present | every file contains "2026-09-26 · Phase 19" |
| Gap register mirrored | every `G1`…`G11` id appears in `README.md` |
| Customer concludes no surface | `customer.md` contains "no app surface" |

## 5. Voice and formatting

- RFC 2119 selectively: `MUST`/`MUST NOT` for conformance, `SHOULD` for defaults, prose for rationale.
- One idea per bullet; no marketing copy; no "simply/just/obviously".
- Cite code by repo-root path, never by line-number ranges that will rot (a single `file:line` for evidence is fine).
- Tables for job fields and scope rows; prose elsewhere.
- The `Not in this file` fence is mandatory in every contract; the `Gaps` section may be empty only if the role genuinely has none (HR and customer never do).
