import {MonthSummaryBar} from "@/src/routes/_authenticated/time-entries/-components/month-summary-bar.tsx";
import {MiniCalendar} from "@/src/routes/_authenticated/time-entries/-components/mini-calendar.tsx";
import {EntryDetail} from "@/src/routes/_authenticated/time-entries/-components/entry-detail.tsx";

export function TimeEntriesPage() {
  return (
    <div className="space-y-4">
      <MonthSummaryBar/>
      <div className="flex gap-6">
        <MiniCalendar/>
        <EntryDetail/>
      </div>
    </div>
  )
}