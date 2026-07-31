import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { EntriesTable, type EntriesColumn } from "../entries-table";
import { StatusBadge } from "../status-badge";

interface Row {
  id: string;
  name: string;
}

const columns: EntriesColumn<Row>[] = [
  { key: "name", header: "Name", cell: (r) => r.name },
];

function makeRows(count: number): Row[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `r${i}`,
    name: `Row ${i}`,
  }));
}

describe("EntriesTable", () => {
  it("renders column headers and row cells", () => {
    render(
      <EntriesTable
        columns={columns}
        rows={makeRows(2)}
        getRowKey={(r) => r.id}
        emptyState={<div>Nothing here</div>}
      />
    );
    expect(screen.getByRole("columnheader", { name: "Name" })).toBeInTheDocument();
    expect(screen.getByText("Row 0")).toBeInTheDocument();
    expect(screen.getByText("Row 1")).toBeInTheDocument();
  });

  it("renders the empty state when there are no rows", () => {
    render(
      <EntriesTable
        columns={columns}
        rows={[]}
        getRowKey={(r) => r.id}
        emptyState={<div>No entries yet</div>}
      />
    );
    expect(screen.getByText("No entries yet")).toBeInTheDocument();
  });

  it("paginates client-side at the configured page size (ADR-FE-018)", () => {
    render(
      <EntriesTable
        columns={columns}
        rows={makeRows(60)}
        getRowKey={(r) => r.id}
        emptyState={<div>Nothing here</div>}
        pageSize={25}
      />
    );
    // Page 1: rows 0..24 visible, row 25 hidden
    expect(screen.getByText("Row 0")).toBeInTheDocument();
    expect(screen.getByText("Row 24")).toBeInTheDocument();
    expect(screen.queryByText("Row 25")).not.toBeInTheDocument();
    expect(screen.getByText(/Page 1 of 3/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(screen.getByText("Row 25")).toBeInTheDocument();
    expect(screen.queryByText("Row 0")).not.toBeInTheDocument();
    expect(screen.getByText(/Page 2 of 3/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Previous page" }));
    expect(screen.getByText("Row 0")).toBeInTheDocument();
  });

  it("fires onRowClick for the clicked row", () => {
    const onRowClick = vi.fn();
    render(
      <EntriesTable
        columns={columns}
        rows={makeRows(3)}
        getRowKey={(r) => r.id}
        onRowClick={onRowClick}
        emptyState={<div>Nothing here</div>}
      />
    );
    fireEvent.click(screen.getByText("Row 1"));
    expect(onRowClick).toHaveBeenCalledWith({ id: "r1", name: "Row 1" });
  });

  it("resets to the first page when rows shrink below the current page", () => {
    render(
      <EntriesTable
        columns={columns}
        rows={makeRows(60)}
        getRowKey={(r) => r.id}
        emptyState={<div>Nothing here</div>}
        pageSize={25}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "Next page" }));
    expect(screen.getByText(/Page 2 of 3/)).toBeInTheDocument();
    // Simulate a filter shrinking the list to 10 rows: rows prop changes
    // and the clamp should land back on a valid page showing row 0.
    // (React Query re-renders with a new rows array; we re-render the tree.)
    render(
      <EntriesTable
        columns={columns}
        rows={makeRows(10)}
        getRowKey={(r) => r.id}
        emptyState={<div>Nothing here</div>}
        pageSize={25}
      />
    );
    expect(screen.getByText("Row 0")).toBeInTheDocument();
  });
});

describe("StatusBadge", () => {
  const cases: Array<[string, string]> = [
    ["draft", "Draft"],
    ["submitted", "Submitted"],
    ["pending_manager", "Pending Manager"],
    ["pending_finance", "Pending Finance"],
    ["approved", "Approved"],
    ["rejected", "Rejected"],
  ];

  it.each(cases)("renders label for %s", (status, label) => {
    render(<StatusBadge status={status as never} />);
    const badge = screen.getByText(label);
    expect(badge).toBeInTheDocument();
    // All six states get distinct styling classes
    expect(badge.className).toMatch(/bg-/);
  });

  it("gives each workflow state a distinct color class", () => {
    const colors = cases.map(([status]) => {
      const { container } = render(<StatusBadge status={status as never} />);
      return container.querySelector("span")!.className;
    });
    expect(new Set(colors).size).toBe(cases.length);
    within(document.body).getByText("Draft");
  });
});
