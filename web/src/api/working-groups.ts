import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type {
  AddWorkingGroupMemberRequest,
  CreateWorkingGroupRequest,
  UpdateWorkingGroupRequest,
  WorkingGroup,
  WorkingGroupMember,
} from "@/types";

// Org-scoped working-group list. 60s staleTime keeps the approval-stage
// derivation (sidebar Review visibility, Plan 10-02) fresh without refetching
// on every mount; the list is invalidated on org switch via the existing
// queryClient.clear() convention. Mutations below invalidate ["working-groups"]
// so the list, the stage derivation, and any card member counts stay in sync.
export const workingGroupsQueryOpts = queryOptions({
  queryKey: ["working-groups"],
  queryFn: () => api<WorkingGroup[]>("/working-groups"),
  staleTime: 60 * 1000,
});

function workingGroupQueryKey(id: string) {
  return ["working-groups", id] as const;
}

function workingGroupMembersQueryKey(id: string) {
  return ["working-groups", id, "members"] as const;
}

const workingGroupQueryOpts = (id: string) =>
  queryOptions({
    queryKey: workingGroupQueryKey(id),
    queryFn: () => api<WorkingGroup>(`/working-groups/${id}`),
    enabled: !!id,
  });

const workingGroupMembersQueryOpts = (id: string) =>
  queryOptions({
    queryKey: workingGroupMembersQueryKey(id),
    queryFn: () => api<WorkingGroupMember[]>(`/working-groups/${id}/members`),
    enabled: !!id,
  });

// Create requires the caller's org id (handler parses req.org_id — threat
// T-10-06-1): the dialog sources it from the route-context profile
// (profile.membership.organization_id), never from user input, so a client can
// never target another org. Backend remains authoritative on membership.
const createWorkingGroupMutationOpts = mutationOptions({
  mutationFn: (data: CreateWorkingGroupRequest) =>
    api<WorkingGroup>("/working-groups", {
      method: "POST",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["working-groups"] });
    toast.success("Working group created");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to create working group");
  },
});

const updateWorkingGroupMutationOpts = mutationOptions({
  mutationFn: ({ id, data }: { id: string; data: UpdateWorkingGroupRequest }) =>
    api<WorkingGroup>(`/working-groups/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, { id }, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["working-groups"] });
    client.invalidateQueries({ queryKey: workingGroupQueryKey(id) });
    toast.success("Working group updated");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to update working group");
  },
});

const deleteWorkingGroupMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/working-groups/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["working-groups"] });
    toast.success("Working group deleted");
  },
  onError: (error: Error) => {
    // Backend guard errors (e.g. "cannot delete working group with members")
    // surface verbatim via the shared client, per threat T-10-06-3 — no
    // client-side cascade logic is invented.
    toast.error(error.message || "Failed to delete working group");
  },
});

const addWorkingGroupMemberMutationOpts = mutationOptions({
  mutationFn: ({
    wgId,
    data,
  }: {
    wgId: string;
    data: AddWorkingGroupMemberRequest;
  }) =>
    api<WorkingGroupMember>(`/working-groups/${wgId}/members`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, { wgId }, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["working-groups"] });
    client.invalidateQueries({ queryKey: workingGroupMembersQueryKey(wgId) });
    toast.success("Member added");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to add member");
  },
});

const removeWorkingGroupMemberMutationOpts = mutationOptions({
  mutationFn: ({ wgId, memberId }: { wgId: string; memberId: string }) =>
    api<void>(`/working-groups/${wgId}/members/${memberId}`, {
      method: "DELETE",
    }),
  onSuccess: (_, { wgId }, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["working-groups"] });
    client.invalidateQueries({ queryKey: workingGroupMembersQueryKey(wgId) });
    toast.success("Member removed");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to remove member");
  },
});

export const WorkingGroupsApis = {
  workingGroupsQueryOpts,
  workingGroupQueryOpts,
  workingGroupMembersQueryOpts,
  createWorkingGroupMutationOpts,
  updateWorkingGroupMutationOpts,
  deleteWorkingGroupMutationOpts,
  addWorkingGroupMemberMutationOpts,
  removeWorkingGroupMemberMutationOpts,
};
