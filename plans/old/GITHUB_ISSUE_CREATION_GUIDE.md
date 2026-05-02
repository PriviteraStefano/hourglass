# GitHub Issue Creation Guide: SurrealDB Migration (Granular)

This guide shows you how to create the master issue and 24 sub-issues in GitHub.

## Overview

**Master Issue:** 1 epic linking all 24 sub-issues
- Status: Ready to create
- Location: `/hourglass/plans/GITHUB_ISSUES_GRANULAR.md`

**Sub-Issues:** 24 granular, independently-grabbable work items
- 4 in Phase 1 (Foundation)
- 5 in Phase 2 (Approval Workflow)
- 4 in Phase 3 (Financial Controls)
- 4 in Phase 4 (Org Hierarchy)
- 5 in Phase 5 (Reporting)

## Option A: Create Issues Manually (GitHub Web UI)

### Step 1: Create Master Issue

1. Go to https://github.com/[owner]/hourglass/issues/new
2. **Title:** `[EPIC] SurrealDB Migration: Full Implementation (6 Weeks)`
3. **Description:** Copy from `GITHUB_ISSUES_GRANULAR.md` → Master Issue Template
4. **Labels:** `epic`, `surrealdb`, `priority-high`
5. **Click "Create"**

Note the issue number (e.g., #42)

### Step 2: Create Phase 1 Sub-Issues

For each sub-issue (1.1, 1.2, 1.3, 1.4):

1. Go to https://github.com/[owner]/hourglass/issues/new
2. Copy title & description from `GITHUB_ISSUES_GRANULAR.md`
3. **Labels:** `phase-1`, `surrealdb`, and appropriate labels (e.g., `backend`, `devops`, `qa`)
4. **In description, add at end:**
   ```
   Part of #[MASTER_ISSUE_NUMBER]
   ```
5. **Click "Create"**

Example for Sub-Issue 1.1:
- **Title:** `[Phase 1.1] Deploy SurrealDB Cluster & Load Schema`
- **Labels:** `phase-1`, `surrealdb`, `devops`, `infrastructure`
- **Description:** (from template) + `\n\nPart of #42`

### Step 3: Repeat for Phases 2-5

Follow same pattern for all sub-issues in Phases 2-5.

---

## Option B: Create Issues with GitHub CLI (gh)

### Prerequisites

```bash
# Install GitHub CLI
brew install gh

# Authenticate
gh auth login
```

### Create Master Issue

```bash
gh issue create \
  --title "[EPIC] SurrealDB Migration: Full Implementation (6 Weeks)" \
  --body "$(cat /Users/stefanoprivitera/Projects/hourglass/plans/GITHUB_ISSUES_GRANULAR.md | sed -n '/## Master Issue Template/,/## Phase 1/p')" \
  --label "epic,surrealdb,priority-high" \
  --repo [owner]/hourglass
```

This returns issue number (e.g., `#42`).

### Create Phase 1 Sub-Issues

```bash
# 1.1: Infrastructure
gh issue create \
  --title "[Phase 1.1] Deploy SurrealDB Cluster & Load Schema" \
  --body "..." \
  --label "phase-1,surrealdb,devops" \
  --repo [owner]/hourglass

# 1.2: Auth
gh issue create \
  --title "[Phase 1.2] Implement JWT Authentication Endpoints" \
  --body "..." \
  --label "phase-1,surrealdb,backend" \
  --repo [owner]/hourglass

# 1.3: CRUD
gh issue create \
  --title "[Phase 1.3] Implement Time Entry CRUD Operations" \
  --body "..." \
  --label "phase-1,surrealdb,backend" \
  --repo [owner]/hourglass

# 1.4: Testing
gh issue create \
  --title "[Phase 1.4] Phase 1 Testing, Integration & Acceptance" \
  --body "..." \
  --label "phase-1,surrealdb,qa,testing" \
  --repo [owner]/hourglass
```

---

## Option C: Bulk Create with Script

Create a bash script to automate:

```bash
#!/bin/bash
# create-issues.sh

REPO="[owner]/hourglass"
MASTER_ISSUE=42  # Update after creating master

declare -a ISSUES=(
  "[Phase 1.1]|Deploy SurrealDB Cluster & Load Schema|phase-1,devops"
  "[Phase 1.2]|Implement JWT Authentication Endpoints|phase-1,backend"
  "[Phase 1.3]|Implement Time Entry CRUD Operations|phase-1,backend"
  "[Phase 1.4]|Phase 1 Testing, Integration & Acceptance|phase-1,qa"
  # ... add all 24 issues
)

for issue in "${ISSUES[@]}"; do
  IFS='|' read -r prefix title labels <<< "$issue"
  
  gh issue create \
    --title "$prefix $title" \
    --label "$labels,surrealdb" \
    --body "See GITHUB_ISSUES_GRANULAR.md for full details.

Part of #$MASTER_ISSUE" \
    --repo "$REPO"
    
  sleep 1  # Rate limit
done
```

---

## Option D: Use GitHub Projects (Recommended)

For better tracking & dependencies:

1. Create Project: https://github.com/[owner]/hourglass/projects/new
   - Name: `SurrealDB Migration`
   - Template: `Table`

2. Create master issue (see Option A, Step 1)

3. Add all sub-issues to project

4. Use project "Status" field:
   - Not Started
   - In Progress
   - In Review
   - Done

5. Use project "Priority" field:
   - High (Phase 1)
   - Medium (Phase 2-3)
   - Low (Phase 4-5)

---

## Quick Reference: Issue Labels

Suggested labels to use:

**By Phase:**
- `phase-1` through `phase-5`

**By Type:**
- `backend` - Backend work
- `frontend` - Frontend work
- `devops` - Infrastructure/deployment
- `qa` - Testing
- `database` - Schema/database

**By Priority:**
- `priority-high` - Blocking other work
- `priority-medium` - Important
- `priority-low` - Nice to have

**By Status:**
- `blocked` - Cannot proceed
- `in-progress` - Actively being worked
- `review` - Under review

---

## Issue Linking & Dependencies

After creating all issues, link them for dependency tracking:

1. In each sub-issue's description, list dependencies:
   ```
   **Dependencies:**
   - Requires #[issue-number]
   ```

2. In GitHub (UI or CLI):
   ```bash
   gh issue link [SUB_ISSUE_NUMBER] --add [DEPENDENCY_NUMBER]
   ```

3. Use GitHub's "Linked issues" feature to show blocking relationships

---

## Acceptance Checklist

Before announcing to team:

- [ ] Master issue created (1/1)
- [ ] Phase 1 sub-issues created (4/4)
- [ ] Phase 2 sub-issues created (5/5)
- [ ] Phase 3 sub-issues created (4/4)
- [ ] Phase 4 sub-issues created (4/4)
- [ ] Phase 5 sub-issues created (5/5)
- [ ] All issues linked to master
- [ ] Labels applied consistently
- [ ] Project board created
- [ ] All issues assigned to project

**Total:** 1 master + 24 sub-issues = 25 issues

---

## Team Notifications

Once issues are created, notify team:

**Message:**
```
📌 SurrealDB Migration Issues Now Open

Master Issue: #[MASTER_NUMBER]
Total Issues: 25 (1 epic + 24 sub-issues)
Timeline: 6 weeks

Phase 1 Issues (1 week): #[1.1], #[1.2], #[1.3], #[1.4]
Phase 2 Issues (1.5 weeks): #[2.1], ... 
...

Project Board: [link]
Documentation: /hourglass/plans/

Next: Assign issues to team members
```

---

## Tips

1. **Start with Phase 1:** Get team comfortable before scaling to full 5 phases
2. **Don't over-assign:** Let developers pick issues as they finish previous work
3. **Update descriptions:** As design evolves, keep issues in sync
4. **Close issues as done:** Mark issues as closed when acceptance criteria met
5. **Use project board:** Drag issues from "Not Started" → "In Progress" → "Done"

---

## Support

For questions about issue creation:
- GitHub CLI: `gh issue create --help`
- GitHub Docs: https://docs.github.com/en/issues/organizing-your-work-with-project-boards
