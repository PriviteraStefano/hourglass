import { describe, it, expect, afterEach, beforeAll, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { routeTree } from "@/routeTree.gen";
import type { Expense, Role, TimeEntry, UserWithMembership, WorkingGroup } from "@/types";

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

function te(overrides: Partial<TimeEntry> & { id: string }): TimeEntry {
  return {
    user_id: "u2",
    org_id: "o1",
    activity_id: "a1",
    activity_name: "Alpha Activity",
    unit_id: "u1",
    hours: 3,
    description: "",
    entry_date: "2026-07-16",
    status: "submitted",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function expense(overrides: Partial<Expense> & { id: string }): Expense {
  return {
    org_id: "o1",
    user_id: "u2",
    activity_id: "a1",
    activity_name: "Gamma Expense",
    category: "meal",
    amount: 42.5,
    description: "",
    entry_date: "2026-07-15",
    status: "submitted",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

// Mutable per-test request state — MSW handlers read them at request time.
// The pending-endpoint counters prove the enabled-gating contract: plain
// employees and HR never fire /time-entries/pending or /expenses/pending.
let currentRole: Role = "employee";
let currentWgs: WorkingGroup[] = [];
let currentPendingTe: TimeEntry[] = [];
let currentPendingExpenses: Expense[] = [];
let pendingTeCalls = 0;
let pendingExpCalls = 0;
let approveTeCalls: string[] = [];
let rejectTeCalls: Array<{ id: string; body: { reason: string } }> = [];

const server = setupServer(
  http.get("/api/auth/me", () =>
    HttpResponse.json({ data: profileFor(currentRole) }),
  ),
  http.get("/api/auth/memberships", () =>
    HttpResponse.json({ data: { memberships: [] } }),
  ),
  http.get("/api/working-groups", () =>
    HttpResponse.json({ data: currentWgs }),
  ),
  http.get("/api/time-entries/pending", () => {
    pendingTeCalls++;
    return HttpResponse.json({ data: currentPendingTe });
  }),
  http.get("/api/expenses/pending", () => {
    pendingExpCalls++;
    return HttpResponse.json({ data: currentPendingExpenses });
  }),
  http.post("/api/time-entries/:id/approve", ({ params }) => {
    approveTeCalls.push(params.id as string);
    return HttpResponse.json({ data: {} });
  }),
  http.post("/api/time-entries/:id/reject", async ({ params, request }) => {
    const body = (await request.json()) as { reason: string };
    rejectTeCalls.push({ id: params.id as string, body });
    return HttpResponse.json({ data: {} });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  currentRole = "employee";
  currentWgs = [];
  currentPendingTe = [];
  currentPendingExpenses = [];
  pendingTeCalls = 0;
  pendingExpCalls = 0;
  approveTeCalls = [];
  rejectTeCalls = [];
});
afterAll(() => server.close());

function renderApprovals() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  });
  const router = createRouter({
    routeTree,
    context: { client: queryClient },
    history: createMemoryHistory({ initialEntries: ["/approvals"] }),
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("ApprovalsPage composition (10-05)", () => {
  it("manager sees the manager-stage queue rows (no tab bar for a single stage)", async () => {
    currentRole = "manager";
    currentPendingTe = [te({ id: "te1", status: "submitted", activity_name: "Alpha" })];
    currentPendingExpenses = [expense({ id: "ex1", status: "submitted", activity_name: "Gamma" })];

    renderApprovals();

    // Header + queue rows render (per-stage manager queue, merged TE+expense).
    expect(
      await screen.findByText("Alpha", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByText("Gamma")).toBeInTheDocument();
    // No tab bar for a single-stage user.
    expect(screen.queryByRole("tab", { name: "Manager" })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Finance" })).not.toBeInTheDocument();
    // Approve/Reject pair renders for manager-stage rows.
    expect(screen.getAllByRole("button", { name: "Approve" }).length).toBe(2);
  });

  it("finance sees the finance-stage queue rows", async () => {
    currentRole = "finance";
    currentPendingTe = [
      te({ id: "te2", status: "pending_finance", activity_name: "Beta" }),
    ];

    renderApprovals();

    expect(
      await screen.findByText("Beta", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    // finance-stage rows carry the Approve/Reject pair.
    expect(screen.getAllByRole("button", { name: "Approve" }).length).toBe(1);
  });

  it("dual-stage user (finance + WG manager) sees both Manager and Finance tabs", async () => {
    currentRole = "finance";
    currentWgs = [
      { id: "wg1", org_id: "o1", manager_id: USER_ID, delegate_ids: [] } as WorkingGroup,
    ];
    currentPendingTe = [te({ id: "te3", status: "pending_finance", activity_name: "Beta" })];

    renderApprovals();

    expect(
      await screen.findByRole("tab", { name: "Manager" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Finance" })).toBeInTheDocument();
  });

  it("approve mutation fires POST /time-entries/{id}/approve and invalidates pending", async () => {
    currentRole = "manager";
    currentPendingTe = [te({ id: "te1", status: "submitted", activity_name: "Alpha" })];

    renderApprovals();

    const approveBtn = await screen.findByRole("button", { name: "Approve" }, { timeout: 5000 });
    // Initial render already fetched pending once.
    await waitFor(() => {
      expect(pendingTeCalls).toBeGreaterThanOrEqual(1);
    });
    fireEvent.click(approveBtn);

    await waitFor(() => {
      expect(approveTeCalls).toEqual(["te1"]);
    });
    // onSuccess invalidates the pending queries → refetch observed on msw
    // (pending GET count grows past the initial load).
    await waitFor(() => {
      expect(pendingTeCalls).toBeGreaterThanOrEqual(2);
    });
  });

  it("reject without a reason cannot submit; with reason fires POST with { reason }", async () => {
    currentRole = "manager";
    currentPendingTe = [te({ id: "te1", status: "submitted", activity_name: "Alpha" })];

    renderApprovals();

    const rejectBtn = await screen.findByRole("button", { name: "Reject" }, { timeout: 5000 });
    fireEvent.click(rejectBtn);

    // Reason panel opens; the confirm button stays disabled until reason ≥ 10.
    const reasonArea = await screen.findByLabelText("Reason for rejection (required)", undefined, {
      timeout: 5000,
    });
    fireEvent.change(reasonArea, { target: { value: "too short" } });
    expect(screen.getByRole("button", { name: "Reject" })).toBeDisabled();
    fireEvent.change(reasonArea, { target: { value: "This entry has wrong hours" } });
    expect(screen.getByRole("button", { name: "Reject" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Reject" }));

    await waitFor(() => {
      expect(rejectTeCalls).toEqual([
        { id: "te1", body: { reason: "This entry has wrong hours" } },
      ]);
    });
  });

  it("403 from pending while tabs render → error state, not 'Queue is clear'", async () => {
    currentRole = "manager";
    server.use(
      http.get("/api/time-entries/pending", () =>
        HttpResponse.json({ error: "only managers and finance can view pending entries" }, { status: 403 }),
      ),
    );

    renderApprovals();

    expect(
      await screen.findByText("We couldn't load Approvals.", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Queue is clear")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("employee/HR direct access → muted notice and zero pending requests", async () => {
    currentRole = "employee";

    renderApprovals();

    expect(
      await screen.findByText(
        "Approvals are for manager and finance stages.",
        undefined,
        { timeout: 5000 },
      ),
    ).toBeInTheDocument();
    // The enabled gate: no stage holder → pending queries never fire.
    await waitFor(() => {
      expect(pendingTeCalls).toBe(0);
      expect(pendingExpCalls).toBe(0);
    });
  });

  it("empty queue → 'Queue is clear' with the stage name interpolated", async () => {
    currentRole = "manager";
    currentPendingTe = [];
    currentPendingExpenses = [];

    renderApprovals();

    expect(
      await screen.findByText("Queue is clear", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "There are no manager approvals waiting. Submitted entries will appear here for review.",
      ),
    ).toBeInTheDocument();
  });
});
