import { FileDown } from 'lucide-react'
import { ExportForm } from '@/components/exports/export-form'

export function ExportsPage() {
  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <FileDown className="h-6 w-6 text-muted-foreground" />
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Exports</h1>
          <p className="text-sm text-muted-foreground">
            Download CSV or Excel reports for timesheets, expenses, or combined data
          </p>
        </div>
      </div>
      <ExportForm
        type="combined"
        showUserFilter={true}
        showProjectFilter={true}
      />
    </div>
  )
}
