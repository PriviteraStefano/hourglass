import { describe, expect, it } from "vitest";
import type { Role, UserWithMembership, WorkingGroup } from "@/types";
import {
  deriveApprovalStages,
  isAdminVisible,
  isEconomicsVisible,
  isReviewVisible,
} from "../role-visibility";

const USER_ID = "user-1";

function profileFor(role: Role): UserWithMembership {
  return {
    user: {
      id: USER_ID,
      email: "u@example.com",
      username: "u",
      name: "U",
      is_active: true,
      created_at: "2026-01-01T00:00:00Z",
    },
    membership: {
      id: "m1",
      user_id: USER_ID,
      organization_id: "o1",
      role,
      is_active: true,
    },
    organization: {
      id: "o1",
      name: "Org",
      slug: "org",
      created_at: "2026-01-01T00:00:00Z",
    },
  };
}

function wg(overrides?: Partial<WorkingGroup>): WorkingGroup {
  return {
    id: "wg-1",
    org_id: "o1",
    subproject_id: "a1",
    name: "Team A",
    description: "",
    unit_ids: [],
    enforce_unit_tuple: false,
    manager_id: "other-user",
    delegate_ids: [],
    is_active: true,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("deriveApprovalStages", () => {
  it("returns [] for employee without WG membership", () => {
    expect(deriveApprovalStages(profileFor("employee"), [wg()])).toEqual([]);
  });

  it("treats an undefined WG list as org-role only", () => {
    expect(deriveApprovalStages(profileFor("employee"), undefined)).toEqual([]);
    expect(deriveApprovalStages(profileFor("manager"), undefined)).toEqual([
      "manager",
    ]);
    expect(deriveApprovalStages(profileFor("finance"), undefined)).toEqual([
      "finance",
    ]);
  });

  it("adds manager stage when an employee is a WG manager_id", () => {
    expect(
      deriveApprovalStages(profileFor("employee"), [wg({ manager_id: USER_ID })]),
    ).toEqual(["manager"]);
  });

  it("adds manager stage on delegate_ids array membership", () => {
    expect(
      deriveApprovalStages(profileFor("employee"), [wg({ delegate_ids: [USER_ID] })]),
    ).toEqual(["manager"]);
  });

  it("keeps org-role manager stage even without WG rows", () => {
    expect(deriveApprovalStages(profileFor("manager"), [])).toEqual([
      "manager",
    ]);
  });

  it("gives finance role the finance stage", () => {
    expect(deriveApprovalStages(profileFor("finance"), [])).toEqual([
      "finance",
    ]);
  });

  it("gives hr NO stage even when listed as WG manager and delegate (ADR-P-008 D-4)", () => {
    const stages = deriveApprovalStages(profileFor("hr"), [
      wg({ manager_id: USER_ID, delegate_ids: [USER_ID] }),
    ]);
    expect(stages).toEqual([]);
  });

  it("unions finance with a WG-derived manager stage", () => {
    const stages = deriveApprovalStages(profileFor("finance"), [
      wg({ manager_id: USER_ID }),
    ]);
    expect(stages).toEqual(expect.arrayContaining(["manager", "finance"]));
  });
});

describe("isReviewVisible", () => {
  it("is false when the user holds no approval stage", () => {
    expect(isReviewVisible("employee", [])).toBe(false);
    expect(isReviewVisible("customer", [])).toBe(false);
  });

  it("is true for employee holding a WG-derived manager stage", () => {
    expect(isReviewVisible("employee", ["manager"])).toBe(true);
  });

  it("is true for manager and finance with their stages", () => {
    expect(isReviewVisible("manager", ["manager"])).toBe(true);
    expect(isReviewVisible("finance", ["finance"])).toBe(true);
  });

  it("is never true for hr, even with stages", () => {
    expect(isReviewVisible("hr", ["manager"])).toBe(false);
    expect(isReviewVisible("hr", ["finance"])).toBe(false);
  });
});

describe("isEconomicsVisible", () => {
  it("is hidden from employee and customer", () => {
    expect(isEconomicsVisible("employee")).toBe(false);
    expect(isEconomicsVisible("customer")).toBe(false);
  });

  it("is visible to manager, finance, and hr (hr read-only per ADR)", () => {
    expect(isEconomicsVisible("manager")).toBe(true);
    expect(isEconomicsVisible("finance")).toBe(true);
    expect(isEconomicsVisible("hr")).toBe(true);
  });
});

describe("isAdminVisible", () => {
  it("is false for every v0.1 role", () => {
    const roles: Role[] = [
      "employee",
      "manager",
      "finance",
      "hr",
      "customer",
    ];
    for (const role of roles) {
      expect(isAdminVisible(role)).toBe(false);
    }
  });
});
