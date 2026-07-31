import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/api.ts";
import { z } from "zod";
import {
  type AddUnitMemberRequest,
  type CreateUnitRequest,
  type Unit,
  UnitMemberSchema,
  UnitSchema,
  type UnitTreeNode,
  UnitTreeNodeSchema,
  type UpdateUnitRequest,
} from "@/types/unit.ts";

const OrgMemberSchema = z.object({
  id: z.string(),
  user_id: z.string().nullable(),
  user_name: z.string().optional(),
  user_email: z.string().optional(),
  role: z.string(),
  is_active: z.boolean(),
});

export type OrgMember = z.infer<typeof OrgMemberSchema>;

export const orgMembersQueryOpts = queryOptions({
  queryKey: ["organizations", "members"],
  queryFn: async () => {
    const data = await api<OrgMember[]>("/organizations/members");
    return OrgMemberSchema.array().parse(data);
  },
  staleTime: 5 * 60 * 1000,
});

export const unitTreeQueryOpts = queryOptions({
  queryKey: ["units", "tree"],
  queryFn: async () => {
    const data = await api<UnitTreeNode[]>("/units/tree");
    return UnitTreeNodeSchema.array().parse(data);
  },
  staleTime: 5 * 60 * 1000,
});

export const unitDetailQueryOpts = (id: string) =>
  queryOptions({
    queryKey: ["units", "detail", id],
    queryFn: async () => {
      const data = await api<Unit>(`/units/${id}`);
      return UnitSchema.parse(data);
    },
    enabled: !!id,
  });

export const unitMembersQueryOpts = (unitId: string) =>
  queryOptions({
    queryKey: ["units", "members", unitId],
    queryFn: async () => {
      const data = await api<import("@/types/unit.ts").UnitMember[]>(
        `/units/${unitId}/members`
      );
      return UnitMemberSchema.array().parse(data);
    },
    enabled: !!unitId,
  });

export const createUnitMutationOpts = mutationOptions({
  mutationFn: (body: CreateUnitRequest) =>
    api<Unit>("/units", { method: "POST", body: JSON.stringify(body) }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "tree"] });
  },
});

export const updateUnitMutationOpts = mutationOptions({
  mutationFn: ({ id, body }: { id: string; body: UpdateUnitRequest }) =>
    api<Unit>(`/units/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "tree"] });
    client.invalidateQueries({ queryKey: ["units", "detail"] });
  },
});

export const reparentUnitMutationOpts = mutationOptions({
  mutationFn: ({
    id,
    parent_unit_id,
  }: {
    id: string;
    parent_unit_id: string | null;
  }) =>
    api<Unit>(`/units/${id}`, {
      method: "PUT",
      body: JSON.stringify({ parent_unit_id }),
    }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "tree"] });
    client.invalidateQueries({ queryKey: ["units", "detail"] });
  },
});

export const deleteUnitMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<{ message: string }>(`/units/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "tree"] });
    client.invalidateQueries({ queryKey: ["units", "detail"] });
  },
});

export const addUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({
    unitId,
    body,
  }: {
    unitId: string;
    body: AddUnitMemberRequest;
  }) =>
    api<import("@/types/unit.ts").UnitMember>(`/units/${unitId}/members`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  onSuccess: (_, { unitId }, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "members", unitId] });
    client.invalidateQueries({ queryKey: ["units", "tree"] });
  },
});

export const removeUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({
    unitId,
    membershipId,
  }: {
    unitId: string;
    membershipId: string;
  }) =>
    api<void>(`/units/${unitId}/members/${membershipId}`, { method: "DELETE" }),
  onSuccess: (_, { unitId }, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "members", unitId] });
    client.invalidateQueries({ queryKey: ["units", "tree"] });
  },
});

export const updateUnitMemberMutationOpts = mutationOptions({
  mutationFn: ({
    unitId,
    membershipId,
    is_primary,
  }: {
    unitId: string;
    membershipId: string;
    is_primary: boolean;
  }) =>
    api<import("@/types/unit.ts").UnitMember>(
      `/units/${unitId}/members/${membershipId}`,
      {
        method: "PUT",
        body: JSON.stringify({ is_primary }),
      }
    ),
  onSuccess: (_, { unitId }, { client }) => {
    client.invalidateQueries({ queryKey: ["units", "members", unitId] });
    client.invalidateQueries({ queryKey: ["units", "tree"] });
  },
});
