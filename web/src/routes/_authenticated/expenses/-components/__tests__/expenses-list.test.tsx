import { describe, it, expect, beforeAll, afterEach, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { format } from "date-fns";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { routeTree } from "@/routeTree.gen";

function makeExpense(overrides: Record<string, unknown> = {}) {
  return {
    id: "ex-1",
    org_id: "o1",
    user_id: "u1",
    project_id: "p1",
    category: "meal",
    amount: 32.5,
    description: "Team lunch",
    entry_date: "2026-05-18",
    status: "draft",
    created_at: "2026-05-18T10:00:00Z",
    updated_at: "2026-05-18T10:00:00Z",
    ...overrides,
  };
}

const handlers = [
  http.get("/api/auth/me", () =>
    HttpResponse.json({
      data: {
        user: { id: "u1", email: "t@example.com", username: "t", name: "Test User", is_active: true, created_at: "2026-01-01T00:00:00Z" },
        membership: { id: "m1", user_id: "u1", organization_id: "o1", role: "employee", is_active: true },
        organization: { id: "o1", name: "Test Org", slug: "test-org", created_at: "2026-01-01T00:00:00Z" },
      },
    })
  ),
  http.get("/api/auth/memberships", () =>
    HttpResponse.json({ data: { memberships: [] } })
  ),
  http.get("/api/expenses", ({ request }) => {
    const url = new URL(request.url);
    if (url.searchParams.get("month")) {
      return HttpResponse.json({
        data: [
          makeExpense(),
          makeExpense({
            id: "ex-2",
            category: "mileage",
            amount: 18.9,
            km_distance: 45,
            description: "Client site visit",
            entry_date: "2026-05-19T00:00:00Z",
            status: "approved",
            receipt_url: "/receipts/ex-2.pdf",
          }),
          makeExpense({
            id: "ex-3",
            category: "accommodation",
            amount: 120,
            description: "Hotel night",
            entry_date: "2026-05-20",
            status: "rejected",
          }),
        ],
      });
    }
    return HttpResponse.json({ data: [] });
  }),
  http.get("/api/projects", () =>
    HttpResponse.json({
      data: [{ id: "p1", name: "Acme Website", type: "billable", contract_id: "c1" }],
    })
  ),
  http.get("/api/contracts", () =>
    HttpResponse.json({
      data: [{ id: "c1", name: "Acme Contract", currency: "EUR" }],
    })
  ),
];

const server = setupServer(...handlers);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const WAIT_TIMEOUT = 5000;

function renderAt(initialEntries: string[]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  });
  const router = createRouter({
    routeTree,
    context: { client: queryClient },
    history: createMemoryHistory({ initialEntries }),
  });
  return {
    queryClient,
    router,
    ...render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    ),
  };
}

describe("ExpensesList (via route)", () => {
  it("renders the populated list with category, amount+currency, receipt and mileage", async () => {
    renderAt(["/expenses?date=2026-05-18&month=2026-05-01"]);
    const table = () => screen.getByRole("table");
    await waitFor(() => {
      expect(screen.getByText("Team lunch")).toBeInTheDocument();
    }, { timeout: WAIT_TIMEOUT });
    // "Meal" also appears in the category filter select — scope to the table
    expect(within(table()).getByText("Meal")).toBeInTheDocument();
    // Amount rendered with contract-derived currency (EUR)
    expect(within(table()).getByText(/€32\.50/)).toBeInTheDocument();
    expect(screen.getByText("Client site visit")).toBeInTheDocument();
    // Receipt indicator for the entry with a receipt_url
    expect(document.querySelector("svg.lucide-paperclip")).not.toBeNull();
    // Mileage distance surfaced alongside the description (Intl 2-decimals)
    expect(screen.getByText(/45[\d.,]*\s*km/)).toBeInTheDocument();
    // Distinct status badges
    expect(within(table()).getByText("Approved")).toBeInTheDocument();
    expect(within(table()).getByText("Rejected")).toBeInTheDocument();
  });

  it("narrows rows via status and category filters with URL persistence", async () => {
    const { router } = renderAt([
      "/expenses?date=2026-05-18&month=2026-05-01",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Team lunch")).toBeInTheDocument();
    }, { timeout: WAIT_TIMEOUT });

    // Category filter
    fireEvent.change(screen.getByLabelText("Category filter"), {
      target: { value: "mileage" },
    });
    await waitFor(() => {
      expect(screen.queryByText("Team lunch")).not.toBeInTheDocument();
    });
    expect(screen.getByText("Client site visit")).toBeInTheDocument();
    let search = new URLSearchParams(router.state.location.searchStr);
    expect(search.get("listCategory")).toBe("mileage");

    // Status filter stacks on top
    fireEvent.click(screen.getByRole("button", { name: /^Status/ }));
    fireEvent.click(screen.getByRole("menuitemcheckbox", { name: "Approved" }));
    await waitFor(() => {
      search = new URLSearchParams(router.state.location.searchStr);
      expect(search.get("listStatuses")).toBe('["approved"]');
    }, { timeout: WAIT_TIMEOUT });
    expect(screen.getByText("Client site visit")).toBeInTheDocument();
  });

  it("applies the date-range filter", async () => {
    renderAt([
      "/expenses?date=2026-05-18&month=2026-05-01&listFrom=2026-05-18&listTo=2026-05-18",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Team lunch")).toBeInTheDocument();
    }, { timeout: WAIT_TIMEOUT });
    expect(screen.queryByText("Client site visit")).not.toBeInTheDocument();
    expect(screen.queryByText("Hotel night")).not.toBeInTheDocument();
  });

  it("row click updates the date search param (opens detail surface)", async () => {
    const { router } = renderAt([
      "/expenses?date=2026-05-18&month=2026-05-01",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Client site visit")).toBeInTheDocument();
    }, { timeout: WAIT_TIMEOUT });

    fireEvent.click(screen.getByText("Client site visit"));

    await waitFor(() => {
      const search = new URLSearchParams(router.state.location.searchStr);
      const date = new Date(search.get("date") ?? "");
      expect(format(date, "yyyy-MM-dd")).toBe("2026-05-19");
    });
  });

  it("shows the filtered empty state when nothing matches", async () => {
    renderAt([
      "/expenses?date=2026-05-18&month=2026-05-01&listCategory=parking",
    ]);
    await waitFor(() => {
      expect(
        screen.getByText("No expenses match the current filters.")
      ).toBeInTheDocument();
    }, { timeout: WAIT_TIMEOUT });
  });
});
