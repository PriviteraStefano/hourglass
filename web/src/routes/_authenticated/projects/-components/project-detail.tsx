import {useNavigate} from '@tanstack/react-router'
import {useQuery} from '@tanstack/react-query'
import {ArrowLeftIcon, GlobeIcon, LockIcon} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {Badge} from '@/components/ui/badge'
import {Card, CardContent, CardHeader, CardTitle} from '@/components/ui/card'
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,} from '@/components/ui/tooltip'
import {ProjectsApis} from '@/api/projects'
import type {Project} from '@/src/types/models'

interface ProjectDetailProps {
  id: string
  fromTab?: 'owned' | 'adopted' | 'all'
}

export function ProjectDetail({id, fromTab = 'owned'}: ProjectDetailProps) {
  const navigate = useNavigate()
  const {data: project, isLoading} = useQuery(ProjectsApis.projectQueryOpts(id))

  if (isLoading) {
    return <div className="text-center py-8 text-muted-foreground">Loading...</div>
  }

  if (!project) {
    return <div className="text-center py-8 text-muted-foreground">Project not found</div>
  }

  const p: Project = project
  const isAdopted = fromTab === 'adopted'

  return (
    <div className="space-y-4">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate({to: '/projects', search: {tab: fromTab}})}
      >
        <ArrowLeftIcon className="w-4 h-4 mr-1" />
        Back to Projects
      </Button>

      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{p.name}</h1>
            {p.is_shared ? (
              <GlobeIcon className="w-5 h-5 text-muted-foreground" />
            ) : (
              <LockIcon className="w-5 h-5 text-muted-foreground" />
            )}
            <Badge variant={p.type === 'billable' ? 'default' : 'secondary'}>
              {p.type === 'billable' ? 'Billable' : 'Internal'}
            </Badge>
            {p.is_shared && <Badge variant="outline">Shared</Badge>}
          </div>
          {isAdopted && p.created_by_org_name && (
            <p className="text-sm text-muted-foreground mt-1">
              Adopted from {p.created_by_org_name}
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="outline" disabled>
                  Edit
                </Button>
              </TooltipTrigger>
              <TooltipContent>Coming soon</TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="destructive" disabled>
                  Delete
                </Button>
              </TooltipTrigger>
              <TooltipContent>Coming soon</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Contract</span>
            <span>{p.contract_name || 'Unknown'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Type</span>
            <span className="capitalize">{p.type}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Governance</span>
            <span className="capitalize">{p.governance_model.replace('_', ' ')}</span>
          </div>
          {p.is_shared && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Adoption Count</span>
              <span>{p.adoption_count ?? 0}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}