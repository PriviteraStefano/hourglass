import { describe, it, expect, beforeAll, afterAll, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { ContractsApis } from "../contracts";

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("ContractsApis", () => {
  it("contractsQueryOpts calls GET /api/contracts with scope", async () => {
    const mockContracts = [
      {
        id: "c1",
        name: "Contract Alpha",
        km_rate: 0.5,
        currency: "EUR",
        governance_model: "creator_controlled" as const,
        is_shared: false,
        is_active: true,
        created_by_org_id: "o1",
        created_at: "2025-01-01T00:00:00Z",
      },
    ];
    server.use(
      http.get("/api/contracts", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("owned");
        return HttpResponse.json({ data: mockContracts });
      })
    );

    const opts = ContractsApis.contractsQueryOpts("owned");
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockContracts);
  });

  it("contractsQueryOpts with isActive adds filter", async () => {
    server.use(
      http.get("/api/contracts", ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("scope")).toBe("adopted");
        expect(url.searchParams.get("is_active")).toBe("true");
        return HttpResponse.json({ data: [] });
      })
    );

    const opts = ContractsApis.contractsQueryOpts("adopted", true);
    await opts.queryFn!(undefined as any);
  });

  it("createContractMutationOpts sends POST /api/contracts with customer_id", async () => {
    const contractData = {
      name: "New Contract",
      km_rate: 0.42,
      currency: "USD",
      governance_model: "majority" as const,
      is_shared: true,
      customer_id: "cust-1",
    };
    const mockContract = {
      id: "c2",
      name: "New Contract",
      km_rate: 0.42,
      currency: "USD",
      governance_model: "majority" as const,
      is_shared: true,
      is_active: true,
      created_by_org_id: "o1",
      customer_id: "cust-1",
      created_at: "2025-01-01T00:00:00Z",
    };

    let capturedBody: unknown = null;
    server.use(
      http.post("/api/contracts", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockContract });
      })
    );

    const result =
      await ContractsApis.createContractMutationOpts.mutationFn!(contractData, undefined as any);
    expect(capturedBody).toEqual(contractData);
    expect(result).toEqual(mockContract);
  });

  it("createContractMutationOpts without customer_id omits the field", async () => {
    const contractData = {
      name: "No Customer Contract",
      km_rate: 0.5,
      currency: "EUR",
      governance_model: "creator_controlled" as const,
      is_shared: false,
    };
    const mockContract = {
      id: "c3",
      name: "No Customer Contract",
      km_rate: 0.5,
      currency: "EUR",
      governance_model: "creator_controlled" as const,
      is_shared: false,
      is_active: true,
      created_by_org_id: "o1",
      created_at: "2025-01-01T00:00:00Z",
    };

    let capturedBody: unknown = null;
    server.use(
      http.post("/api/contracts", async ({ request }) => {
        capturedBody = await request.json();
        return HttpResponse.json({ data: mockContract });
      })
    );

    const result =
      await ContractsApis.createContractMutationOpts.mutationFn!(contractData, undefined as any);
    expect(capturedBody).toEqual(contractData);
    expect(
      (capturedBody as Record<string, unknown>).customer_id
    ).toBeUndefined();
    expect(result).toEqual(mockContract);
  });

  it("contractQueryOpts calls GET /api/contracts/:id", async () => {
    const mockContract = {
      id: "c1",
      name: "Contract Alpha",
      km_rate: 0.5,
      currency: "EUR",
      governance_model: "creator_controlled" as const,
      is_shared: false,
      is_active: true,
      created_by_org_id: "o1",
      created_at: "2025-01-01T00:00:00Z",
    };
    server.use(
      http.get("/api/contracts/c1", () =>
        HttpResponse.json({ data: mockContract })
      )
    );

    const opts = ContractsApis.contractQueryOpts("c1");
    const result = await opts.queryFn!(undefined as any);
    expect(result).toEqual(mockContract);
  });
});
