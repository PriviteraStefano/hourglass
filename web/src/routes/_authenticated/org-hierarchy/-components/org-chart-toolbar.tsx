import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'
import {Plus, Search} from 'lucide-react'
import {useOrgHierarchyStore} from '../-context/org-hierarchy-context'
import {Tabs, TabsList, TabsTrigger} from "@/components/ui/tabs.tsx";

export function OrgChartToolbar() {
  const searchQuery = useOrgHierarchyStore(s => s.searchQuery)
  const setSearchQuery = useOrgHierarchyStore(s => s.setSearchQuery)
  const viewMode = useOrgHierarchyStore(s => s.viewMode)
  const setViewMode = useOrgHierarchyStore(s => s.setViewMode)
  const addUnit = useOrgHierarchyStore(s => s.addUnit)

  return (
    <div className="flex items-center justify-between z-10 w-full">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-semibold">Organization Hierarchy</h1>
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"/>
          <Input
            type="text"
            placeholder="Search units..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-9 w-[250px]"
          />
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Tabs value={viewMode} onValueChange={setViewMode}>
          <TabsList>
            <TabsTrigger value={"tree"}>Tree</TabsTrigger>
            <TabsTrigger value={"members"}>Members</TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="h-6 w-px bg-border mx-2"/>
        <Button size="sm" onClick={() => addUnit(null)} className="h-9">
          <Plus className="h-4 w-4 mr-1"/>
          Add Root Unit
        </Button>
      </div>
    </div>
  )
}
