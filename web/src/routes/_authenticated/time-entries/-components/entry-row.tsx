import {type TimeEntryItem} from '@/types'
import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'
import {XIcon} from 'lucide-react'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue,} from '@/components/ui/select'
import type {ChangeEvent} from "react";
import {useSuspenseQuery} from "@tanstack/react-query";
import {ProjectsApis} from "@/api/projects.ts";

interface EntryRowProps {
  item: TimeEntryItem
  index: number
  editable: boolean
  onUpdate: (index: number, field: keyof TimeEntryItem, value: string | number) => void
  onRemove: (index: number) => void
}

export function EntryRow({ item, index, editable, onUpdate, onRemove }: EntryRowProps) {
  const { data: projects } = useSuspenseQuery(ProjectsApis.projectsQueryOpts("all"))

  return (
    <div className="flex items-center gap-2 p-2 bg-muted/30 rounded">
      <Select
        value={item.project_id}
        onValueChange={(v) => v !== null ? onUpdate(index, 'project_id', v) : undefined}
        disabled={!editable}
      >
        <SelectTrigger className="w-48">
          <SelectValue placeholder="Select project" />
        </SelectTrigger>
        <SelectContent>
          {projects?.map((p: { id: string; name: string }) => (
            <SelectItem key={p.id} value={p.id}>
              {p.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Input
        type="number"
        step="0.25"
        min="0"
        max="24"
        value={item.hours}
        onChange={(e: ChangeEvent<HTMLInputElement>) => onUpdate(index, 'hours', parseFloat(e.target.value) || 0)}
        disabled={!editable}
        className="w-20"
      />
      <span className="text-sm">hours</span>

      <Input
        value={item.description || ''}
        onChange={(e: ChangeEvent<HTMLInputElement>) => onUpdate(index, 'description', e.target.value)}
        placeholder="Description (optional)"
        disabled={!editable}
        className="flex-1"
      />

      {editable && (
        <Button variant="ghost" size="sm" onClick={() => onRemove(index)}>
          <XIcon className="w-4 h-4" />
        </Button>
      )}
    </div>
  )
}