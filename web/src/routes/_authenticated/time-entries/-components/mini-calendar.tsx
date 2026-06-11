import {useMemo} from 'react'
import {format} from 'date-fns'
import {type EntryStatus, type TimeEntry} from '@/types'
import {Calendar} from '@/components/ui/calendar'
import {useNavigate, useSearch} from "@tanstack/react-router";
import {useSuspenseQuery} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/api/time-entries.ts";
import {Separator} from "@/components/ui/separator.tsx";
import {Label} from "@/components/ui/label.tsx";

const statusPriority: Record<string, number> = {
  approved: 5,
  rejected: 4,
  pending_finance: 3,
  pending_manager: 2,
  submitted: 1,
  draft: 0,
}

export function MiniCalendar() {
  const navigate = useNavigate()
  const {month, date} = useSearch({from: "/_authenticated/time-entries/"})
  const {data: entries} = useSuspenseQuery(
    TimeEntriesApis.timeEntriesForMonthQueryOpts(
      month.getMonth() + 1,
      month.getFullYear()
    )
  )

  const statusByDate = useMemo(() => {
    const map = new Map<string, EntryStatus>()
    entries?.forEach((entry: TimeEntry) => {
      const dateStr = format(new Date(entry.entry_date), 'yyyy-MM-dd')
      const existing = map.get(dateStr)
      if (!existing || (statusPriority[entry.status] ?? 0) > (statusPriority[existing] ?? 0)) {
        map.set(dateStr, entry.status)
      }
    })
    return map
  }, [entries])

  const modifiers = useMemo(() => {
    const datesByStatus = new Map<EntryStatus, Date[]>()

    statusByDate.forEach((status: EntryStatus, dateStr: string) => {
      const d = new Date(dateStr + 'T00:00:00')
      const arr = datesByStatus.get(status) || []
      arr.push(d)
      datesByStatus.set(status, arr)
    })

    return {
      draft: datesByStatus.get('draft') || [],
      submitted: datesByStatus.get('submitted') || [],
      pending_manager: datesByStatus.get('pending_manager') || [],
      pending_finance: datesByStatus.get('pending_finance') || [],
      approved: datesByStatus.get('approved') || [],
      rejected: datesByStatus.get('rejected') || [],
    }
  }, [statusByDate])

  return (
    <div className="w-fit p-3 bg-accent space-y-2 rounded-xl">
      <Calendar
        mode="single"
        classNames={{
          "root": "rounded-xl p-1.5",
        }}
        weekStartsOn={1}
        showOutsideDays={false}
        showWeekNumber={true}
        selected={date}
        onSelect={(d) => d && navigate({
          to: "/time-entries",
          search: {date: d, month: month}
        })}
        month={month}
        onMonthChange={(m) => navigate({
          to: "/time-entries",
          search: {date: date, month: m}
        })}
        className="p-0 [--cell-size:--spacing(11)]"
        modifiers={modifiers}
        modifiersStyles={{
          draft: {backgroundColor: '#fef08a'},
          submitted: {backgroundColor: '#bfdbfe'},
          pending_manager: {backgroundColor: '#bbf7d0'},
          pending_finance: {backgroundColor: '#e9d5ff'},
          approved: {backgroundColor: '#bfdbfe'},
          rejected: {backgroundColor: '#fecaca'},
        }}
      />
      <Separator/>
      <Label>Legend</Label>
      <div className="mt-3 flex flex-col gap-3 text-xs">
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-yellow-200"/>
          <span>Draft</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-blue-200"/>
          <span>Submitted</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-green-200"/>
          <span>Pending Manager</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-purple-200"/>
          <span>Pending Finance</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-blue-200"/>
          <span>Approved</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="size-4 rounded bg-red-200"/>
          <span>Rejected</span>
        </div>
      </div>
    </div>
  )
}
