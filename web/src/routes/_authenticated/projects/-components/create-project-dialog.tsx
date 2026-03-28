import {useState} from 'react'
import {useNavigate} from '@tanstack/react-router'
import {useMutation, useQuery} from '@tanstack/react-query'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Checkbox} from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue,} from '@/components/ui/select'
import {ProjectsApis} from '@/api/projects'
import {ContractsApis} from '@/api/contracts'
import type {Project} from '@/types/models'

const GOVERNANCE_OPTIONS = [
  {
    value: 'creator_controlled',
    label: 'Creator Controlled',
    description: 'Only your organization can approve changes to this project',
  },
  {
    value: 'unanimous',
    label: 'Unanimous',
    description: 'All organizations using this project must approve changes',
  },
  {
    value: 'majority',
    label: 'Majority',
    description: 'More than half of organizations using this project must approve changes',
  },
]

const PROJECT_TYPES = [
  {value: 'billable', label: 'Billable'},
  {value: 'internal', label: 'Internal'},
]

interface CreateProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: (project: Project) => void
}

export function CreateProjectDialog({open, onOpenChange, onSuccess}: CreateProjectDialogProps) {
  const navigate = useNavigate()
  const createProject = useMutation(ProjectsApis.createProjectMutationOpts)
  const {data: contracts} = useQuery(ContractsApis.contractsQueryOpts('all'))

  const [name, setName] = useState('')
  const [type, setType] = useState<'billable' | 'internal'>('billable')
  const [contractId, setContractId] = useState('')
  const [governanceModel, setGovernanceModel] = useState<'creator_controlled' | 'unanimous' | 'majority'>('creator_controlled')
  const [isShared, setIsShared] = useState(false)

  const handleSubmit = () => {
    if (!name.trim() || !contractId) return

    createProject.mutate(
      {
        name: name.trim(),
        type,
        contract_id: contractId,
        governance_model: governanceModel,
        is_shared: isShared,
      },
      {
        onSuccess: (data) => {
          onOpenChange(false)
          resetForm()
          if (onSuccess) {
            onSuccess(data as Project)
          } else {
            navigate({to: '/projects/$id', params: {id: (data as Project).id}})
          }
        },
      }
    )
  }

  const resetForm = () => {
    setName('')
    setType('billable')
    setContractId('')
    setGovernanceModel('creator_controlled')
    setIsShared(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create Project</DialogTitle>
          <DialogDescription>
            Add a new project for your organization.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Project name"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Type</label>
            <Select value={type} onValueChange={(v) => setType(v as 'billable' | 'internal')}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PROJECT_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Contract *</label>
            <Select value={contractId} onValueChange={setContractId}>
              <SelectTrigger>
                <SelectValue placeholder="Select a contract" />
              </SelectTrigger>
              <SelectContent>
                {contracts?.data?.map((c) => (
                  <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Governance Model</label>
            <Select value={governanceModel} onValueChange={(v) => setGovernanceModel(v as typeof governanceModel)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {GOVERNANCE_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    <div>
                      <div className="font-medium">{opt.label}</div>
                      <div className="text-xs text-muted-foreground">{opt.description}</div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="project-is-shared"
              checked={isShared}
              onCheckedChange={(checked) => setIsShared(checked)}
            />
            <label htmlFor="project-is-shared" className="text-sm">
              Share with other organizations
            </label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || !contractId || createProject.isPending}
          >
            {createProject.isPending ? 'Creating...' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}