import { queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/api.ts";

const API_BASE = import.meta.env.VITE_API_URL || "/api";

function exportCountQueryKey(type: string, from: string, to: string) {
  return ["exports", "count", type, from, to] as const;
}

export function exportCountQueryOpts(
  type: "timesheets" | "expenses" | "combined",
  from: string,
  to: string,
  projectId?: string,
  userId?: string
) {
  const params = new URLSearchParams({ from, to });
  if (projectId) params.set("project_id", projectId);
  if (userId) params.set("user_id", userId);

  return queryOptions({
    queryKey: exportCountQueryKey(type, from, to),
    queryFn: () => api<{ count: number }>(`/exports/${type}/count?${params}`),
  });
}

export function getExportUrl(
  type: "timesheets" | "expenses" | "combined",
  from: string,
  to: string,
  format: "csv" | "xlsx",
  projectId?: string,
  userId?: string
): string {
  const params = new URLSearchParams({ from, to, format });
  if (projectId) params.set("project_id", projectId);
  if (userId) params.set("user_id", userId);
  return `${API_BASE}/exports/${type}?${params}`;
}

export const ExportApis = {
  exportCountQueryOpts,
  getExportUrl,
};
