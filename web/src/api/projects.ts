import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type { CreateProjectRequest, UpdateProjectRequest } from "@/types";
import type { Project, Subproject } from "@/types/models";

function projectsQueryKey(
  scope: "owned" | "adopted" | "all",
  contractId?: string
) {
  return ["projects", scope, contractId] as const;
}

function projectQueryKey(id: string) {
  return ["projects", id] as const;
}

function projectsQueryOpts(
  scope: "owned" | "adopted" | "all" = "owned",
  contractId?: string
) {
  let url = `/projects?scope=${scope}`;
  if (contractId) {
    url += `&contract_id=${contractId}`;
  }
  return queryOptions({
    queryKey: projectsQueryKey(scope, contractId),
    queryFn: () => api<Project[]>(url),
    staleTime: 5 * 60 * 1000,
  });
}

function projectQueryOpts(id: string) {
  return queryOptions({
    queryKey: projectQueryKey(id),
    queryFn: () => api<Project>(`/projects/${id}`),
    enabled: !!id,
  });
}

const createProjectMutationOpts = mutationOptions({
  mutationFn: (data: CreateProjectRequest) =>
    api<Project>("/projects", { method: "POST", body: JSON.stringify(data) }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["projects"] });
    toast.success("Project created");
  },
});

const adoptProjectMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/projects/${id}/adopt`, { method: "POST" }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["projects"] });
    toast.success("Project added to your organization");
  },
});

const updateProjectMutationOpts = mutationOptions({
  mutationFn: ({ id, data }: { id: string; data: UpdateProjectRequest }) =>
    api<Project>(`/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, { id }, { client }) => {
    client.invalidateQueries({ queryKey: ["projects"] });
    client.invalidateQueries({ queryKey: ["projects", id] });
    toast.success("Project updated");
  },
});

const deleteProjectMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/projects/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, { client }) => {
    client.invalidateQueries({ queryKey: ["projects"] });
    toast.success("Project deleted");
  },
  onError: (error: Error) => {
    toast.error(error.message || "Failed to delete project");
  },
});

const subprojectsQueryOpts = (id: string) =>
  queryOptions({
    queryKey: ["projects", id, "subprojects"],
    queryFn: () => api<Subproject[]>(`/projects/${id}/subprojects`),
    enabled: !!id,
  });

export const ProjectsApis = {
  projectsQueryOpts,
  projectQueryOpts,
  createProjectMutationOpts,
  adoptProjectMutationOpts,
  updateProjectMutationOpts,
  deleteProjectMutationOpts,
  subprojectsQueryOpts,
};
