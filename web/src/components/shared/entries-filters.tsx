import { useState } from "react";
import { format } from "date-fns";
import { DayPicker } from "react-day-picker";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { CalendarRangeIcon, ChevronDownIcon } from "lucide-react";
import { cn } from "@/lib/utils.ts";

export interface StatusFilterOption {
  value: string;
  label: string;
}

interface StatusFilterSelectProps {
  options: StatusFilterOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
}

/**
 * Multi-select status filter. URL-agnostic: the parent page owns the
 * search-param schema and passes value/onChange in.
 */
export function StatusFilterSelect({
  options,
  selected,
  onChange,
}: StatusFilterSelectProps) {
  const allSelected = selected.length === options.length;
  const noneSelected = selected.length === 0;

  const toggle = (value: string) => {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value));
    } else {
      onChange([...selected, value]);
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="sm" className="gap-1.5">
            <span>Status</span>
            {!noneSelected && (
              <span className="rounded bg-muted px-1.5 text-xs">
                {selected.length}
              </span>
            )}
            <ChevronDownIcon className="h-3.5 w-3.5 text-muted-foreground" />
          </Button>
        }
      />
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="flex items-center justify-between">
            <span>Status</span>
            <button
              type="button"
              className="text-xs text-primary hover:underline"
              onClick={() =>
                onChange(allSelected ? [] : options.map((o) => o.value))
              }
            >
              {allSelected ? "Clear" : "Select all"}
            </button>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          {options.map((option) => {
            const checked = selected.includes(option.value);
            return (
              <DropdownMenuCheckboxItem
                key={option.value}
                checked={checked}
                onCheckedChange={() => toggle(option.value)}
                className={cn(!checked && "pl-7")}
              >
                {option.label}
              </DropdownMenuCheckboxItem>
            );
          })}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface DateRangeFilterProps {
  /** ISO yyyy-MM-dd or undefined for "no lower bound". */
  from?: string;
  /** ISO yyyy-MM-dd or undefined for "no upper bound". */
  to?: string;
  onChange: (from: string | undefined, to: string | undefined) => void;
}

/** Date-range picker. URL-agnostic; parent owns the search-schema wiring. */
export function DateRangeFilter({ from, to, onChange }: DateRangeFilterProps) {
  const [open, setOpen] = useState(false);

  const range = {
    from: from ? new Date(`${from}T00:00:00`) : undefined,
    to: to ? new Date(`${to}T00:00:00`) : undefined,
  };

  const hasRange = !!from || !!to;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            className="gap-1.5 font-normal"
          >
            <CalendarRangeIcon className="h-3.5 w-3.5 text-muted-foreground" />
            {hasRange ? (
              <span>
                {from && format(new Date(`${from}T00:00:00`), "dd MMM")}
                {from && to && " - "}
                {to && format(new Date(`${to}T00:00:00`), "dd MMM")}
              </span>
            ) : (
              <span className="text-muted-foreground">Date range</span>
            )}
          </Button>
        }
      />
      <PopoverContent align="start" className="w-auto p-0">
        <DayPicker
          mode="range"
          selected={range}
          onSelect={(r) => {
            if (r?.from && r.to) {
              onChange(
                format(r.from, "yyyy-MM-dd"),
                format(r.to, "yyyy-MM-dd")
              );
            } else if (r?.from) {
              onChange(format(r.from, "yyyy-MM-dd"), undefined);
            }
          }}
          numberOfMonths={2}
        />
        {hasRange && (
          <div className="flex justify-end p-2 pt-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                onChange(undefined, undefined);
                setOpen(false);
              }}
            >
              Clear
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}
