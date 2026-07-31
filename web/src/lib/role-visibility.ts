import type { Role, UserWithMembership, WorkingGroup } from "@/types";

/**
 * The approval stage(s) a user holds, driving sidebar Review-group visibility
 * and the Approvals page stage tabs (ADR-P-011 D-5).
 *
 * UX scoping only — every role-restricted surface stays backend-gated. Hiding
 * a group here must never be mistaken for authorization.
 */
export type ApprovalStage = "manager" | "finance";

/**
 * Derive the user's approval stages from the org membership role plus
 * working-group manager/delegate membership (from GET /working-groups
 * payloads + auth/me user id — no new endpoint, RESEARCH §2.1).
 *
 * - org-role finance → `["finance"]`
 * - org-role manager → `["manager"]`
 * - any non-hr role (incl. employee, customer) gains `"manager"` when the
 *   user id matches a WG `manager_id` or appears in `delegate_ids`
 * - HR never receives a stage (ADR-P-008 D-4): even when hr is listed as a WG
 *   manager/delegate, the WG-derived stage is stripped
 * - `undefined` WG list (query not loaded yet) → org-role stages only
 */
export function deriveApprovalStages(
  profile: UserWithMembership,
  workingGroups: WorkingGroup[] | undefined,
): ApprovalStage[] {
  const role = profile.membership.role;

  // HR never holds an approval stage — strip WG-derived manager stage even
  // when hr rows appear as WG manager/delegate (ADR-P-008 D-4).
  if (role === "hr") return [];

  const stages = new Set<ApprovalStage>();
  if (role === "finance") stages.add("finance");
  if (role === "manager") stages.add("manager");

  const uid = profile.user.id;
  const isWgManager =
    workingGroups?.some((wg) => wg.manager_id === uid) ?? false;
  const isWgDelegate =
    workingGroups?.some((wg) => (wg.delegate_ids ?? []).includes(uid)) ??
    false;
  if (isWgManager || isWgDelegate) stages.add("manager");

  return Array.from(stages);
}

/** Review group renders when the user holds an approval stage, never for hr. */
export function isReviewVisible(role: Role, stages: ApprovalStage[]): boolean {
  return stages.length > 0 && role !== "hr";
}

/** Economics group is hidden from employee and customer (ADR matrix). */
export function isEconomicsVisible(role: Role): boolean {
  return role !== "employee" && role !== "customer";
}

/**
 * Admin group is hidden from every v0.1 role — no org-admin role exists yet.
 * Revisit when an org-admin role lands.
 */
export function isAdminVisible(role: Role): boolean {
  void role;
  return false;
}
