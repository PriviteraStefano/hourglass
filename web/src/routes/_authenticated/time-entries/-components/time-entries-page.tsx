import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { format } from "date-fns";
import { MiniCalendar } from "@/routes/_authenticated/time-entries/-components/mini-calendar.tsx";
import { EntryDetail } from "@/routes/_authenticated/time-entries/-components/entry-detail.tsx";
import { TimeEntriesList } from "@/routes/_authenticated/time-entries/-components/time-entries-list.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ExportForm } from "@/components/exports/export-form";
import { type TimeEntry } from "@/types";

export function TimeEntriesPage() {
  const navigate = useNavigate();
  const [tab, setTab] = useState("list");

  const openDetail = (entry: TimeEntry) => {
    // entry_date arrives as RFC3339 from the API; new Date handles both that
    // and plain yyyy-MM-dd fixtures.
    navigate({
      to: "/time-entries",
      search: (prev) => ({ ...prev, date: new Date(entry.entry_date) }),
    });
    setTab("calendar");
  };

  const openNewEntry = () => {
    navigate({
      to: "/time-entries",
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
        <TimeEntriesList onOpenDetail={openDetail} onNewEntry={openNewEntry} />
      </TabsContent>
      <TabsContent value="calendar">
        <div className="flex">
          <MiniCalendar />
          <EntryDetail />
        </div>
      </TabsContent>
      <TabsContent value="export">
        <ExportForm type="timesheets" />
      </TabsContent>
    </Tabs>
  );
}
