import { describe, it, expect, afterEach, beforeAll, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { format, subDays } from "date-fns";
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

function entry(overrides: Partial<TimeEntry> & { id: string }): TimeEntry {
  return {
    user_id: USER_ID,
    org_id: "o1",
    activity_id: "a1",
    activity_name: "E2E Activity",
    unit_id: "u1",
    hours: 2.5,
    description: "",
    entry_date: format(new Date(), "yyyy-MM-dd"),
    status: "draft",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function expense(overrides: Partial<Expense> & { id: string }): Expense {
  return {
    org_id: "o1",
    user_id: USER_ID,
    activity_id: "a1",
    activity_name: "E2E Activity",
    category: "meal",
    amount: 12.5,
    description: "",
    entry_date: format(new Date(), "yyyy-MM-dd"),
    status: "submitted",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

// Mutable per-test request state — MSW handlers read it at request time.
// The pending-endpoint counters prove the enabled-gating contract: plain
// employees and HR never fire /time-entries/pending or /expenses/pending.
let currentRole: Role = "employee";
let currentWgs: WorkingGroup[] = [];
let currentMonthEntries: TimeEntry[] = [];
let currentPendingTe: TimeEntry[] = [];
let currentPendingExpenses: Expense[] = [];
let pendingTeCalls = 0;
let pendingExpCalls = 0;

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
  http.get("/api/time-entries", () =>
    HttpResponse.json({ data: currentMonthEntries }),
  ),
  http.get("/api/time-entries/pending", () => {
    pendingTeCalls++;
    return HttpResponse.json({ data: currentPendingTe });
  }),
  http.get("/api/expenses", () => HttpResponse.json({ data: [] })),
  http.get("/api/expenses/pending", () => {
    pendingExpCalls++;
    return HttpResponse.json({ data: currentPendingExpenses });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  currentRole = "employee";
  currentWgs = [];
  currentMonthEntries = [];
  currentPendingTe = [];
  currentPendingExpenses = [];
  pendingTeCalls = 0;
  pendingExpCalls = 0;
});
afterAll(() => server.close());

function renderToday() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  });
  const router = createRouter({
    routeTree,
    context: { client: queryClient },
    history: createMemoryHistory({ initialEntries: ["/"] }),
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("TodayPage composition (10-04)", () => {
  it("approver (manager) sees 'Waiting on you' with seeded pending entries and 'Review now' CTA", async () => {
    currentRole = "manager";
    currentPendingTe = [
      entry({ id: "te1", status: "submitted", hours: 4, activity_name: "Alpha" }),
      entry({ id: "te2", status: "pending_finance", hours: 2, activity_name: "Beta" }),
    ];
    currentPendingExpenses = [
      expense({ id: "ex1", status: "pending_manager", amount: 42.5, activity_name: "Gamma" }),
    ];
    // In-week entry so the page renders sections (not an empty state).
    currentMonthEntries = [entry({ id: "te3", status: "draft" })];

    renderToday();

    // Full-suite runs contend for CPU across parallel test files — the
    // route-context hydration + three queries can take >1s. Give the async
    // assertions a generous window.
    const T = { timeout: 5000 };
    expect(await screen.findByText("Waiting on you", undefined, T)).toBeInTheDocument();
    // The pending queries resolve independently — wait for the merged count.
    expect(await screen.findByText("3 waiting for your review", undefined, T)).toBeInTheDocument();
    // Preview rows show the pending activity names.
    expect(await screen.findByText("Alpha", undefined, T)).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
    expect(screen.getByText("Gamma")).toBeInTheDocument();
    // .font-text numeric hours render for time entries.
    expect(screen.getByText("4")).toBeInTheDocument();
    // 'Review now' appears in the header CTA and the section accent link,
    // both pointing at /approvals.
    const reviewLinks = screen.getAllByRole("link", { name: "Review now" });
    expect(reviewLinks.length).toBeGreaterThanOrEqual(2);
    expect(reviewLinks.every((l) => l.getAttribute("href") === "/approvals")).toBe(true);
    expect(screen.getByText("Your week")).toBeInTheDocument();
  });

  it("plain employee does NOT see 'Waiting on you', sees 'Log time' CTA, and pending endpoints are never called", async () => {
    currentRole = "employee";
    currentMonthEntries = [entry({ id: "te1", status: "draft" })];

    renderToday();

    expect(
      await screen.findByText("Your week", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Waiting on you")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Log time" })).toBeInTheDocument();
    // The enabled gate: no stage holder → pending queries never fire.
    await waitFor(() => {
      expect(pendingTeCalls).toBe(0);
      expect(pendingExpCalls).toBe(0);
    });
  });

  it("'Your week' excludes entries outside the current ISO week and approved entries", async () => {
    currentRole = "employee";
    const now = new Date();
    currentMonthEntries = [
      entry({
        id: "in-week-draft",
        status: "draft",
        entry_date: format(now, "yyyy-MM-dd"),
        activity_name: "In Week Draft",
      }),
      entry({
        id: "in-week-submitted",
        status: "submitted",
        entry_date: format(now, "yyyy-MM-dd"),
        activity_name: "In Week Submitted",
      }),
      entry({
        id: "last-week",
        status: "draft",
        entry_date: format(subDays(now, 7), "yyyy-MM-dd"),
        activity_name: "Last Week Draft",
      }),
      entry({
        id: "approved-today",
        status: "approved",
        entry_date: format(now, "yyyy-MM-dd"),
        activity_name: "Approved Today",
      }),
    ];

    renderToday();

    expect(
      await screen.findByText("In Week Draft", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByText("In Week Submitted")).toBeInTheDocument();
    expect(screen.queryByText("Last Week Draft")).not.toBeInTheDocument();
    expect(screen.queryByText("Approved Today")).not.toBeInTheDocument();
  });

  it("caught-up empty state renders the locked copy when nothing is pending and no week entries exist", async () => {
    currentRole = "manager";
    // Data exists (earlier this month) but falls outside the current week.
    currentMonthEntries = [
      entry({ id: "old", status: "draft", entry_date: format(subDays(new Date(), 7), "yyyy-MM-dd") }),
    ];
    currentPendingTe = [];
    currentPendingExpenses = [];

    renderToday();

    expect(
      await screen.findByText("You're all caught up", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Nothing is waiting on you. When drafts, rejections, or approvals land, they'll show up here.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText("Waiting on you")).not.toBeInTheDocument();
    expect(screen.queryByText("Your week")).not.toBeInTheDocument();
  });

  it("new-user empty state renders the locked copy with a 'Log time' CTA", async () => {
    currentRole = "employee";
    currentMonthEntries = [];

    renderToday();

    expect(
      await screen.findByText("Welcome to Hourglass", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Start by logging time against an activity. Your week and anything waiting on you will appear here.",
      ),
    ).toBeInTheDocument();
    // Header CTA + empty-state CTA both read 'Log time'.
    expect(
      screen.getAllByRole("link", { name: "Log time" }).length,
    ).toBeGreaterThanOrEqual(2);
  });

  it("HR with WG manager_id never sees 'Waiting on you' (ADR-P-008 D-4) and pending endpoints stay uncalled", async () => {
    currentRole = "hr";
    currentWgs = [{ manager_id: USER_ID } as WorkingGroup];
    currentMonthEntries = [entry({ id: "te1", status: "draft" })];

    renderToday();

    expect(
      await screen.findByText("Your week", undefined, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Waiting on you")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Log time" })).toBeInTheDocument();
    await waitFor(() => {
      expect(pendingTeCalls).toBe(0);
      expect(pendingExpCalls).toBe(0);
    });
  });
});
