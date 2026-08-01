# README Claim-to-Vault Trace (M001/S01)

Every claim the README narrative makes must resolve to a real vault file. This
document is the machine-check contract that S04 verification consumes: a claim
in `README.md` without a row here (or a row here whose vault source does not
exist) is a defect.

## Fixed section outline (binding for T02)

The README top half (everything above `## Tech stack`) follows this exact
section order:

1. **Hero mission sentence** — one sentence in plain customer language
2. **The problem** — 2–3 short paragraphs
3. **The three questions** — verbatim, one line each, with the audience each serves
4. **The four pillars** — one plain sentence each (pillar word + plain-language gloss)
5. **Hourglass for adopters**
   - *What you get* — v0.1 features only (F05–F13), grouped by pillar
   - *What it is not* — non-goals: no chat, no task boards, no payroll/HR machinery
   - *What is coming* — vision features V1–V8, explicitly labeled planned, not in v0.1
6. **Hourglass for employees**
   - *The daily loop* — capture work, submit, manager approves, finance confirms, see status anytime
   - *One worked example* — engineer logs 6 hours against a project activity on Tuesday
7. **Roadmap** — v0.1 (Capture + Structure + Control) then the vision path, each marked as direction, not current scope
8. **For developers** — short transition marker into the technical half

## Flag semantics

| Flag | Meaning |
|------|---------|
| `v0.1` | The claim describes a capability that ships in v0.1 (features F05–F13). |
| `vision` | The claim describes the mission, problem framing, the three questions, the pillar model, a permanent out-of-scope boundary, or a planned feature (V1–V8) that is **not** in v0.1. |

A row is flagged `vision` whenever the README claim could be misread as "ships
today" but must not be. The README must never present a `vision` row as current
scope.

## Claim table

| Claim | Vault source | Pillar | v0.1 or vision |
|-------|--------------|--------|----------------|
| Hourglass removes the meta-work around work — so every person knows what to do, every project knows where it stands, and every contract knows what it's worth | `hourglass-vault/VISION.md` §2 (Mission) | All (mission) | vision |
| The meta-work around work — recording what happened, chasing approvals, reconciling numbers — is what Hourglass removes | `hourglass-vault/VISION.md` §1, §2 | All (mission) | vision |
| Hourglass is a work operations platform, not just a time-and-expense tracker | `hourglass-vault/VISION.md` §1 | All | vision |
| The product exists to answer "What should I be working on?" — direction for people | `hourglass-vault/VISION.md` §3 (Q1) | Capture | vision |
| The product exists to answer "Is the work on track?" — steering for managers | `hourglass-vault/VISION.md` §3 (Q2) | Control | vision |
| The product exists to answer "What does the work cost and earn?" — economics for finance | `hourglass-vault/VISION.md` §3 (Q3) | Insight | vision |
| Four pillars organize every feature: Capture, Structure, Control, Insight, each answering the questions using data from the pillars below it | `hourglass-vault/VISION.md` §4; `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md` | All | vision |
| Capture pillar: record the ground truth of work, once, at the source | `hourglass-vault/VISION.md` §4; `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md` | Capture | v0.1 |
| Structure pillar: map who exists and what they are working on | `hourglass-vault/VISION.md` §4; `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md` | Structure | v0.1 |
| Control pillar: decide who approves and who sees what | `hourglass-vault/VISION.md` §4; `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md` | Control | v0.1 |
| Insight pillar: turn captured data into steering signals | `hourglass-vault/VISION.md` §4; `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md` | Insight | vision |
| v0.1 is Capture + Structure + Control; Insight ships as exports only | `hourglass-vault/VISION.md` §4 (key insight) | All | v0.1 |
| Employees log hours against an activity, one entry per activity per date, captured at the source | `hourglass-vault/01-Features/F11-Time-Entries.md`; `hourglass-vault/VISION.md` §5 | Capture | v0.1 |
| Time entries become trusted through a two-stage approval chain (manager, then finance) | `hourglass-vault/01-Features/F11-Time-Entries.md`; `hourglass-vault/VISION.md` §5 (approval workflows) | Control | v0.1 |
| Entries move through a status flow: draft, submitted, pending manager, pending finance, approved or rejected — visible to the employee anytime | `hourglass-vault/01-Features/F11-Time-Entries.md` (status machine) | Control | v0.1 |
| Approved and rejected entries are immutable; rejected entries show a reason and can be corrected | `hourglass-vault/01-Features/F11-Time-Entries.md` (US-002) | Capture | v0.1 |
| Employees claim expenses — mileage, meal, accommodation, and more — with amounts and receipt upload, captured with the same fidelity as time | `hourglass-vault/01-Features/F12-Expenses.md`; `hourglass-vault/VISION.md` §5 | Capture | v0.1 |
| Expenses route through the same two-stage approval chain as time entries | `hourglass-vault/01-Features/F12-Expenses.md` | Control | v0.1 |
| Org hierarchy maps who is accountable for whom, with multi-unit membership; managers see their subtree's data | `hourglass-vault/01-Features/F07-Org-Hierarchy-Units.md`; `hourglass-vault/VISION.md` §5 | Structure | v0.1 |
| Activities are recursive work containers at any granularity, with internal (non-commercial) work first-class | `hourglass-vault/01-Features/F10-Activities.md`; `hourglass-vault/decisions/project/ADR-P-007 — Activity Ontology.md` | Structure | v0.1 |
| Contracts bind a customer to a scope of work and its economics, and are the source of the billability default | `hourglass-vault/01-Features/F09-Contracts.md`; `hourglass-vault/VISION.md` §5 | Structure | v0.1 |
| Customers are the commercial counterparty — including the internal customer that anchors non-commercial work | `hourglass-vault/01-Features/F08-Customers.md` | Structure | v0.1 |
| Governance models define whose approval counts per activity: creator-controlled, unanimous, or majority | `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md`; `hourglass-vault/VISION.md` §5 | Control | v0.1 |
| An organization and its admin account are created in one atomic step | `hourglass-vault/01-Features/F05-Org-Bootstrap.md` | Control | v0.1 |
| Admins invite members via code or link; invitations expire and invitees arrive as employees by default | `hourglass-vault/01-Features/F06-Invitation-System.md` | Control | v0.1 |
| Exports hand trusted data outward: timesheet, expense, and combined reports as CSV/XLSX, date-ranged and role-scoped | `hourglass-vault/01-Features/F13-Exports.md`; `hourglass-vault/VISION.md` §5, §8 | Insight | v0.1 |
| No chat: Hourglass references work, it does not host conversation about work | `hourglass-vault/VISION.md` §8 (chat anchor); `hourglass-vault/decisions/project/ADR-P-006 — Out-of-Scope Enforcement.md` (rejection log: comment threads) | Control (boundary) | vision |
| No task boards: Hourglass is not a Jira board, sprint planner, or Kanban tool — tickets track demand, not execution | `hourglass-vault/VISION.md` §8 (task execution anchor); `hourglass-vault/decisions/project/ADR-P-006 — Out-of-Scope Enforcement.md` (rejection log: kanban board); `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` | Control (boundary) | vision |
| No payroll or HR machinery: Hourglass produces trusted data for those systems; it does not become them | `hourglass-vault/VISION.md` §8 (payroll / HR management anchors); `hourglass-vault/decisions/project/ADR-P-006 — Out-of-Scope Enforcement.md` (rejection log: invoice/payroll generation, work permits & holidays); `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` | Control (boundary) | vision |
| V1 Today view / priorities: one screen answering "what now" from tickets, projects, and pending approvals | `hourglass-vault/VISION.md` §6 (V1) | Insight | vision |
| V2 Tickets / request tracking (internal + external): the missing capture layer for demand | `hourglass-vault/VISION.md` §6 (V2); `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` | Capture | vision |
| V3 Employee knowledge at a glance: skills, current load, and project history per person | `hourglass-vault/VISION.md` §6 (V3) | Structure | vision |
| V4 Project knowledge maps, evolutions, and change requests: living state for projects, scope changes as first-class events | `hourglass-vault/VISION.md` §6 (V4) | Structure | vision |
| V5 Contract helper / pricing analytics: "what did similar work actually cost us?" from captured history | `hourglass-vault/VISION.md` §6 (V5) | Insight | vision |
| V6 Real-time project finance overview: burn versus budget, live | `hourglass-vault/VISION.md` §6 (V6) | Insight | vision |
| V7 Outcome knowledge capture: entries record what was made, learnt, taught, or shown | `hourglass-vault/VISION.md` §6 (V7) | Capture | vision |
| V8 Company knowledge wiki: an LLM/RAG/knowledge-graph query surface over captured outcomes, inside Hourglass | `hourglass-vault/VISION.md` §6 (V8), §8 (knowledge-product anchor) | Insight | vision |
| Vision features V1–V8 are planned, not in v0.1 — the roadmap marks each as direction, not current scope | `hourglass-vault/VISION.md` §6 (phase-fit column) | All | vision |
| The daily loop: capture work, submit, manager approves, finance confirms, see status anytime | `hourglass-vault/01-Features/F11-Time-Entries.md`; `hourglass-vault/01-Features/F12-Expenses.md`; `hourglass-vault/VISION.md` §3 (stakeholder map: management and finance edges) | Control | v0.1 |
| Worked example: an engineer logs 6 hours against a project activity on Tuesday, submits, manager approves, finance confirms — no spreadsheets, no chasing | `hourglass-vault/01-Features/F11-Time-Entries.md` (status machine, US-001/US-003) | Capture | v0.1 |
| Roadmap v0.1: Capture + Structure + Control, with exports as the first Insight capability | `hourglass-vault/VISION.md` §4 (key insight), §5 | All | v0.1 |
| Roadmap vision path: Today view, tickets, knowledge profiles, project maps, pricing analytics, live project finance | `hourglass-vault/VISION.md` §6 (V1–V6) | Insight | vision |
| For developers: Hourglass is a full-stack application with a Go backend and a React frontend | `README.md` (`## Tech stack` onward); `hourglass-vault/VISION.md` §9 (relationship to existing docs) | n/a | v0.1 |

## Source file existence check (S04 will re-verify)

Vault sources referenced above:

- `hourglass-vault/VISION.md`
- `hourglass-vault/01-Features/F05-Org-Bootstrap.md`
- `hourglass-vault/01-Features/F06-Invitation-System.md`
- `hourglass-vault/01-Features/F07-Org-Hierarchy-Units.md`
- `hourglass-vault/01-Features/F08-Customers.md`
- `hourglass-vault/01-Features/F09-Contracts.md`
- `hourglass-vault/01-Features/F10-Activities.md`
- `hourglass-vault/01-Features/F11-Time-Entries.md`
- `hourglass-vault/01-Features/F12-Expenses.md`
- `hourglass-vault/01-Features/F13-Exports.md`
- `hourglass-vault/decisions/project/ADR-P-002 — Four Pillars & Feature Purposes.md`
- `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md`
- `hourglass-vault/decisions/project/ADR-P-006 — Out-of-Scope Enforcement.md`
- `hourglass-vault/decisions/project/ADR-P-007 — Activity Ontology.md`
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md`

**README constraints enforced by this trace:**

- No links into `hourglass-vault/` from `README.md` (the vault is internal; S04 machine-checks this).
- Blocklisted jargon absent from `README.md`: "steering test", "edges" (stakeholder-map sense), "ADR", "stakeholder map".
- Every `v0.1` row above must be supported by a shipped feature (F05–F13, all "Implemented" except F10 which ships with the P-007 migration — activities and working groups are v0.1 structure).

## Mapping note on F10 (activities)

`F10-Activities.md` is marked "In progress (ships big-bang pre-deploy, P-007 D-6)".
The README claims activities as v0.1 Structure because the activity ontology is
scheduled to land with the v0.1 pre-deployment migration (`VISION §9`). T02 must
phrase the activities claim in shipped terms (recursive work containers, first-class
internal work) without asserting UI surfaces that do not exist yet.
