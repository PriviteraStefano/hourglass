import { FileDown } from "lucide-react";
import { ExportForm } from "@/components/exports/export-form";
import { Header, Body } from "@/components/layout";

export function ExportsPage() {
  return (
    <>
      <Header>
        <h1 className="text-xl font-semibold">Exports</h1>
      </Header>
      <Body>
        <div className="h-full overflow-y-auto p-6 space-y-6">
          <div className="flex items-center gap-3">
            <FileDown className="h-6 w-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Download CSV or Excel reports for timesheets, expenses, or
              combined data
            </p>
          </div>
          <ExportForm
            type="combined"
            showUserFilter={true}
            showProjectFilter={true}
          />
        </div>
      </Body>
    </>
  );
}
