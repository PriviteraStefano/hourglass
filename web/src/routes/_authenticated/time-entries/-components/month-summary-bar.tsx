import {addMonths, format, subMonths} from 'date-fns'
import {Button} from '@/src/components/ui/button'
import type {TimeEntryMonthlySummary} from "@/src/types";
import {useMutation} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/src/api/time-entries.ts";

interface MonthSummaryBarProps {
  month: Date
  onMonthChange: (date: Date) => void
  summary: TimeEntryMonthlySummary | undefined
}

export function MonthSummaryBar({ month, onMonthChange, summary }: MonthSummaryBarProps) {
  const submitMonth = useMutation(TimeEntriesApis.submitMonthMutationOpts)
  const draftCount = summary?.days.filter((d: { has_draft: boolean }) => d.has_draft).length ?? 0

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onMonthChange(subMonths(month, 1))}
        >
          ←
        </Button>
        <h1 className="text-xl font-semibold">
          {format(month, 'MMMM yyyy')}
        </h1>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onMonthChange(addMonths(month, 1))}
        >
          →
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="text-sm text-muted-foreground">
          Total: {summary?.totals?.total_hours ?? 0}h
        </div>

        {draftCount > 0 && (
          <Button
            onClick={() =>
              submitMonth.mutate({
                month: month.getMonth() + 1,
                year: month.getFullYear(),
              })
            }
            disabled={submitMonth.isPending}
          >
            {submitMonth.isPending ? 'Submitting...' : `Submit All Drafts (${draftCount})`}
          </Button>
        )}
      </div>
    </div>
  )
}