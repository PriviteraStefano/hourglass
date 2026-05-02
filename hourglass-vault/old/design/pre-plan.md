# SurrealDB Migration Architecture - Visual Design

## 1. Organization Hierarchy Structure

The foundational org structure showing units at different levels:

```
┌─────────────────────────────────────────────┐
│           Company (Organization)            │
│         └─ hierarchy_level: 0               │
└─────────────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   ┌─────────┐  ┌─────────┐  ┌─────────┐
   │Division │  │Division │  │Division │
   │  Tech   │  │Product  │  │Finance  │
   │(Level 1)│  │ (Lvl 1) │  │ (Lvl 1) │
   └────┬────┘  └────┬────┘  └─────────┘
        │            │
    ┌───┴───┐      ┌──┴──┐
    ▼       ▼      ▼     ▼
  ┌──────┐┌──────┐┌──────┐┌──────┐
  │ R&D  ││Infra ││Web   ││Mobile│
  │(L.2) ││(L.2) ││(L.2) ││(L.2) │
  └──┬───┘└──────┘└──────┘└──────┘
     │
  ┌──┴──┬──────┐
  ▼     ▼      ▼
┌────┐┌────┐┌────┐
│ ML ││Data││Core│
│(L3)││(L3)││(L3)│
└────┘└────┘└────┘
```

**Key Points:**
- Each unit has a unique path from root (Company)
- Units track parent_unit_id for hierarchy queries
- Users can belong to multiple units at any level
- Roles in units cascade down (e.g., Division manager has authority over all subordinate departments/teams)

---

## 2. Working Groups & Projects Relationship

Working groups are separate from org hierarchy — they're execution units:

```
┌──────────────────────────────────────────────────────────────┐
│                        PROJECT                               │
│  "Build New Analytics Platform"  [Composite Project]         │
└──────────────────────────────────────────────────────────────┘
        │
        ├─────────────────────────────────────┐
        ▼                                     ▼
┌─────────────────────┐          ┌─────────────────────┐
│   SUBPROJECT #1     │          │   SUBPROJECT #2     │
│  "Backend API"      │          │  "Frontend UI"      │
│  [Technical]        │          │  [Technical]        │
└────────┬────────────┘          └────────┬────────────┘
         │                               │
    ┌────┴─────────────┐            ┌────┴─────────────┐
    ▼                  ▼            ▼                  ▼
┌─────────────┐  ┌──────────────┐┌────────────┐┌──────────────┐
│Working Grp  │  │Working Grp   ││Working Grp ││Working Grp   │
│"Backend"    │  │"Integration" ││"Frontend"  ││"Mobile"      │
│             │  │              ││            ││              │
│Units:R&D+   │  │Units:Infra   ││Units:Web   ││Units:Mobile  │
│Infra        │  │+External     ││            ││              │
│Manager:John │  │Manager:Alice ││Manager:Bob ││Manager:Carol │
│             │  │              ││            ││              │
└──────┬──────┘  └──────┬───────┘└────────┬───┘└──────┬───────┘
       │                │                 │           │
   Users from       Users from         Users from   Users from
   R&D+Infra       Infra+External     Web unit     Mobile unit
   units           units              units        units
```

**Key Insight:**
- **Working groups** != **Units**
- Working group pulls users from one or more units
- Each working group has its own manager(s)
- Working groups execute projects/subprojects
- Units exist in org hierarchy (reporting, KPIs, resource tracking)

---

## 3. User & Unit Relationship

A single user can belong to multiple units and multiple working groups:

```
┌────────────────────────────────────────┐
│         USER: Sarah (ID: user-123)     │
└────────────────────────────────────────┘
         │
         ├─ Unit Membership #1
         │  └─ Unit: Tech/R&D (Level 2)
         │     └─ Role: engineer
         │     └─ Start Date: 2024-01-01
         │
         ├─ Unit Membership #2
         │  └─ Unit: Tech/R&D/ML (Level 3)
         │     └─ Role: senior_engineer
         │     └─ Start Date: 2024-06-01
         │
         └─ Working Group Memberships
            ├─ WG: "Backend" [Project: New Platform]
            │  └─ Manager: John
            │  └─ Role: backend_engineer
            │
            ├─ WG: "Data Pipeline" [Project: Analytics]
            │  └─ Manager: Alice
            │  └─ Role: data_engineer
            │
            └─ WG: "Infra" [Project: Cloud Migration]
               └─ Manager: Bob
               └─ Role: infra_engineer

ORGANIZATION VIEW (Hierarchy):
  - Sarah works in R&D
  - Also assigned to ML team (subunit)
  - Resource allocation: 60% R&D, 40% Process Improvement projects

WORKING GROUP VIEW (Projects):
  - Sarah codes on Backend (reports to John for Backend work)
  - Sarah codes on Data Pipeline (reports to Alice for Data work)
  - Sarah codes on Infra (reports to Bob for Infra work)
```

---

## 4. Time Entry & Approval Flow

How a time entry moves through the system:

```
┌─────────────────────────────────────────────────────────────┐
│  STEP 1: User Submits Time Entry                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Sarah logs: "Worked on Backend API"                        │
│  ├─ Project: "Build New Analytics Platform"                 │
│  ├─ Subproject: "Backend API"  (auto-filled from default)   │
│  │  └─ This determines: Unit = Tech/R&D, WG = Backend       │
│  ├─ Hours: 8                                                │
│  ├─ Date: 2026-04-11                                        │
│  └─ Status: DRAFT                                           │
│                                                             │
│  NOTE: Sarah COULD override to "Data Pipeline" if needed    │
│        but Backend is her default for this project          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 2: User Submits Entry                                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Status: SUBMITTED                                          │
│  Current Approver: Backend Working Group Manager (John)     │
│                                                             │
│  System determines approver by:                             │
│  1. Find working group for subproject                       │
│  2. Get working group manager(s)                            │
│  3. Route to manager                                        │
│                                                             │
│  IMPORTANT: Org hierarchy NOT consulted for approval!       │
│  Sarah's manager in R&D is not involved.                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 3: Manager Reviews & Approves                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  John (Backend WG Manager) reviews:                         │
│  ├─ Sarah worked 8 hours on Backend API               ✓     │
│  ├─ Scope is within project                           ✓     │
│  └─ Approves                                          ✓     │
│                                                             │
│  Status: APPROVED                                           │
│  Approval recorded immutably                                │
│                                                             │
│  NOTE: R&D Division manager (Sarah's org manager)           │
│        never sees this entry                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  STEP 4: Org Hierarchy Uses for Tracking                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  R&D Manager (Sarah's org manager) can later run reports:   │
│  ├─ "Sarah spent 30% time on Analytics project"             │
│  ├─ "Sarah primarily in R&D but 40% in Process Improv"      │
│  ├─ "We're overalloc'd: Sarah split 3 ways"                 │
│  └─ "Recommend: dedicate to one team or hire specialist"    │
│                                                             │
│  This is REPORTING, not APPROVAL                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Complete Example Scenario

**Real-world composite project with multiple units:**

```
┌────────────────────────────────────────────────────────────────┐
│             PROJECT: "ERP System Implementation"               │
│             Type: Composite (multiple subprojects)             │
└────────────────────────────────────────────────────────────────┘
              │
    ┌─────────┼─────────┬──────────┐
    ▼         ▼         ▼          ▼
┌────────┐┌────────┐┌────────┐┌────────┐
│  Mgmt  ││ Tech   ││ Change ││ Vendor │
│Config  ││ Build  ││ Mgmt   ││ Mgmt   │
└───┬────┘└───┬────┘└───┬────┘└───┬────┘
    │         │         │         │
    │         │         │         └─ External vendor (separate)
    │         │         │
    ▼         ▼         ▼
  ┌──────────────────────────────────────┐
  │    ORGANIZATION UNITS                │
  │  (where time gets attributed)        │
  ├──────────────────────────────────────┤
  │  Finance/Ops (Mgmt)                  │
  │  Tech/R&D (Tech Build)               │
  │  HR/Org Dev (Change Mgmt)            │
  └──────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│               WORKING GROUPS & MANAGERS                        │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  WG: "ERP Management"                                          │
│  ├─ Subproject: "Management & Config"                          │
│  ├─ Units: Finance/Ops                                         │
│  ├─ Manager: Sarah (Finance Director)                          │
│  └─ Users: [Finance team members]                              │
│                                                                │
│  WG: "ERP Technical"                                           │
│  ├─ Subproject: "Technical Build"                              │
│  ├─ Units: Tech/R&D, Tech/Infra                                │
│  ├─ Manager: John (Engineering Lead)                           │
│  └─ Users: [Backend engineers, Infra engineers]                │
│                                                                │
│  WG: "ERP Change Management"                                   │
│  ├─ Subproject: "Change Management"                            │
│  ├─ Units: HR/Org Dev, Finance/Ops                             │
│  ├─ Manager: Alice (Change Director)                           │
│  └─ Users: [HR specialists, Trainers]                          │
│                                                                │
│  WG: "Vendor Management" (External)                            │
│  ├─ Subproject: "Vendor Coordination"                          │
│  ├─ Units: N/A (external vendor)                               │
│  ├─ Manager: Bob (Vendor PM)                                   │
│  └─ Users: [Vendor staff]                                      │
│                                                                │
└────────────────────────────────────────────────────────────────┘

TIME ENTRY EXAMPLES:

User: Mike (Tech/R&D/Backend engineer)
  └─ Logs 6 hours "ERP System Implementation"
     Subproject options shown:
     ✓ Technical Build (DEFAULT - his assignment)
     ○ Management & Config
     ○ Change Management
     Action: Selects "Technical Build" (default) → Status: APPROVED by John
     Effect: Hours attributed to Tech/R&D unit, Technical WG

User: Lisa (Finance/Ops analyst)
  └─ Logs 4 hours "ERP System Implementation"
     Subproject options shown:
     ○ Technical Build
     ✓ Management & Config (DEFAULT - her assignment)
     ○ Change Management
     Action: Selects "Management & Config" (default) → Status: APPROVED by Sarah
     Effect: Hours attributed to Finance/Ops unit, Management WG

User: Tom (HR/Org Dev specialist)
  └─ Logs 3 hours "ERP System Implementation"
     Subproject options shown:
     ○ Technical Build
     ○ Management & Config
     ✓ Change Management (DEFAULT - his assignment)
     Action: Selects "Change Management" (default) → Status: APPROVED by Alice
     Effect: Hours attributed to HR/Org Dev unit, Change Management WG

ORGANIZATIONAL REPORTING (R&D Manager perspective):
  ├─ Mike: 6h ERP (5% allocation)  ← "Too much on project work, need on core R&D"
  └─ Capacity planning decision

FINANCIAL REPORTING (Finance Director perspective):
  ├─ Mgmt & Config cost: Lisa 4h @ $X = $4X
  ├─ Tech Build cost: Mike 6h @ $Y = $6Y
  ├─ Change Mgmt cost: Tom 3h @ $Z = $3Z
  └─ Project profitability: Cost breakdown by unit
```

---

## 6. Data Isolation & Authority Cascade

How authority flows in the hierarchy for non-approval tasks:

```
┌─────────────────────────────────────────────────────────────┐
│  ORGANIZATION HIERARCHY (for reporting/permissions)         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│            Company (CEO)                                    │
│            ├─ can_manage: all units                         │
│                                                             │
│            ├─ Tech Division (VP Eng)                        │
│            │  ├─ can_manage: all tech units below           │
│            │  │                                             │
│            │  ├─ R&D Department (R&D Manager)               │
│            │  │  ├─ can_manage: R&D + all subunits          │
│            │  │  │                                          │
│            │  │  ├─ ML Team (ML Lead)                       │
│            │  │  │  └─ can_manage: ML team only             │
│            │  │  │                                          │
│            │  │  └─ Data Team (Data Lead)                   │
│            │  │     └─ can_manage: Data team only           │
│            │  │                                             │
│            │  └─ Infra Department (Infra Manager)           │
│            │     ├─ can_manage: Infra + all subunits        │
│            │     ├─ Cloud Team                              │
│            │     └─ On-Prem Team                            │
│            │                                                │
│            └─ Product Division (VP Product)                 │
│               ├─ Web Department                             │
│               └─ Mobile Department                          │
│                                                             │
└─────────────────────────────────────────────────────────────┘

AUTHORITY CASCADE RULES:

1. View Reports: A manager can see all units below them
   └─ R&D Manager sees: R&D + ML + Data
   └─ ML Lead sees: ML team only
   └─ CEO sees: everything

2. Resource Allocation: A manager can view allocation of their unit members
   └─ R&D Manager: "What % of time does each R&D person spend on projects?"
   └─ ML Lead: "What % of ML team is on project X?"

3. Manage Structure: A manager can add subunits below them
   └─ R&D Manager CAN create "QA Team" under R&D
   └─ ML Lead CANNOT create units

4. Approve Time Entries: NOT inherited from hierarchy
   └─ Only working group managers approve
   └─ Org hierarchy managers do NOT approve entries
```

---

## 7. Schema Relationships Summary

```
ORGANIZATION LAYER:
  Organization
    ├─ Units (hierarchy: parent_unit_id)
    ├─ Users (via unit_memberships: multiple units per user)
    ├─ Settings
    └─ Customers

WORKING GROUP LAYER:
  Projects
    ├─ Project Subprojects (optional, for composite projects)
    ├─ Working Groups
    │  └─ Working Group Members (from various units)
    │  └─ Working Group Managers
    └─ Project Managers

TIME TRACKING LAYER:
  Time Entries
    ├─ Project reference
    ├─ Subproject reference (optional)
    ├─ User reference
    ├─ Unit reference (which unit's work this is for)
    └─ Working Group reference (who approves)
  
  Time Entry Approvals (WG Manager level, not org hierarchy)
    └─ immutable approval history

SAME FOR EXPENSES:
  Expenses
    ├─ Unit reference
    └─ Working Group Managers approve

SEPARATION:
  ✗ Time entries NOT filtered/managed by org hierarchy managers
  ✓ Time entries APPROVED by working group managers
  ✓ Org hierarchy used for: reporting, KPIs, resource tracking, permission levels
  ✓ Working groups used for: project execution, time tracking, approval
```

---

This is what we'll migrate to SurrealDB. Next step: detailed schema design with SurrealDB tables and relationships.
