import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { getExportUrl } from '@/api/exports'
import { Download, FileDown } from 'lucide-react'

const EXPORT_TYPES = [
  { value: 'timesheets' as const, label: 'Timesheets', description: 'Time entries with hours per project' },
  { value: 'expenses' as const, label: 'Expenses', description: 'Expense entries with amounts and categories' },
  { value: 'combined' as const, label: 'Combined', description: 'Time entries and expenses in a single report' },
]

function today(): string {
  const d = new Date()
  return d.toISOString().slice(0, 10)
}

function monthStart(): string {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toISOString().slice(0, 10)
}

export function ExportsPage() {
  const [exportType, setExportType] = useState<string>('timesheets')
  const [from, setFrom] = useState(monthStart)
  const [to, setTo] = useState(today)
  const [downloading, setDownloading] = useState(false)

  const selectedType = EXPORT_TYPES.find(t => t.value === exportType)

  function handleDownload() {
    setDownloading(true)
    const url = getExportUrl(exportType as 'timesheets' | 'expenses' | 'combined', from, to)
    const a = document.createElement('a')
    a.href = url
    a.download = ''
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    setTimeout(() => setDownloading(false), 1000)
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <FileDown className="h-6 w-6 text-muted-foreground" />
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Exports</h1>
          <p className="text-sm text-muted-foreground">
            Download CSV reports for timesheets, expenses, or combined data
          </p>
        </div>
      </div>

      <Card className="max-w-lg">
        <CardHeader>
          <CardTitle>Export Report</CardTitle>
          <CardDescription>
            Select a report type and date range to download as CSV
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="export-type">Report type</Label>
            <Select value={exportType} onValueChange={(v) => v && setExportType(v)}>
              <SelectTrigger id="export-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {EXPORT_TYPES.map(t => (
                  <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {selectedType && (
              <p className="text-xs text-muted-foreground">{selectedType.description}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="from">From</Label>
              <Input
                id="from"
                type="date"
                value={from}
                onChange={e => setFrom(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="to">To</Label>
              <Input
                id="to"
                type="date"
                value={to}
                onChange={e => setTo(e.target.value)}
              />
            </div>
          </div>

          <Button
            className="w-full"
            onClick={handleDownload}
            disabled={!from || !to || downloading}
          >
            <Download className="mr-2 h-4 w-4" />
            {downloading ? 'Downloading...' : 'Download CSV'}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
