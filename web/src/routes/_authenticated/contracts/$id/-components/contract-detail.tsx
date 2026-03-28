import {useNavigate} from '@tanstack/react-router'
import {useSuspenseQuery} from '@tanstack/react-query'
import {ArrowLeftIcon, GlobeIcon, LockIcon} from 'lucide-react'
import {Button} from '@/src/components/ui/button.tsx'
import {Badge} from '@/src/components/ui/badge.tsx'
import {Card, CardContent, CardHeader, CardTitle} from '@/src/components/ui/card.tsx'
import {Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,} from '@/src/components/ui/tooltip.tsx'
import {ContractsApis} from '@/src/api/contracts.ts'
import {Route} from "@/src/routes/_authenticated/contracts/$id";

export function ContractDetail() {
  const {id} = Route.useParams()
  const {from: fromTab} = Route.useSearch()
  const navigate = useNavigate()
  const {data: c} = useSuspenseQuery(ContractsApis.contractQueryOpts(id))

  const isAdopted = fromTab === 'adopted'

  return (
    <div className="space-y-4">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate({to: '/contracts', search: {tab: fromTab}})}
      >
        <ArrowLeftIcon className="w-4 h-4 mr-1"/>
        Back to Contracts
      </Button>

      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{c.name}</h1>
            {c.is_shared ? (
              <GlobeIcon className="w-5 h-5 text-muted-foreground"/>
            ) : (
              <LockIcon className="w-5 h-5 text-muted-foreground"/>
            )}
            {c.is_shared && <Badge variant="secondary">Shared</Badge>}
          </div>
          {isAdopted && c.created_by_org_name && (
            <p className="text-sm text-muted-foreground mt-1">
              Adopted from {c.created_by_org_name}
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger render={
                <Button variant="outline" disabled>
                  Edit
                </Button>
              }/>
              <TooltipContent>Coming soon</TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger render={
                <Button variant="destructive" disabled>
                  Delete
                </Button>
              }/>
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
            <span className="text-muted-foreground">KM Rate</span>
            <span>{c.currency} {c.km_rate.toFixed(2)}/km</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Currency</span>
            <span>{c.currency}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Governance</span>
            <span className="capitalize">{c.governance_model.replace('_', ' ')}</span>
          </div>
          {c.is_shared && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Adoption Count</span>
              <span>{c.adoption_count ?? 0}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}