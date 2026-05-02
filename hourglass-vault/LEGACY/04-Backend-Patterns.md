# Backend Patterns

Development conventions and architectural patterns for Hourglass backend.

## Handler Pattern

Every feature is implemented as a `*Handler` struct with dependency injection.

### Structure

```go
type TimeEntryHandler struct {
    db *sql.DB
}

func NewTimeEntryHandler(db *sql.DB) *TimeEntryHandler {
    return &TimeEntryHandler{db: db}
}

func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### Registration

In `cmd/server/main.go`, use Go 1.22+ pattern:

```go
timeEntryHandler := handlers.NewTimeEntryHandler(database.DB)
mux.HandleFunc("POST /time-entries", middleware.Auth(authService, timeEntryHandler.Create))
mux.HandleFunc("GET /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Get))
mux.HandleFunc("PUT /time-entries/{id}", middleware.Auth(authService, timeEntryHandler.Update))
```

### Key Points

- **Receiver**: Always `(h *Handler)` — pointer receiver
- **Dependency Injection**: Handler receives `*sql.DB` from main
- **Middleware Wrapping**: Protected routes wrapped with `middleware.Auth(authService, handler)`
- **Route Format**: Kebab-case (`/time-entries`, not `/timeEntries`)

---

## API Response Format

All responses use consistent envelope from `pkg/api/response.go`:

```go
type Response struct {
    Data   interface{} `json:"data,omitempty"`
    Error  string      `json:"error,omitempty"`
    Status int         `json:"-"`
}
```

### Success Response

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(api.Response{Data: myData})

// Produces: { "data": { ... } }
```

### Error Response

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusBadRequest)
json.NewEncoder(w).Encode(api.Response{Error: "Invalid request"})

// Produces: { "error": "Invalid request" }
```

### Common Status Codes

| Code | Use Case |
|------|----------|
| 200 | Successful GET, PUT, DELETE |
| 201 | Successful POST (creation) |
| 400 | Invalid request data |
| 401 | Missing/invalid JWT token |
| 403 | Authorized but no permission |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate email) |
| 500 | Server error (log it!) |

---

## Request Parsing

### URL Parameters

Use `chi`-like path parameters:

```go
func (h *TimeEntryHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")  // From /time-entries/{id}
    // Parse UUID
    entryID, err := uuid.Parse(id)
}
```

### JSON Body Parsing

```go
var req struct {
    ProjectID uuid.UUID `json:"project_id"`
    Hours     float64   `json:"hours"`
    Notes     string    `json:"notes,omitempty"`
}

if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    w.WriteHeader(http.StatusBadRequest)
    json.NewEncoder(w).Encode(api.Response{Error: "Invalid JSON"})
    return
}
```

### Query Parameters

```go
limit := r.URL.Query().Get("limit")  // Default ""
offset := r.URL.Query().Get("offset")
status := r.URL.Query().Get("status")  // Filter by status
```

---

## Database Access

### Pattern

All handlers receive `*sql.DB`. Use standard `database/sql` package:

```go
// Single row
var user User
err := h.db.QueryRowContext(ctx, 
    "SELECT id, email, name FROM users WHERE id = $1", 
    userID).Scan(&user.ID, &user.Email, &user.Name)

// Multiple rows
rows, err := h.db.QueryContext(ctx, 
    "SELECT id, email, name FROM users WHERE organization_id = $1 AND is_active = true", 
    orgID)
defer rows.Close()

for rows.Next() {
    var user User
    if err := rows.Scan(&user.ID, &user.Email, &user.Name); err != nil {
        // Handle error
    }
}
```

### Context Usage

Always use `r.Context()` for request-scoped operations:

```go
func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    err := h.db.QueryRowContext(ctx, query, args...).Scan(&id)
}
```

---

## Auth Context

Protected routes automatically inject auth context via middleware.

### Extracting User Info

```go
func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(uuid.UUID)
    orgID := r.Context().Value("org_id").(uuid.UUID)
    role := r.Context().Value("role").(models.Role)
}
```

See [[05-Auth-System]] for auth details.

---

## Common Handler Pattern: List with Filters

```go
func (h *TimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(uuid.UUID)
    orgID := r.Context().Value("org_id").(uuid.UUID)
    
    status := r.URL.Query().Get("status")  // Optional filter
    limit := 50
    offset := 0
    
    // Build query
    query := `
        SELECT id, user_id, status, created_at 
        FROM time_entries 
        WHERE organization_id = $1 AND user_id = $2
    `
    args := []interface{}{orgID, userID}
    
    // Add status filter if provided
    if status != "" {
        query += " AND status = $" + strconv.Itoa(len(args)+1)
        args = append(args, status)
    }
    
    query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + 
             " OFFSET $" + strconv.Itoa(len(args)+2)
    args = append(args, limit, offset)
    
    rows, err := h.db.QueryContext(r.Context(), query, args...)
    // ...handle rows
}
```

---

## Common Handler Pattern: Create with Validation

```go
func (h *TimeEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateTimeEntryRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Validate
    if req.Hours <= 0 || req.Hours > 24 {
        http.Error(w, "Hours must be between 0 and 24", http.StatusBadRequest)
        return
    }
    
    if len(req.ProjectIDs) == 0 {
        http.Error(w, "At least one project required", http.StatusBadRequest)
        return
    }
    
    // Create entry
    entryID := uuid.New()
    userID := r.Context().Value("user_id").(uuid.UUID)
    orgID := r.Context().Value("org_id").(uuid.UUID)
    
    _, err := h.db.ExecContext(r.Context(),
        `INSERT INTO time_entries (id, user_id, organization_id, status, work_date, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
        entryID, userID, orgID, "draft", req.WorkDate)
    
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(api.Response{Data: map[string]interface{}{"id": entryID}})
}
```

---

## Error Handling Best Practices

### Don't Log & Return Error

**Bad:**
```go
if err != nil {
    log.Printf("Error: %v", err)  // Logs
    http.Error(w, "Error", 500)    // Also returns to client
}
```

**Good:**
```go
if err != nil {
    // Log with context
    log.Printf("Failed to create time entry for user %s: %v", userID, err)
    // Return appropriate status
    http.Error(w, "Failed to create entry", http.StatusInternalServerError)
    return
}
```

### Distinguish User Errors from System Errors

```go
// User error (bad request)
http.Error(w, "Hours must be positive", http.StatusBadRequest)

// System error (database failure)
http.Error(w, "Failed to save entry", http.StatusInternalServerError)
log.Printf("Database error: %v", err)
```

---

## Naming Conventions

| Item | Convention | Example |
|------|-----------|---------|
| Handlers | `*Handler` suffix | `TimeEntryHandler`, `UserHandler` |
| Methods | PascalCase | `Create`, `Get`, `Update`, `List` |
| Request types | `CreateFoo`, `UpdateFoo` structs | `CreateTimeEntryRequest` |
| URL routes | kebab-case | `/time-entries`, `/org-settings` |
| Database table | snake_case | `time_entries`, `org_settings` |
| Model fields | camelCase (JSON), snake_case (DB) | `"user_id"` in JSON maps to `user_id` column |
| Constants | SCREAMING_SNAKE_CASE | `RoleEmployee`, `StatusApproved` |

---

## Testing Patterns

See [[17-Testing]] for comprehensive testing guide.

Quick pattern:
```go
func TestTimeEntryHandler_Create(t *testing.T) {
    handler := NewTimeEntryHandler(db)
    
    w := httptest.NewRecorder()
    r := httptest.NewRequest("POST", "/time-entries", 
        strings.NewReader(`{"hours": 8}`))
    r = r.WithContext(context.WithValue(r.Context(), "user_id", userID))
    
    handler.Create(w, r)
    
    if w.Code != http.StatusCreated {
        t.Fatalf("Expected 201, got %d", w.Code)
    }
}
```

---

**Next**: [[05-Auth-System]] for authentication details, or [[06-Middleware]] for request processing.
