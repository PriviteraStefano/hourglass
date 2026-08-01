# Hourglass — For Adopters

Hourglass is a work operations platform, not just a time-and-expense tracker. If you are evaluating it for your organization, this page tells you what v0.1 actually ships, how roles and approvals work, how Hourglass keeps the data secure, what it deliberately does not do, and how it runs. Anything labeled **planned, not in v0.1** is future direction — it is not available today.

_Back to [Home](Home.md) · [Vision](Vision.md) · [FAQ](FAQ.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

## What you get in v0.1

v0.1 ships all four pillars working together. Everything below is current scope.

### Capture — work is recorded once, where it happens

- Employees log hours against an activity — one entry per activity per date — so the record is captured at the source.
- Approved and rejected entries are final. A rejected entry shows the reason and can be corrected, so a mistake never silently sticks.
- Employees claim expenses — mileage, meal, accommodation, and more — with an amount and a receipt upload, captured with the same fidelity as time.

### Structure — a map of who works on what

- The org hierarchy is a tree of units that maps who is accountable for whom, with multi-unit membership and a designated primary unit per person. Managers see the data for the people under them.
- Activities are work containers at any granularity — from a whole program to a single task — and internal (non-commercial) work is first-class.
- Contracts bind a customer to a scope of work and its economics, and are the source of the billability default that flows down to the work under them.
- Customers are the commercial counterparty — including the internal customer that anchors non-commercial work.

### Control — who approves and who sees what

- Time entries and expenses become trusted through a two-stage approval chain: a **manager approves first**, then **finance confirms**.
- Every entry moves through a visible status flow: draft, submitted, pending manager, pending finance, approved or rejected.
- Governance models define whose approval counts: creator-controlled, unanimous, or majority.
- An organization and its admin account are created in one atomic step — no orphaned users, no half-created orgs. Admins invite members via code or link; invitations expire after 7 days and invitees arrive as employees by default, so membership stays deliberate.

### Insight — steering signals from captured data

- Exports hand trusted data outward: timesheet, expense, and combined reports as CSV or XLSX, date-ranged and role-scoped.

## Roles and visibility

Four roles exist, and they determine what you can see and which approval step you perform:

| Role | In v0.1 |
|------|---------|
| **Employee** | Records time and expenses; sees their own entries at every status |
| **Manager** | Approves the first stage of the chain; sees the data for the people under them |
| **Finance** | Confirms the second stage; sees what the approval chain routes to them |
| **Customer** | The commercial counterparty the work is done for — the org itself is the internal customer that anchors non-commercial work |

Visibility follows the org hierarchy: managers see their part of the tree, employees see their own entries, and finance sees what the approval chain routes to them. Exports are role-scoped on top of that. Employees see this loop from their side on the [Employees](Employees.md) page.

## Security posture

- Sign-in uses signed tokens held in **HttpOnly, same-site cookies** — not exposed to page scripts.
- Visibility and approval rights are enforced per role, and exports are role-scoped.
- Authentication endpoints are rate-limited to blunt brute-force attempts.
- The server accepts connections only from a configured list of allowed origins (CORS).

## What it is not

Hourglass stays deliberately narrow. It is not:

- **A chat tool** — it references work; it does not host conversation about work
- **A task board** — no Jira board, sprint planner, or Kanban tool; tickets track demand, not execution, and tickets are a planned feature, not part of v0.1
- **Payroll or HR machinery** — Hourglass produces trusted data for those systems via exports; it does not become them

## What is coming — planned, not in v0.1

The vision path builds on the data v0.1 captures: a Today view, tickets, knowledge profiles, project knowledge maps, pricing analytics, live project finance, outcome capture, and a company knowledge wiki. Each item is **planned, not in v0.1** — see the [Vision](Vision.md) page for the full list and the logic behind the order.

## Running it

Hourglass is self-hosted software: you run it in your own environment, and the repository includes Docker and docker-compose to make that easy. There is no hosted trial yet. To evaluate it, follow the [Getting started](Getting-Started.md) page or the quickstart in the [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md).

---

_Back to [Home](Home.md) · [Vision](Vision.md) · [FAQ](FAQ.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

_Source: README.md, the feature specifications (F05–F13), the claim trace (docs/README-claim-trace.md), and the repository code (cookie handling, middleware, domain models), 2026-08-01._
