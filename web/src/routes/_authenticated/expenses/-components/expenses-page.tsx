import { ExpenseCalendar } from "@/routes/_authenticated/expenses/-components/expense-calendar.tsx";
import { ExpenseDetail } from "@/routes/_authenticated/expenses/-components/expense-detail.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ExportForm } from "@/components/exports/export-form";

export function ExpensesPage() {
  return (
    <Tabs defaultValue="list" className="p-2">
      <TabsList>
        <TabsTrigger value="list">List</TabsTrigger>
        <TabsTrigger value="calendar">Calendar</TabsTrigger>
        <TabsTrigger value="export">Export</TabsTrigger>
      </TabsList>
      <TabsContent value="list">
        {/* Placeholder for future list view */}
      </TabsContent>
      <TabsContent value="calendar">
        <div className="flex">
          <ExpenseCalendar />
          <ExpenseDetail />
        </div>
      </TabsContent>
      <TabsContent value="export">
        <ExportForm type="expenses" />
      </TabsContent>
    </Tabs>
  );
}
