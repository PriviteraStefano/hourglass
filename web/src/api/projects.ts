import {queryOptions} from '@tanstack/react-query'
import {api} from '@/src/lib/api.ts'
import type {Project} from '@/src/types'

const projectsQueryOpts = queryOptions({
  queryKey: ['projects'],
  queryFn: () => api<Project[]>('/projects?scope=all'),
  staleTime: 5 * 60 * 1000,
})

export const ProjectsApis = {
  projectsQueryOpts,
}

