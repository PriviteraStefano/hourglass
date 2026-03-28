import { useState } from 'react'
import { format } from 'date-fns'
import { type TimeEntry, type TimeEntryItem } from '@/src/types'
import { Button } from '@/src/components/ui/button.tsx'
import { StatusBadge } from './status-badge.tsx'
import { EntryRow } from './entry-row.tsx'
import {useMutation} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/src/api/time-entries.ts";

interface EntryDetailProps {
  date: Date
  entry: TimeEntry | null | undefined
}

export function EntryDetail({ date, entry }: EntryDetailProps) {
  const createEntry = useMutation(TimeEntriesApis.createTimeEntryMutationOpts)
  const updateEntry = useMutation(TimeEntriesApis.updateTimeEntryMutationOpts)
  const deleteEntry = useMutation(TimeEntriesApis.deleteTimeEntryMutationOpts)
  const submitEntry = useMutation(TimeEntriesApis.submitTimeEntryMutationOpts)
  
  const [items, setItems] = useState<TimeEntryItem[]>(
    entry?.items ?? [{ id: '', time_entry_id: '', project_id: '', hours: 0, description: '' }]
  )

  const isEditable = !entry || entry.status === 'draft' || entry.status === 'rejected'
  
  const totalHours = items.reduce((sum, item) => sum + item.hours, 0)

  const handleCreate = () => {
    createEntry.mutate({
      date: format(date, 'yyyy-MM-dd'),
      items: items.map((item) => ({
        project_id: item.project_id,
        hours: item.hours,
        description: item.description,
      })),
    })
  }

  const handleUpdate = () => {
    if (entry) {
      updateEntry.mutate({
        id: entry.id,
        items: items.map((item) => ({
          project_id: item.project_id,
          hours: item.hours,
          description: item.description,
        })),
      })
    }
  }

  const handleDelete = () => {
    if (entry && confirm('Are you sure you want to delete this entry?')) {
      deleteEntry.mutate(entry.id)
    }
  }

  const handleSubmit = () => {
    if (entry) {
      submitEntry.mutate(entry.id)
    }
  }

  const handleAddRow = () => {
    setItems([...items, { id: '', time_entry_id: '', project_id: '', hours: 0, description: '' }])
  }

  const handleRemoveRow = (index: number) => {
    setItems(items.filter((_, i) => i !== index))
  }

  const handleUpdateItem = (index: number, field: keyof TimeEntryItem, value: string | number) => {
    const newItems = [...items]
    newItems[index] = { ...newItems[index], [field]: value }
    setItems(newItems)
  }

  if (!entry) {
    return (
      <div className="flex-1 flex items-center justify-center border rounded-lg p-8">
        <Button onClick={handleCreate} disabled={createEntry.isPending}>
          {createEntry.isPending ? 'Creating...' : `Create Entry for ${format(date, 'MMMM d')}`}
        </Button>
      </div>
    )
  }

  return (
    <div className="flex-1 border rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">
            {format(date, 'EEEE, MMMM d, yyyy')}
          </h2>
          <div className="flex items-center gap-2 mt-1">
            <StatusBadge status={entry.status} />
            <span className="text-sm text-muted-foreground">
              Total: {totalHours}h
            </span>
          </div>
        </div>
        {isEditable && (
          <Button variant="destructive" size="sm" onClick={handleDelete}>
            Delete
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {items.map((item, index) => (
          <EntryRow
            key={index}
            item={item}
            index={index}
            editable={isEditable}
            onUpdate={handleUpdateItem}
            onRemove={handleRemoveRow}
          />
        ))}
      </div>

      {isEditable && (
        <Button variant="outline" size="sm" className="mt-3" onClick={handleAddRow}>
          + Add Project Row
        </Button>
      )}

      {isEditable && (
        <div className="mt-4 flex gap-2">
          <Button 
            onClick={handleUpdate}
            disabled={totalHours === 0 || updateEntry.isPending}
          >
            {updateEntry.isPending ? 'Saving...' : 'Save Draft'}
          </Button>
          <Button
            variant="default"
            disabled={totalHours === 0 || submitEntry.isPending}
            onClick={handleSubmit}
          >
            {submitEntry.isPending ? 'Submitting...' : 'Submit Entry'}
          </Button>
        </div>
      )}
    </div>
  )
}