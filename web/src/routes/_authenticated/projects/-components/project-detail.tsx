import {useState} from 'react'
import {useNavigate} from '@tanstack/react-router'
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query'
import {toast} from 'sonner'
import {ArrowLeftIcon, GlobeIcon, LockIcon, Trash2} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {Badge} from '@/components/ui/badge'
import {Card, CardContent, CardHeader, CardTitle} from '@/components/ui/card'
import {Skeleton} from '@/components/ui/skeleton'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {ProjectsApis} from '@/api/projects'
import {EditProjectDialog} from './edit-project-dialog'
import type {Project, Subproject} from '@/types/models'

interface ProjectDetailProps {
  id: string
  fromTab?: 'owned' | 'adopted' | 'all'
}

export function ProjectDetail({id, fromTab = 'owned'}: ProjectDetailProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const {data: project, isLoading} = useQuery(ProjectsApis.projectQueryOpts(id))

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const deleteProject = useMutation({
    mutationFn: ProjectsApis.deleteProjectMutationOpts.mutationFn,
    onSuccess: () => {
      queryClient.invalidateQueries({queryKey: ['projects']})
      toast.success('Project deleted')
      navigate({to: '/projects', search: {tab: fromTab}})
    },
    onError: (error: Error) => {
      setDeleteError(error.message)
    },
  })

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
          <Button variant="outline" onClick={() => setEditOpen(true)}>
            Edit
          </Button>
          <Button variant="destructive" onClick={() => { setDeleteOpen(true); setDeleteError(null) }}>
            <Trash2 className="w-4 h-4 mr-1" />
            Delete
          </Button>
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

      <Accordion>
        <AccordionItem value="subprojects">
          <AccordionTrigger>Subprojects</AccordionTrigger>
          <AccordionContent>
            <SubprojectSection id={id} />
          </AccordionContent>
        </AccordionItem>
      </Accordion>

      <EditProjectDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        project={p}
        onSuccess={() => {
          queryClient.invalidateQueries({queryKey: ['projects', id]})
        }}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete "{p.name}"?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete this project, remove all adoption records, and cannot be undone.
              If this project or its subprojects have active time entries, deletion will be blocked.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleteError && (
            <div className="text-sm text-destructive bg-destructive/10 rounded-md p-3">
              {deleteError}
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteProject.isPending}
              onClick={(e) => {
                e.preventDefault()
                setDeleteError(null)
                deleteProject.mutate(p.id)
              }}
            >
              {deleteProject.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function SubprojectSection({id}: {id: string}) {
  const {data, isLoading, isError, refetch} = useQuery(ProjectsApis.subprojectsQueryOpts(id))
  const subprojects = data as Subproject[] | undefined

  if (isLoading) {
    return <div className="space-y-2">
      {[1, 2, 3].map(i => <Skeleton key={i} className="h-12 w-full" />)}
    </div>
  }

  if (isError) {
    return <div className="text-center py-4">
      <p className="text-sm text-destructive">Failed to load subprojects.</p>
      <Button variant="ghost" size="sm" onClick={() => refetch()} className="mt-2">
        Retry
      </Button>
    </div>
  }

  if (!subprojects?.length) {
    return <div className="text-center py-8 text-muted-foreground">
      <p className="text-sm">No subprojects</p>
      <p className="text-xs mt-1">Subprojects can be created through time entries or project settings.</p>
    </div>
  }

  return <div className="space-y-2">
    {subprojects.map(sp => (
      <div key={sp.id} className="flex items-center justify-between py-2 border-b last:border-0">
        <div>
          <span className="font-medium">{sp.name}</span>
          {sp.description && (
            <p className="text-sm text-muted-foreground">{sp.description}</p>
          )}
        </div>
        <Badge variant={sp.is_active ? 'default' : 'secondary'}>
          {sp.is_active ? 'Active' : 'Inactive'}
        </Badge>
      </div>
    ))}
  </div>
}
