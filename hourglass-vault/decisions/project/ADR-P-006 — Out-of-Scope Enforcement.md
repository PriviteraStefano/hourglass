# ADR-P-006 — Out-of-Scope Enforcement

---
tags: ["adr", "idea-layer", "scope", "governance"]

---

# ADR-P-006 — Out-of-Scope Enforcement

**Status:** Proposed
**Date:** 2026-07-28
**Operationalizes:** [[VISION]] §8

## Context

The author of this vault self-reports a tendency to steer. [[VISION]] §8 lists explicit out-of-scope anchors (chat, task execution, payroll/invoicing, HR management, real-time collaboration) — but a list in a vision doc is advisory. Without an enforcement mechanism, a tempting idea gets re-proposed in a new costume three months later and relitigated from scratch.

## Decision

Out-of-scope anchors are **binding rejections**, enforced procedurally:

1. **Rejected ideas are logged, not deleted.** When an idea fails the steering test or hits a §8 anchor, it is recorded in the Rejection Log below (one line: idea, date, anchor/test failed). Future proposals are checked against the log *before* discussion.
2. **Reopening requires a vision revision.** A logged rejection can only be reversed by editing [[VISION]] §8 with a recorded reason (VISION §10 revision log) — never by a planning conversation.
3. **Planning docs must cite rejections.** When a SPEC or phase touches a boundary (e.g. tickets vs. task boards), it must cite the relevant anchor in its scope section.

## Rejection Log

| Date | Idea | Anchor / test failed |
|------|------|----------------------|
| 2026-07-28 | Kanban/sprint board for tickets | §8 task execution; ADR-P-003 hard boundary |
| 2026-07-28 | Comment threads on tickets | §8 chat/communication |
| 2026-07-28 | Invoice/payroll generation from approved entries | §8 payroll-invoicing (we produce trusted *data*, not documents) |
| 2026-07-28 | Charts/KPIs on the Today view | ADR-P-004 design rules (composition, not dashboards) |
| 2026-07-29 | Work permits & holidays (leave/absence tracking, permit management) | §8 HR management; §7 belonging — absence/compliance data answers "is this person available/legal", not one of the three questions. **Carved 2026-07-29:** availability windows + employment-validity dates *as staffing structure data* are in scope per [[ADR-P-008 — Availability & Employment Validity]]; leave machinery, balances, and document storage remain rejected here. **HR-in-the-loop carve (2026-07-29, same ADR):** an `hr` org role curates/consumes that data but is *never* an approval stage — HR as approver stays rejected here. |
| 2026-07-29 | Wiki / authored knowledge base | Knowledge in Hourglass is *derived* from captured data (V3 profiles, V4 maps), not authored content; §7 belonging — a wiki answers no question the captured data feeds |
| 2026-07-29 | Illness / sick-leave tracking | **Superseded before logging:** absence kinds incl. `medical` admitted same-day per [[ADR-P-008 — Availability & Employment Validity]] D-1 (wages need the classification). What remains rejected is *medical content* — certificate uploads, diagnoses, sub-types; only the `certificate_ref` number is stored. Kept here so the full consideration trail is visible. |

## Consequences

* Rejected ideas die once, documented — they stop consuming planning energy.
* §8 anchors become enforceable rather than aspirational.
* A legitimate scope change is possible but expensive by design (vision revision), which is the correct friction for a self-steering author.

## Related

* [[VISION]] §8 (anchors), §7 (steering test), §10 (revision rules)
* All ADR-P documents cite this when narrowing scope
