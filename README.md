# Hourglass

**Hourglass removes the meta-work around work** — so every person knows what to do, every project knows where it stands, and every contract knows what it's worth.

Hourglass is a work operations platform, not just a time-and-expense tracker. It records what actually happened at work, once and at the source, and turns it into trusted data that answers three questions at any moment.

---

## The problem

Work generates meta-work: recording what happened, chasing approvals, answering status questions, reconciling numbers at month end. In most organizations that meta-work lives in spreadsheets, email threads, and last-minute report-building — and nobody sees the same picture twice.

Hourglass exists so the data of work is captured once, at the source, and every question reads from that same record. No re-entry, no chasing, no cost surprises.

---

## The three questions

Everything in Hourglass exists to answer three questions, from captured data:

1. **"What should I be working on?"** — direction for people
2. **"Is the work on track?"** — steering for managers
3. **"What does the work cost and earn?"** — economics for finance

---

## The four pillars

Every feature belongs to one of four pillars, each answering the questions above using data from the pillars below it:

- **Capture** — work is recorded once, where it happens: time entries and expenses
- **Structure** — a map of who works on what: org units, activities, contracts, customers
- **Control** — who approves and who sees what: roles, governance, the approval chain
- **Insight** — steering signals from captured data: exports today, dashboards coming

v0.1 is **Capture + Structure + Control**, with exports as the first Insight capability.

---

## Hourglass for adopters

### What you get

v0.1 ships all four pillars working together:

- **Capture** — Employees log hours against an activity, one entry per activity per date, captured at the source. Approved and rejected entries are immutable; rejected entries show a reason and can be corrected. Employees claim expenses — mileage, meal, accommodation, and more — with amounts and receipt upload, captured with the same fidelity as time.
- **Structure** — The org hierarchy maps who is accountable for whom, with multi-unit membership, so managers see their subtree's data. Activities are recursive work containers at any granularity, with internal (non-commercial) work first-class. Contracts bind a customer to a scope of work and its economics, and are the source of the billability default. Customers are the commercial counterparty — including the internal customer that anchors non-commercial work.
- **Control** — Time entries and expenses become trusted through a two-stage approval chain: manager, then finance. Entries move through a status flow — draft, submitted, pending manager, pending finance, approved or rejected — visible to the employee anytime. Governance models define whose approval counts per activity: creator-controlled, unanimous, or majority. An organization and its admin account are created in one atomic step, and admins invite members via code or link.
- **Insight** — Exports hand trusted data outward: timesheet, expense, and combined reports as CSV or XLSX, date-ranged and role-scoped.

**Trying it:** the [Getting started](#getting-started) quickstart below runs the whole stack in a few commands.

### What it is not

Hourglass stays deliberately narrow. It is not:

- **A chat tool** — it references work; it does not host conversation about work
- **A task board** — no Jira board, sprint planner, or Kanban tool; tickets track demand, not execution
- **Payroll or HR machinery** — Hourglass produces trusted data for those systems; it does not become them

### What is coming

Planned after v0.1 — **not in v0.1**:

- **V1 Today view** — one screen answering "what now" from tickets, projects, and pending approvals
- **V2 Tickets** — request tracking for internal and external demand
- **V3 Knowledge profiles** — skills, current load, and project history per person
- **V4 Project knowledge maps** — living state for projects, scope changes as first-class events
- **V5 Pricing analytics** — "what did similar work actually cost us?" from captured history
- **V6 Live project finance** — burn versus budget, in real time
- **V7 Outcome capture** — entries record what was made, learnt, taught, or shown
- **V8 Company knowledge wiki** — a query surface over captured outcomes, inside Hourglass

---

## Hourglass for employees

### The daily loop

1. **Capture** your work — hours against an activity, expenses as they happen
2. **Submit** — your entries leave your desk
3. **Manager approves** — one check, with a reason if anything is rejected
4. **Finance confirms** — the second stage of the chain closes
5. **See status anytime** — draft, submitted, pending manager, pending finance, approved or rejected, always visible

### One worked example

An engineer logs 6 hours against a project activity on Tuesday and submits. The manager approves; finance confirms. Done — no spreadsheets, no chasing.

That is the whole loop for an employee: record once, submit, and watch the status move until the entry is approved.

---

## Roadmap

**v0.1 — Capture + Structure + Control.** Time entries, expenses, the org hierarchy, activities, contracts, customers, governance models, invitations, and the exports that make the data useful elsewhere. This is what ships today.

**The vision path — direction, not current scope.** Today view, tickets, knowledge profiles, project maps, pricing analytics, live project finance. Each step builds on the data v0.1 captures; none of it is in v0.1.

**More on the wiki:** the [Home](wiki/Home.md) page introduces the four pillars, with deeper guides for [getting started](wiki/Getting-Started.md), [adopters](wiki/Adopters.md), [employees](wiki/Employees.md), the [vision](wiki/Vision.md), and the [FAQ](wiki/FAQ.md).

---

## For developers

Hourglass is a full-stack application with a Go backend and a React frontend. If you are evaluating or contributing, the technical documentation — stack, architecture, setup, and configuration — starts below, and the full developer reference — every environment variable, Makefile target, testing command, and the domain model — lives on [wiki/Developer.md](wiki/Developer.md).

## Tech stack

| Layer     | Technology                                                              |
|-----------|-------------------------------------------------------------------------|
| Backend   | Go 1.26.1, standard library `net/http`, hexagonal (ports & adapters)    |
| Frontend  | React 19, TanStack Router v1, TanStack React Query v5, Vite, TypeScript |
| Styling   | Tailwind CSS v4, shadcn/ui                                              |
| Database  | PostgreSQL 15                                                           |
| Auth      | JWT (`golang-jwt/jwt/v5`), bcrypt (`golang.org/x/crypto`)               |
| Testing   | `stretchr/testify`, `testcontainers-go`, Vitest, Playwright             |
| Container | Docker (multi-stage), docker-compose                                    |

---

## Project structure

```
hourglass/
├── cmd/             # Server entry, migration CLI, schema tooling
├── internal/
│   ├── core/        # Domain, ports, application services
│   ├── adapters/    # primary/http (handlers) + secondary/postgres (repos)
│   └── …            # auth, cookies, db, handlers, middleware, models
├── pkg/api/         # Shared response envelope
├── migrations/      # SQL migrations (*.up.sql / *.down.sql)
├── web/             # React frontend (Vite + TanStack)
├── Makefile
└── docker-compose.yml
```

This is the shallow view — the full per-directory tree lives on [wiki/Developer.md](wiki/Developer.md).

---

## Getting started

### Prerequisites

- **Go** >= 1.26.1
- **Node.js** / **Bun** (for the frontend)
- **PostgreSQL** >= 15 (or use the bundled docker-compose service)
- **Docker** + **docker-compose** (optional, for containerized runs)

### 1. Clone

```bash
git clone https://github.com/PriviteraStefano/hourglass.git
cd hourglass
```

### 2. Run with Docker (easiest)

```bash
make docker-up          # starts postgres + app on :8080
make docker-down        # stop
```

### 3. Local development

**Backend** (terminal 1):

```bash
docker-compose up -d postgres   # start only PostgreSQL (or make docker-up for the full stack in containers)
make migrate-up                 # apply migrations (go run ./cmd/migrate -up -dir migrations)
make run                        # http://localhost:8080
```

**Frontend** (terminal 2):

```bash
cd web
bun install
bun run dev             # http://localhost:3000 (proxies /api → :8080)
```

---

## Configuration

Local development needs only two variables; the full reference — every backend
and frontend environment variable, Makefile target, testing command, and the
domain model — lives on [wiki/Developer.md](wiki/Developer.md).

| Variable       | Description                 | Default                                             |
|----------------|-----------------------------|-----------------------------------------------------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://hourglass:hourglass@localhost:5432/...` |
| `JWT_SECRET`   | JWT signing key (**required in production**) | `dev-secret-change-in-production`                   |

---

## License

Proprietary — see [LICENSE](./LICENSE).

This software is the proprietary property of Stefano Privitera. No license to
use, copy, modify, or distribute is granted without prior written permission.
Anyone granted permission to evaluate or use the software **must report usage
and any issues** to the copyright holder. See the [LICENSE](./LICENSE) file for
full terms.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
