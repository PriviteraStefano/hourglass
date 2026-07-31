import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { ActivitiesApis } from "../activities";

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeActivity(overrides: Record<string, unknown> = {}) {
  return {
    id: "a1",
    org_id: "o1",
    name: "Activity Alpha",
    description: "",
    kind: "engagement",
    governance_model: "creator_controlled" as const,
    created_by_org_id: "o1",
    is_shared: false,
    is_active: true,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    contract_name: "",
    created_by_org_name: "Org One",
    adoption_count: 0,
    is_adopted: false,
    ...overrides,
  };
}

describe("ActivitiesApis", () => {
  it("activitiesQueryOpts calls GET /api/activities with scope", async () => {
    const mockActivities = [makeActivity()];
    server.use(
      http.get("/api/activities", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("owned");
        return HttpResponse.json({ data: mockActivities });
      })
    );

    const opts = ActivitiesApis.activitiesQueryOpts("owned");
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockActivities);
  });

  it("activitiesQueryOpts with contract_id adds query param", async () => {
    server.use(
      http.get("/api/activities", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("adopted");
        expect(url.searchParams.get("contract_id")).toBe("c1");
        return HttpResponse.json({ data: [] });
      })
    );

    const opts = ActivitiesApis.activitiesQueryOpts("adopted", "c1");
    await opts.queryFn!(undefined as any);
  });

  it("createActivityMutationOpts sends POST /api/activities with body", async () => {
    const activityData = {
      name: "New Activity",
      description: "",
      kind: "engagement",
      governance_model: "unanimous" as const,
      is_shared: false,
    };
    const mockActivity = makeActivity({ name: "New Activity" });

    let capturedBody: unknown = null;
    server.use(
      http.post("/api/activities", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockActivity });
      })
    );

    const result =
      await ActivitiesApis.createActivityMutationOpts.mutationFn!(activityData, undefined as any);
    expect(capturedBody).toEqual(activityData);
    expect(result).toEqual(mockActivity);
  });

  it("activityQueryOpts calls GET /api/activities/:id returning detail", async () => {
    const mockDetail = {
      activity: makeActivity(),
      ancestry: [],
      commercial_context: null,
      billable: true,
    };
    server.use(
      http.get("/api/activities/a1", () =>
        HttpResponse.json({ data: mockDetail })
      )
    );

    const opts = ActivitiesApis.activityQueryOpts("a1");
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockDetail);
  });

  it("activityChildrenQueryOpts calls GET /api/activities/:id/children", async () => {
    server.use(
      http.get("/api/activities/a1/children", ({ request }) => {
        expect(request.url).toContain("/activities/a1/children");
        return HttpResponse.json({ data: [] });
      })
    );

    const opts = ActivitiesApis.activityChildrenQueryOpts("a1");
    await opts.queryFn!(undefined as any);
  });

  it("activityKindsQueryOpts calls GET /api/activity-kinds", async () => {
    server.use(
      http.get("/api/activity-kinds", () =>
        HttpResponse.json({ data: ["engagement", "phase", "task"] })
      )
    );

    const opts = ActivitiesApis.activityKindsQueryOpts;
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(["engagement", "phase", "task"]);
  });

  it("updateActivityMutationOpts sends PUT /api/activities/:id", async () => {
    const updateData = { name: "Renamed" };
    const mockActivity = makeActivity({ name: "Renamed" });

    let capturedBody: unknown = null;
    server.use(
      http.put("/api/activities/a1", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockActivity });
      })
    );

    const result = await ActivitiesApis.updateActivityMutationOpts.mutationFn!(
      { id: "a1", data: updateData },
      undefined as any
    );
    expect(capturedBody).toEqual(updateData);
    expect(result).toEqual(mockActivity);
  });

  it("deleteActivityMutationOpts sends DELETE /api/activities/:id", async () => {
    server.use(
      http.delete("/api/activities/a1", () => HttpResponse.json({ data: {} }))
    );

    await ActivitiesApis.deleteActivityMutationOpts.mutationFn!("a1", undefined as any);
  });
});
