import {createFileRoute} from '@tanstack/react-router'
import {ProjectDetail} from './-components/project-detail'

export const Route = createFileRoute('/_authenticated/projects/$id')({
  component: ProjectDetailPage,
  validateSearch: (search: Record<string, unknown>) => ({
    from: (search.from as 'owned' | 'adopted' | 'all') || 'owned',
  }),
})

function ProjectDetailPage() {
  const {id} = Route.useParams()
  const search = Route.useSearch()
  
  return <ProjectDetail id={id} fromTab={search.from} />
}