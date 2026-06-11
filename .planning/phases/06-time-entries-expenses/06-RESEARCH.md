# Phase 6: Time Entries + Expenses - Research

**Researched:** 2026-06-11
**Domain:** Time entry & expense CRUD with two-stage approval workflow
**Confidence:** HIGH

## Summary

Phase 6 delivers full CRUD + two-stage approval workflow for time entries and expenses. The backend has significant existing infrastructure: `TimeEntryService`/`TimeEntryHandler` (needs two-stage approval extension), `ExpenseRepository` (needs filter methods), and full schema (`time_entry_approvals`, `expense_approvals`, `financial_cutoff_periods` tables exist). The frontend has a partial time-entries page with items-based model (needs flat-model rewrite) and no expense UI. Two-stage approval workflow (draft → submitted → pending_manager → pending_finance → approved/rejected) mirrors between both domains and routes via WG manager ID on the working group.

**Primary recommendation:** Extend existing time entry service/handler for two-stage workflow, build expense domain/service/handler from scratch following hexagonal patterns, adapt frontend MiniCalendar to client-side status computation, build expense UI with same calendar layout, and create shared approval components. Plan in 3 waves: (1) Backend domain/service/repo changes, (2) Backend HTTP handlers + route wiring + tests, (3) Frontend rewrite + expense UI.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Time entry CRUD | API / Backend | Database | Service handles validation, repo persists. Frontend renders. |
| Expense CRUD | API / Backend | Database | Same pattern — built from scratch. |
| Two-stage approval workflow | API / Backend | Database | Approve/reject logic in service, approval records in DB. |
| Month calendar statuses | Browser / Client | API / Backend | D-14: client-side computation from `GET /time-entries?month=&year=` |
| Receipt upload | API / Backend | Filesystem | Handler receives file, stores locally, sets `receipt_url` on expense. |
| Approval UI (buttons/history) | Browser / Client | — | Shared components rendered by role-aware logic. |
| Period locking | API / Backend | Database | `IsPeriodLocked` already exists for time entries; reuse for expenses. |
| Pending approvals view | Browser / Client | API / Backend | `GET /time-entries/pending` and `GET /expenses/pending` return approver-scoped lists. |

## User Constraints (from CONTEXT.md)

### Locked Decisions (D-01 through D-23)

**D-01:** Flat model — One entry per project per date. No items/sub-items. Matches existing `domain/time_entry` model and current DB schema.

**D-02:** Existing domain/service/handler extended — Not rewritten. Add `pending_manager`, `pending_finance`, `rejected` statuses. Approve handler differentiates WG manager approval (→ pending_finance) from finance approval (→ approved).

**D-03:** Flat expense model — One expense per entry. No expense items table. Matches current `expenses` DB schema.

**D-04:** 9 expense categories — mileage, meal, accommodation, parking, travel_tickets, tolls, taxi, equipment, other (from models.go, already in DB CHECK constraint).

**D-05:** Full hexagonal structure for expense — Create `domain/expense/` with proper domain types (not using models.go structs directly). Service in `services/expense/`. Handler in `http/expense.go`.

**D-06:** Receipt upload in scope — Simple `POST /expenses/{id}/receipt` file upload. Stores file, sets `receipt_url` on expense row. Local filesystem storage for MVP. No OCR, no multiple receipts, no ExpenseReceipt model table.

**D-07:** Km distance per category — Only meaningful for `mileage` category. Other categories ignore km_distance.

**D-08:** Two-stage workflow — `draft → submitted → pending_manager → pending_finance → approved/rejected`. Same for both time entries and expenses.

**D-09:** Route via project WG manager — When entry is submitted, lookup the project's working group manager(s). WG manager is first-stage approver. Finance role is second-stage.

**D-10:** Both can reject — WG manager or finance can reject at their stage (→ rejected with reason).

**D-11:** Self-approval prevention — If entry creator's user ID matches the WG manager, skip manager approval and route directly to pending_finance.

**D-12:** Full immutable approval history — Each approve/reject action creates a record in `time_entry_approvals` or `expense_approvals` tables. Current goroutine-based `CreateAuditLog` replaced with proper synchronous history.

**D-13:** Proper approval flow engine deferred — Current two-stage is hardcoded. Configurable flow engine is post-MVP.

**D-14:** Client-side monthly computation — No dedicated monthly-summary endpoint. Frontend fetches via `GET /time-entries?month=&year=` and computes calendar day-statuses client-side.

**D-15:** No submit-month endpoint — `POST /time-entries/submit-month` removed. Each entry submitted individually.

**D-16:** Adapt existing time entry UI — MiniCalendar stays as main navigation. Day shows flat list (one row per project). No more items/row.

**D-17:** Expense frontend built from scratch — Same calendar-style layout as time entries.

**D-18:** Shared approval components — `approval-buttons.tsx` and `approval-history.tsx` shared between time entries and expenses.

**D-19:** Migration: `time_entries` status CHECK — Change from `(draft, submitted, approved)` to `(draft, submitted, pending_manager, pending_finance, approved, rejected)`.

**D-20:** Migration: `expenses` status CHECK — Change from `(draft, submitted, approved, rejected)` to `(draft, submitted, pending_manager, pending_finance, approved, rejected)`.

**D-21:** Existing tables suffice — No new tables needed. `time_entry_approvals`, `expense_approvals` already exist. `expenses.receipt_url` already exists.

**D-22:** TimeEntry response includes `current_approver_role` — Add to domain model for frontend approval UI rendering.

**D-23:** Expense response mirrors TimeEntry — Same structure: id, user_id, org_id, project_id, date, status, current_approver_role, submitted_at, created_at, updated_at.

### Agent's Discretion
- Approve/reject handler response shape (API envelope)
- Receipt upload file path format and storage location
- Frontend component layout within calendar + day-detail pattern
- Mutation `onSuccess` behavior (inline feedback vs navigation)
- Test file locations within established patterns
- WG manager lookup implementation detail

### Deferred Ideas (OUT OF SCOPE)
- Proper approval flow engine (configurable workflows)
- Expense items (multi-item per expense)
- Time entry items (multi-project per entry)
- Multiple receipts per expense
- Receipt OCR processing
- Submit-month / batch operations
- Exports integration (Phase 7)
- Dashboard/analytics

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TIME-01 | Time entry list with filtering (status, date range, project, user) | `TimeEntryHandler.List` exists with dynamic query builder. Extend ListPending for two-stage. |
| TIME-02 | Create time entry (date, hours, project, subproject, WG, description) | `TimeEntryService.Create` exists and works for flat model. No changes needed. |
| TIME-03 | Edit time entry (draft/submitted only) | `TimeEntryService.Update` exists, checks `CanEdit()` (draft only). Per D-02, extend to `submitted` + two-stage. |
| TIME-04 | Submit for approval | `TimeEntryService.Submit` exists. Needs two-stage: set `status=submitted` + `current_approver_role=manager`. |
| TIME-05 | Cannot edit approved/rejected entries | `CanEdit()` check already in Update/Delete. Add new statuses to check. |
| TIME-06 | Cannot delete entries with approvals | Already enforced via `CanEdit()` check. Approvals table is immutable. |
| TIME-07 | Employee cannot self-approve | Role check in `middleware.GetRole()`. Deny if role != "manager" or "finance". |
| TIME-08 | Manager cannot approve own entries | D-11: If creator == approver, skip that stage. Check `e.UserID == userID` in Approve. |
| EXPN-01 | Expense list with filtering | Extend `ExpenseRepository.ListByOrg` → `List` with dynamic filters ala `buildTimeEntryListQuery`. |
| EXPN-02 | Create expense (date, amount, category, project, description) | `ExpenseRepository.Create` exists. Build service layer + handler. |
| EXPN-03 | Edit expense (draft/submitted only) | `ExpenseRepository.Update` exists. Add draft-only check in service. |
| EXPN-04 | Submit for approval | Analogous to time entry submit. No existing expense submit. |
| EXPN-05 | Expense categories (9) | Already in DB CHECK constraint and models.go. Reuse constants. |
| EXPN-06 | Same approval constraints as time entries | Mirror time entry two-stage workflow in expense service. |
| APPR-01 | Approval history is immutable | Tables use immutable INSERT-only pattern. No UPDATE/DELETE. |
| APPR-02 | Rejected entries show reason | `comment` column in `*_approvals` tables. Pass reason in Reject. |
| APPR-03 | Workflow: employee → submits → manager approves → finance approves/rejects | D-08/D-09: Two-stage hardcoded workflow via WG manager → finance. |
| APPR-04 | Status badge component | `StatusBadge` already supports all 6 statuses. Reuse as-is. |
| APPR-05 | Approval buttons component | Build shared `approval-buttons.tsx` with role-aware visibility. |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | 1.26.1 | HTTP routing (ServeMux with method+path patterns) | Project standard since Phase Pg-3 |
| pgx/v5 | v5.x | PostgreSQL driver | Project standard — all repos use pgxpool |
| TanStack Router | v1 | File-based routing | Project standard |
| TanStack React Query | v5 | Server state + mutations | Project standard |
| shadcn/ui | latest | UI component library | Project standard |
| Tailwind CSS | v4 | Styling | Project standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|--------------|
| `date-fns` | — | Date formatting | Existing frontend dependency for calendar |
| `uuid` (Go) | google/uuid | UUID generation | Existing dependency |
| `uuid` (JS) | — | UUID generation | For frontend expense types |
| `zod` | — | Search params validation | Already used in route `validateSearch` |
| `lucide-react` | — | Icons | Existing frontend dependency |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending `TimeEntryService` | Rewriting domain model | Rewrite risks breaking existing seed data + tests. Extension is minimal diff. |
| `models.Expense` directly | New `domain/expense` types | Per D-05, hexagonal requires domain layer. New types = more code but proper isolation. |
| Server-side monthly summary | Client-side computation | D-14: client-side avoids new endpoint, but more frontend logic. |

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  Browser (React 19 / TanStack Router)                              │
│  ┌──────────────────────┐  ┌──────────────────────────────────┐    │
│  │ MiniCalendar         │  │ Day Detail Panel                  │    │
│  │ (month statuses via  │  │ ┌─────────────┐ ┌─────────────┐ │    │
│  │  client-side compute)│  │ │ Entry/Expense│ │ Approval    │ │    │
│  │                      │  │ │ Rows        │ │ Buttons     │ │    │
│  └──────────┬───────────┘  │ └─────────────┘ └─────────────┘ │    │
│             │              │ ┌──────────────────────────────┐ │    │
│             │              │ │ Approval History (immutable) │ │    │
│             │              │ └──────────────────────────────┘ │    │
│             └──────────────┴──────────────────────────────────┘    │
│                           │                                        │
│                    TanStack React Query                            │
│              (auto-refresh on 401, cache invalidation)             │
└───────────────────────────┬────────────────────────────────────────┘
                            │ HTTP (JSON) via Vite proxy → :8080
                            ▼
┌────────────────────────────────────────────────────────────────────┐
│  Backend (Go 1.26, net/http ServeMux)                             │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ TimeEntryHandler  │  │ ExpenseHandler    │  │ Shared Middleware │  │
│  │ (extend for 2-   │  │ (build from       │  │ (Auth, Role,      │  │
│  │  stage approval)  │  │  scratch)         │  │  CORS, Logging)   │  │
│  └────────┬─────────┘  └────────┬─────────┘  └──────────────────┘  │
│           │                     │                                   │
│  ┌────────▼─────────────────────▼──────────┐                       │
│  │        Service Layer                     │                       │
│  │  ┌────────────────┐ ┌────────────────┐  │                       │
│  │  │ TimeEntryService│ │ ExpenseService│  │                       │
│  │  │ (extend for    │ │ (build from    │  │                       │
│  │  │ 2-stage + sync │ │  scratch,      │  │                       │
│  │  │ approval hist.)│ │  mirror TE)    │  │                       │
│  │  └────────┬───────┘ └───────┬────────┘  │                       │
│  └───────────┼──────────────────┼───────────┘                       │
│              │                  │                                   │
│  ┌───────────▼──────────────────▼───────────┐                       │
│  │        Repository Layer (PostgreSQL)      │                       │
│  │  ┌────────────────┐ ┌────────────────┐   │                       │
│  │  │ TimeEntryRepo   │ │ ExpenseRepo     │   │                       │
│  │  │ (minor extend)  │ │ (extend: List   │   │                       │
│  │  │                 │ │  with filters,  │   │                       │
│  │  │                 │ │  ListPending)   │   │                       │
│  │  └────────────────┘ └────────────────┘   │                       │
│  └──────────────────────────────────────────┘                       │
│                                                                     │
│  ┌──────────────────────────────────────────┐                       │
│  │  WorkingGroupRepository (for WG manager  │                       │
│  │  lookup during approval routing)          │                       │
│  └──────────────────────────────────────────┘                       │
│                                                                     │
│  ┌──────────────────────────────────────────┐                       │
│  │  Filesystem (receipts storage)            │                       │
│  │  uploads/receipts/{org_id}/{expense_id}/  │                       │
│  └──────────────────────────────────────────┘                       │
└────────────────────────────────────────────────────────────────────┘
                            │ SQL
                            ▼
┌────────────────────────────────────────────────────────────────────┐
│  PostgreSQL                                                        │
│  ┌──────────────┐ ┌──────────────┐ ┌─────────────────────────┐    │
│  │ time_entries  │ │ expenses     │ │ time_entry_approvals    │    │
│  │ (status: add  │ │ (status: add │ │ expense_approvals       │    │
│  │  pending_*)   │ │  pending_*)  │ │ (already exist)         │    │
│  └──────────────┘ └──────────────┘ └─────────────────────────┘    │
│  ┌──────────────────────────────────────────────┐                  │
│  │ financial_cutoff_periods (IsPeriodLocked)    │                  │
│  └──────────────────────────────────────────────┘                  │
└────────────────────────────────────────────────────────────────────┘
```

**Flow for primary use case (time entry create → submit → manager approve → finance approve):**

```
Browser ──POST /time-entries──► Handler ──► Service ──► Repo (INSERT draft)
Browser ◄── 201 {entry, status:draft} ──────── Handler
Browser ──POST /time-entries/{id}/submit──► Handler ──► Service (status→submitted, current_approver_role→manager)
Browser ◄── 200 {entry, status:submitted} ────── Handler
Browser ──POST /time-entries/{id}/approve (role:manager)──► Handler
    ──► Service: check role==manager, e.UserID != approver, status==submitted
    ──► status→pending_finance, current_approver_role→finance
    ──► Repo: UPDATE status + INSERT time_entry_approvals
Browser ◄── 200 {entry, status:pending_finance}
Browser ──POST /time-entries/{id}/approve (role:finance)──► Handler
    ──► Service: check role==finance, status==pending_finance
    ──► status→approved, current_approver_role→nil
    ──► Repo: UPDATE status + INSERT time_entry_approvals
Browser ◄── 200 {entry, status:approved}
```

### Recommended Project Structure

```
# New files (★) and modified files (✎)

internal/core/domain/expense/         # D-05: New expense domain
★ expense.go                          # Expense types, sentinel errors, status constants

internal/core/ports/
✎ expense_repository.go               # Add List(ListFilters), ListPending, IsPeriodLocked

internal/core/services/expense/       # D-05: New expense service
★ expense.go                          # Full CRUD + approval workflow (mirrors time_entry)
★ expense_test.go                     # Unit tests

internal/core/services/time_entry/
✎ time_entry.go                       # Two-stage approval, sync approval history
✎ time_entry_test.go                  # Update for two-stage tests

internal/core/services/testdata/
★ factories.go                        # Add NewExpenseDomain (for domain/expense types)
✎ mocks.go                            # Add MockExpenseRepo

internal/adapters/primary/http/
★ expense.go                          # New: 10 expense endpoints
★ expense_test.go                     # Handler integration tests
✎ time_entry.go                       # Two-stage approve/reject, role differentiation

internal/adapters/secondary/postgres/
✎ expense_repository.go               # Add List with filters, ListPending, IsPeriodLocked
✎ expense_repository_test.go          # Tests for new methods

cmd/server/main.go
✎ main.go                             # Register expense routes + NewExpenseHandler

migrations/
★ 004_time_entries_status_check.up.sql    # Update CHECK constraint
★ 004_time_entries_status_check.down.sql  # Revert
★ 005_expenses_status_check.up.sql        # Update CHECK constraint
★ 005_expenses_status_check.down.sql      # Revert

uploads/receipts/                     # Created at runtime (not committed)

web/src/api/
★ expenses.ts                         # All expense query/mutation options
✎ time-entries.ts                     # Remove monthly-summary, submit-month. Add approve/reject

web/src/types/
★ expense-types.ts                    # Expense request/response types
✎ api.ts                              # Add expense types
✎ models.ts                           # (already has EntryStatus, add Expense if needed)

web/src/routes/_authenticated/expenses/
★ index.tsx                           # Expense route + page component
★ -components/
★   expenses-page.tsx                 # Layout: MiniCalendar + day detail
★   expense-detail.tsx                # Day detail with expense form
★   expense-row.tsx                   # Single expense entry row

web/src/routes/_authenticated/time-entries/
✎ index.tsx                           # Remove monthly-summary/submit-month deps
✎ -components/
✎   time-entries-page.tsx             # Adapt layout
✎   entry-detail.tsx                  # Rewrite for flat model
✎   entry-row.tsx                     # Adapt for flat model
✎   mini-calendar.tsx                 # Client-side status computation
  (status-badge.tsx unchanged — already supports 6 statuses)

web/src/components/
★ approval/                           # Shared approval components
★   approval-buttons.tsx              # Approve/reject with role-aware visibility
★   approval-history.tsx              # Immutable history timeline
```

### Pattern 1: Two-Stage Approval Workflow
**What:** The approve handler on both time entries and expenses differentiates between WG manager approval (stage 1) and finance approval (stage 2). Each stage transitions to a specific `pending_*` status and sets `current_approver_role` for the next approver.

**When to use:** In both `TimeEntryService.Approve` and `ExpenseService.Approve`.

**Approval state machine:**
```
draft ──submit──► submitted (current_approver_role=manager)
                    │
                    ├── [WG manager approves] ──► pending_finance (current_approver_role=finance)
                    ├── [WG manager rejects]  ──► rejected
                    │
                    ▼ (if self-approve guard triggers, skip to pending_finance)
pending_finance
    │
    ├── [Finance approves] ──► approved
    └── [Finance rejects]  ──► rejected
```

**Key rule (D-11):** If `entry.UserID == approver.UserID` (self-approve guard), skip `pending_manager` entirely and go directly to `pending_finance`.

**Key rule (D-09):** WG manager is determined by `working_group.manager_id` on the WG linked to the entry via `entry.WGID`.

### Pattern 2: Synchronous Approval History (D-12)
**What:** Replace current async `go s.auditRepo.Create()` with a synchronous `s.approvalRepo.Create()` call inside the service method, after the status update succeeds.

**When to use:** In `Approve`, `Reject`, and `Submit` for both time entries and expenses.

```
func (s *Service) Approve(ctx, id, userID, role, reason) (*TimeEntry, error) {
    // 1. Validate: role allowed, entry at correct stage, not self-approve
    // 2. Update: entry.Status = newStatus, entry.CurrentApproverRole = nextRole
    // 3. Save: repo.Update(ctx, entry)
    // 4. Log: repo.CreateApproval(ctx, &Approval{entryID, action, userID, role, reason})
    return entry, nil
}
```

### Anti-Patterns to Avoid
- **Async audit log:** Current `go s.auditRepo.Create()` creates a race where approval action succeeds but history is lost if server crashes. Replace with synchronous call (D-12).
- **Using `models.Expense` directly in service:** Per D-05, create proper domain types in `domain/expense/`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP routing | Custom mux | Go 1.22+ `http.ServeMux` with method+path | Already project standard, supports `GET /path/{id}` |
| UUID generation | Custom ID | `uuid.New()` from google/uuid | Already project standard |
| JSON encoding | Custom serializer | `encoding/json` stdlib | Already project standard |
| DB connection | Custom pool | `pgxpool` from pgx/v5 | Already project standard |
| File validation | Full MIME detection | Simple extension check (.pdf, .jpg, .png) + size check | D-06: MVP scope, no need for deep inspection |
| Authentication | Custom JWT | Existing `internal/auth` package | Already handles token generation/validation |
| Response format | Custom envelope | `pkg/api.RespondWithJSON` / `RespondWithError` | Already project standard |

**Key insight:** All infrastructure (routing, DB, auth, API responses) already exists in the project. Phase 6 is purely adding domain logic following existing patterns.

## Common Pitfalls

### Pitfall 1: Approval Status Transition Gap
**What goes wrong:** The two-stage approval state machine can have invalid transitions (e.g., approving a `draft` entry, or a finance user approving a `submitted` entry before WG manager).
**Why it happens:** The status validation in the current `Approve` only checks if status is `submitted`. The two-stage workflow needs more granular checks.
**How to avoid:** Implement precise status checks per role:
- WG manager can approve: `submitted` → `pending_finance`
- Finance can approve: `pending_finance` → `approved`
- Either role can reject: `submitted` or `pending_finance` → `rejected`
- Rejected entries can be re-submitted: `rejected` → `submitted` (not `draft`)

### Pitfall 2: Self-Approval Gap
**What goes wrong:** WG manager creates an entry, submits it, then approves it themselves (bypassing the intended separation).
**Why it happens:** The current `Approve` handler only checks role, not ownership.
**How to avoid:** In `Approve`, check `e.UserID == approverUserID`. If match and role is WG manager, skip `pending_manager` stage (go directly to `pending_finance`). If match and role is finance, deny with 403 (per D-08, D-11).

### Pitfall 3: Missing `current_approver_role` on Responses
**What goes wrong:** Frontend can't determine which approval buttons to show because the API response doesn't indicate the current expected approver role.
**Why it happens:** The existing `TimeEntry` domain model doesn't include `CurrentApproverRole`.
**How to avoid:** Add `CurrentApproverRole *string` to both `TimeEntry` and `Expense` domain models (D-22, D-23). The field maps to who should act next: `"manager"`, `"finance"`, or `nil` (terminal state).

### Pitfall 4: Stale Query Cache After Status Change
**What goes wrong:** After approve/reject, the MiniCalendar still shows old status colors because the monthly query wasn't invalidated.
**Why it happens:** React Query cache invalidation only targets specific query keys.
**How to avoid:** In all approve/reject mutations, invalidate both day-level (`['time-entries', 'date', ...]`) AND month-level (`['time-entries', 'month', ...]`) query keys. Same for expenses.

## Code Examples

### TimeEntryService Two-Stage Approve
```go
// Source: [VERIFIED: existing `internal/core/services/time_entry/time_entry.go` — to be extended]
func (s *Service) Approve(ctx context.Context, id, userID uuid.UUID, role string) (*time_entry.TimeEntry, error) {
    e, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Self-approval prevention (D-11)
    if e.UserID == userID {
        return nil, time_entry.ErrForbidden
    }

    // Route by role and current status
    switch {
    case role == "manager" && e.Status == time_entry.StatusSubmitted:
        // WG manager approves → pending_finance
        e.Status = time_entry.StatusPendingFinance
        e.CurrentApproverRole = ptr("finance")
    case role == "finance" && e.Status == time_entry.StatusPendingFinance:
        // Finance approves → approved
        e.Status = time_entry.StatusApproved
        e.CurrentApproverRole = nil
    default:
        return nil, time_entry.ErrEntryNotSubmitted
    }

    e.UpdatedAt = time.Now()

    // Save status change
    updated, err := s.repo.Update(ctx, e)
    if err != nil {
        return nil, err
    }

    // Synchronous approval history (D-12)
    approval := &time_entry.Approval{
        ID:          uuid.New(),
        EntryID:     id,
        Action:      "approve",
        ActorUserID: userID,
        ActorRole:   role,
        CreatedAt:   time.Now(),
    }
    if err := s.approvalRepo.Create(ctx, approval); err != nil {
        return nil, err
    }

    return updated, nil
}
```

### Expense Repository List with Filters
```go
// Source: [VERIFIED: existing `internal/adapters/secondary/postgres/expense_repository.go` — to be extended]
// Pattern follows `buildTimeEntryListQuery` from time_entry_repository.go
func (r *ExpenseRepository) List(ctx context.Context, orgID uuid.UUID, filters ports.ExpenseListFilters) ([]domainexpense.Expense, error) {
    query, args := buildExpenseListQuery(orgID, filters)
    rows, err := r.pool.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("list expenses: %w", err)
    }
    defer rows.Close()
    return scanDomainExpenses(rows)
}
```

### Frontend Month Calendar (Client-Side Computation)
```typescript
// Source: [VERIFIED: existing `mini-calendar.tsx` — adapted per D-14]
// Replaces server-side monthly-summary with client-side computation
import { useSuspenseQuery } from '@tanstack/react-query'

export function MiniCalendar() {
  const { month } = useSearch({ from: '/_authenticated/time-entries/' })
  const { data: entries } = useSuspenseQuery(
    TimeEntriesApis.timeEntriesForMonthQueryOpts(
      month.getMonth() + 1,
      month.getFullYear()
    )
  )

  // Compute day statuses client-side
  const statusByDate = useMemo(() => {
    const map = new Map<string, EntryStatus>()
    entries?.forEach((entry: TimeEntry) => {
      const dateStr = format(new Date(entry.entry_date), 'yyyy-MM-dd')
      // Most significant status wins: approved > rejected > submitted > draft
      if (!map.has(dateStr) || statusPriority(entry.status) > statusPriority(map.get(dateStr)!)) {
        map.set(dateStr, entry.status)
      }
    })
    return map
  }, [entries])

  // Render calendar with status-based color modifiers
  // ... (same as current mini-calendar.tsx)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| One-stage approval (draft→submitted→approved) | Two-stage (draft→submitted→pending_manager→pending_finance→approved/rejected) | This phase | Approve/Reject handlers need major rework, frontend needs role-aware UI |
| `go s.auditRepo.Create()` (async goroutine) | Synchronous `s.approvalRepo.Create()` inside service | This phase (D-12) | Race condition removed, but approvalRepo dependency added to service constructor |
| `GET /time-entries/monthly-summary` | Client-side computation from `GET /time-entries?month=&year=` | This phase (D-14) | Remove monthly-summary API endpoint and frontend call |
| `POST /time-entries/submit-month` | Per-entry submit | This phase (D-15) | Remove endpoint and frontend mutation |
| Items-based time entry frontend (many rows per entry) | Flat model (one row per entry) | This phase (D-01, D-16) | `entry-detail.tsx` and `entry-row.tsx` rewrite |

**Deprecated/outdated:**
- `models.TimeEntryCreateRequest` and `models.ExpenseCreateRequest` (items-based) — domain models in `domain/time_entry/` and `domain/expense/` should be used instead
- `TimeEntriesApis.timeEntriesMonthlySummaryQueryOpts` and `submitMonthMutationOpts` — remove from frontend code

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | WG manager is determined by `working_group.manager_id` linked via `time_entry.wg_id` | Architecture | If WG manager lookup changes (e.g., to project managers), approval routing breaks. Per D-09, route via project WG manager — verify against `working_groups` table during implementation. |
| A2 | The `time_entry_approvals` table's `user_id` column maps to the actor (approver/rejecter) | Architecture | The existing `AuditLogRepository` uses this mapping. Verify column semantics match `creator` vs `actor`. |
| A3 | Receipt upload to local filesystem is acceptable for MVP | Standard Stack | D-06 specifies local filesystem. If demo/staging env uses container without persistent volume, uploads will be lost on restart. Acceptable for MVP per D-06. |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.
**Note:** Three assumptions identified above remain unverified and flagged for discuss/plan.

## Open Questions

1. **WG manager lookup implementation detail**
   - What we know: Working group has `manager_id` field. The WG is linked to time entries via `entry.WGID`.
   - What's unclear: Exact implementation approach — should the "time entry" service accept a `WorkingGroupRepository` dependency? Or should we query `working_groups` directly in the repository layer? Or use `TimeEntryRepository.ListPending` approach which already joins on `working_groups.manager_id`?
   - Recommendation: Inject a `WorkingGroupRepository` (or a new narrow port interface) into the time entry service for the `ListPending` call and the approval routing check. Follow existing pattern from `TimeEntryRepository.ListPending`.

2. **Approval repository interface**
   - What we know: The current `AuditLogRepository` interface has `Create(ctx, *AuditLog)`. The `time_entry_approvals` table exists. D-12 says use synchronous history.
   - What's unclear: Should we create a new `ApprovalRepository` port or reuse/extend the existing `AuditLogRepository`? The table schema (`time_entry_approvals`) differs from what `AuditLog` stores.
   - Recommendation: Create a new `ApprovalRepository` port interface scoped to the approval tables (specific to time_entries and expenses), and deprecate `AuditLogRepository`.

3. **Self-approval bypass for WG manager == creator**
   - What we know: D-11 says if creator == WG manager, skip manager approval and route directly to pending_finance.
   - What's unclear: Should the `Submit` handler immediately transition to `pending_finance` (bypassing `submitted`), or should `Approve` detect the self-approval case and skip the stage? The former is simpler but changes the API contract.
   - Recommendation: Handle in `Approve` — if role==manager and e.UserID == userID, transition to `pending_finance` directly. `Submit` always goes to `submitted` first.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26 | Backend | ✓ | 1.26.1 | — |
| PostgreSQL | Database | ✓ | (docker-compose) | — |
| pgx/v5 | Backend DB driver | ✓ | (go.sum) | — |
| bun | Frontend dev | ✓ | (web/package.json) | — |
| Vite | Frontend build | ✓ | (web/vite.config.ts) | — |
| testcontainers-go | Integration tests | ✓ | (Phase 0 infra) | — |

**Missing dependencies with no fallback:** None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify + testcontainers-go |
| Config file | `internal/core/services/postgres/testpool.go` (Phase 0) |
| Quick run command | `go test -count=1 -timeout 120s ./internal/core/services/expense/...` |
| Full suite command | `go test -count=1 -timeout 300s ./internal/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TIME-01 | Time entry list filtering | unit | `go test ./internal/core/services/time_entry/... -run TestService_List` | ✅ `time_entry_test.go` |
| TIME-02 | Create time entry | unit | `go test ./internal/core/services/time_entry/... -run TestService_Create` | ✅ |
| TIME-03 | Edit time entry (draft/submitted) | unit | `go test ./internal/core/services/time_entry/... -run TestService_Update` | ✅ (needs extension for two-stage) |
| TIME-04 | Submit for approval | unit | `go test ./internal/core/services/time_entry/... -run TestService_Submit` | ✅ (needs extension) |
| TIME-05 | Cannot edit approved/rejected | unit | `go test ./internal/core/services/time_entry/... -run TestService_Update` | ✅ |
| TIME-06 | Cannot delete with approvals | unit | `go test ./internal/core/services/time_entry/... -run TestService_Delete` | ✅ |
| TIME-07 | Employee cannot self-approve | unit | `go test ./internal/core/services/time_entry/... -run TestService_Approve` | ✅ (needs self-approve case) |
| TIME-08 | Manager cannot approve own entries | unit | `go test ./internal/core/services/time_entry/... -run TestService_Approve` | ❌ new test needed |
| EXPN-01 | Expense list filtering | unit | `go test ./internal/core/services/expense/... -run TestExpenseService_List` | ❌ Wave 0 |
| EXPN-02 | Create expense | unit | `go test ./internal/core/services/expense/... -run TestExpenseService_Create` | ❌ Wave 0 |
| EXPN-03 | Edit expense (draft/submitted) | unit | `go test ./internal/core/services/expense/... -run TestExpenseService_Update` | ❌ Wave 0 |
| EXPN-04 | Submit for approval | unit | `go test ./internal/core/services/expense/... -run TestExpenseService_Submit` | ❌ Wave 0 |
| EXPN-05 | Expense categories | unit | Test expense create with each category | ❌ Wave 0 |
| EXPN-06 | Same approval constraints | unit | Mirror time entry approval tests | ❌ Wave 0 |
| APPR-01 | Approval history immutable | integration | Insert-duplicate check or read-only test | ❌ Wave 0 |
| APPR-02 | Rejected entries show reason | unit | Test Reject with reason string | ❌ Wave 0 |
| APPR-03 | Full workflow | integration | End-to-end: create→submit→manager_approve→finance_approve | ❌ Wave 0 |
| APPR-04 | Status badge component | frontend | Visual snapshot or unit test | ❌ Wave 0 |
| APPR-05 | Approval buttons component | frontend | Test role visibility logic | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/services/time_entry/... -count=1 -timeout 60s`
- **Per wave merge:** `go test -count=1 -timeout 300s ./internal/...`
- **Phase gate:** Full suite green + `bun run build` on web/

### Wave 0 Gaps
- [ ] `internal/core/services/expense/expense_test.go` — covers EXPN-01 through EXPN-06
- [ ] `internal/adapters/primary/http/expense_test.go` — handler integration tests
- [ ] `internal/adapters/secondary/postgres/expense_repository_test.go` — new list/pending methods
- [ ] `web/src/components/approval/approval-buttons.test.tsx` — role visibility logic
- [ ] `web/src/components/approval/approval-history.test.tsx` — render test

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Auth already handled by existing middleware |
| V3 Session Management | no | Cookie-based auth already implemented |
| V4 Access Control | yes | Role checks in middleware + service layer (WG manager vs finance) |
| V5 Input Validation | yes | Zod (frontend), manual parsing (backend) |
| V6 Cryptography | no | No new crypto requirements |

### Known Threat Patterns for This Phase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized approval (employee approves own entry) | Elevation of Privilege | D-11: self-approval guard — check `e.UserID == userID` in Approve handler |
| Wrong-role approval (employee tries to approve as finance) | Spoofing | `middleware.GetRole()` returns actual JWT role. Handler rejects non-manager/finance. |
| Deleted expense file enumeration | Information Disclosure | Receipt URLs use UUID-based paths. No sequential IDs in file storage. |
| Mass assignment in update | Tampering | Domain models control which fields are updatable. Handler maps only allowed fields. |
| Receipt upload: large files / wrong types | Denial of Service | Validate file size (<10MB) and extension (.pdf, .jpg, .png) in upload handler. |

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase] `internal/core/domain/time_entry/time_entry.go` — Domain model with status constants, sentinel errors
- [VERIFIED: codebase] `internal/core/services/time_entry/time_entry.go` — Service with CRUD + one-stage approval
- [VERIFIED: codebase] `internal/adapters/primary/http/time_entry.go` — HTTP handler with all 9 endpoints
- [VERIFIED: codebase] `internal/adapters/secondary/postgres/time_entry_repository.go` — PG repo with dynamic query builder
- [VERIFIED: codebase] `internal/adapters/secondary/postgres/expense_repository.go` — PG repo with basic CRUD
- [VERIFIED: codebase] `internal/core/ports/expense_repository.go` — Port interface (needs extension)
- [VERIFIED: codebase] `internal/core/ports/time_entry_repository.go` — Port interface with ListFilters
- [VERIFIED: codebase] `internal/core/services/testdata/mocks.go` — Mock repos, factory functions
- [VERIFIED: codebase] `internal/core/services/testdata/factories.go` — NewTimeEntry, NewExpense factories
- [VERIFIED: codebase] `migrations/000_full_schema.up.sql` — Schema: expenses, time_entries, time_entry_approvals, expense_approvals
- [VERIFIED: codebase] `internal/middleware/middleware.go` — Auth middleware: GetUserID, GetRole, GetOrganizationID
- [VERIFIED: codebase] `cmd/server/main.go` — Route registration (lines 198-210 for time entries)
- [VERIFIED: codebase] `web/src/api/time-entries.ts` — Frontend API module (needs removal of monthly-summary/submit-month)
- [VERIFIED: codebase] `web/src/routes/_authenticated/time-entries/index.tsx` — Route definition with zod search validation
- [VERIFIED: codebase] `web/src/routes/_authenticated/time-entries/-components/mini-calendar.tsx` — Calendar with status colors
- [VERIFIED: codebase] `web/src/routes/_authenticated/time-entries/-components/entry-detail.tsx` — Items-based detail page
- [VERIFIED: codebase] `web/src/routes/_authenticated/time-entries/-components/entry-row.tsx` — Items-based row component
- [VERIFIED: codebase] `web/src/routes/_authenticated/time-entries/-components/status-badge.tsx` — All 6 statuses
- [VERIFIED: codebase] `web/src/types/api.ts` — Frontend API types
- [VERIFIED: codebase] `web/src/types/models.ts` — Frontend model types
- [VERIFIED: codebase] `internal/models/models.go` — Full model definitions: Expense, ExpenseCategory, TimeEntryApproval, etc.
- [VERIFIED: codebase] `internal/core/domain/working_group/working_group.go` — WorkingGroup with ManagerID field
- [VERIFIED: codebase] `internal/core/ports/working_group_repository.go` — WG repo port interface
- [VERIFIED: codebase] `.planning/phases/06-time-entries-expenses/06-CONTEXT.md` — 23 locked decisions (D-01 through D-23)
- [VERIFIED: codebase] `.planning/REQUIREMENTS.md` — All TIME-*, EXPN-*, APPR-* requirement definitions

### Secondary (MEDIUM confidence)
- None needed — all claims verified from codebase sources.

### Tertiary (LOW confidence)
- None — all claims verified.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified from codebase imports
- Architecture: HIGH — existing code patterns confirmed from 5+ Go source files and frontend components
- Pitfalls: HIGH — derived from examination of existing approval logic gaps

**Research date:** 2026-06-11
**Valid until:** 30 days (stable Go + React stack)
