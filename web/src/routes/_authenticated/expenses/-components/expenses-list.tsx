import { useMemo } from "react";
import { format } from "date-fns";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useSuspenseQuery } from "@tanstack/react-query";
import { EyeIcon, PaperclipIcon, PlusIcon } from "lucide-react";
import {
  type Expense,
  type ExpenseCategory,
  CATEGORY_LABELS,
} from "@/types/expense-types";
import { type EntryStatus } from "@/types";
import { ExpensesApis } from "@/api/expenses.ts";
import { ActivitiesApis } from "@/api/activities.ts";
import { ContractsApis } from "@/api/contracts.ts";
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
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { Button } from "@/components/ui/button";

const STATUS_OPTIONS: StatusFilterOption[] = [
  { value: "draft", label: "Draft" },
  { value: "submitted", label: "Submitted" },
  { value: "pending_manager", label: "Pending Manager" },
  { value: "pending_finance", label: "Pending Finance" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
];

const CATEGORY_OPTIONS = Object.keys(CATEGORY_LABELS) as ExpenseCategory[];

function truncate(text: string, max = 60) {
  if (!text || text.length <= max) return text;
  return `${text.slice(0, max - 1)}…`;
}

// API returns RFC3339 (2026-07-15T00:00:00Z); some fixtures/tests use yyyy-MM-dd.
function expenseDateOf(entryDate: string): Date {
  return new Date(entryDate.includes("T") ? entryDate : `${entryDate}T00:00:00`);
}

const numberFormatter = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

export function ExpensesList({
  onOpenDetail,
  onNewExpense,
}: {
  onOpenDetail: (expense: Expense) => void;
  onNewExpense: () => void;
}) {
  const navigate = useNavigate();
  const { month, listStatuses, listCategory, listFrom, listTo } = useSearch({
    from: "/_authenticated/expenses/",
  });
  const { data: expenses } = useSuspenseQuery(
    ExpensesApis.expensesForMonthQueryOpts(
      month.getMonth() + 1,
      month.getFullYear()
    )
  );
  const { data: activities } = useSuspenseQuery(
    ActivitiesApis.activitiesQueryOpts("all")
  );
  const { data: contracts } = useSuspenseQuery(
    ContractsApis.contractsQueryOpts("all")
  );

  // Expenses carry no currency field; resolve it through the existing
  // activity → contract relationship (both endpoints already exist).
  const activityMeta = useMemo(() => {
    const currency = new Map<string, string>();
    const contractCurrency = new Map<string, string>();
    for (const c of contracts ?? []) contractCurrency.set(c.id, c.currency);
    for (const a of activities ?? []) {
      const cc = a.contract_id ? contractCurrency.get(a.contract_id) : undefined;
      if (cc) currency.set(a.id, cc);
    }
    return { currency };
  }, [activities, contracts]);

  const rows = useMemo(() => {
    let filtered = expenses ?? [];
    if (listStatuses && listStatuses.length > 0) {
      filtered = filtered.filter((e) =>
        (listStatuses as EntryStatus[]).includes(e.status)
      );
    }
    if (listCategory) {
      filtered = filtered.filter((e) => e.category === listCategory);
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
  }, [expenses, listStatuses, listCategory, listFrom, listTo]);

  const filtersActive =
    (listStatuses && listStatuses.length > 0) ||
    !!listCategory ||
    !!listFrom ||
    !!listTo;

  const setFilters = (patch: {
    listStatuses?: EntryStatus[];
    listCategory?: string;
    listFrom?: string;
    listTo?: string;
  }) => {
    navigate({
      to: "/expenses",
      search: (prev) => ({
        ...prev,
        listStatuses: patch.listStatuses ?? listStatuses,
        listCategory: patch.listCategory ?? listCategory,
        listFrom: patch.listFrom ?? listFrom,
        listTo: patch.listTo ?? listTo,
      }),
    });
  };

  const clearFilters = () =>
    navigate({
      to: "/expenses",
      search: (prev) => ({
        ...prev,
        listStatuses: undefined,
        listCategory: undefined,
        listFrom: undefined,
        listTo: undefined,
      }),
    });

  const columns: EntriesColumn<Expense>[] = [
    {
      key: "date",
      header: "Date",
      cell: (e) => (
        <span className="whitespace-nowrap">
          {format(expenseDateOf(e.entry_date), "dd MMM yyyy")}
        </span>
      ),
    },
    {
      key: "activity",
      header: "Activity",
      cell: (e) => e.activity_name || "—",
    },
    {
      key: "category",
      header: "Category",
      cell: (e) => (
        <span className="capitalize">
          {CATEGORY_LABELS[e.category] ?? e.category}
        </span>
      ),
    },
    {
      key: "description",
      header: "Description",
      cell: (e) => {
        const km =
          e.category === "mileage" && e.km_distance != null
            ? `${numberFormatter.format(e.km_distance)} km`
            : null;
        const label = e.description
          ? truncate(e.description)
          : km ?? "—";
        return (
          <span
            title={e.description || km || undefined}
            className="text-muted-foreground"
          >
            {label}
            {e.description && km ? (
              <span className="whitespace-nowrap"> · {km}</span>
            ) : null}
          </span>
        );
      },
      cellClassName: "max-w-xs",
    },
    {
      key: "amount",
      header: "Amount",
      cell: (e) => {
        if (e.amount == null) return "—";
        const currency = activityMeta.currency.get(e.activity_id);
        const formatted = currency
          ? new Intl.NumberFormat(undefined, {
              style: "currency",
              currency,
            }).format(e.amount)
          : numberFormatter.format(e.amount);
        return <span className="whitespace-nowrap">{formatted}</span>;
      },
    },
    {
      key: "status",
      header: "Status",
      cell: (e) => <StatusBadge status={e.status} />,
    },
    {
      key: "receipt",
      header: "",
      cell: (e) =>
        e.receipt_url ? (
          <span title="Receipt attached">
            <PaperclipIcon className="h-3.5 w-3.5 text-muted-foreground" />
          </span>
        ) : (
          <span className="inline-block w-3.5" />
        ),
    },
    {
      key: "actions",
      header: "",
      cell: (e) => (
        <Button
          variant="ghost"
          size="sm"
          aria-label={`View expense ${format(expenseDateOf(e.entry_date), "dd MMM")}`}
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
        <NativeSelect
          aria-label="Category filter"
          value={listCategory ?? ""}
          onChange={(e) =>
            setFilters({
              listCategory: e.target.value || undefined,
            })
          }
          className="w-44"
        >
          <NativeSelectOption value="">All categories</NativeSelectOption>
          {CATEGORY_OPTIONS.map((c) => (
            <NativeSelectOption key={c} value={c}>
              {CATEGORY_LABELS[c]}
            </NativeSelectOption>
          ))}
        </NativeSelect>
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
                ? "No expenses match the current filters."
                : "No expenses in this period."}
            </p>
            {!filtersActive && (
              <Button size="sm" onClick={onNewExpense}>
                <PlusIcon className="h-4 w-4 mr-1" />
                New expense
              </Button>
            )}
          </div>
        }
        ariaLabel="Expenses list"
      />
    </div>
  );
}
