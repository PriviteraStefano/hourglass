import {useState, useCallback} from 'react'
import {format} from 'date-fns'
import {type TimeEntry} from '@/types'
import {Button} from '@/components/ui/button.tsx'
import {StatusBadge} from './status-badge.tsx'
import {EntryRow} from './entry-row.tsx'
import {ApprovalButtons} from '@/components/approval/approval-buttons.tsx'
import {ApprovalHistory} from '@/components/approval/approval-history.tsx'
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
import {useMutation, useSuspenseQuery, useQueryClient} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/api/time-entries.ts";
import {useSearch} from "@tanstack/react-router";
import {api} from '@/lib/api.ts'
import {toast} from 'sonner'

export function EntryDetail() {
  const queryClient = useQueryClient()
  const {date} = useSearch({from: "/_authenticated/time-entries/"})
  const {data: entries} = useSuspenseQuery(TimeEntriesApis.timeEntryQueryOpts(date))

  const createEntry = useMutation(TimeEntriesApis.createTimeEntryMutationOpts)
  const updateEntry = useMutation(TimeEntriesApis.updateTimeEntryMutationOpts)
  const deleteEntry = useMutation(TimeEntriesApis.deleteTimeEntryMutationOpts)
  const submitEntry = useMutation(TimeEntriesApis.submitTimeEntryMutationOpts)

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const hasEntries = entries && entries.length > 0
  const totalHours = entries?.reduce((sum, e) => sum + e.hours, 0) ?? 0

  const isEditable = (entry: TimeEntry) =>
    entry.status === 'draft' || entry.status === 'rejected'

  const handleCreate = () => {
    createEntry.mutate({
      date: format(date, 'yyyy-MM-dd'),
      project_id: '',
      subproject_id: '',
      wg_id: '',
      unit_id: '',
      hours: 0,
      description: '',
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

  if (!hasEntries) {
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
