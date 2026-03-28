import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {toast} from 'sonner'
import {api} from '@/src/lib/api.ts'
import type {ApiResponse, CreateProjectRequest} from '@/src/types'
import type {Project} from '@/src/types/models'

function projectsQueryKey(scope: 'owned' | 'adopted' | 'all', contractId?: string) {
  return ['projects', scope, contractId] as const
}

function projectQueryKey(id: string) {
  return ['projects', id] as const
}

function projectsQueryOpts(scope: 'owned' | 'adopted' | 'all' = 'owned', contractId?: string) {
  let url = `/projects?scope=${scope}`
  if (contractId) {
    url += `&contract_id=${contractId}`
  }
  return queryOptions({
    queryKey: projectsQueryKey(scope, contractId),
    queryFn: () => api<Project[]>(url),
    staleTime: 5 * 60 * 1000,
  })
}

function projectQueryOpts(id: string) {
  return queryOptions({
    queryKey: projectQueryKey(id),
    queryFn: () => api<Project>(`/projects/${id}`),
    enabled: !!id,
  })
}

const createProjectMutationOpts = mutationOptions({
  mutationFn: (data: CreateProjectRequest) =>
    api<Project>('/projects', {method: 'POST', body: JSON.stringify(data)}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    toast.success('Project created')
  },
})

const adoptProjectMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/projects/${id}/adopt`, {method: 'POST'}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['projects']})
    toast.success('Project added to your organization')
  },
})

export const ProjectsApis = {
  projectsQueryOpts,
  projectQueryOpts,
  createProjectMutationOpts,
  adoptProjectMutationOpts,
}