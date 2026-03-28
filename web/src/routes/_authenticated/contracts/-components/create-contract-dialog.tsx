import {useState} from 'react'
import {useNavigate} from '@tanstack/react-router'
import {useMutation} from '@tanstack/react-query'
import {Button} from '@/src/components/ui/button'
import {Input} from '@/src/components/ui/input'
import {Checkbox} from '@/src/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/src/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/src/components/ui/select'
import {ContractsApis} from '@/src/api/contracts'
import type {Contract} from '@/src/types/models'

const CURRENCIES = ['EUR', 'USD', 'GBP', 'CHF', 'JPY', 'CAD', 'AUD']

const GOVERNANCE_OPTIONS = [
  {
    value: 'creator_controlled',
    label: 'Creator Controlled',
    description: 'Only your organization can approve changes to this contract',
  },
  {
    value: 'unanimous',
    label: 'Unanimous',
    description: 'All organizations using this contract must approve changes',
  },
  {
    value: 'majority',
    label: 'Majority',
    description: 'More than half of organizations using this contract must approve changes',
  },
]

interface CreateContractDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: (contract: Contract) => void
}

export function CreateContractDialog({open, onOpenChange, onSuccess}: CreateContractDialogProps) {
  const navigate = useNavigate()
  const createContract = useMutation(ContractsApis.createContractMutationOpts)

  const [name, setName] = useState('')
  const [kmRate, setKmRate] = useState('0')
  const [currency, setCurrency] = useState('EUR')
  const [governanceModel, setGovernanceModel] = useState<'creator_controlled' | 'unanimous' | 'majority'>('creator_controlled')
  const [isShared, setIsShared] = useState(false)

  const handleSubmit = () => {
    if (!name.trim()) return

    createContract.mutate(
      {
        name: name.trim(),
        km_rate: parseFloat(kmRate) || 0,
        currency,
        governance_model: governanceModel,
        is_shared: isShared,
      },
      {
        onSuccess: (data) => {
          onOpenChange(false)
          resetForm()
          if (onSuccess) {
            onSuccess(data)
          } else {
            navigate({to: '/contracts/$id', params: {id: data.id}})
          }
        },
      }
    )
  }

  const resetForm = () => {
    setName('')
    setKmRate('0')
    setCurrency('EUR')
    setGovernanceModel('creator_controlled')
    setIsShared(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create Contract</DialogTitle>
          <DialogDescription>
            Add a new contract for your organization.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Contract name"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">KM Rate</label>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={kmRate}
                onChange={(e) => setKmRate(e.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Currency</label>
              <Select value={currency} onValueChange={setCurrency}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CURRENCIES.map((c) => (
                    <SelectItem key={c} value={c}>{c}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
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
              id="is-shared"
              checked={isShared}
              onCheckedChange={(checked) => setIsShared(checked === true)}
            />
            <label htmlFor="is-shared" className="text-sm">
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
            disabled={!name.trim() || createContract.isPending}
          >
            {createContract.isPending ? 'Creating...' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}