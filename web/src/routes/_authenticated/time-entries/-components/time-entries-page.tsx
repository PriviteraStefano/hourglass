import {useState} from "react";
import {MonthSummaryBar} from "@/src/routes/_authenticated/time-entries/-components/month-summary-bar.tsx";
import {MiniCalendar} from "@/src/routes/_authenticated/time-entries/-components/mini-calendar.tsx";
import {EntryDetail} from "@/src/routes/_authenticated/time-entries/-components/entry-detail.tsx";
import {useQuery} from "@tanstack/react-query";
import {TimeEntriesApis} from "@/src/api/time-entries.ts";

export function TimeEntriesPage() {
  const [selectedDate, setSelectedDate] = useState<Date>(new Date())
  const [month, setMonth] = useState<Date>(new Date())

  const {data: summary} = useQuery(TimeEntriesApis.timeEntriesMonthlySummaryQueryOpts(
    month.getMonth() + 1,
    month.getFullYear()
  ))

  const {data: entry} = useQuery(TimeEntriesApis.timeEntryQueryOpts(selectedDate))

  return (
    <div className="space-y-4">
      <MonthSummaryBar
        month={month}
        onMonthChange={setMonth}
        summary={summary?.data}
      />

      <div className="flex gap-6">
        <MiniCalendar
          month={month}
          selectedDate={selectedDate}
          onSelectDate={setSelectedDate}
          onMonthChange={setMonth}
          summary={summary?.data}
        />

        <EntryDetail date={selectedDate} entry={entry?.data}/>
      </div>
    </div>
  )
}