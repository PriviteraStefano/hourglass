# Organization & User Management

Multi-tenant organization model and role-based user management.

---

## Overview

Hourglass uses an **organization-based multi-tenancy model**:
- Users belong to one or more organizations
- Each org has its own data (contracts, projects, time entries, etc.)
- User roles (employee, manager, finance, customer) control permissions
- Users can switch between orgs they belong to

---

## Data Model

### organizations

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Organization identifier |
| name | VARCHAR | Company name |
| slug | VARCHAR UNIQUE | URL-friendly identifier |
| created_at | TIMESTAMP | Creation time |

**Indexes:** `slug`

### organization_memberships

Maps users to organizations with roles:

| Column | Type | Purpose |
|--------|------|---------|
| id | UUID PK | Membership record |
| user_id | UUID FK | References users(id) |
| organization_id | UUID FK | References organizations(id) |
| role | VARCHAR CHECK | employee, manager, finance, customer |
| is_active | BOOLEAN | Soft delete flag |
| invited_by | UUID FK | Who sent invite (nullable) |
| invited_at | TIMESTAMP | When invited (nullable) |
| activated_at | TIMESTAMP | When user accepted invite (nullable) |

**Constraint:** UNIQUE(user_id, organization_id) — one role per user per org

**Indexes:** `user_id`, `organization_id`

### organization_settings

Per-organization configuration:

| Column | Type | Purpose |
|--------|------|---------|
| organization_id | UUID PK FK | References organizations(id) |
| default_km_rate | DECIMAL | Mileage rate if no contract rate (nullable) |
| currency | VARCHAR | Default currency (e.g., \"USD\") |
| week_start_day | INTEGER | 0=Sunday, 1=Monday, etc. |
| timezone | VARCHAR | e.g., \"UTC\", \"America/New_York\" |
| show_approval_history | BOOLEAN | Display approval chain to users |
| created_at | TIMESTAMP | Creation time |
| updated_at | TIMESTAMP | Last update |

---

## Roles

### employee

**Permissions:**
- Create own time entries, expenses
- View own submissions
- Cannot approve others
- Cannot see other employees' submissions

### manager

**Permissions:**
- All employee permissions
- Approve time entries and expenses
- View all org entries
- Assign managers to projects
- Cannot approve at finance level

### finance

**Permissions:**
- All manager permissions
- Final approval of entries
- Approve/reject at finance level
- Export reports and CSV
- Access financial summaries

### customer

**Permissions:**
- Read-only access
- View invoicing info (future)
- See submitted (approved) entries related to their contracts
- Cannot approve or modify

---

## API Endpoints

### Organizations

#### POST /organizations (Create)

**Request:**
```json
{
  \"name\": \"My Company\",
  \"slug\": \"my-company\"
}
```

**Flow:**
1. Validate unique slug
2. Create organization
3. Create organization_settings with defaults
4. Create membership for creator as manager
5. Return organization

**Response (201):**
```json
{
  \"data\": {
    \"id\": \"org-uuid\",
    \"name\": \"My Company\",
    \"slug\": \"my-company\",
    \"created_at\": \"2024-04-01T10:00:00Z\"
  }
}
```

#### GET /organizations/{id} (Get)

Returns org with basic info (slug, name, created_at)

#### GET /organizations/{id}/settings (Get Settings)

Returns organization_settings

**Response (200):**
```json
{
  \"data\": {
    \"organization_id\": \"org-uuid\",
    \"default_km_rate\": 0.50,
    \"currency\": \"EUR\",
    \"week_start_day\": 1,
    \"timezone\": \"Europe/Berlin\",
    \"show_approval_history\": true,
    \"created_at\": \"2024-04-01T10:00:00Z\",
    \"updated_at\": \"2024-04-01T10:00:00Z\"
  }
}
```

#### PUT /organizations/{id}/settings (Update Settings)

**Request:**
```json
{
  \"default_km_rate\": 0.55,
  \"currency\": \"EUR\",
  \"week_start_day\": 1,
  \"timezone\": \"Europe/Berlin\",
  \"show_approval_history\": true
}
```

**Permissions:** Org manager or finance only

---

### User Management

#### POST /organizations/{id}/invite (Invite Member)

**Request:**
```json
{
  \"email\": \"newuser@company.com\",
  \"role\": \"employee\"
}
```

**Flow:**
1. Check if user with email exists
   - If yes: create membership record with role
   - If no: create unverified user + membership (pending activation)
2. Send invite email with activation link
3. Return membership record

**Response (201):**
```json
{
  \"data\": {
    \"id\": \"membership-uuid\",
    \"user_id\": \"user-uuid\",
    \"organization_id\": \"org-uuid\",
    \"role\": \"employee\",
    \"is_active\": false,
    \"invited_at\": \"2024-04-01T10:00:00Z\",
    \"activated_at\": null
  }
}
```

#### GET /organizations/{id}/members (List Members)

**Query:**
```
?role=manager  # Filter by role
?is_active=true
```

**Response (200):**
```json
{
  \"data\": [
    {
      \"id\": \"membership-uuid\",
      \"user\": {
        \"id\": \"user-uuid\",
        \"email\": \"user@company.com\",
        \"name\": \"John Doe\"
      },
      \"role\": \"manager\",
      \"is_active\": true,
      \"activated_at\": \"2024-04-01T11:00:00Z\"
    }
  ]
}
```

#### PUT /organizations/{id}/members/{member_id}/roles (Update Role)

**Request:**
```json
{
  \"role\": \"finance\"
}
```

**Permissions:** Org manager or finance only

**Response (200):** Updated membership

#### DELETE /organizations/{id}/members/{member_id} (Remove Member)

Soft-deletes membership (sets is_active = false)

**Permissions:** Org manager or finance only

---

### User Switching

#### POST /auth/switch-org (Switch Organization)

**Request:**
```json
{
  \"organization_id\": \"different-org-uuid\"
}
```

**Flow:**
1. Validate user has active membership in target org
2. Generate new JWT with updated org_id
3. Return new token

**Response (200):**
```json
{
  \"data\": {
    \"access_token\": \"new-jwt-token\",
    \"organization_id\": \"different-org-uuid\"
  }
}
```

**Frontend Usage:**
```javascript
// User switches org via dropdown
const newOrg = selectOrgDropdown()
const { data } = await api.post('/auth/switch-org', {
  organization_id: newOrg.id
})
localStorage.setItem('access_token', data.data.access_token)
localStorage.setItem('current_org', newOrg.id)
// Refresh UI with new org context
```

---

## Frontend Components

### Organization Switcher

Located in app shell / header

**Features:**
- Dropdown showing all user's orgs
- Current org highlighted
- Click to switch
- Shows user's role in each org

```typescript
function OrgSwitcher() {
  const { user } = useAuth()
  const [selectedOrg, setSelectedOrg] = useState(user.organization)
  const switchOrgMutation = useMutation(switchOrgMutation())
  
  const handleOrgChange = async (orgId: string) => {
    await switchOrgMutation.mutateAsync({ organization_id: orgId })
    // Refetch profile, data
  }
  
  return (
    <select value={selectedOrg.id} onChange={e => handleOrgChange(e.target.value)}>
      {user.organizations.map(org => (
        <option key={org.id} value={org.id}>
          {org.name} ({user.roleInOrg[org.id]})
        </option>
      ))}
    </select>
  )
}
```

### Member Management

**Org Settings → Members Page**

**Features:**
- List current members with roles
- Add member (invite by email)
- Edit member role (dropdown)
- Remove member button
- Permissions: only manager/finance can manage

### Settings Page

**Org Settings → General**

**Features:**
- Organization name (display only for non-admins)
- Timezone selector
- Currency selector
- Week start day selector
- Default km rate input
- Approval history visibility toggle
- Save button

---

## Business Rules

### Membership Activation

**New user invited:**
1. Membership created with is_active=false
2. Email sent with activation link
3. User clicks link, completes registration
4. Membership updated: is_active=true, activated_at=now

**Existing user invited:**
1. Membership created immediately with is_active=true
2. User can start using org right away

### Role Permissions

| Action | Employee | Manager | Finance | Customer |
|--------|----------|---------|---------|----------|
| Create time entry | ✓ | ✓ | ✓ | ✗ |
| View own | ✓ | ✓ | ✓ | ✗ |
| View all | ✗ | ✓ | ✓ | ~ |
| Approve (manager) | ✗ | ✓ | ✓ | ✗ |
| Approve (finance) | ✗ | ✗ | ✓ | ✗ |
| Manage org | ✗ | ✓ | ✓ | ✗ |
| Export CSV | ✗ | ✓ | ✓ | ✗ |
| Edit settings | ✗ | ✓ | ✓ | ✗ |
| Invite members | ✗ | ✓ | ✓ | ✗ |

---

## Workflows

### Onboard New Employee

```
1. Manager navigates to Org Settings → Members
2. Enters employee email and selects role=\"employee\"
3. System sends invite email
4. Employee clicks link, creates password
5. Employee membership activated
6. Employee can now log time entries
7. Manager sees entries for approval
```

### Change Employee Role

```
1. Manager navigates to Members page
2. Finds employee in list
3. Clicks edit role → selects \"manager\"
4. Role updated immediately
5. Employee can now approve entries
6. UI updates on employee's next interaction
```

### Multi-Org User

```
1. User belongs to Org A (as manager) and Org B (as employee)
2. User logs in (defaults to Org A)
3. Creates time entry in Org A context
4. Switches org via dropdown → Org B
5. New JWT token issued with org_id=Org B
6. Sees Org B data
7. Can create time entry in Org B (as employee)
8. Cannot approve (lacks manager role in Org B)
```

---

## Data Isolation

**Key principle:** All queries filter by current org_id

```go
// Bad: Returns data from all orgs
SELECT * FROM time_entries WHERE user_id = $1

// Good: Org-scoped
SELECT * FROM time_entries 
WHERE user_id = $1 AND organization_id = (
  SELECT organization_id FROM organization_memberships 
  WHERE user_id = $1 AND is_active = true LIMIT 1
)
```

**Frontend equivalent:**
```typescript
// Always include current org in queries
const { data } = await api.get('/time-entries', {
  params: { organization_id: currentOrg.id }
})
```

---

## Type System

```typescript
export interface Organization {
  id: string
  name: string
  slug: string
  created_at: string
}

export interface OrganizationMembership {
  id: string
  user_id: string
  organization_id: string
  role: 'employee' | 'manager' | 'finance' | 'customer'
  is_active: boolean
  invited_at?: string
  activated_at?: string
}

export interface OrganizationSettings {
  organization_id: string
  default_km_rate?: number
  currency: string
  week_start_day: number
  timezone: string
  show_approval_history: boolean
  created_at: string
  updated_at: string
}
```

---

**Next**: [[14-CSV-Exports]] for reporting.
