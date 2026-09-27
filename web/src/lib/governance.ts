/**
 * Governance models (CONTEXT.md) — how a contract/activity is approved.
 * Shared so the create-activity and ticket-triage forms offer identical copy
 * for the same enum. The contract dialog keeps its own noun-specific wording.
 */
export const GOVERNANCE_OPTIONS = [
  {
    value: "creator_controlled",
    label: "Creator Controlled",
    description: "Only your organization can approve changes to this activity",
  },
  {
    value: "unanimous",
    label: "Unanimous",
    description: "All organizations using this activity must approve changes",
  },
  {
    value: "majority",
    label: "Majority",
    description:
      "More than half of organizations using this activity must approve changes",
  },
];
