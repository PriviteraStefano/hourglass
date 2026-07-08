import {MiniCalendar} from "@/routes/_authenticated/time-entries/-components/mini-calendar.tsx";
import {EntryDetail} from "@/routes/_authenticated/time-entries/-components/entry-detail.tsx";
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs";
import {ExportForm} from "@/components/exports/export-form";

export function TimeEntriesPage() {
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
          <MiniCalendar/>
          <EntryDetail/>
        </div>
      </TabsContent>
      <TabsContent value="export">
        <ExportForm type="timesheets"/>
      </TabsContent>
    </Tabs>
  )
}