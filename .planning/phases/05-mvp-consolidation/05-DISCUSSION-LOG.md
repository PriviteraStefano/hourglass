# Phase 5: Projects - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-11
**Phase:** 05-Projects
**Areas discussed:** Edit project UI, Delete protection scope, Subproject display, Contract requirement, Editable fields

---

## Edit Project UI

| Option | Description | Selected |
|--------|-------------|----------|
| Dialog-based | Reuse CreateProjectDialog as edit dialog — quick, consistent | ✓ |
| Inline page edit | View/edit toggle on detail page — more polished but more work | |

**User's choice:** Dialog-based (Recommended)
**Notes:** MVP deadline Monday — dialog is faster to build, consistent with existing CreateProjectDialog pattern.

---

## Delete Protection Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Block on ANY time entries | Simple, consistent with contract delete pattern | |
| Block only on active entries | Allow delete if all entries are rejected/approved | ✓ |
| Block on time entries OR subprojects | Both checks | ✓ (implied — including subproject time entries) |
| Block shared project deletion entirely | No delete if adopted by other orgs | |
| Cascade-clean adoptions | Delete adoption records too | ✓ |

**User's choice:** Block only on active time entries (including subproject time entries). Cascade-clean adoptions when shared project is deleted.
**Notes:** Delete protection must check time entry statuses for both the project itself and all its subprojects. Active = draft/submitted/pending statuses. If only approved/rejected, deletion allowed.

---

## Subproject Display on Detail

| Option | Description | Selected |
|--------|-------------|----------|
| Inline list | Simple list on detail page | |
| Expandable section | Collapsible accordion | ✓ |
| Separate tab | Third tab on detail page | |
| Just count with link | Show count, defer full implementation | |

**User's choice:** Expandable section
**Nesting:** One level only (no grandchildren)

---

## Contract Requirement

| Option | Description | Selected |
|--------|-------------|----------|
| Required (Recommended) | All projects must have a contract | ✓ |
| Optional — allow no contract | Nullable contract_id, internal project support | |

**User's choice:** Required. Internal projects use an "Internal Operations" contract. Seed data should include one.
**Notes:** User considered nullable contract_id but preferred a consistent data model where all projects reference a contract. Internal contracts can be seeded.

---

## Editable Fields

| Option | Description | Selected |
|--------|-------------|----------|
| All fields | Name, type, contract, governance, shared | ✓ |
| Name + governance only | Lock type, contract, shared | |
| Name + governance + type | Lock contract and shared | |

**User's choice:** All fields editable

---

## the agent's Discretion

- Exact update/delete handler response format (API envelope consistent with existing patterns)
- Subproject display component design within the expandable section
- Delete confirmation dialog wording and UX flow
- Test file locations and specific test cases within existing patterns
- Frontend mutation `onSuccess` behavior (navigation vs inline feedback)

## Deferred Ideas

### Subproject CRUD Management
Creating, editing, deleting subprojects from the frontend. Deferred — display only for MVP. Backend CRUD already exists.

### Multi-level Subproject Nesting
Recursive subproject hierarchy. Not needed for MVP — one level is sufficient.
