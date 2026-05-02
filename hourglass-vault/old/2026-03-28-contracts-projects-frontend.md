# Contracts & Projects Frontend (Phase 2)

**Vertical slice: Contract and Project management with shared resources**

---

## 1. Overview

This phase implements the frontend for managing contracts and projects - shared resources that time entries and expenses are linked to. Users can view, create, and adopt contracts and projects through a tabbed interface.

### Scope

**In scope:**
- Contracts list page with Owned/Adopted/All tabs
- Projects list page with Owned/Adopted/All tabs
- Search by name filter
- Create contract/project dialogs
- Detail pages for viewing resource information
- Adopt flow with confirmation dialog
- Backend update to include `created_by_org_name`

**Out of scope:**
- Edit/delete contracts/projects (Phase 6 governance flow)
- Governance edit requests and voting
- Customer access restrictions

---

## 2. URL Structure

```
/contracts                              → Contracts list (default: Owned tab)
/contracts?tab=adopted                  → Contracts list (Adopted tab)
/contracts?tab=all                      → Contracts list (All/shared catalog)
/contracts/:id                          → Contract detail
/contracts/:id?from=owned               → Contract detail (back context)

/projects                               → Projects list (default: Owned tab)
/projects?tab=adopted                   → Projects list (Adopted tab)
/projects?tab=all                       → Projects list (All/shared catalog)
/projects/:id                           → Project detail
/projects/:id?from=all                  → Project detail (back context)
/contracts/:contractId/projects/:projectId → Project detail (nested route)
```

---

## 3. List Page Components

### Contracts List (`/contracts`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Contracts                                          [Search...]  │
├─────────────────────────────────────────────────────────────────┤
│ [Owned] [Adopted] [All]                           [+ Create]   │
├─────────────────────────────────────────────────────────────────┤
│  🔒 Acme Contract                                               │
│     Creator Controlled                                          │
├─────────────────────────────────────────────────────────────────┤
│  🌐 Shared Services                            [Adopt]          │
│     Unanimous · Adopted by 3 orgs                               │
└─────────────────────────────────────────────────────────────────┘
```

**Owned tab:**
- Shows contracts created by user's org
- Create button visible
- Private icon (🔒) for `is_shared: false`
- Shared badge for `is_shared: true`
- Click row → detail page

**Adopted tab:**
- Shows contracts adopted by user's org
- Each row shows "Adopted from [Org Name]"
- Shared badge
- No create button

**All tab (catalog):**
- Shows all shared contracts in system
- Adopted items show "Already adopted" badge + disabled adopt button
- Non-adopted items show "Adopt" button
- No create button

**Row content (minimal):**
- Icon/badge indicating shared/private status
- Name
- Governance model badge

---

### Projects List (`/projects`)

Same structure as contracts, with:

**Row content (minimal):**
- Icon/badge indicating shared/private status
- Name
- Type badge (Billable/Internal)

**Create button:** Only visible in Owned tab

---

## 4. Create Dialogs

### Create Contract Dialog

**Fields:**
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| Name | text input | Yes | |
| KM Rate | number input | No | Default: 0 |
| Currency | select dropdown | Yes | Options: EUR, USD, GBP, CHF, JPY, CAD, AUD |
| Governance Model | select dropdown | Yes | With descriptions |
| Shared | checkbox | No | Default: unchecked |

**Governance Model Options:**
```
[Creator Controlled]
"Only your organization can approve changes to this contract"

[Unanimous]
"All organizations using this contract must approve changes"

[Majority]
"More than half of organizations using this contract must approve changes"
```

**On success:**
- Close dialog
- Show success toast: "Contract created"
- Navigate to contract detail page

---

### Create Project Dialog

**Fields:**
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| Name | text input | Yes | |
| Type | select dropdown | Yes | Options: Billable, Internal |
| Contract | select dropdown | Yes | All accessible contracts (owned + adopted) |
| Governance Model | select dropdown | Yes | Same options as contract |
| Shared | checkbox | No | Default: unchecked |

**On success:**
- Close dialog
- Show success toast: "Project created"
- Navigate to project detail page

---

## 5. Detail Pages

### Contract Detail (`/contracts/:id`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ ← Back to Contracts                                              │
├─────────────────────────────────────────────────────────────────┤
│ Acme Contract                                    [Edit] [Delete]│
│ 🔒 Private (or 🌐 Shared)                                        │
│ Adopted from: [Org Name] (only if adopted)                       │
├─────────────────────────────────────────────────────────────────┤
│ Details                                                          │
│ ─────────────────────────────────────────────────────────────── │
│ KM Rate: €0.50/km                                               │
│ Currency: EUR                                                   │
│ Governance: Creator Controlled                                   │
│ Adoption Count: 0 (or N if shared)                               │
├─────────────────────────────────────────────────────────────────┤
│ [Edit and Delete buttons disabled with "Coming soon" tooltip]   │
└─────────────────────────────────────────────────────────────────┘
```

**Content sections:**
- Header: Name, shared/private indicator
- If adopted: "Adopted from [Org Name]" text
- Info card: KM Rate, Currency, Governance Model, Adoption Count
- Actions: Edit (disabled), Delete (disabled) with "Coming soon" tooltip

---

### Project Detail (`/projects/:id`)

**Layout:**
```
┌─────────────────────────────────────────────────────────────────┐
│ ← Back to Projects                                               │
├─────────────────────────────────────────────────────────────────┤
│ Client Portal                                    [Edit] [Delete] │
│ [Billable] 🌐 Shared                                             │
│ Adopted from: [Org Name] (only if adopted)                       │
├─────────────────────────────────────────────────────────────────┤
│ Details                                                          │
│ ─────────────────────────────────────────────────────────────── │
│ Contract: Acme Contract                                          │
│ Type: Billable                                                   │
│ Governance: Unanimous                                            │
│ Adoption Count: 3                                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Adoption Flow

**Trigger:** User clicks "Adopt" on a shared item in the "All" tab

**Confirmation Dialog:**
```
┌─────────────────────────────────────────────────────────────────┐
│ Adopt [Name]?                                                   │
│                                                                 │
│ This will make it available for your organization's            │
│ time entries and expenses.                                      │
│                                                                 │
│ [Cancel] [Adopt]                                                │
└─────────────────────────────────────────────────────────────────┘
```

**On success:**
- Close dialog
- Success toast: "[Name] has been added to your organization"
- Refresh list (item now appears in Adopted tab)
- In All tab: item shows "Already adopted" badge with disabled adopt button

---

## 7. API Client Changes

### contracts.ts (new file)

```typescript
// Query Options
contractsQueryOpts(scope: 'owned' | 'adopted' | 'all') 
contractQueryOpts(id: string)

// Mutations
createContractMutationOpts(data: CreateContractRequest)
adoptContractMutationOpts(id: string)
```

### projects.ts (updated)

```typescript
// Add to existing file
projectsQueryOpts(scope: 'owned' | 'adopted' | 'all', contractId?: string)
projectQueryOpts(id: string)
createProjectMutationOpts(data: CreateProjectRequest)
adoptProjectMutationOpts(id: string)
```

---

## 8. Backend Changes

**Add `created_by_org_name` to responses:**

1. `GET /contracts` - JOIN with `organizations` table
2. `GET /contracts/:id` - JOIN with `organizations` table
3. `GET /projects` - JOIN with `organizations` table
4. `GET /projects/:id` - JOIN with `organizations` table

**Response changes:**

```go
type ContractResponse struct {
    models.Contract
    CreatedByOrgName string `json:"created_by_org_name,omitempty"`
    AdoptionCount    int    `json:"adoption_count,omitempty"`
}
```

```go
type ProjectResponse struct {
    models.Project
    ContractName     string `json:"contract_name,omitempty"`
    CreatedByOrgName string `json:"created_by_org_name,omitempty"`
    AdoptionCount    int    `json:"adoption_count,omitempty"`
}
```

---

## 9. Types (api.ts)

```typescript
export interface Contract {
  id: string
  name: string
  km_rate: number
  currency: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
  is_active: boolean
  created_by_org_id: string
  created_by_org_name?: string
  adoption_count?: number
  created_at: string
}

export interface Project {
  id: string
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  contract_name?: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
  is_active: boolean
  created_by_org_id: string
  created_by_org_name?: string
  adoption_count?: number
  created_at: string
}

export interface CreateContractRequest {
  name: string
  km_rate: number
  currency: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
}

export interface CreateProjectRequest {
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
}
```

---

## 10. File Structure

**New files:**

```
web/src/
├── api/
│   └── contracts.ts              # Contract API client
├── routes/
│   └── _authenticated/
│       ├── contracts/
│       │   ├── index.tsx         # Contracts list page
│       │   ├── index.tsx           # Contract detail page
│       │   └── -components/
│       │       ├── contract-list.tsx
│       │       ├── contract-row.tsx
│       │       └── create-contract-dialog.tsx
│       └── projects/
│           ├── index.tsx         # Projects list page
│           ├── index.tsx           # Project detail page
│           └── -components/
│               ├── project-list.tsx
│               ├── project-row.tsx
│               └── create-project-dialog.tsx
```

**Modified files:**

```
web/src/
├── api/
│   └── projects.ts               # Add mutations
├── types/
│   └── api.ts                    # Add contract/project types
├── components/
│   └── layout/
│       └── sidebar.tsx           # Enable nav links
```

**Backend files:**

```
internal/handlers/
├── contract_handler.go           # Add created_by_org_name
└── project_handler.go            # Add created_by_org_name
```

---

## 11. Implementation Order

1. **Backend update** - Add `created_by_org_name` to contract/project responses
2. **Types** - Add Contract, Project, CreateContractRequest, CreateProjectRequest
3. **API clients** - Create `contracts.ts`, update `projects.ts`
4. **Contracts list page** - Index route with tabs and search
5. **Contract detail page** - Show contract info
6. **Create contract dialog** - Form with governance dropdown
7. **Adopt contract dialog** - Confirmation flow
8. **Projects list page** - Index route with tabs and search
9. **Project detail page** - Show project info
10. **Create project dialog** - Form with contract dropdown
11. **Adopt project dialog** - Confirmation flow
12. **Sidebar** - Enable nav links for Contracts and Projects

---

## 12. Success Criteria

**When complete, a user can:**

1. Navigate to Contracts page from sidebar
2. See owned contracts in default tab
3. Switch to Adopted tab to see adopted contracts
4. Switch to All tab to browse shared contracts catalog
5. Search contracts by name
6. Create a new contract from Owned tab
7. View contract details
8. Adopt a shared contract from All tab (with confirmation)
9. See "Already adopted" badge on owned items in All tab

10. Navigate to Projects page from sidebar
11. See owned projects in default tab
12. Switch between Owned/Adopted/All tabs
13. Search projects by name
14. Create a new project (selecting from accessible contracts)
15. View project details
16. Adopt a shared project from All tab
