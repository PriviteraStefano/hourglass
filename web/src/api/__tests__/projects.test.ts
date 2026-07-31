import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { ProjectsApis } from "../projects";

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("ProjectsApis", () => {
  it("projectsQueryOpts calls GET /api/projects with scope", async () => {
    const mockProjects = [
      {
        id: "p1",
        name: "Project Alpha",
        type: "billable" as const,
        contract_id: "c1",
        governance_model: "creator_controlled" as const,
        is_shared: false,
        is_active: true,
        created_by_org_id: "o1",
        created_at: "2025-01-01T00:00:00Z",
      },
    ];
    server.use(
      http.get("/api/projects", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("owned");
        return HttpResponse.json({ data: mockProjects });
      })
    );

    const opts = ProjectsApis.projectsQueryOpts("owned");
    const result = await opts.queryFn();
    expect(result).toEqual(mockProjects);
  });

  it("projectsQueryOpts with contract_id adds query param", async () => {
    server.use(
      http.get("/api/projects", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("adopted");
        expect(url.searchParams.get("contract_id")).toBe("c1");
        return HttpResponse.json({ data: [] });
      })
    );

    const opts = ProjectsApis.projectsQueryOpts("adopted", "c1");
    await opts.queryFn();
  });

  it("createProjectMutationOpts sends POST /api/projects with body", async () => {
    const projectData = {
      name: "New Project",
      type: "internal" as const,
      contract_id: "c1",
      governance_model: "unanimous" as const,
      is_shared: false,
    };
    const mockProject = {
      id: "p2",
      name: "New Project",
      type: "internal" as const,
      contract_id: "c1",
      governance_model: "unanimous" as const,
      is_shared: false,
      is_active: true,
      created_by_org_id: "o1",
      created_at: "2025-01-01T00:00:00Z",
    };

    let capturedBody: unknown = null;
    server.use(
      http.post("/api/projects", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockProject });
      })
    );

    const result =
      await ProjectsApis.createProjectMutationOpts.mutationFn(projectData);
    expect(capturedBody).toEqual(projectData);
    expect(result).toEqual(mockProject);
  });

  it("projectQueryOpts calls GET /api/projects/:id", async () => {
    const mockProject = {
      id: "p1",
      name: "Project Alpha",
      type: "billable" as const,
      contract_id: "c1",
      governance_model: "creator_controlled" as const,
      is_shared: false,
      is_active: true,
      created_by_org_id: "o1",
      created_at: "2025-01-01T00:00:00Z",
    };
    server.use(
      http.get("/api/projects/p1", () =>
        HttpResponse.json({ data: mockProject })
      )
    );

    const opts = ProjectsApis.projectQueryOpts("p1");
    const result = await opts.queryFn();
    expect(result).toEqual(mockProject);
  });
});
