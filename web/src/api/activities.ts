import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type {
  ActivityDetail,
  ActivityResponse,
  CreateActivityRequest,
  UpdateActivityRequest,
} from "@/types/models";

function activitiesQueryKey(scope: "owned" | "adopted" | "all", contractId?: string) {
  return ["activities", scope, contractId] as const;
}

function activityQueryKey(id: string) {
  return ["activities", id] as const;
}

function activitiesQueryOpts(
  scope: "owned" | "adopted" | "all" = "owned",
  contractId?: string
) {
  let url = `/activities?scope=${scope}`;
  if (contractId) {
    url += `&contract_id=${contractId}`;
  }
  return queryOptions({
    queryKey: activitiesQueryKey(scope, contractId),
    queryFn: () => api<ActivityResponse[]>(url),
    staleTime: 5 * 60 * 1000,
  });
}

function activityQueryOpts(id: string) {
  return queryOptions({
    queryKey: activityQueryKey(id),
    queryFn: () => api<ActivityDetail>(`/activities/${id}`),
    enabled: !!id,
  });
}

function activityChildrenQueryOpts(id: string) {
  return queryOptions({
    queryKey: ["activities", id, "children"],
    queryFn: () => api<ActivityResponse[]>(`/activities/${id}/children`),
    enabled: !!id,
  });
}

const activityKindsQueryOpts = queryOptions({
  queryKey: ["activity-kinds"],
  queryFn: () => api<string[]>("/activity-kinds"),
  staleTime: 5 * 60 * 1000,
});

const createActivityMutationOpts = mutationOptions({
  mutationFn: (data: CreateActivityRequest) =>
    api<ActivityResponse>("/activities", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["activities"] });
    toast.success("Activity created");
  },
});

const updateActivityMutationOpts = mutationOptions({
  mutationFn: ({ id, data }: { id: string; data: UpdateActivityRequest }) =>
    api<ActivityResponse>(`/activities/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, { id }, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["activities"] });
    toast.success("Activity updated");
  },
});

const deleteActivityMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/activities/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["activities"] });
    toast.success("Activity deleted");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to delete activity");
  },
});

export const ActivitiesApis = {
  activitiesQueryOpts,
  activityQueryOpts,
  activityChildrenQueryOpts,
  activityKindsQueryOpts,
  createActivityMutationOpts,
  updateActivityMutationOpts,
  deleteActivityMutationOpts,
};
