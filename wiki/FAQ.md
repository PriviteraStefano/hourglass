# Hourglass — Frequently Asked Questions

Quick answers about what Hourglass is, what v0.1 includes, and what comes after. Every answer here reflects the current scope — anything planned after v0.1 is labeled as such.

_Back to [Home](Home.md) · [Vision](Vision.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

### Is Hourglass just a time tracker?

No. Hourglass is a work operations platform, not just a time-and-expense tracker. Time tracking is only the first layer (Capture). The product records what actually happened at work, once and at the source, and turns it into trusted data that answers three questions at any moment: **what should I be working on**, **is the work on track**, and **what does the work cost and earn**.

### What is in v0.1?

v0.1 is **Capture + Structure + Control**, with exports as the first Insight capability. Concretely: time entries, expenses with receipt upload, the org hierarchy, activities, contracts, customers, governance models, invitations, and exports (timesheet, expense, and combined reports as CSV or XLSX, date-ranged and role-scoped).

### How do approvals work?

Time entries and expenses move through a two-stage approval chain: a **manager approves first**, then **finance confirms**. You can watch an entry's status at any time — draft, submitted, pending manager, pending finance, approved or rejected.

### Can I edit an approved entry?

No. Approved entries are final. A **rejected** entry shows the reason for rejection and can be corrected, so a mistake never silently sticks.

### What roles exist, and who sees what?

Four roles: **employee**, **manager**, **finance**, and **customer**. Roles determine what you can see and which approval step you perform. Visibility follows the org hierarchy: managers see the data for the people under them, employees see their own entries, and finance sees what the approval chain routes to them. Exports are role-scoped on top of that.

### Can I export my data?

Yes. v0.1 exports trusted data outward — timesheet, expense, and combined reports as CSV or XLSX, with date ranges and role scoping applied. This is the data Hourglass hands to the systems that need it.

### Does Hourglass replace payroll or HR?

No. Hourglass produces **trusted data** for payroll and HR systems via exports; it does not become them. It does not do payroll math, leave balances, or HR management.

### Is Hourglass a chat tool or a task board?

No. Hourglass references work — it does not host conversation about work. It is also not a Jira board, sprint planner, or Kanban tool: tickets track demand, not execution, and tickets are a planned feature, not part of v0.1.

### Do we host it for you?

No. Hourglass is self-hosted software: you run it in your own environment, and the repository includes Docker and docker-compose to make that easy. There is no hosted trial yet — to try it, follow the [Getting started](Getting-Started.md) page or the quickstart in the [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md).

### What is coming after v0.1?

The vision roadmap — each item **planned, not in v0.1**: a Today view, tickets, knowledge profiles, project knowledge maps, pricing analytics, live project finance, outcome capture, and a company knowledge wiki. Each step builds on the data v0.1 captures. See the [Vision](Vision.md) page for the full list.

---

_Back to [Home](Home.md) · [Vision](Vision.md) · [README](https://github.com/PriviteraStefano/hourglass/blob/main/README.md)_

---

_Source: README.md, the internal product vision (VISION.md), the feature specifications (F05–F13), and the claim trace (docs/README-claim-trace.md), 2026-08-01._
