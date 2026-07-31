import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { DateRangeFilter, StatusFilterSelect } from "../entries-filters";

function triggerText(): string {
  const buttons = screen.getAllByRole("button");
  return buttons.length ? (buttons[0].textContent ?? "") : "";
}

describe("DateRangeFilter", () => {
  it("renders a single-date label without crashing when only from is set", () => {
    render(<DateRangeFilter from="2026-05-18" to={undefined} onChange={vi.fn()} />);
    expect(triggerText()).toContain("18 May");
    expect(triggerText()).not.toContain("undefined");
  });

  it("renders a single-date label without crashing when only to is set", () => {
    render(<DateRangeFilter from={undefined} to="2026-05-20" onChange={vi.fn()} />);
    expect(triggerText()).toContain("20 May");
    expect(triggerText()).not.toContain("undefined");
  });

  it("renders a joined range label when both bounds are set", () => {
    render(
      <DateRangeFilter from="2026-05-18" to="2026-05-20" onChange={vi.fn()} />
    );
    expect(triggerText()).toContain("18 May");
    expect(triggerText()).toContain("20 May");
    expect(triggerText()).toContain("-");
  });

  it("renders the placeholder when no bounds are set", () => {
    render(<DateRangeFilter from={undefined} to={undefined} onChange={vi.fn()} />);
    expect(triggerText()).toContain("Date range");
  });
});

describe("StatusFilterSelect", () => {
  const options = [
    { value: "draft", label: "Draft" },
    { value: "approved", label: "Approved" },
  ];

  it("shows a count badge when selections exist", () => {
    render(
      <StatusFilterSelect
        options={options}
        selected={["draft"]}
        onChange={vi.fn()}
      />
    );
    expect(triggerText()).toContain("1");
  });
});
