import { describe, it, expect, beforeAll, afterEach, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { routeTree } from "@/routeTree.gen";
import { AuthRouteError } from "../route-error";

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
  http.get("/api/activities", () =>
    HttpResponse.json({ data: [{ id: "a1", name: "Acme Website", kind: "engagement" }] })
  ),
];

// The time-entries loader is forced to fail initially, then succeeds once
// "Try again" re-runs it (simulates a transient API outage).
let failTimeEntries = true;
const timeEntriesHandler = http.get("/api/time-entries", () => {
  if (failTimeEntries) {
    return HttpResponse.json({ error: "boom" }, { status: 500 });
  }
  return HttpResponse.json({
    data: [
      {
        id: "te-1",
        user_id: "u1",
        org_id: "o1",
        activity_id: "a1",
        activity_name: "Acme Website",
        activity_kind: "engagement",
        unit_id: "u1",
        hours: 7.5,
        description: "Recovered entry",
        entry_date: "2026-05-18",
        status: "draft",
        created_at: "",
        updated_at: "",
      },
    ],
  });
});

const server = setupServer(...handlers, timeEntriesHandler);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  failTimeEntries = true;
});
afterAll(() => server.close());

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

describe("RouteError boundary (P0-4)", () => {
  it("renders the error panel instead of a blank screen when a loader fails", async () => {
    renderAt(["/time-entries?date=2026-05-18&month=2026-05-01"]);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
    // Error message from the failed loader surfaced, recovery affordances present
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /try again/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /go to today/i })
    ).toBeInTheDocument();
  });

  it("re-runs the loader on Try again and recovers to content", async () => {
    const { router } = renderAt([
      "/time-entries?date=2026-05-18&month=2026-05-01",
    ]);

    await waitFor(() => {
      expect(screen.getByText("boom")).toBeInTheDocument();
    });

    // Simulate the API recovering, then click Try again
    failTimeEntries = false;
    fireEvent.click(screen.getByRole("button", { name: /try again/i }));

    await waitFor(() => {
      expect(screen.getByText("Recovered entry")).toBeInTheDocument();
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    // Navigation state intact after retry
    expect(router.state.location.pathname).toBe("/time-entries");
  });
});

describe("AuthRouteError slim variant (P0-4)", () => {
  it("renders the slim panel with retry and no home link", async () => {
    // Mount AuthRouteError through a minimal router whose component throws —
    // the same boundary mechanism the _auth layout registers (login/register/
    // password-reset pages) uses for any error escaping its layout.
    const rootRoute = createRootRoute();
    const boomRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: "/boom",
      errorComponent: AuthRouteError,
      component: () => {
        throw new Error("auth boom");
      },
    });
    const router = createRouter({
      routeTree: rootRoute.addChildren([boomRoute]),
      history: createMemoryHistory({ initialEntries: ["/boom"] }),
    });

    render(<RouterProvider router={router} />);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("auth boom")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /try again/i })
    ).toBeInTheDocument();
    // Slim variant: no home link (auth pages have no app shell)
    expect(
      screen.queryByRole("link", { name: /go to today/i })
    ).not.toBeInTheDocument();
  });
});
