import {MiniCalendar} from "@/routes/_authenticated/time-entries/-components/mini-calendar.tsx";
import {EntryDetail} from "@/routes/_authenticated/time-entries/-components/entry-detail.tsx";

export function TimeEntriesPage() {
  return (
    <div className="flex p-2">
      <MiniCalendar/>
      <EntryDetail/>
    </div>
  )
}