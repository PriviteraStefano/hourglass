import { useState, useCallback } from 'react'
import {
  format,
  startOfMonth,
  endOfMonth,
  subMonths,
  startOfQuarter,
  endOfQuarter,
  startOfYear,
  endOfYear,
} from 'date-fns'
import { DayPicker, type DateRange } from 'react-day-picker'
import { Download } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useDownload } from '@/lib/use-download'
import {
  exportCountQueryOpts,
  getExportUrl,
} from '@/api/exports'
import { queryClient } from '@/lib/query-client'

interface ExportFormProps {
  type: 'timesheets' | 'expenses' | 'combined'
  showUserFilter?: boolean
  showProjectFilter?: boolean
}

const presets = [
  {
    label: 'This Month',
    getRange: () => ({
      from: startOfMonth(new Date()),
      to: endOfMonth(new Date()),
    }),
  },
  {
    label: 'Last Month',
    getRange: () => ({
      from: startOfMonth(subMonths(new Date(), 1)),
      to: endOfMonth(subMonths(new Date(), 1)),
    }),
  },
  {
    label: 'This Quarter',
    getRange: () => ({
      from: startOfQuarter(new Date()),
      to: endOfQuarter(new Date()),
    }),
  },
  {
    label: 'This Year',
    getRange: () => ({
      from: startOfYear(new Date()),
      to: endOfYear(new Date()),
    }),
  },
]

export function ExportForm({
  type,
  showUserFilter,
  showProjectFilter,
}: ExportFormProps) {
  const [range, setRange] = useState<DateRange | undefined>({
    from: startOfMonth(new Date()),
    to: endOfMonth(new Date()),
  })
  const [exportFormat, setExportFormat] = useState<'csv' | 'xlsx'>('csv')
  const { download, isPending } = useDownload()

  const isValid =
    range?.from != null &&
    range?.to != null &&
    range.from < range.to &&
    range.to.getTime() - range.from.getTime() <= 365 * 24 * 60 * 60 * 1000

  const isRangeTooLong =
    range?.from != null && range?.to != null && range.to.getTime() - range.from.getTime() > 365 * 24 * 60 * 60 * 1000
  const isRangeReversed =
    range?.from != null && range?.to != null && range.from >= range.to

  function isPresetActive(preset: (typeof presets)[number]) {
    if (!range?.from || !range?.to) return false
    const p = preset.getRange()
    return (
      range.from.getTime() === p.from.getTime() &&
      range.to.getTime() === p.to.getTime()
    )
  }

  const handleDownload = useCallback(async () => {
    if (!range?.from || !range?.to) return
    if (!isValid) {
      if (isRangeTooLong) {
        toast.error('Date range must be within 1 year')
      } else if (isRangeReversed) {
        toast.error('Start date must be before end date')
      }
      return
    }

    const fromISO = format(range.from, 'yyyy-MM-dd')
    const toISO = format(range.to, 'yyyy-MM-dd')

    const countPromise = queryClient
      .fetchQuery(exportCountQueryOpts(type, fromISO, toISO))
      .then((result) => {
        if (result.count === 0) {
          toast('No data to export')
          return null
        }
        return getExportUrl(type, fromISO, toISO, exportFormat)
      })

    toast.promise(countPromise, {
      loading: 'Generating export...',
      success: (url) => {
        if (!url) return 'No data to export'
        download(url)
        return 'Download started'
      },
      error: 'Export failed. Please try again.',
    })
  }, [range, type, exportFormat, download, isValid, isRangeTooLong, isRangeReversed])

  return (
    <Card className="max-w-lg">
      <CardHeader>
        <CardTitle>Export Report</CardTitle>
        <CardDescription>
          {type === 'combined'
            ? 'Download CSV or Excel reports for timesheets, expenses, or combined data'
            : `Download ${type === 'timesheets' ? 'timesheet' : 'expense'} reports as CSV or Excel`}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Date range section */}
        <div className="space-y-2">
          <Label>Date range</Label>
          <div className="flex gap-1.5 overflow-x-auto pb-1">
            {presets.map((p) => (
              <Button
                key={p.label}
                variant={isPresetActive(p) ? 'secondary' : 'outline'}
                size="sm"
                onClick={() => setRange(p.getRange())}
              >
                {p.label}
              </Button>
            ))}
          </div>
          <Popover>
            <PopoverTrigger
              render={
                <Button variant="outline" className="w-full justify-start text-left font-normal">
                  {range?.from && range?.to ? (
                    <span>
                      {format(range.from, 'dd MMM yyyy')} - {format(range.to, 'dd MMM yyyy')}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">Select date range</span>
                  )}
                </Button>
              }
            />
            <PopoverContent>
              <DayPicker
                mode="range"
                selected={range}
                onSelect={setRange}
              />
            </PopoverContent>
          </Popover>
          {isRangeTooLong && (
            <p className="text-xs text-destructive">Date range must be within 1 year</p>
          )}
          {isRangeReversed && (
            <p className="text-xs text-destructive">Start date must be before end date</p>
          )}
        </div>

        {/* Format selector section */}
        <div className="space-y-2">
          <Label>Format</Label>
          <div className="flex rounded-lg border p-1">
            {(['csv', 'xlsx'] as const).map((f) => (
              <Button
                key={f}
                variant={exportFormat === f ? 'default' : 'ghost'}
                size="sm"
                className="flex-1"
                onClick={() => setExportFormat(f)}
              >
                {f.toUpperCase()}
              </Button>
            ))}
          </div>
        </div>

        {/* Download button */}
        <Button
          className="w-full"
          onClick={handleDownload}
          disabled={!isValid || isPending}
        >
          <Download className="mr-2 h-4 w-4" />
          {isPending ? 'Downloading...' : 'Download'}
        </Button>
      </CardContent>
    </Card>
  )
}
