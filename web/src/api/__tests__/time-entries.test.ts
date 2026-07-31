import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { TimeEntriesApis } from "../time-entries";

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("TimeEntriesApis", () => {
  it("timeEntriesForMonthQueryOpts calls GET /time-entries with month/year", async () => {
    const mockEntries: Array<{ id: string; date: string; status: string }> = [
      { id: "te1", date: "2026-05-01", status: "draft" },
    ];
    server.use(
      http.get("/api/time-entries", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("month")).toBe("5");
        expect(url.searchParams.get("year")).toBe("2026");
        return HttpResponse.json({ data: mockEntries });
      })
    );

    const opts = TimeEntriesApis.timeEntriesForMonthQueryOpts(5, 2026);
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockEntries);
  });

  it("createTimeEntryMutationOpts sends POST /time-entries with entry data", async () => {
    const entryData = {
      activity_id: "a1",
      unit_id: "u1",
      hours: 8,
      description: "Work",
      date: "2026-05-18",
    };
    const mockEntry = {
      id: "te1",
      user_id: "u1",
      org_id: "o1",
      activity_id: "a1",
      unit_id: "u1",
      hours: 8,
      description: "Work",
      entry_date: "2026-05-18",
      status: "draft" as const,
      created_at: "2026-05-18T10:00:00Z",
      updated_at: "2026-05-18T10:00:00Z",
    };

    let capturedBody: unknown = null;
    server.use(
      http.post("/api/time-entries", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockEntry });
      })
    );

    const result =
      await TimeEntriesApis.createTimeEntryMutationOpts.mutationFn!(entryData, undefined as any);
    expect(capturedBody).toEqual(entryData);
    expect(result).toEqual(mockEntry);
  });

  it("submitTimeEntryMutationOpts sends POST /time-entries/{id}/submit", async () => {
    server.use(
      http.post("/api/time-entries/:id/submit", async ({ params }) => {
        expect(params.id).toBe("te1");
        return HttpResponse.json({ data: { id: "te1", status: "submitted" } });
      })
    );

    const result =
      await TimeEntriesApis.submitTimeEntryMutationOpts.mutationFn!("te1", undefined as any);
    expect(result).toEqual({ id: "te1", status: "submitted" });
  });

  it("timeEntryQueryOpts calls GET /time-entries with date", async () => {
    const mockEntry = {
      id: "te1",
      user_id: "u1",
      organization_id: "o1",
      date: "2026-05-18",
      status: "draft" as const,
      created_at: "2026-05-18T10:00:00Z",
      updated_at: "2026-05-18T10:00:00Z",
      items: [],
    };
    server.use(
      http.get("/api/time-entries", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("date")).toBe("2026-05-18");
        return HttpResponse.json({ data: mockEntry });
      })
    );

    const opts = TimeEntriesApis.timeEntryQueryOpts(new Date("2026-05-18"));
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockEntry);
  });
});
