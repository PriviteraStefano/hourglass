import { queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/api.ts";
import type { WorkingGroup } from "@/types";

// Org-scoped working-group list. 60s staleTime keeps the approval-stage
// derivation (sidebar Review visibility, Plan 10-02) fresh without refetching
// on every mount; the list is invalidated on org switch via the existing
// queryClient.clear() convention. Mutations (create/update/delete/
// addMember/removeMember) land with the Working Groups surface in Plan 10-06.
export const workingGroupsQueryOpts = queryOptions({
  queryKey: ["working-groups"],
  queryFn: () => api<WorkingGroup[]>("/working-groups"),
  staleTime: 60 * 1000,
});

export const WorkingGroupsApis = {
  workingGroupsQueryOpts,
};
