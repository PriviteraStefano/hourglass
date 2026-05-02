# Quick Start: Creating GitHub Issues for SurrealDB Migration

**Status:** ✅ All 25 issue templates ready to create

**Files you need:**
- `GITHUB_ISSUES_GRANULAR.md` — Full issue templates (copy-paste ready)
- `GITHUB_ISSUE_CREATION_GUIDE.md` — Step-by-step creation instructions
- `ISSUE_STRUCTURE_OVERVIEW.txt` — Visual overview & dependencies

---

## 30-Second Summary

**What you're creating:**
- 1 master epic issue (umbrella for entire SurrealDB migration)
- 24 granular sub-issues (independently grabbable work items)

**Timeline:** 6 weeks to production

**Team:** 2 backend, 1 frontend, 1 QA, 1 DevOps

**Phases:**
1. **Phase 1** (1 week) - Foundation: Deploy SurrealDB, load schema, auth, CRUD
2. **Phase 2** (1.5 weeks) - Approvals: WG manager approvals, splits, moves
3. **Phase 3** (1 week) - Finance: Cutoff enforcement, Finance override, expenses
4. **Phase 4** (1 week) - Org: Org manager reallocation, hierarchy queries, dashboard
5. **Phase 5** (1 week) - Reporting: Audit queries, compliance reports, dashboards

---

## Fastest Way to Create Issues (5 minutes)

### Option A: GitHub CLI (Recommended)

```bash
# 1. Create master issue
gh issue create \
  --title "[EPIC] SurrealDB Migration: Full Implementation (6 Weeks)" \
  --label "epic,surrealdb,priority-high" \
  --body "Master issue for SurrealDB migration. Connects 24 sub-issues.

Timeline: 6 weeks
Team: 2 backend, 1 frontend, 1 QA, 1 DevOps
Goal: Complete migration with all features, reporting, compliance.

See GITHUB_ISSUES_GRANULAR.md for full sub-issue details." \
  --repo [owner]/hourglass
```

Note the master issue number (e.g., #42).

```bash
# 2. Create Phase 1 sub-issues (4 issues)
# Copy-paste from GITHUB_ISSUES_GRANULAR.md, one per command

gh issue create --title "[Phase 1.1] ..." --label "phase-1,devops" ... --repo [owner]/hourglass
gh issue create --title "[Phase 1.2] ..." --label "phase-1,backend" ... --repo [owner]/hourglass
gh issue create --title "[Phase 1.3] ..." --label "phase-1,backend" ... --repo [owner]/hourglass
gh issue create --title "[Phase 1.4] ..." --label "phase-1,qa" ... --repo [owner]/hourglass
```

### Option B: GitHub Web (5 minutes)

1. Open https://github.com/[owner]/hourglass/issues/new
2. Create master issue (copy title & body from `GITHUB_ISSUES_GRANULAR.md`)
3. Create Phase 1 sub-issues (4 issues, copy-paste from template)
4. Add labels: `phase-1`, `surrealdb`, role-based (`backend`, `devops`, `qa`)

### Option C: GitHub Projects (Recommended Long-Term)

1. Create project: https://github.com/[owner]/hourglass/projects/new
2. Create master + Phase 1 issues (Option A or B)
3. Add issues to project board
4. Track progress with Status field (Not Started → In Progress → Done)

---

## What's in Each Phase

### Phase 1: Foundation (4 issues)
- **1.1** Deploy SurrealDB cluster, load schema
- **1.2** JWT auth endpoints (register, login, me)
- **1.3** Time entry CRUD (create, read, list)
- **1.4** Phase 1 testing & acceptance

### Phase 2: Approval Workflow (5 issues)
- **2.1** Entry submission & status tracking
- **2.2** WG manager approval & rejection
- **2.3** Time entry splitting (5h accepted + 3h rejected)
- **2.4** Time entry moving to different projects
- **2.5** Phase 2 testing & performance

### Phase 3: Financial Controls (4 issues)
- **3.1** Financial cutoff enforcement
- **3.2** Finance override capability
- **3.3** Expense CRUD & budget caps
- **3.4** Phase 3 testing & migration prep

### Phase 4: Org Hierarchy (4 issues)
- **4.1** Org manager reallocation
- **4.2** Recursive hierarchy queries
- **4.3** Org structure dashboard (UI)
- **4.4** Phase 4 testing & acceptance

### Phase 5: Reporting (5 issues)
- **5.1** Audit trail query endpoints
- **5.2** Compliance & allocation reports
- **5.3** Reporting dashboards (UI)
- **5.4** Performance tuning & load tests
- **5.5** Final acceptance & production deployment

---

## Issue Format

Each issue includes:

```
Title: [Phase X.Y] Brief Description

Description:
- What to build (end-to-end behavior)
- Tasks (detailed checklist)
- Acceptance criteria (what "done" looks like)
- Testing strategy
- Dependencies
- Estimated duration
- Team allocation
```

---

## Team Assignment

After creating Phase 1 issues:

1. Assign **1.1** to DevOps person
2. Assign **1.2** to Backend person #1
3. Assign **1.3** to Backend person #2
4. Assign **1.4** to QA person

Start Phase 1 work immediately. Create Phase 2+ issues as Phase 1 nears completion.

---

## Labels to Use

```
Phases:        phase-1, phase-2, phase-3, phase-4, phase-5
Roles:         backend, frontend, devops, qa
Types:         infrastructure, api, database, ui, testing, performance
Status:        blocked, in-progress, review
Priority:      priority-high, priority-medium
Surrealdb:     surrealdb, migration
```

---

## Key Decisions (Already Made)

✅ **Approval Authority:** WG manager approves, org manager reallocates post-approval  
✅ **Financial Cutoff:** Entries lock after date, Finance can override with reason  
✅ **Time Status:** Draft → Submitted → Approved (3 states only)  
✅ **Audit Trail:** Immutable, tracks all actions  
✅ **Org Hierarchy:** Unlimited nesting via parent_unit_id  

---

## Success Metrics Per Phase

**Phase 1:** Time entry CRUD working, schema deployed ✓  
**Phase 2:** WG manager approvals working, 100+ concurrent ops ✓  
**Phase 3:** Cutoff enforced, Finance override working, ready for migration ✓  
**Phase 4:** Org hierarchy & reallocation working, < 200ms queries ✓  
**Phase 5:** All reporting working, production ready ✓  

---

## Files in `/hourglass/plans/`

- `README.md` — Team orientation
- `surrealdb-implementation-plan.md` — 5-phase plan with day-by-day breakdown
- `github-issues-template.md` — Original issue templates (5 per phase)
- **`GITHUB_ISSUES_GRANULAR.md`** ← 25 granular issue templates (copy-paste ready)
- **`GITHUB_ISSUE_CREATION_GUIDE.md`** ← Step-by-step creation instructions
- **`ISSUE_STRUCTURE_OVERVIEW.txt`** ← Visual overview & dependencies
- **`QUICK_START.md`** ← This file

---

## Next: Team Kickoff

**Once issues are created:**

1. Share master issue link with team
2. Assign Phase 1 issues (1.1, 1.2, 1.3, 1.4)
3. Have standups to discuss Phase 1 plan
4. Start work!

**Week 1:** Phase 1 (Foundation)  
**Week 2-3:** Phase 2 (Approvals)  
**Week 3-4:** Phase 3 (Finance)  
**Week 4-5:** Phase 4 (Org)  
**Week 5-6:** Phase 5 (Reporting)  

---

## Support

**Questions?**
- See `GITHUB_ISSUE_CREATION_GUIDE.md` for detailed instructions
- See `GITHUB_ISSUES_GRANULAR.md` for full issue content
- See `ISSUE_STRUCTURE_OVERVIEW.txt` for dependencies
- See `surrealdb-implementation-plan.md` for architectural context

**Ready to create?**
- Pick Option A (CLI), B (Web), or C (Projects) above
- Start with master issue
- Create Phase 1 issues (4 issues)
- Assign to team
- Begin Phase 1 work

---

**Status:** ✅ READY TO CREATE IN GITHUB

All templates complete. No more design decisions needed.
Proceed directly to issue creation in GitHub.
