import { describe, it, expect, beforeAll, afterEach, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { format } from "date-fns";
import {
  render,
  screen,
  fireEvent,
  waitFor,
} from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { routeTree } from "@/routeTree.gen";

function makeEntry(overrides: Record<string, unknown> = {}) {
  return {
    id: "te-1",
    user_id: "u1",
    org_id: "o1",
    activity_id: "a1",
    activity_name: "Acme Website",
    activity_kind: "engagement",
    unit_id: "u1",
    hours: 7.5,
    description: "Client workshop",
    entry_date: "2026-05-18",
    status: "draft",
    created_at: "2026-05-18T10:00:00Z",
    updated_at: "2026-05-18T10:00:00Z",
    ...overrides,
  };
}

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

const handlers = [
  http.get("/api/auth/me", () =>
    HttpResponse.json({
      data: {
        user: {
          id: "u1",
          email: "t@example.com",
          username: "t",
          name: "Test User",
          is_active: true,
          created_at: "2026-01-01T00:00:00Z",
        },
        membership: {
          id: "m1",
          user_id: "u1",
          organization_id: "o1",
          role: "employee",
          is_active: true,
        },
        organization: {
          id: "o1",
          name: "Test Org",
          slug: "test-org",
          created_at: "2026-01-01T00:00:00Z",
        },
      },
    })
  ),
  http.get("/api/auth/memberships", () =>
    HttpResponse.json({
      data: {
        memberships: [
          {
            membership: {
              id: "m1",
              user_id: "u1",
              organization_id: "o1",
              role: "employee",
              is_active: true,
            },
            organization: {
              id: "o1",
              name: "Test Org",
              slug: "test-org",
              created_at: "2026-01-01T00:00:00Z",
            },
          },
        ],
      },
    })
  ),
  http.get("/api/time-entries", ({ request }) => {
    const url = new URL(request.url);
    const month = url.searchParams.get("month");
    const year = url.searchParams.get("year");
    if (month && year) {
      return HttpResponse.json({
        data: [
          makeEntry(),
          makeEntry({
            id: "te-2",
            // Real API returns RFC3339, not yyyy-MM-dd — pin the parse path
            entry_date: "2026-05-19T00:00:00Z",
            status: "approved",
            hours: 3,
            description: "Internal sync",
          }),
          makeEntry({
            id: "te-3",
            entry_date: "2026-05-20",
            status: "rejected",
            hours: 1.5,
            description: "Expensed but rejected",
          }),
        ],
      });
    }
    return HttpResponse.json({ data: [] });
  }),
  http.get("/api/activities", () =>
    HttpResponse.json({
      data: [{ id: "a1", name: "Acme Website", kind: "engagement" }],
    })
  ),
];

const server = setupServer(...handlers);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("TimeEntriesList (via route)", () => {
  it("renders the populated list table with project names and status badges", async () => {
    renderAt(["/time-entries?date=2026-05-18&month=2026-05-01"]);
    await waitFor(() => {
      expect(screen.getByText("Client workshop")).toBeInTheDocument();
    });
    // Activity name appears in the table (and in the calendar tab's select, so
    // getAllByText is required)
    expect(screen.getAllByText("Acme Website").length).toBeGreaterThan(0);
    expect(screen.getByText("7.5h")).toBeInTheDocument();
    // All three rows render
    expect(screen.getByText("Internal sync")).toBeInTheDocument();
    expect(screen.getByText("Expensed but rejected")).toBeInTheDocument();
    // Distinct badges for the distinct statuses
    expect(screen.getByText("Draft")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("Rejected")).toBeInTheDocument();
  });

  it("narrows rows via the status filter and persists it in the URL", async () => {
    const { router } = renderAt([
      "/time-entries?date=2026-05-18&month=2026-05-01",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Client workshop")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /^Status/ }));
    fireEvent.click(screen.getByRole("menuitemcheckbox", { name: "Draft" }));

    await waitFor(() => {
      expect(screen.queryByText("Internal sync")).not.toBeInTheDocument();
    });
    expect(screen.getByText("Client workshop")).toBeInTheDocument();

    // URL now carries the filter (URL-shareable per ADR-FE-017); arrays are
    // JSON-serialized by TanStack Router's default parseSearch
    const search = new URLSearchParams(router.state.location.searchStr);
    expect(search.get("listStatuses")).toBe('["draft"]');
  });

  it("applies the date-range filter", async () => {
    // Render already-filtered URL: listFrom/listTo are URL-shareable search
    // params (ADR-FE-017); the interactive picker is a thin controlled shell
    // proven by the status-filter test above.
    renderAt([
      "/time-entries?date=2026-05-18&month=2026-05-01&listFrom=2026-05-18&listTo=2026-05-18",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Client workshop")).toBeInTheDocument();
    });
    expect(screen.queryByText("Internal sync")).not.toBeInTheDocument();
    expect(screen.queryByText("Expensed but rejected")).not.toBeInTheDocument();
  });

  it("row click updates the date search param (opens detail surface)", async () => {
    const { router } = renderAt([
      "/time-entries?date=2026-05-18&month=2026-05-01",
    ]);
    await waitFor(() => {
      expect(screen.getByText("Internal sync")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Internal sync"));

    await waitFor(() => {
      const search = new URLSearchParams(router.state.location.searchStr);
      const date = new Date(search.get("date") ?? "");
      expect(format(date, "yyyy-MM-dd")).toBe("2026-05-19");
    });
  });

  it("shows the filtered empty state when nothing matches", async () => {
    renderAt([
      "/time-entries?date=2026-05-18&month=2026-05-01&listStatuses=approved&listFrom=2026-05-20&listTo=2026-05-20",
    ]);
    await waitFor(() => {
      expect(
        screen.getByText("No entries match the current filters.")
      ).toBeInTheDocument();
    });
  });
});
