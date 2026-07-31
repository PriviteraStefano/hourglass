import { describe, it, expect, beforeAll, afterEach, afterAll } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
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
import type { Role, UserWithMembership, WorkingGroup } from "@/types";

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

// Mutable per-test request state — MSW handlers read them at request time.
let currentRole: Role = "employee";
let currentWgs: WorkingGroup[] = [];

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
  http.get("/api/activities", () => HttpResponse.json({ data: [] })),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  server.resetHandlers();
  currentRole = "employee";
  currentWgs = [];
});
afterAll(() => server.close());

function renderAt(initialEntries = ["/activities"]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 0 } },
  });
  const router = createRouter({
    routeTree,
    context: { client: queryClient },
    history: createMemoryHistory({ initialEntries }),
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

function groupLabels(): string[] {
  return Array.from(
    document.querySelectorAll('[data-slot="sidebar-group-label"]'),
  ).map((el) => el.textContent?.trim() ?? "");
}

describe("AppSidebar D-1 regroup (10-02)", () => {
  it("renders group labels in locked order for a manager (all visible groups)", async () => {
    currentRole = "manager";
    renderAt();

    await waitFor(() => {
      expect(groupLabels()).toHaveLength(6);
    });
    expect(groupLabels()).toEqual([
      "Track",
      "Work",
      "People",
      "Economics",
      "Review",
      "Reports",
    ]);
  });

  it("renders the Today item first with href /", async () => {
    currentRole = "employee";
    renderAt();

    const today = await screen.findByRole("link", { name: /today/i });
    expect(today).toHaveAttribute("href", "/");
    // Today sits above the first labeled group (no label on the top item)
    const firstLabel = document.querySelector(
      '[data-slot="sidebar-group-label"]',
    );
    expect(firstLabel?.textContent?.trim()).toBe("Track");
    expect(
      today.compareDocumentPosition(firstLabel!) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("hides Economics and Review for an employee without a WG stage", async () => {
    currentRole = "employee";
    currentWgs = [];
    renderAt();

    await waitFor(() => {
      expect(groupLabels()).toEqual(["Track", "Work", "People", "Reports"]);
    });
    expect(
      screen.queryByRole("link", { name: /contracts/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /approvals/i }),
    ).not.toBeInTheDocument();
  });

  it("shows Review for an employee who is a WG manager", async () => {
    currentRole = "employee";
    currentWgs = [{ manager_id: USER_ID } as WorkingGroup];
    renderAt();

    await waitFor(() => {
      expect(
        screen.getByRole("link", { name: /approvals/i }),
      ).toBeInTheDocument();
    });
    expect(groupLabels()).toContain("Review");
    // Economics stays hidden for employee even with a stage
    expect(
      screen.queryByRole("link", { name: /contracts/i }),
    ).not.toBeInTheDocument();
  });

  it("never renders Admin/Settings for any role", async () => {
    currentRole = "manager";
    renderAt();

    await waitFor(() => {
      expect(groupLabels()).toHaveLength(6);
    });
    expect(
      screen.queryByRole("link", { name: /settings/i }),
    ).not.toBeInTheDocument();
    expect(groupLabels()).not.toContain("Admin");
  });

  it("never renders Review for hr even when hr is a WG manager", async () => {
    currentRole = "hr";
    currentWgs = [{ manager_id: USER_ID } as WorkingGroup];
    renderAt();

    await waitFor(() => {
      expect(groupLabels()).toContain("People");
    });
    expect(
      screen.queryByRole("link", { name: /approvals/i }),
    ).not.toBeInTheDocument();
    expect(groupLabels()).not.toContain("Review");
  });

  it("marks Tickets and Availability disabled with locked tooltip copy", async () => {
    currentRole = "employee";
    renderAt();

    const tickets = await screen.findByRole("link", { name: /tickets/i });
    const availability = screen.getByRole("link", {
      name: /availability/i,
    });
    expect(tickets).toHaveAttribute("aria-disabled", "true");
    expect(availability).toHaveAttribute("aria-disabled", "true");

    // Tooltips only render in the collapsed sidebar — collapse, then hover
    // (base-ui opens on mouseenter after the 600ms rest delay).
    fireEvent.click(screen.getByText("Collapse"));
    const collapsedTickets = await screen.findByRole("link", {
      name: /tickets/i,
    });
    fireEvent.mouseEnter(collapsedTickets);
    await waitFor(
      () => {
        expect(
          screen.getByText("Tickets arrive in v0.2"),
        ).toBeInTheDocument();
      },
      { timeout: 2000 },
    );
  });

  it("contains no Tracking/Management legacy labels", async () => {
    currentRole = "manager";
    renderAt();

    await waitFor(() => {
      expect(groupLabels()).toHaveLength(6);
    });
    expect(screen.queryByText("Tracking")).not.toBeInTheDocument();
    expect(screen.queryByText("Management")).not.toBeInTheDocument();
  });
});
