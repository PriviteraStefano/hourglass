import {useQuery} from '@tanstack/react-query'
import {ProjectsApis} from '@/src/api/projects.ts'

export function useProjects() {
  return useQuery(ProjectsApis.projectsQueryOpts)
}