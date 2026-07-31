import { useState } from "react";
import type { ReactNode } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { cn } from "@/lib/utils.ts";

export interface EntriesColumn<T> {
  /** Stable key for the column. */
  key: string;
  /** Header content. */
  header: ReactNode;
  /** Cell renderer for a row. */
  cell: (row: T) => ReactNode;
  /** Applied to the <th> element. */
  className?: string;
  /** Applied to the <td> element. */
  cellClassName?: string;
}

export interface EntriesTableProps<T> {
  columns: EntriesColumn<T>[];
  rows: T[];
  getRowKey: (row: T) => string;
  /** Called when a row body is clicked (not the header). */
  onRowClick?: (row: T) => void;
  /** Rendered when rows is empty (e.g. friendly empty state with CTA). */
  emptyState: ReactNode;
  /** Client-side page size (ADR-FE-018 conventions; default 25). */
  pageSize?: number;
  ariaLabel?: string;
}

/**
 * Generic, type-safe table shell for the time-entries / expenses list views.
 * Column defs keep the two lists visually identical while the caller owns the
 * row type and any URL-param wiring (filters live in the page components).
 */
export function EntriesTable<T>({
  columns,
  rows,
  getRowKey,
  onRowClick,
  emptyState,
  pageSize = 25,
  ariaLabel,
}: EntriesTableProps<T>) {
  const [page, setPage] = useState(0);

  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  // Clamp so filter changes that shrink the list never leave an empty page.
  const currentPage = Math.min(page, pageCount - 1);
  const pageRows = rows.slice(
    currentPage * pageSize,
    currentPage * pageSize + pageSize
  );

  if (rows.length === 0) {
    return <div className="py-8">{emptyState}</div>;
  }

  return (
    <div className="space-y-2">
      <div className="rounded-lg border overflow-x-auto">
        <Table aria-label={ariaLabel}>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {columns.map((col) => (
                <TableHead key={col.key} className={col.className}>
                  {col.header}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {pageRows.map((row) => (
              <TableRow
                key={getRowKey(row)}
                className={cn(onRowClick && "cursor-pointer")}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {columns.map((col) => (
                  <TableCell key={col.key} className={col.cellClassName}>
                    {col.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {pageCount > 1 && (
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>
            Page {currentPage + 1} of {pageCount} · {rows.length} entries
          </span>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              aria-label="Previous page"
            >
              <ChevronLeftIcon className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={currentPage >= pageCount - 1}
              onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
              aria-label="Next page"
            >
              <ChevronRightIcon className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
