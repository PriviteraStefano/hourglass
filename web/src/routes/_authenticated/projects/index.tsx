import {createFileRoute} from '@tanstack/react-router'
import {ProjectList} from './-components/project-list'

export const Route = createFileRoute('/_authenticated/projects/')({
  component: ProjectList,
  validateSearch: (search: Record<string, unknown>) => ({
    tab: (search.tab as 'owned' | 'adopted' | 'all') || 'owned',
  }),
})