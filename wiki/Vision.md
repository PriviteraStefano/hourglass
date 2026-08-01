# Hourglass — Vision

**Hourglass removes the meta-work around work** — so every person knows what to do, every project knows where it stands, and every contract knows what it's worth.

Hourglass is a work operations platform, not just a time-and-expense tracker. It records what actually happened at work, once and at the source, and turns it into trusted data. This page explains why Hourglass exists, what v0.1 actually ships, and what is planned after it. Anything labeled **planned, not in v0.1** is future direction — it is not available today.

_Back to [Home](Home.md) · [FAQ](FAQ.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

## The problem

Work generates meta-work: recording what happened, chasing approvals, answering status questions, reconciling numbers at month end. In most organizations that meta-work lives in spreadsheets, email threads, and last-minute report-building — and nobody sees the same picture twice.

Hourglass exists so the data of work is captured once, at the source, and every question reads from that same record. No re-entry, no chasing, no cost surprises.

## The three questions

Everything in Hourglass exists to answer three questions, from captured data:

1. **"What should I be working on?"** — direction for people
2. **"Is the work on track?"** — steering for managers
3. **"What does the work cost and earn?"** — economics for finance

## The four pillars

Every feature belongs to one of four pillars. Each pillar answers the questions above using data from the pillars below it:

| Pillar | Purpose | In v0.1 |
|--------|---------|---------|
| **Capture** | Work is recorded once, where it happens | Time entries, expenses, receipts |
| **Structure** | A map of who works on what | Org hierarchy, units, working groups, activities, contracts, customers |
| **Control** | Who approves and who sees what | Roles, the approval chain, governance models, invitations |
| **Insight** | Steering signals from captured data | Exports — the only Insight feature in v0.1 |

The order is intentional. Insight is nearly empty in v0.1 because there is no insight without ground truth: first the work is captured, then it is structured, then controlled, and only then does it produce steering signals. **v0.1 is Capture + Structure + Control, with exports as the first Insight capability.**

## What v0.1 ships

v0.1 delivers the four pillars working together:

- **Capture** — employees log hours against an activity, one entry per activity per date. Approved and rejected entries are final; a rejected entry shows the reason and can be corrected. Employees claim expenses — mileage, meal, accommodation, and more — with amounts and receipt upload.
- **Structure** — the org hierarchy maps who is accountable for whom, with multi-unit membership, so managers see their part of the tree. Activities are work containers at any granularity, with internal (non-commercial) work first-class. Contracts bind a customer to a scope of work and its economics, and are the source of the billability default. Customers are the commercial counterparty — including the internal customer that anchors non-commercial work.
- **Control** — time entries and expenses become trusted through a two-stage approval chain: manager, then finance. Entries move through a status flow — draft, submitted, pending manager, pending finance, approved or rejected — visible to the employee anytime. Governance models define whose approval counts: creator-controlled, unanimous, or majority. An organization and its admin account are created in one atomic step, and admins invite members via code or link.
- **Insight** — exports hand trusted data outward: timesheet, expense, and combined reports as CSV or XLSX, date-ranged and role-scoped.

## What is coming — planned, not in v0.1

The vision path builds on the data v0.1 captures. **None of this is in v0.1** — it is direction, not current scope:

- **V1 Today view** — one screen answering "what now" from tickets, projects, and pending approvals. *Planned, not in v0.1.*
- **V2 Tickets** — request tracking for internal and external demand. *Planned, not in v0.1.*
- **V3 Knowledge profiles** — skills, current load, and project history per person. *Planned, not in v0.1.*
- **V4 Project knowledge maps** — living state for projects, scope changes as first-class events. *Planned, not in v0.1.*
- **V5 Pricing analytics** — "what did similar work actually cost us?" from captured history. *Planned, not in v0.1.*
- **V6 Live project finance** — burn versus budget, in real time. *Planned, not in v0.1.*
- **V7 Outcome capture** — entries record what was made, learnt, taught, or shown. *Planned, not in v0.1.*
- **V8 Company knowledge wiki** — a query surface over captured outcomes, inside Hourglass. *Planned, not in v0.1.*

The sequence is deliberate, not arbitrary: each step depends on data the earlier steps capture — demand tracking before a "today" view, history before pricing analytics, outcomes before a company knowledge wiki.

## What Hourglass is not

Hourglass stays deliberately narrow. It is not:

- **A chat tool** — it references work; it does not host conversation about work
- **A task board** — no Jira board, sprint planner, or Kanban tool; tickets track demand, not execution
- **Payroll or HR machinery** — Hourglass produces trusted data for those systems; it does not become them

---

_Back to [Home](Home.md) · [FAQ](FAQ.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

_Source: README.md, the internal product vision (VISION.md), and the claim trace (docs/README-claim-trace.md), 2026-08-01._
