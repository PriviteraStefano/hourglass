import { useMemo } from 'react'
import { type TimeEntryMonthlySummary, type EntryStatus } from '@/src/types'
import { Calendar } from '@/src/components/ui/calendar'

interface MiniCalendarProps {
  month: Date
  selectedDate: Date
  onSelectDate: (date: Date) => void
  onMonthChange: (date: Date) => void
  summary: TimeEntryMonthlySummary | undefined
}

interface DaySummary {
  date: string
  has_approved: boolean
  has_rejected: boolean
  has_submitted: boolean
  has_draft: boolean
}

function inferStatus(day: DaySummary): EntryStatus | null {
  if (day.has_approved) return 'approved'
  if (day.has_rejected) return 'rejected'
  if (day.has_submitted) return 'submitted'
  if (day.has_draft) return 'draft'
  return null
}

export function MiniCalendar({ month, selectedDate, onSelectDate, onMonthChange, summary }: MiniCalendarProps) {
  const statusByDate = useMemo(() => {
    const map = new Map<string, EntryStatus>()
    summary?.days.forEach((d: DaySummary) => {
      const status = inferStatus(d)
      if (status) {
        map.set(d.date, status)
      }
    })
    return map
  }, [summary])

  const modifiers = useMemo(() => {
    const datesByStatus = new Map<EntryStatus, Date[]>()
    
    statusByDate.forEach((status: EntryStatus, dateStr: string) => {
      const date = new Date(dateStr)
      const dates = datesByStatus.get(status) || []
      dates.push(date)
      datesByStatus.set(status, dates)
    })

    return {
      draft: datesByStatus.get('draft') || [],
      submitted: datesByStatus.get('submitted') || [],
      approved: datesByStatus.get('approved') || [],
      rejected: datesByStatus.get('rejected') || [],
    }
  }, [statusByDate])

  return (
    <div className="w-80 p-4 border rounded-lg">
      <Calendar
        mode="single"
        selected={selectedDate}
        onSelect={(d) => d && onSelectDate(d)}
        month={month}
        onMonthChange={onMonthChange}
        modifiers={modifiers}
        modifiersStyles={{
          draft: { backgroundColor: '#fef3c7' },
          submitted: { backgroundColor: '#d1fae5' },
          approved: { backgroundColor: '#dbeafe' },
          rejected: { backgroundColor: '#fee2e2' },
        }}
      />
      <div className="mt-3 flex flex-wrap gap-3 text-xs">
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-yellow-100" />
          <span>Draft</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-green-100" />
          <span>Submitted</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-blue-100" />
          <span>Approved</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded bg-red-100" />
          <span>Rejected</span>
        </div>
      </div>
    </div>
  )
}