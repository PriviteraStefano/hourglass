# Remaining Backend Migration Plan

## Current State

| Handler | File | Status | Wired in main.go |
|---------|------|--------|-----------------|
| Auth | `internal/handlers/auth_handler.go` | DEPRECATED (superseded by hex) | NO |
| Invitation | `internal/handlers/invitation_handler.go` | Old `*db.SurrealDB` | YES |
| PasswordReset | `internal/handlers/password_reset_handler.go` | Old `*db.SurrealDB` | YES |
| Unit | `internal/handlers/unit_handler.go` | Old `*db.SurrealDB` | YES |
| WorkingGroup | `internal/handlers/working_group_handler.go` | Old `*db.SurrealDB` | YES |
| SurrealTimeEntry | `internal/handlers/surreal_time_entry_handler.go` | Old `*db.SurrealDB` | YES |
| TimeEntry | `internal/handlers/time_entry_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Organization | `internal/handlers/organization_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Project | `internal/handlers/project_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Expense | `internal/handlers/expense_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Export | `internal/handlers/export_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Customer | `internal/handlers/customer_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Contract | `internal/handlers/contract_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |
| Approval | `internal/handlers/approval_handler.go` | PostgreSQL `*sql.DB` | OUT OF SCOPE |

**All 14 handlers need migration. 6 → SurrealDB typed SDK. 8 → hexagonal + SurrealDB SDK.**

**Note:** 8 handlers (`time_entry_handler.go`, `organization_handler.go`, `project_handler.go`, `expense_handler.go`, `export_handler.go`, `customer_handler.go`, `contract_handler.go`, `approval_handler.go`) currently use PostgreSQL `*sql.DB`. They are OUT OF SCOPE for this plan — must be migrated to SurrealDB separately.

---

## Phase 1: SurrealDB Typed SDK Migration (6 handlers)

Convert from raw `h.sdb.Query()` + `json.Marshal/Unmarshal` → `sdb.DB` + typed `sdb.Query[T]`, `sdb.Create[T]`, `sdb.Select[T]`, `sdb.Merge[T]`, `sdb.Delete[T]`.

### 1.1 Add Models to `internal/adapters/secondary/surrealdb/models.go`

Add 6 typed structs:

```
SurrealInvitation      — id, org_id, code, invite_token, email, status, expires_at, created_by, created_at
SurrealPasswordReset   — id, user_id, code_hash, expires_at, used_at, created_at
SurrealUnit            — id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at
SurrealWorkingGroup    — id, org_id, subproject_id, name, description, unit_ids, enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at
SurrealWorkingGroupMember — id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, end_date, created_at
SurrealTimeEntry       — id, org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status, is_deleted, created_from_entry_id, created_at, updated_at
```

### 1.2 Update Each Handler

#### InvitationHandler — 4 methods

| Method | New Pattern |
|--------|-------------|
| Create | `sdb.Create[SurrealInvitation](ctx, h.db, models.Table("invitations"), data)` |
| ValidateCode | `sdb.Query[[]SurrealInvitation]` |
| ValidateToken | `sdb.Query[[]SurrealInvitation]` |
| Accept | `sdb.Query[[]SurrealInvitation]` + `sdb.Merge` for status update |

#### PasswordResetHandler — 2 methods

| Method | New Pattern |
|--------|-------------|
| Request.FindUser | `sdb.Query[[]map[string]any]` then extract ID |
| Request.CreateReset | `sdb.Create[SurrealPasswordReset]` |
| Verify.FindUser | `sdb.Query[[]map[string]any]` |
| Verify.FindReset | `sdb.Query[[]SurrealPasswordReset]` |
| Verify.UpdatePassword | `sdb.Merge` with `models.NewRecordID("users", userID)` |
| Verify.MarkUsed | `sdb.Merge[SurrealPasswordReset]` |

#### UnitHandler — 7 methods

| Method | New Pattern |
|--------|-------------|
| List | `sdb.Query[[]SurrealUnit]` |
| Get | `sdb.Select[SurrealUnit]` with `models.NewRecordID("units", id)` |
| Create | `sdb.Create[SurrealUnit]` |
| Create.GetParentLevel | `sdb.Select[SurrealUnit]` |
| Update | `sdb.Merge[SurrealUnit]` |
| Delete | `sdb.Delete[SurrealUnit]` |
| Delete.CheckMembers | `sdb.Query[[]map[string]any]` count check |
| GetTree | `sdb.Query[[]SurrealUnit]` |
| GetDescendants | `sdb.Query[[]SurrealUnit]` |

#### WorkingGroupHandler — 8 methods

| Method | New Pattern |
|--------|-------------|
| List | `sdb.Query[[]SurrealWorkingGroup]` |
| Get | `sdb.Select[SurrealWorkingGroup]` |
| Create | `sdb.Create[SurrealWorkingGroup]` |
| Update | `sdb.Merge[SurrealWorkingGroup]` |
| Delete | `sdb.Delete[SurrealWorkingGroup]` |
| ListMembers | `sdb.Query[[]SurrealWorkingGroupMember]` |
| AddMember | `sdb.Create[SurrealWorkingGroupMember]` |
| RemoveMember | `sdb.Delete[SurrealWorkingGroupMember]` |

#### SurrealTimeEntryHandler — 10+ methods

| Method | New Pattern |
|--------|-------------|
| List | `sdb.Query[[]SurrealTimeEntry]` |
| Get | `sdb.Select[SurrealTimeEntry]` |
| Create | `sdb.Create[SurrealTimeEntry]` |
| Update | `sdb.Merge[SurrealTimeEntry]` |
| Delete | `sdb.Merge` (soft delete: is_deleted=true) |
| Submit | `sdb.Merge` + `sdb.Create` audit log (fire-and-forget goroutine) |
| Approve | `sdb.Merge` + audit |
| Reject | `sdb.Merge` + audit |
| ListPending | `sdb.Query[[]SurrealTimeEntry]` |
| isPeriodLocked | `sdb.Query[[]map[string]any]` |

### 1.3 Update main.go Wiring

Change all `NewXxxHandler(sdbConn)` → `NewXxxHandler(sdbConn.DB())`.

### 1.4 Deprecate Old AuthHandler

Delete `internal/handlers/auth_handler.go` — fully superseded by `internal/adapters/primary/http/auth.go`.

---

## Phase 2: Hexagonal Architecture Migration (5 handlers)

Pattern per handler: Domain → Ports → Service → Secondary Adapter → Primary Adapter.

### 2.1 InvitationHandler

Files (5):
- `internal/core/domain/invitation/invitation.go`
- `internal/core/ports/invitation_repository.go`
- `internal/core/services/invitation/invitation.go`
- `internal/adapters/secondary/surrealdb/invitation_repository.go`
- `internal/adapters/primary/http/invitation.go`

### 2.2 PasswordResetHandler

Files (5):
- `internal/core/domain/password_reset/password_reset.go`
- `internal/core/ports/password_reset_repository.go`
- `internal/core/services/password_reset/password_reset.go`
- `internal/adapters/secondary/surrealdb/password_reset_repository.go`
- `internal/adapters/primary/http/password_reset.go`

### 2.3 UnitHandler

Files (5):
- `internal/core/domain/unit/unit.go`
- `internal/core/ports/unit_repository.go`
- `internal/core/services/unit/unit.go`
- `internal/adapters/secondary/surrealdb/unit_repository.go`
- `internal/adapters/primary/http/unit.go`

### 2.4 WorkingGroupHandler

Files (5):
- `internal/core/domain/working_group/working_group.go`
- `internal/core/ports/working_group_repository.go`
- `internal/core/services/working_group/working_group.go`
- `internal/adapters/secondary/surrealdb/working_group_repository.go`
- `internal/adapters/primary/http/working_group.go`

### 2.5 SurrealTimeEntryHandler

Files (5):
- `internal/core/domain/time_entry/time_entry.go`
- `internal/core/ports/time_entry_repository.go`
- `internal/core/services/time_entry/time_entry.go`
- `internal/adapters/secondary/surrealdb/time_entry_repository.go`
- `internal/adapters/primary/http/time_entry.go`

---

## Execution Order

### Phase 1 (SurrealDB Typed SDK)
1. Add models to `internal/adapters/secondary/surrealdb/models.go`
2. PasswordResetHandler — simplest, 2 methods
3. InvitationHandler — 4 methods, straightforward CRUD
4. UnitHandler — 7 methods, tree building logic
5. WorkingGroupHandler — 8 methods, member management
6. SurrealTimeEntryHandler — 10+ methods, most complex
7. Update main.go wiring (`sdbConn` → `sdbConn.DB()`)
8. Delete `internal/handlers/auth_handler.go`
9. Verify: `go build ./...`

### Phase 2 (Hexagonal — per handler)
For each of 5 handlers:
1. Create domain entity
2. Create port interface(s)
3. Create service layer
4. Create SurrealDB secondary adapter
5. Create HTTP primary adapter
6. Update main.go wiring
7. Verify: `go build ./...`

---

## Key Technical Decisions

**`$variable` table name access:** SurrealDB allows `SELECT * FROM $table_name`. SDK uses `models.NewRecordID(table, id)` for record-level ops. For variable table names, fall back to `sdb.Query`.

**Fire-and-forget audit logs:** Keep existing pattern — wrap `sdb.Create` in goroutine, ignore errors.

**Query response unwrapping:**
```go
results, err := sdb.Query[[]T](ctx, h.db, query, vars)
if len(*results) == 0 || len((*results)[0].Result) == 0 { return nil, ErrNotFound }
return (*results)[0].Result, nil
```

**Error mapping:** Use `wrapErr` from `helpers.go` for consistent error translation. Add domain-specific errors to `internal/core/ports/` packages.

---

## File Count Summary

### Phase 1: 6 handlers updated, 1 file modified
- `internal/adapters/secondary/surrealdb/models.go` — add 6 typed structs
- `internal/handlers/invitation_handler.go`
- `internal/handlers/password_reset_handler.go`
- `internal/handlers/unit_handler.go`
- `internal/handlers/working_group_handler.go`
- `internal/handlers/surreal_time_entry_handler.go`
- `cmd/server/main.go` — update constructor calls
- Delete `internal/handlers/auth_handler.go`

### Phase 2: 25 new files
- 5 domain files
- 8 port interface files
- 5 service files
- 5 secondary adapter files
- 5 primary adapter files
