import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type { Contract } from "@/types/models";
import type { CreateContractRequest } from "@/types";

function contractsQueryKey(
  scope: "owned" | "adopted" | "all",
  isActive?: boolean
) {
  return ["contracts", scope, isActive] as const;
}

function contractQueryKey(id: string) {
  return ["contracts", id] as const;
}

function contractsQueryOpts(
  scope: "owned" | "adopted" | "all" = "owned",
  isActive?: boolean
) {
  const queryFn = () => {
    const params = new URLSearchParams({ scope });
    if (isActive !== undefined) {
      params.set("is_active", String(isActive));
    }
    return api<Contract[]>(`/contracts?${params.toString()}`);
  };
  return queryOptions({
    queryKey: contractsQueryKey(scope, isActive),
    queryFn,
  });
}

function contractQueryOpts(id: string) {
  return queryOptions({
    queryKey: contractQueryKey(id),
    queryFn: () => api<Contract>(`/contracts/${id}`),
    enabled: !!id,
  });
}

const createContractMutationOpts = mutationOptions({
  mutationFn: (data: CreateContractRequest) =>
    api<Contract>("/contracts", { method: "POST", body: JSON.stringify(data) }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["contracts"] });
    toast.success("Contract created");
  },
});

const adoptContractMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/contracts/${id}/adopt`, { method: "POST" }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["contracts"] });
    toast.success("Contract added to your organization");
  },
});

interface UpdateContractRequest {
  name: string;
  km_rate: number;
  currency: string;
  governance_model: string;
  is_shared: boolean;
  is_active: boolean;
  customer_id?: string;
}

const updateContractMutationOpts = mutationOptions({
  mutationFn: ({ id, data }: { id: string; data: UpdateContractRequest }) =>
    api<{ contract: Contract; affected_mileage_count: number }>(
      `/contracts/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      }
    ),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["contracts"] });
    toast.success("Contract updated");
  },
});

const deleteContractMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/contracts/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["contracts"] });
    toast.success("Contract deleted");
  },
});

const recalculateMileageMutationOpts = mutationOptions({
  mutationFn: ({ id, fromDate }: { id: string; fromDate: string }) =>
    api<{ recalculated_count: number }>(
      `/contracts/${id}/recalculate-mileage`,
      {
        method: "POST",
        body: JSON.stringify({ from_date: fromDate }),
      }
    ),
});

export const ContractsApis = {
  contractsQueryOpts,
  contractQueryOpts,
  createContractMutationOpts,
  adoptContractMutationOpts,
  updateContractMutationOpts,
  deleteContractMutationOpts,
  recalculateMileageMutationOpts,
};
