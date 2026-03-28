import {useState} from 'react'
import {useNavigate} from '@tanstack/react-router'
import {useQuery, useMutation} from '@tanstack/react-query'
import {PlusIcon, SearchIcon, GlobeIcon, LockIcon} from 'lucide-react'
import {Button} from '@/src/components/ui/button'
import {Input} from '@/src/components/ui/input'
import {Badge} from '@/src/components/ui/badge'
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/src/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/src/components/ui/dialog'
import {ProjectsApis} from '@/src/api/projects'
import {CreateProjectDialog} from './create-project-dialog'
import type {Project} from '@/src/types/models'

interface ProjectListProps {
  initialTab?: 'owned' | 'adopted' | 'all'
}

export function ProjectList({initialTab = 'owned'}: ProjectListProps) {
  const navigate = useNavigate()

  const [tab, setTab] = useState<'owned' | 'adopted' | 'all'>(initialTab)
  const [searchQuery, setSearchQuery] = useState('')
  const [adoptDialogOpen, setAdoptDialogOpen] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [selectedProject, setSelectedProject] = useState<Project | null>(null)

  const {data: projects, isLoading} = useQuery(ProjectsApis.projectsQueryOpts(tab))
  const adoptProject = useMutation(ProjectsApis.adoptProjectMutationOpts)

  const filteredProjects = projects?.filter((p: Project) =>
    p.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const handleTabChange = (newTab: string) => {
    setTab(newTab as 'owned' | 'adopted' | 'all')
    navigate({search: {tab: newTab}})
  }

  const handleAdoptClick = (project: Project) => {
    setSelectedProject(project)
    setAdoptDialogOpen(true)
  }

  const handleAdoptConfirm = () => {
    if (selectedProject) {
      adoptProject.mutate(selectedProject.id, {
        onSuccess: () => {
          setAdoptDialogOpen(false)
          setSelectedProject(null)
        },
      })
    }
  }

  const handleRowClick = (project: Project) => {
    navigate({to: '/projects/$id', params: {id: project.id}, search: {from: tab}})
  }

  return (
    <>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-semibold">Projects</h1>
        <div className="flex items-center gap-4">
          <div className="relative">
            <SearchIcon className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground"/>
            <Input
              placeholder="Search projects..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-64 pl-8"
            />
          </div>
          {tab === 'owned' && (
            <Button onClick={() => setCreateDialogOpen(true)}>
              <PlusIcon className="w-4 h-4 mr-1"/>
              Create
            </Button>
          )}
        </div>
      </div>

      <Tabs value={tab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="owned">Owned</TabsTrigger>
          <TabsTrigger value="adopted">Adopted</TabsTrigger>
          <TabsTrigger value="all">All</TabsTrigger>
        </TabsList>

        <TabsContent value={tab} className="mt-4">
          {isLoading ? (
            <div className="text-center py-8 text-muted-foreground">Loading...</div>
          ) : filteredProjects?.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchQuery ? 'No projects match your search' : `No ${tab} projects`}
            </div>
          ) : (
            <div className="border rounded-lg divide-y">
              {filteredProjects?.map((project: Project) => (
                <div
                  key={project.id}
                  className="flex items-center justify-between p-4 hover:bg-muted/50 cursor-pointer"
                  onClick={() => handleRowClick(project)}
                >
                  <div className="flex items-center gap-3">
                    {project.is_shared ? (
                      <GlobeIcon className="w-4 h-4 text-muted-foreground"/>
                    ) : (
                      <LockIcon className="w-4 h-4 text-muted-foreground"/>
                    )}
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{project.name}</span>
                        <Badge variant={project.type === 'billable' ? 'default' : 'secondary'} className="text-xs">
                          {project.type === 'billable' ? 'Billable' : 'Internal'}
                        </Badge>
                        {project.is_shared && (
                          <Badge variant="outline" className="text-xs">Shared</Badge>
                        )}
                        {tab === 'adopted' && project.created_by_org_name && (
                          <span className="text-xs text-muted-foreground">
                            from {project.created_by_org_name}
                          </span>
                        )}
                        {tab === 'all' && project.is_adopted && (
                          <Badge variant="outline" className="text-xs">Already adopted</Badge>
                        )}
                      </div>
                    </div>
                  </div>
                  {tab === 'all' && !project.is_adopted && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleAdoptClick(project)
                      }}
                    >
                      Adopt
                    </Button>
                  )}
                </div>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <Dialog open={adoptDialogOpen} onOpenChange={setAdoptDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Adopt {selectedProject?.name}?</DialogTitle>
            <DialogDescription>
              This will make it available for your organization's time entries and expenses.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAdoptDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleAdoptConfirm} disabled={adoptProject.isPending}>
              {adoptProject.isPending ? 'Adopting...' : 'Adopt'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <CreateProjectDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
    </>
  )
}