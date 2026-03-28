import {useState} from 'react'
import {useNavigate, useSearch} from '@tanstack/react-router'
import {useMutation, useSuspenseQuery} from '@tanstack/react-query'
import {GlobeIcon, LockIcon, PlusIcon, SearchIcon} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {Badge} from '@/components/ui/badge'
import {Tabs, TabsContent, TabsList, TabsTrigger} from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {ContractsApis} from '@/api/contracts'
import {CreateContractDialog} from './create-contract-dialog'
import type {Contract} from '@/types/models'
import {z} from "zod";
import {InputGroup, InputGroupAddon, InputGroupInput,} from "@/components/ui/input-group.tsx"

export const tabsSchema = z.enum(['owned', 'adopted', 'all'])
export type TabType = z.infer<typeof tabsSchema>

export function ContractList() {
  const navigate = useNavigate()
  const {tab, searchQuery} = useSearch({from: "/_authenticated/contracts/"})

  const [adoptDialogOpen, setAdoptDialogOpen] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [selectedContract, setSelectedContract] = useState<Contract | null>(null)

  const {data: contracts, isLoading} = useSuspenseQuery(ContractsApis.contractsQueryOpts(tab))
  const adoptContract = useMutation(ContractsApis.adoptContractMutationOpts)

  const filteredContracts = contracts.filter((c: Contract) =>
    c.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const handleTabChange = (newTab: string) => {
    const parsedTab = tabsSchema.safeParse(newTab)
    if (parsedTab.error) {
      navigate({to: "/contracts", search: {tab: 'owned'}})
      return
    }
    navigate({to: "/contracts", search: {tab: parsedTab.data}})
  }

  const handleAdoptClick = (contract: Contract) => {
    setSelectedContract(contract)
    setAdoptDialogOpen(true)
  }

  const handleAdoptConfirm = () => {
    if (selectedContract) {
      adoptContract.mutate(selectedContract.id, {
        onSuccess: () => {
          setAdoptDialogOpen(false)
          setSelectedContract(null)
        },
      })
    }
  }

  const handleRowClick = (contract: Contract) => {
    navigate({to: '/contracts/$id', params: {id: contract.id}, search: {from: tab}})
  }

  return (
    <>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-lg font-semibold">Contracts</h1>
        <div className="flex items-center gap-4">
          <InputGroup>
            <InputGroupInput
              placeholder="Search contracts..."
              value={searchQuery}
              onChange={(e) => navigate({
                to: '/contracts',
                search: {tab, searchQuery: e.target.value}
              })}
            />
            <InputGroupAddon>
              <SearchIcon/>
            </InputGroupAddon>
          </InputGroup>
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
          ) : filteredContracts?.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchQuery ? 'No contracts match your search' : `No ${tab} contracts`}
            </div>
          ) : (
            <div className="border rounded-lg divide-y">
              {filteredContracts?.map((contract: Contract) => (
                <div
                  key={contract.id}
                  className="flex items-center justify-between p-4 hover:bg-muted/50 cursor-pointer"
                  onClick={() => handleRowClick(contract)}
                >
                  <div className="flex items-center gap-3">
                    {contract.is_shared ? (
                      <GlobeIcon className="w-4 h-4 text-muted-foreground"/>
                    ) : (
                      <LockIcon className="w-4 h-4 text-muted-foreground"/>
                    )}
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{contract.name}</span>
                        {contract.is_shared && (
                          <Badge variant="secondary" className="text-xs">Shared</Badge>
                        )}
                        {tab === 'adopted' && contract.created_by_org_name && (
                          <span className="text-xs text-muted-foreground">
                            from {contract.created_by_org_name}
                          </span>
                        )}
                        {tab === 'all' && contract.is_adopted && (
                          <Badge variant="outline" className="text-xs">Already adopted</Badge>
                        )}
                      </div>
                      <div className="text-sm text-muted-foreground capitalize">
                        {contract.governance_model.replace('_', ' ')}
                      </div>
                    </div>
                  </div>
                  {tab === 'all' && !contract.is_adopted && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleAdoptClick(contract)
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
            <DialogTitle>Adopt {selectedContract?.name}?</DialogTitle>
            <DialogDescription>
              This will make it available for your organization's time entries and expenses.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAdoptDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleAdoptConfirm} disabled={adoptContract.isPending}>
              {adoptContract.isPending ? 'Adopting...' : 'Adopt'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <CreateContractDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
    </>
  )
}