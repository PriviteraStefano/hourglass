import {useCallback, useEffect, useState} from 'react'
import {format} from 'date-fns'
import {type CreateTimeEntryRequest, type TimeEntry} from '@/types'
import type {Subproject} from '@/types/models'
import {Button} from '@/components/ui/button.tsx'
import {Input} from '@/components/ui/input.tsx'
import {EntryRow} from './entry-row.tsx'
import {ApprovalButtons} from '@/components/approval/approval-buttons.tsx'
import {ApprovalHistory} from '@/components/approval/approval-history.tsx'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select.tsx'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog.tsx'
import {useMutation, useQueryClient, useSuspenseQuery} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/api/time-entries.ts";
import {ProjectsApis} from "@/api/projects.ts";
import {useSearch} from "@tanstack/react-router";
import {api} from '@/lib/api.ts'
import {toast} from 'sonner'

export function EntryDetail() {
  const queryClient = useQueryClient()
  const {date} = useSearch({from: "/_authenticated/time-entries/"})
  const {data: entries} = useSuspenseQuery(TimeEntriesApis.timeEntryQueryOpts(date))
  const {data: projects} = useSuspenseQuery(ProjectsApis.projectsQueryOpts("all"))

  const createEntry = useMutation(TimeEntriesApis.createTimeEntryMutationOpts)
  const updateEntry = useMutation(TimeEntriesApis.updateTimeEntryMutationOpts)
  const deleteEntry = useMutation(TimeEntriesApis.deleteTimeEntryMutationOpts)
  const submitEntry = useMutation(TimeEntriesApis.submitTimeEntryMutationOpts)

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [newEntry, setNewEntry] = useState<CreateTimeEntryRequest>({
    date: format(date, 'yyyy-MM-dd'),
    project_id: '',
    subproject_id: '',
    wg_id: '',
    unit_id: '',
    hours: 0,
    description: '',
  })
  const [subprojects, setSubprojects] = useState<Subproject[]>([])
  const [wgs, setWgs] = useState<{ id: string; name: string; unit_ids: string[] }[]>([])
  const [loadingSub, setLoadingSub] = useState(false)
  const [loadingWg, setLoadingWg] = useState(false)

  useEffect(() => {
    if (!newEntry.project_id) {
      setSubprojects([])
      setWgs([])
      return
    }
    setLoadingSub(true)
    api<Subproject[]>(`/projects/${newEntry.project_id}/subprojects`).then(subs => {
      setSubprojects(subs)
      const spId = subs.length > 0 ? subs[0].id : ''
      setNewEntry(prev => ({...prev, subproject_id: spId, wg_id: '', unit_id: ''}))
    }).catch(() => toast.error('Failed to load subprojects'))
      .finally(() => setLoadingSub(false))
  }, [newEntry.project_id])

  useEffect(() => {
    if (!newEntry.subproject_id) {
      setWgs([])
      return
    }
    setLoadingWg(true)
    api<{
      id: string;
      name: string;
      unit_ids: string[]
    }[]>(`/working-groups?subproject_id=${newEntry.subproject_id}`).then(wgList => {
      setWgs(wgList)
      const wgId = wgList.length > 0 ? wgList[0].id : ''
      const unitId = wgList.length > 0 && wgList[0].unit_ids.length > 0 ? wgList[0].unit_ids[0] : ''
      setNewEntry(prev => ({...prev, wg_id: wgId, unit_id: unitId}))
    }).catch(() => toast.error('Failed to load working groups'))
      .finally(() => setLoadingWg(false))
  }, [newEntry.subproject_id])

  const hasEntries = entries && entries.length > 0
  const totalHours = entries?.reduce((sum, e) => sum + e.hours, 0) ?? 0

  const isEditable = (entry: TimeEntry) =>
    entry.status === 'draft' || entry.status === 'rejected'

  const handleCreate = () => {
    setNewEntry({
      date: format(date, 'yyyy-MM-dd'),
      project_id: '',
      subproject_id: '',
      wg_id: '',
      unit_id: '',
      hours: 0,
      description: '',
    })
    setSubprojects([])
    setWgs([])
    setIsCreating(true)
  }

  const handleSaveNewEntry = () => {
    if (!newEntry.project_id) {
      toast.error('Please select a project')
      return
    }
    if (!newEntry.subproject_id) {
      toast.error('No subproject available for this project')
      return
    }
    if (!newEntry.wg_id) {
      toast.error('No working group available for this subproject')
      return
    }
    if (!newEntry.unit_id) {
      toast.error('No unit available for this working group')
      return
    }
    if (!newEntry.hours || newEntry.hours <= 0) {
      toast.error('Hours must be greater than 0')
      return
    }
    createEntry.mutate(newEntry, {
      onSuccess: () => setIsCreating(false),
    })
  }

  const handleUpdate = useCallback((id: string, entry: Partial<TimeEntry>) => {
    updateEntry.mutate({id, ...entry} as Parameters<typeof updateEntry.mutate>[0])
  }, [updateEntry])

  const handleDelete = () => {
    if (entries && entries[0]) {
      deleteEntry.mutate(entries[0].id)
    }
    setDeleteDialogOpen(false)
  }

  const handleSubmit = (id: string) => {
    submitEntry.mutate(id)
  }

  const handleApprove = (id: string) => {
    api<TimeEntry>(`/time-entries/${id}/approve`, {method: 'POST'}).then(() => {
      queryClient.invalidateQueries({queryKey: ['time-entries']})
      toast.success('Entry approved')
    }).catch(() => {
      toast.error('Failed to approve entry')
    })
  }

  const handleReject = (id: string, reason: string) => {
    api<TimeEntry>(`/time-entries/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({reason}),
    }).then(() => {
      queryClient.invalidateQueries({queryKey: ['time-entries']})
      toast.success('Entry rejected')
    }).catch(() => {
      toast.error('Failed to reject entry')
    })
  }

  const handleUpdateEntry = (id: string, field: string, value: string | number) => {
    handleUpdate(id, {[field]: value} as Partial<TimeEntry>)
  }

  if (!hasEntries && !isCreating) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center rounded-lg p-8 gap-3">
        <p className="text-sm text-muted-foreground">
          No entries for {format(date, 'MMMM d, yyyy')}
        </p>
        <p className="text-xs text-muted-foreground">
          Create a time entry to track your hours for this day.
        </p>
        <Button onClick={handleCreate} disabled={createEntry.isPending}>
          {createEntry.isPending ? 'Creating...' : 'Create Entry'}
        </Button>
      </div>
    )
  }

  if (isCreating) {
    return (
      <div className="flex-1 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">
            {format(date, 'EEEE, MMMM d, yyyy')}
          </h2>
        </div>
        <div className="flex flex-col gap-2 p-2 bg-muted/30 rounded">
          <div className="flex items-center gap-2">
            <Select
              value={newEntry.project_id}
              onValueChange={(v) => v !== null && setNewEntry({...newEntry, project_id: v})}
            >
              <SelectTrigger className="w-48">
                <SelectValue placeholder="Select project"/>
              </SelectTrigger>
              <SelectContent>
                {projects?.map((p: { id: string; name: string }) => (
                  <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              type="number"
              step="0.25"
              min="0"
              max="24"
              value={newEntry.hours || ''}
              onChange={(e) => setNewEntry({...newEntry, hours: parseFloat(e.target.value) || 0})}
              className="w-20"
              placeholder="Hours"
            />
            <span className="text-sm">hours</span>
            <Input
              value={newEntry.description}
              onChange={(e) => setNewEntry({...newEntry, description: e.target.value})}
              placeholder="Description (optional)"
              className="flex-1"
            />
            <Button onClick={handleSaveNewEntry}
                    disabled={createEntry.isPending || !newEntry.project_id || !newEntry.subproject_id || !newEntry.wg_id || !newEntry.unit_id}>
              {createEntry.isPending ? 'Saving...' : 'Save Draft'}
            </Button>
            <Button variant="ghost" onClick={() => setIsCreating(false)} disabled={createEntry.isPending}>
              Cancel
            </Button>
          </div>
          {(loadingSub || loadingWg) && (
            <p className="text-xs text-muted-foreground">
              {loadingSub ? 'Loading subprojects...' : 'Loading working group...'}
            </p>
          )}
          {!loadingSub && !loadingWg && newEntry.project_id && !newEntry.subproject_id && (
            <p className="text-xs text-destructive">No subproject found for this project</p>
          )}
          {!loadingSub && !loadingWg && newEntry.subproject_id && !newEntry.wg_id && (
            <p className="text-xs text-destructive">No working group found for this subproject</p>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">
            {format(date, 'EEEE, MMMM d, yyyy')}
          </h2>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-sm text-muted-foreground">
              Total: {totalHours}h
            </span>
          </div>
        </div>
        {hasEntries && entries?.every(e => e.status === 'draft' || e.status === 'rejected') && (
          <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
            <AlertDialogTrigger render={<Button variant="destructive" size="sm">Delete</Button>} />
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Entry?</AlertDialogTitle>
                <AlertDialogDescription>
                  This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      <div className="space-y-2">
        {entries?.map((entry) => (
          <div key={entry.id}>
            <EntryRow
              entry={entry}
              editable={isEditable(entry)}
              onUpdate={(field, value) => handleUpdateEntry(entry.id, field, value)}
              onDelete={() => {
                deleteEntry.mutate(entry.id)
              }}
              onSubmit={() => handleSubmit(entry.id)}
            />
            {(entry.status === 'pending_manager' || entry.status === 'pending_finance') && (
              <div className="mt-2 pl-2 space-y-2">
                <ApprovalButtons
                  status={entry.status}
                  currentApproverRole={entry.current_approver_role}
                  userRole="employee"
                  onApprove={() => handleApprove(entry.id)}
                  onReject={(reason) => handleReject(entry.id, reason)}
                  isPending={false}
                />
              </div>
            )}
            {(entry.status === 'approved' || entry.status === 'rejected') && (
              <div className="mt-2 pl-2">
                <ApprovalHistory approvals={[]} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
