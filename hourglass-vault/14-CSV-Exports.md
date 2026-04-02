# CSV Exports & Reporting

Generate exportable reports with role-based data filtering.

---

## Overview

**CSV Exports** allow users to download data as spreadsheets:
- Time entries by date range, project, status
- Expenses with cost breakdown
- Reports filtered by role permissions
- Role-based scoping (employee sees own, manager sees team, finance sees all)

**Feature added:** Issue #13 (CSV exports with role-based scoping)

---

## API Endpoints

### GET /exports/time-entries

Download time entries as CSV.

**Query Parameters:**
```
?start_date=2024-04-01    # Date range
?end_date=2024-04-30
?status=approved          # Filter by status
?project_id=proj-uuid     # Filter by project
?format=csv               # csv, json (default: csv)
```

**Response:**
```
Content-Type: text/csv
Content-Disposition: attachment; filename="time-entries-2024-04-01.csv"

entry_id,date,user_email,project,hours,status,submitted_at,approved_at
uuid1,2024-04-01,john@company.com,Frontend Development,8,approved,2024-04-02T10:00:00Z,2024-04-03T14:00:00Z
uuid2,2024-04-01,jane@company.com,Backend Services,8,approved,2024-04-02T11:00:00Z,2024-04-03T15:00:00Z
```

**Permissions:**
- **Employee:** Only own entries
- **Manager:** All org entries
- **Finance:** All org entries

---

### GET /exports/expenses

Download expenses as CSV.

**Query Parameters:**
```
?start_date=2024-04-01
?end_date=2024-04-30
?category=mileage          # Filter by category
?status=approved
?format=csv
```

**Response:**
```
Content-Type: text/csv
Content-Disposition: attachment; filename="expenses-2024-04-01.csv"

expense_id,date,user_email,category,amount,currency,description,status,submitted_at,approved_at
uuid1,2024-04-01,john@company.com,mileage,60.25,EUR,Client visit,approved,2024-04-02T10:00:00Z,2024-04-03T14:00:00Z
uuid2,2024-04-05,jane@company.com,meal,45.50,EUR,Team lunch,approved,2024-04-06T11:00:00Z,2024-04-07T15:00:00Z
```

**Permissions:** Same as time entries

---

### GET /exports/summary

High-level financial summary.

**Query Parameters:**
```
?start_date=2024-04-01
?end_date=2024-04-30
?group_by=project         # project, user, category, status
```

**Response (JSON):**
```json
{
  "data": {
    "period": "2024-04-01 to 2024-04-30",
    "total_hours": 160,
    "total_expenses": 2500.50,
    "by_project": [
      {
        "project_name": "Frontend Development",
        "hours": 80,
        "hours_cost": 4000,
        "expenses": 500
      }
    ],
    "by_status": {
      "approved": 2300,
      "pending_finance": 200.50
    }
  }
}
```

**Permissions:** Finance only (or manager for own team)

---

## Frontend Integration

### Export Button Component

```typescript
function ExportTimeEntriesButton() {
  const [dateRange, setDateRange] = useState({
    start: format(new Date(), 'yyyy-MM-dd'),
    end: format(new Date(), 'yyyy-MM-dd'),
  })
  const [status, setStatus] = useState('approved')
  
  const handleDownload = async () => {
    const response = await api.get('/exports/time-entries', {
      params: {
        start_date: dateRange.start,
        end_date: dateRange.end,
        status,
      },
      responseType: 'blob',  // Binary data
    })
    
    // Create download link
    const url = window.URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `time-entries-${dateRange.start}.csv`)
    document.body.appendChild(link)
    link.click()
    link.remove()
  }
  
  return (
    <div>
      <DateRangePicker value={dateRange} onChange={setDateRange} />
      <select value={status} onChange={(e) => setStatus(e.target.value)}>
        <option value="">All statuses</option>
        <option value="approved">Approved Only</option>
        <option value="pending">Pending</option>
      </select>
      <button onClick={handleDownload}>Download CSV</button>
    </div>
  )
}
```

### Reports Page

**Route:** `/reports`

**Features:**
- Date range selector
- Filter options (status, project, category)
- Export buttons (CSV, JSON)
- Summary stats
- Charts (hours by project, expenses by category)

---

## CSV Format Specifications

### Time Entries CSV

**Headers:**
```
entry_id
work_date
user_email
user_name
project_name
project_type
hours
status
current_approver_role
submitted_at
notes
```

**Example Row:**
```
550e8400-e29b-41d4-a716-446655440000,2024-04-01,john@company.com,John Doe,Frontend Development,billable,8,approved,,2024-04-02T10:00:00Z,Sprint work
```

**Notes:**
- Dates in ISO 8601 format
- Hours as decimal (8.5 = 8 hours 30 minutes)
- Approved entries show as approved status
- Pending entries show current approver role

### Expenses CSV

**Headers:**
```
expense_id
expense_date
user_email
user_name
category
amount
currency
description
status
submitted_at
contract_name (mileage only)
distance_km (mileage only)
rate_per_km (mileage only)
```

**Example Rows:**
```
uuid1,2024-04-01,john@company.com,John Doe,mileage,60.25,EUR,Client visit to Berlin,approved,2024-04-02T10:00:00Z,Enterprise Support,120.5,0.50
uuid2,2024-04-05,jane@company.com,Jane Smith,meal,45.50,EUR,Team lunch,approved,2024-04-06T11:00:00Z,,,
```

### Summary JSON

Contains aggregated stats:
- Total hours, total expenses
- Breakdown by project, user, status
- Average approval time
- Pending items

---

## Backend Implementation

### Export Handler

**File:** `internal/handlers/export.go`

```go
type ExportHandler struct {
    db *sql.DB
}

func (h *ExportHandler) TimeEntriesCSV(w http.ResponseWriter, r *http.Request) {
    // 1. Parse query params
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    status := r.URL.Query().Get("status")
    
    userID := r.Context().Value("user_id").(uuid.UUID)
    orgID := r.Context().Value("org_id").(uuid.UUID)
    role := r.Context().Value("role").(models.Role)
    
    // 2. Build query with role-based filtering
    query := `
        SELECT 
            te.id, te.work_date, u.email, u.name, p.name, p.type,
            SUM(tei.hours), te.status, te.current_approver_role,
            te.submitted_at, te.notes
        FROM time_entries te
        JOIN users u ON te.user_id = u.id
        LEFT JOIN time_entry_items tei ON te.id = tei.time_entry_id
        LEFT JOIN projects p ON tei.project_id = p.id
        WHERE te.organization_id = $1
    `
    args := []interface{}{orgID}
    
    // 3. Add role-based filtering
    switch role {
    case models.RoleEmployee:
        query += " AND te.user_id = $2"
        args = append(args, userID)
    // managers and finance see all
    }
    
    // 4. Add date/status filters
    if startDate != "" {
        query += fmt.Sprintf(" AND te.work_date >= $%d", len(args)+1)
        args = append(args, startDate)
    }
    if status != "" {
        query += fmt.Sprintf(" AND te.status = $%d", len(args)+1)
        args = append(args, status)
    }
    
    // 5. Execute query
    rows, err := h.db.QueryContext(r.Context(), query, args...)
    if err != nil {
        http.Error(w, "Query failed", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    // 6. Write CSV headers
    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", "attachment; filename=\"time-entries.csv\"")
    
    writer := csv.NewWriter(w)
    writer.Write([]string{"entry_id", "date", "user_email", "project", "hours", "status", "submitted_at"})
    
    // 7. Write rows
    for rows.Next() {
        var entryID, date, email, project, status, submittedAt string
        var hours float64
        
        if err := rows.Scan(&entryID, &date, &email, &project, &hours, &status, &submittedAt); err != nil {
            continue
        }
        
        writer.Write([]string{entryID, date, email, project, fmt.Sprintf("%.2f", hours), status, submittedAt})
    }
    
    writer.Flush()
}
```

---

## Permissions & Privacy

### Role-Based Filtering

**Employee:**
- Can only export own time entries and expenses
- Cannot see other employees' data
- Cannot export org-wide summaries

**Manager:**
- Can export all org data (team members)
- Cannot see finance-specific summaries (revenue details)
- Can filter by own projects

**Finance:**
- Can export all data
- Full access to summaries and financial reports
- Can see all organizations they belong to

### Data Protection

- Salary/hourly rates NOT included in export
- Email addresses included (for mapping to payroll)
- No sensitive fields beyond what user can see in app

---

## Common Use Cases

### Payroll Export

Finance exports approved time entries:
```
GET /exports/time-entries?status=approved&start_date=2024-04-01&end_date=2024-04-30
```

Result: CSV with all approved hours → import to payroll system

### Expense Reimbursement

Manager exports submitted expenses:
```
GET /exports/expenses?status=approved&start_date=2024-04-01
```

Result: CSV with amounts and descriptions → submit to accounts payable

### Project Billing

Finance exports by project:
```
GET /exports/summary?group_by=project&start_date=2024-04-01&end_date=2024-04-30
```

Result: JSON summary → generate invoice

---

## Future Enhancements

- **PDF export** — formatted invoices
- **Email delivery** — automated weekly exports
- **Scheduled exports** — weekly, monthly reports
- **Custom templates** — user-defined export formats
- **Filters** — saved filter sets for repeated exports

---

**Next**: [[17-Testing]] for testing patterns, or [[18-Deployment]] for production setup.
