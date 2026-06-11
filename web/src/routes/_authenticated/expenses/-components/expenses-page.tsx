import {ExpenseCalendar} from "@/routes/_authenticated/expenses/-components/expense-calendar.tsx";
import {ExpenseDetail} from "@/routes/_authenticated/expenses/-components/expense-detail.tsx";

export function ExpensesPage() {
  return (
    <div className="flex p-2">
      <ExpenseCalendar/>
      <ExpenseDetail/>
    </div>
  )
}
