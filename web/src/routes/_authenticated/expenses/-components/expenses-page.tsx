import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ExpenseCalendar } from "@/routes/_authenticated/expenses/-components/expense-calendar.tsx";
import { ExpenseDetail } from "@/routes/_authenticated/expenses/-components/expense-detail.tsx";
import { ExpensesList } from "@/routes/_authenticated/expenses/-components/expenses-list.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ExportForm } from "@/components/exports/export-form";
import { type Expense } from "@/types/expense-types";

export function ExpensesPage() {
  const navigate = useNavigate();
  const [tab, setTab] = useState("list");

  const openDetail = (expense: Expense) => {
    // entry_date arrives as RFC3339 from the API; new Date handles both that
    // and plain yyyy-MM-dd fixtures.
    navigate({
      to: "/expenses",
      search: (prev) => ({ ...prev, date: new Date(expense.entry_date) }),
    });
    setTab("calendar");
  };

  const openNewExpense = () => {
    navigate({
      to: "/expenses",
      search: (prev) => ({ ...prev, date: new Date() }),
    });
    setTab("calendar");
  };

  return (
    <Tabs value={tab} onValueChange={setTab} className="p-2">
      <TabsList>
        <TabsTrigger value="list">List</TabsTrigger>
        <TabsTrigger value="calendar">Calendar</TabsTrigger>
        <TabsTrigger value="export">Export</TabsTrigger>
      </TabsList>
      <TabsContent value="list">
        <ExpensesList onOpenDetail={openDetail} onNewExpense={openNewExpense} />
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
