import { useMemo } from "react";
import { format } from "date-fns";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useSuspenseQuery } from "@tanstack/react-query";
import { EyeIcon, PlusIcon } from "lucide-react";
import { type TimeEntry, type EntryStatus } from "@/types";
import { TimeEntriesApis } from "@/api/time-entries.ts";
import {
  EntriesTable,
  type EntriesColumn,
} from "@/components/shared/entries-table";
import { StatusBadge } from "@/components/shared/status-badge";
import {
  DateRangeFilter,
  StatusFilterSelect,
  type StatusFilterOption,
} from "@/components/shared/entries-filters";
import { Button } from "@/components/ui/button";

const STATUS_OPTIONS: StatusFilterOption[] = [
  { value: "draft", label: "Draft" },
  { value: "submitted", label: "Submitted" },
  { value: "pending_manager", label: "Pending Manager" },
  { value: "pending_finance", label: "Pending Finance" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
];

function truncate(text: string, max = 60) {
  if (!text || text.length <= max) return text;
  return `${text.slice(0, max - 1)}…`;
}

// API returns RFC3339 (2026-07-15T00:00:00Z); some fixtures/tests use yyyy-MM-dd.
function entryDateOf(entryDate: string): Date {
  return new Date(entryDate.includes("T") ? entryDate : `${entryDate}T00:00:00`);
}

export function TimeEntriesList({
  onOpenDetail,
  onNewEntry,
}: {
  onOpenDetail: (entry: TimeEntry) => void;
  onNewEntry: () => void;
}) {
  const navigate = useNavigate();
  const { month, listStatuses, listFrom, listTo } = useSearch({
    from: "/_authenticated/time-entries/",
  });
  const { data: entries } = useSuspenseQuery(
    TimeEntriesApis.timeEntriesForMonthQueryOpts(
      month.getMonth() + 1,
      month.getFullYear()
    )
  );

  const activityName = (entry: TimeEntry) => entry.activity_name || "—";

  const rows = useMemo(() => {
    let filtered = entries ?? [];
    if (listStatuses && listStatuses.length > 0) {
      filtered = filtered.filter((e) =>
        (listStatuses as EntryStatus[]).includes(e.status)
      );
    }
    if (listFrom) {
      filtered = filtered.filter((e) => e.entry_date.slice(0, 10) >= listFrom);
    }
    if (listTo) {
      filtered = filtered.filter((e) => e.entry_date.slice(0, 10) <= listTo);
    }
    return [...filtered].sort((a, b) =>
      b.entry_date.slice(0, 10).localeCompare(a.entry_date.slice(0, 10))
    );
  }, [entries, listStatuses, listFrom, listTo]);

  const filtersActive =
    (listStatuses && listStatuses.length > 0) || !!listFrom || !!listTo;

  const setFilters = (patch: {
    listStatuses?: EntryStatus[];
    listFrom?: string;
    listTo?: string;
  }) => {
    navigate({
      to: "/time-entries",
      search: (prev) => ({
        ...prev,
        listStatuses: patch.listStatuses ?? listStatuses,
        listFrom: patch.listFrom ?? listFrom,
        listTo: patch.listTo ?? listTo,
      }),
    });
  };

  const clearFilters = () =>
    navigate({
      to: "/time-entries",
      search: (prev) => ({
        ...prev,
        listStatuses: undefined,
        listFrom: undefined,
        listTo: undefined,
      }),
    });

  const columns: EntriesColumn<TimeEntry>[] = [
    {
      key: "date",
      header: "Date",
      cell: (e) => (
        <span className="whitespace-nowrap">
          {format(entryDateOf(e.entry_date), "dd MMM yyyy")}
        </span>
      ),
    },
    {
      key: "activity",
      header: "Activity",
      cell: (e) => activityName(e),
    },
    {
      key: "description",
      header: "Description",
      cell: (e) => (
        <span
          title={e.description || undefined}
          className="text-muted-foreground"
        >
          {truncate(e.description) || "—"}
        </span>
      ),
      cellClassName: "max-w-xs",
    },
    {
      key: "hours",
      header: "Hours",
      cell: (e) => <span className="whitespace-nowrap">{e.hours}h</span>,
    },
    {
      key: "status",
      header: "Status",
      cell: (e) => <StatusBadge status={e.status} />,
    },
    {
      key: "actions",
      header: "",
      cell: (e) => (
        <Button
          variant="ghost"
          size="sm"
          aria-label={`View entry ${format(entryDateOf(e.entry_date), "dd MMM")}`}
          onClick={(ev) => {
            ev.stopPropagation();
            onOpenDetail(e);
          }}
        >
          <EyeIcon className="h-4 w-4" />
        </Button>
      ),
      cellClassName: "text-right",
    },
  ];

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusFilterSelect
          options={STATUS_OPTIONS}
          selected={listStatuses ?? []}
          onChange={(v) => setFilters({ listStatuses: v as EntryStatus[] })}
        />
        <DateRangeFilter
          from={listFrom}
          to={listTo}
          onChange={(from, to) => setFilters({ listFrom: from, listTo: to })}
        />
        {filtersActive && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            Clear filters
          </Button>
        )}
      </div>

      <EntriesTable
        columns={columns}
        rows={rows}
        getRowKey={(e) => e.id}
        onRowClick={onOpenDetail}
        emptyState={
          <div className="flex flex-col items-center gap-2 text-center">
            <p className="text-sm text-muted-foreground">
              {filtersActive
                ? "No entries match the current filters."
                : "No time entries in this period."}
            </p>
            {!filtersActive && (
              <Button size="sm" onClick={onNewEntry}>
                <PlusIcon className="h-4 w-4 mr-1" />
                New entry
              </Button>
            )}
          </div>
        }
        ariaLabel="Time entries list"
      />
    </div>
  );
}
