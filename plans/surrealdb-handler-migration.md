# SurrealDB Handler Migration Plan

## Overview

Migrate remaining handlers from raw `h.sdb.Query()` + JSON marshal/unmarshal pattern to typed SDK functions (`sdb.Create`, `sdb.Select`, `sdb.Query[T]`).

**Previous migration** (auth repos): See `internal/adapters/secondary/surrealdb/` - established pattern with `models.go`, `helpers.go`, typed structs with `models.RecordID`.

---

## Files to Migrate

### Handlers Using `*db.SurrealDB`

| Handler | File | Tables | Complexity |
|---------|------|--------|------------|
| `AuthHandler` | `internal/handlers/auth_handler.go` | users, organizations, refresh_tokens | Medium |
| `InvitationHandler` | `internal/handlers/invitation_handler.go` | invitations | Medium |
| `PasswordResetHandler` | `internal/handlers/password_reset_handler.go` | verification_tokens | Low |
| `UnitHandler` | `internal/handlers/unit_handler.go` | units | High |
| `WorkingGroupHandler` | `internal/handlers/working_group_handler.go` | working_groups | High |
| `SurrealTimeEntryHandler` | `internal/handlers/surreal_time_entry_handler.go` | time_entries, audit_logs | Very High |

---

## Migration Pattern

### Before (Raw Query + JSON)
```go
results, err := h.sdb.Query(ctx, query, vars)
if err != nil {
    return err
}
var entries []models.SurrTimeEntry
resultBytes, _ := json.Marshal((*results)[0].Result)
if err := json.Unmarshal(resultBytes, &entries); err != nil {
    return err
}
```

### After (Typed SDK)
```go
results, err := sdb.Query[[]SurrealTimeEntry](ctx, h.db, query, vars)
if err != nil {
    return wrapErr(err, "list time entries")
}
if len(*results) == 0 || len((*results)[0].Result) == 0 {
    return nil, ErrNotFound
}
return (*results)[0].Result, nil
```

---

## Step-by-Step Instructions

### Phase 1: Auth Handler

**File:** `internal/handlers/auth_handler.go`

1. **Add imports**
   ```go
   sdb "github.com/surrealdb/surrealdb.go"
   "github.com/surrealdb/surrealdb.go/pkg/models"
   ```

2. **Change struct field**
   ```go
   type AuthHandler struct {
       authService *auth.Service
       db         *sdb.DB
   }
   
   func NewAuthHandler(db *sdb.DB, authService *auth.Service) *AuthHandler {
       return &AuthHandler{db: db, authService: authService}
   }
   ```

3. **Update `cmd/server/main.go` calls**
   ```go
   hexAuthHandler := http.NewAuthHandler(sdbConn.DB(), hexAuthService)
   ```

4. **Replace each query** with typed `sdb.Query[T]`:
   - Bootstrap (creates user + org)
   - Login queries
   - Any other raw queries

**Schema:** Uses `users`, `organizations`, `refresh_tokens` tables (already migrated in repos, but handler has direct queries too)

---

### Phase 2: Invitation Handler

**File:** `internal/handlers/invitation_handler.go`

1. **Change struct field**
   ```go
   type InvitationHandler struct {
       db *sdb.DB
   }
   ```

2. **Add models** in `internal/models/surreal_models.go` or create `invitation.go` in surrealdb adapter:
   ```go
   type SurrealInvitation struct {
       ID             models.RecordID `json:"id,omitempty"`
       OrganizationID models.RecordID `json:"organization_id"`
       Code           string          `json:"code"`
       InviteToken    string          `json:"invite_token"`
       Email          string          `json:"email,omitempty"`
       Status         string          `json:"status"`
       ExpiresAt      time.Time       `json:"expires_at"`
       CreatedBy      models.RecordID `json:"created_by"`
       CreatedAt      time.Time       `json:"created_at"`
   }
   ```

3. **Replace queries** with typed `sdb.Create[SurrealInvitation]()` for Create
4. **Replace queries** with typed `sdb.Query[[]SurrealInvitation]()` for Validate/Accept

**Schema:** Uses `invitations` table (see `schema/007_invitations.surql`)

---

### Phase 3: Password Reset Handler

**File:** `internal/handlers/password_reset_handler.go`

1. **Change struct field** to `*sdb.DB`
2. **Replace queries** using `sdb.Create` for create, `sdb.Query` for find

**Schema:** Uses `verification_tokens` table (see `schema/008_password_resets.surql`)

---

### Phase 4: Unit Handler

**File:** `internal/handlers/unit_handler.go`

1. **Change struct field** to `*sdb.DB`
2. **Add model** in adapter models:
   ```go
   type SurrealUnit struct {
       ID            models.RecordID `json:"id,omitempty"`
       OrgID         models.RecordID `json:"org_id"`
       Name          string         `json:"name"`
       Description   string         `json:"description,omitempty"`
       ParentUnitID  models.RecordID `json:"parent_unit_id,omitempty"`
       HierarchyLevel int           `json:"hierarchy_level"`
       Code          string         `json:"code,omitempty"`
       CreatedAt     time.Time      `json:"created_at"`
       UpdatedAt     time.Time      `json:"updated_at"`
   }
   ```

3. **Replace all queries**:
   - List: `sdb.Query[[]SurrealUnit](...)` 
   - Get: `sdb.Select[SurrealUnit](...)` with `models.NewRecordID("units", id)`
   - Create: `sdb.Create[SurrealUnit](...)`
   - Update: `sdb.Update[SurrealUnit](...)` or `sdb.Merge(...)`
   - Delete: `sdb.Delete[SurrealUnit](...)`
   - GetTree: `sdb.Query[[]SurrealUnit](...)`

**Complex patterns:** `$unit_id` syntax (parameterized table access) may need adaptation

**Schema:** Uses `units` table (see `schema/001_organizations.surql`)

---

### Phase 5: Working Group Handler

**File:** `internal/handlers/working_group_handler.go`

1. **Change struct field** to `*sdb.DB`
2. **Add model**:
   ```go
   type SurrealWorkingGroup struct {
       ID          models.RecordID `json:"id,omitempty"`
       OrgID      models.RecordID `json:"org_id"`
       Name       string          `json:"name"`
       Description string         `json:"description,omitempty"`
       ManagerID  models.RecordID `json:"manager_id"`
       DelegateIDs []string       `json:"delegate_ids,omitempty"`
       IsActive   bool            `json:"is_active"`
       CreatedAt  time.Time       `json:"created_at"`
       UpdatedAt  time.Time       `json:"updated_at"`
   }
   ```

3. **Replace all queries** with typed SDK calls

**Schema:** Uses `working_groups` table (see `schema/002_projects.surql`)

---

### Phase 6: Time Entry Handler

**File:** `internal/handlers/surreal_time_entry_handler.go`

**This is the most complex handler.** Contains:
- List with 6+ optional filters
- Get, Create, Update, Delete, Submit, Approve, Reject, ListPending
- Period lock checks
- Audit log creation

1. **Change struct field** to `*sdb.DB`
2. **Add model**:
   ```go
   type SurrealTimeEntry struct {
       ID              models.RecordID `json:"id,omitempty"`
       OrgID          models.RecordID `json:"org_id"`
       UserID        models.RecordID `json:"user_id"`
       ProjectID     models.RecordID `json:"project_id"`
       SubprojectID   models.RecordID `json:"subproject_id"`
       WGID          models.RecordID `json:"wg_id"`
       UnitID        models.RecordID `json:"unit_id"`
       Hours         float64        `json:"hours"`
       Description   string          `json:"description"`
       EntryDate     time.Time       `json:"entry_date"`
       Status        string          `json:"status"`
       IsDeleted     bool            `json:"is_deleted"`
       CreatedFromEntryID models.RecordID `json:"created_from_entry_id,omitempty"`
       CreatedAt     time.Time       `json:"created_at"`
       UpdatedAt     time.Time       `json:"updated_at"`
   }
   
   type SurrealAuditLog struct {
       ID         models.RecordID `json:"id,omitempty"`
       OrgID     models.RecordID `json:"org_id"`
       EntryID   string         `json:"entry_id"`
       EntryType string         `json:"entry_type"`
       Action    string         `json:"action"`
       ActorRole string         `json:"actor_role"`
       ActorID   models.RecordID `json:"actor_id"`
       Reason    string         `json:"reason,omitempty"`
       Changes   map[string]any `json:"changes,omitempty"`
       Timestamp time.Time      `json:"timestamp"`
       IPAddress string         `json:"ip_address,omitempty"`
   }
   ```

3. **Key patterns to handle**:
   - `SELECT * FROM $entry_id` → `sdb.Select[SurrealTimeEntry](ctx, db, recordID)`
   - Complex WHERE clauses → `sdb.Query[[]SurrealTimeEntry](...)`
   - UPDATE with MERGE → `sdb.Update[SurrealTimeEntry](...)`
   - Soft delete → `sdb.Update` with `is_deleted = true`
   - Audit logs → `sdb.Create[SurrealAuditLog](...)` (fire-and-forget, don't check error)

4. **Queries requiring special handling**:
   - `$entry_id` as table name (line 111, 260, etc.) → use `models.NewRecordID("time_entries", id)`
   - Subquery for wg_ids in ListPending (lines 756-769) → wrap properly

**Schema:** Uses `time_entries`, `audit_logs`, `financial_cutoff_periods`, `working_groups` tables (see `schema/003_time_entries.surql`)

---

## Current status

- Migrated or retired: auth, invitation, password reset, unit, working group, time entry, approval cleanup, expense retirement
- No remaining live PostgreSQL handlers are wired into the server
- The active server wiring now uses hex adapters for the live routes

---

## Key API Calls Reference

### Generic CRUD
```go
// Create
sdb.Create[T](ctx, db, models.Table("table"), data) (*T, error)

// Select one
sdb.Select[T](ctx, db, models.NewRecordID("table", id)) (*T, error)

// Select many
sdb.Select[[]T](ctx, db, models.Table("table")) (*[]T, error)

// Update (full replace)
sdb.Update[T](ctx, db, recordID, data) (*T, error)

// Merge (partial update)
sdb.Merge[T](ctx, db, recordID, data) (*T, error)

// Delete
sdb.Delete[T](ctx, db, recordID) (*T, error)

// Query
sdb.Query[[]T](ctx, db, "SELECT ... WHERE ...", vars) (*[]QueryResult[[]T], error)
```

### Query Response Structure
```go
type QueryResult[T] struct {
    Result []T `json:"result"`
}
// For Query[T], response is *[]QueryResult[T]
// Access: (*results)[0].Result[0]
```

### RecordID Construction
```go
// From UUID string
recordID := models.NewRecordID("table", "uuid-string")

// From uuid.UUID (via existing helpers)
recordID := uuidToRecordID("users", userID)  // from helpers.go
```

---

## Error Handling Pattern
```go
func wrapErr(err error, op string) error {
    if err == nil { return nil }
    if isNotFound(err) { return ports.ErrUserNotFound }
    return fmt.Errorf("%s: %w", op, err)
}

func isNotFound(err error) bool {
    if err == nil { return false }
    return strings.Contains(err.Error(), "not found") ||
           strings.Contains(err.Error(), "record not found")
}
```

---

## Testing Pattern

For each handler, tests should:
1. Skip if `SURREALDB_URL` not set (integration tests)
2. Use `GetDB()` from helpers to get connection
3. Create test data, perform operation, verify result
4. Clean up (or use unique IDs to avoid conflicts)

```go
func TestTimeEntryHandler_Create(t *testing.T) {
    if os.Getenv("SURREALDB_URL") == "" {
        t.Skip("SURREALDB_URL not set, skipping")
    }
    db := GetDB()
    handler := NewSurrealTimeEntryHandler(db)
    // ... test
}
```

---

## Execution Order

1. **AuthHandler** - Medium, uses repos (already migrated) for data, only direct queries for bootstrap
2. **InvitationHandler** - Medium, straightforward CRUD
3. **PasswordResetHandler** - Low, simple find/create
4. **UnitHandler** - High, recursive tree queries
5. **WorkingGroupHandler** - High, member management
6. **TimeEntryHandler** - Very High, most complex, many edge cases

Each phase: create models → update struct → update main.go → replace queries → test.

---

## Verification

After each handler migration:
```bash
go build ./...
go test ./internal/handlers/... -run TestSurreal -v
```

Integration tests will skip if `SURREALDB_URL` not set.

---

## Notes

- `db.SurrealDB` wrapper in `internal/db/surreal.go` can be deleted after ALL handlers migrated
- Handlers currently using `h.sdb.Query(...)` extensively - each query needs individual review
- Some queries use SurrealDB-specific syntax like `$entry_id` (parameterized table name) - SDK may not support directly, may need `sdb.Query` fallback
- Audit log creates are fire-and-forget (error ignored) - keep that pattern
