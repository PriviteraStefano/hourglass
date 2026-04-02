# Testing

Testing strategies and patterns for Hourglass backend and frontend.

---

## Backend Testing (Go)

### Test File Structure

Each package has a `*_test.go` file:

```
internal/handlers/
  user.go
  user_test.go          # Tests for user handler
  time_entry.go
  time_entry_test.go    # Tests for time entry handler
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/handlers

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run single test
go test -run TestCreateTimeEntry ./internal/handlers
```

### Test Pattern: Handler Test

```go
package handlers

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/google/uuid"
    "github.com/stefanoprivitera/hourglass/internal/models"
)

func TestTimeEntryHandler_Create(t *testing.T) {
    // 1. Setup: Create test database and handler
    db := setupTestDB(t)
    defer db.Close()
    handler := NewTimeEntryHandler(db)
    
    // 2. Create test user context
    userID := uuid.New()
    orgID := uuid.New()
    ctx := context.Background()
    ctx = context.WithValue(ctx, "user_id", userID)
    ctx = context.WithValue(ctx, "org_id", orgID)
    
    // 3. Prepare request
    reqBody := map[string]interface{}{
        "work_date": "2024-04-01",
        "items": []map[string]interface{}{
            {
                "project_id": uuid.New(),
                "hours":      8,
            },
        },
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest("POST", "/time-entries", bytes.NewReader(body))
    req = req.WithContext(ctx)
    w := httptest.NewRecorder()
    
    // 4. Execute handler
    handler.Create(w, req)
    
    // 5. Assert response
    if w.Code != http.StatusCreated {
        t.Fatalf("Expected 201, got %d", w.Code)
    }
    
    var resp struct {
        Data struct {
            ID     string `json:"id"`
            Status string `json:"status"`
        } `json:"data"`
    }
    json.NewDecoder(w.Body).Decode(&resp)
    
    if resp.Data.Status != "draft" {
        t.Errorf("Expected draft status, got %s", resp.Data.Status)
    }
}
```

### Test Pattern: Database Test

```go
func TestUpdateTimeEntryStatus(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Create test entry
    entryID := createTestTimeEntry(t, db)
    
    // Update status
    _, err := db.Exec(
        "UPDATE time_entries SET status = $1 WHERE id = $2",
        "submitted", entryID,
    )
    if err != nil {
        t.Fatalf("Failed to update: %v", err)
    }
    
    // Verify update
    var status string
    err = db.QueryRow("SELECT status FROM time_entries WHERE id = $1", entryID).Scan(&status)
    if err != nil {
        t.Fatalf("Failed to query: %v", err)
    }
    
    if status != "submitted" {
        t.Errorf("Expected submitted, got %s", status)
    }
}
```

### Test Database Setup

**File:** `internal/testutil/db.go`

```go
package testutil

import (
    "database/sql"
    "testing"
)

func SetupTestDB(t *testing.T) *sql.DB {
    // Connect to test database
    db, err := sql.Open("postgres", 
        "postgres://hourglass:hourglass@localhost:5432/hourglass_test?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    
    // Run migrations
    runMigrations(t, db)
    
    t.Cleanup(func() {
        truncateTables(t, db)
        db.Close()
    })
    
    return db
}

func runMigrations(t *testing.T, db *sql.DB) {
    // Execute migration files in order
    for i := 1; i <= 8; i++ {
        migPath := fmt.Sprintf("../../migrations/%03d_*.up.sql", i)
        // Execute SQL from file
    }
}

func truncateTables(t *testing.T, db *sql.DB) {
    tables := []string{
        "time_entry_approvals", "time_entry_items", "time_entries",
        "expense_approvals", "expenses",
        "project_managers", "projects", "contracts",
        "organization_settings", "organization_memberships",
        "organizations", "users",
    }
    
    for _, table := range tables {
        if _, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE"); err != nil {
            t.Logf("Failed to truncate %s: %v", table, err)
        }
    }
}
```

### Table-Driven Tests

```go
func TestApprovalWorkflow(t *testing.T) {
    tests := []struct {
        name            string
        initialStatus   string
        approverRole    string
        expectedStatus  string
        shouldSucceed   bool
    }{
        {
            name:            "Manager approves submitted entry",
            initialStatus:   "submitted",
            approverRole:    "manager",
            expectedStatus:  "pending_finance",
            shouldSucceed:   true,
        },
        {
            name:            "Employee cannot approve",
            initialStatus:   "submitted",
            approverRole:    "employee",
            expectedStatus:  "submitted",
            shouldSucceed:   false,
        },
        {
            name:            "Finance approves pending entry",
            initialStatus:   "pending_finance",
            approverRole:    "finance",
            expectedStatus:  "approved",
            shouldSucceed:   true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic using tt.* values
        })
    }
}
```

---

## Frontend Testing (React/TypeScript)

### Test Setup

**File:** `web/vitest.config.ts`

```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

### Component Test Pattern

**File:** `web/src/components/__tests__/time-entry-form.test.tsx`

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import TimeEntryForm from '@/components/time-entry-form'

describe('TimeEntryForm', () => {
  it('renders form fields', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <TimeEntryForm />
      </QueryClientProvider>
    )
    
    expect(screen.getByLabelText(/work date/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/hours/i)).toBeInTheDocument()
  })
  
  it('submits form with valid data', async () => {
    const mockMutate = vi.fn()
    vi.mock('@/api/time-entries', () => ({
      createTimeEntryMutation: () => ({
        mutationFn: mockMutate,
      }),
    }))
    
    const user = userEvent.setup()
    
    render(
      <QueryClientProvider client={queryClient}>
        <TimeEntryForm />
      </QueryClientProvider>
    )
    
    const dateInput = screen.getByLabelText(/work date/i)
    const hoursInput = screen.getByLabelText(/hours/i)
    const submitButton = screen.getByRole('button', { name: /submit/i })
    
    await user.type(dateInput, '2024-04-01')
    await user.type(hoursInput, '8')
    await user.click(submitButton)
    
    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalled()
    })
  })
  
  it('shows validation error for invalid hours', async () => {
    const user = userEvent.setup()
    
    render(
      <QueryClientProvider client={queryClient}>
        <TimeEntryForm />
      </QueryClientProvider>
    )
    
    const hoursInput = screen.getByLabelText(/hours/i)
    const submitButton = screen.getByRole('button', { name: /submit/i })
    
    await user.type(hoursInput, '25')  // Invalid: > 24
    await user.click(submitButton)
    
    expect(screen.getByText(/must be between 0 and 24/i)).toBeInTheDocument()
  })
})
```

### Hook Test Pattern

**File:** `web/src/hooks/__tests__/useTimeEntries.test.ts`

```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { useTimeEntriesQuery } from '@/hooks'

function wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}

describe('useTimeEntriesQuery', () => {
  it('fetches time entries', async () => {
    const { result } = renderHook(() => useTimeEntriesQuery(), { wrapper })
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    
    expect(result.current.data).toBeDefined()
    expect(Array.isArray(result.current.data)).toBe(true)
  })
})
```

---

## Integration Tests

### End-to-End Workflow

```go
func TestCompleteTimeEntryWorkflow(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // 1. Create user and org
    userID, orgID := createTestUserAndOrg(t, db)
    
    // 2. Create entry (draft)
    entryID := createTimeEntry(t, db, userID, orgID)
    
    // 3. Submit entry
    submitTimeEntry(t, db, entryID)
    
    // 4. Manager approves
    managerID := createManager(t, db, orgID)
    approveTimeEntry(t, db, entryID, managerID, "manager")
    
    // 5. Finance approves
    financeID := createFinance(t, db, orgID)
    approveTimeEntry(t, db, entryID, financeID, "finance")
    
    // 6. Verify final status
    var status string
    err := db.QueryRow("SELECT status FROM time_entries WHERE id = $1", entryID).
        Scan(&status)
    if err != nil {
        t.Fatalf("Query failed: %v", err)
    }
    
    if status != "approved" {
        t.Errorf("Expected approved, got %s", status)
    }
}
```

---

## Running Tests

### All Tests
```bash
go test ./...
npm run test
```

### With Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Watch Mode (Frontend)
```bash
npm run test:watch
```

### Specific Tests
```bash
go test -run TestCreateTimeEntry ./internal/handlers
npm run test -- time-entry-form
```

---

## CI/CD Integration

**.github/workflows/test.yml**

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_USER: hourglass
          POSTGRES_PASSWORD: hourglass
          POSTGRES_DB: hourglass_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: 1.26.1
      
      - name: Backend Tests
        run: make test
        env:
          DATABASE_URL: postgres://hourglass:hourglass@localhost:5432/hourglass_test?sslmode=disable
      
      - name: Frontend Tests
        run: cd web && npm install && npm run test
```

---

## Best Practices

1. **Test behavior, not implementation** — Test what user sees, not internal details
2. **Use table-driven tests** — Multiple scenarios in one test
3. **Mock external services** — Don't hit real APIs in tests
4. **Keep tests focused** — One assertion per test when possible
5. **Use descriptive names** — Test name should explain what it tests
6. **Isolate tests** — No test dependencies on other tests

---

**Next**: [[18-Deployment]] for production setup.
