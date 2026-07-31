import { useQuery } from "@tanstack/react-query";
import {
  Link,
  ToPathOption,
  useRouteContext,
  useRouter,
} from "@tanstack/react-router";
import { endOfWeek, format, parseISO, startOfWeek } from "date-fns";
import {
  AlertTriangleIcon,
  CheckCircle2Icon,
  SparklesIcon,
} from "lucide-react";
import { ExpensesApis } from "@/api/expenses.ts";
import { TimeEntriesApis } from "@/api/time-entries.ts";
import { WorkingGroupsApis } from "@/api/working-groups.ts";
import { Body } from "@/components/layout/body.tsx";
import { Header } from "@/components/layout/header.tsx";
import { StatusBadge } from "@/components/shared/status-badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { deriveApprovalStages } from "@/lib/role-visibility.ts";
import type { TimeEntry } from "@/types";

/**
 * Today landing page (ADR-P-011 D-2 / ADR-P-004).
 *
 * Read-only composition — no new state, no charts/KPIs, never blank. Sections
 * render top-down: "Waiting on you" (approvers only, gated on the WG-derived
 * approval stage) then "Your week" (own draft/submitted/rejected entries in
 * the current ISO week). Locked empty states replace the sections when there
 * is nothing to show; a query failure renders the locked error copy inside
 * Body with router.invalidate() recovery (Phase 8 convention).
 */
const WEEK_STATUSES = new Set(["draft", "submitted", "rejected"]);

type PendingRow = {
  id: string;
  date: string;
  activityName: string;
  value: string;
  status: TimeEntry["status"];
};

function dateLabel(dateStr: string): string {
  // Normalize on the stored date part (yyyy-MM-dd) — entry_date arrives as
  // RFC3339; slicing keeps the label stable across timezones and matches the
  // list-view convention (time-entries-list.tsx uses entry_date.slice(0, 10)).
  return format(parseISO(dateStr.slice(0, 10)), "EEE d MMM");
}

function isInCurrentWeek(dateStr: string, weekStart: string, weekEnd: string): boolean {
  const d = dateStr.slice(0, 10);
  return d >= weekStart && d <= weekEnd;
}

export function TodayPage() {
  const router = useRouter();
  // Profile is hydrated by the _authenticated layout's beforeLoad — same
  // source the sidebar and ProfileMenu render from (Plan 02 pattern).
  const { profile } = useRouteContext({ from: "/_authenticated" });
  // Non-suspense: an unloaded WG list simply means "no WG-derived stages yet".
  const { data: workingGroups } = useQuery(WorkingGroupsApis.workingGroupsQueryOpts);

  const stages = deriveApprovalStages(profile, workingGroups);
  const isApprover = stages.length > 0;

  // "Your week" reuses the month query and filters client-side to the current
  // ISO week (no new endpoint, RESEARCH §2.3). `now` is intentionally recomputed
  // per render — the week boundary window is tiny and correctness matters more
  // than memoizing a single Date.
  const now = new Date();
  const month = now.getMonth() + 1;
  const year = now.getFullYear();
  const weekStart = format(startOfWeek(now, { weekStartsOn: 1 }), "yyyy-MM-dd");
  const weekEnd = format(endOfWeek(now, { weekStartsOn: 1 }), "yyyy-MM-dd");

  const monthQuery = useQuery(TimeEntriesApis.timeEntriesForMonthQueryOpts(month, year));

  // Pending queries are gated on the approval stage: a plain employee never
  // touches /time-entries/pending or /expenses/pending (no 403 spam, threat
  // T-10-04-3). The backend remains authoritative — this is UX scoping only.
  const pendingTimeEntriesQuery = useQuery({
    ...TimeEntriesApis.pendingTimeEntriesQueryOpts,
    enabled: isApprover,
  });
  const pendingExpensesQuery = useQuery({
    ...ExpensesApis.pendingExpensesQueryOpts,
    enabled: isApprover,
  });

  const monthEntries = monthQuery.data ?? [];
  const weekEntries = monthEntries.filter(
    (e) =>
      WEEK_STATUSES.has(e.status) &&
      isInCurrentWeek(e.entry_date, weekStart, weekEnd),
  );
  const pendingTimeEntries = pendingTimeEntriesQuery.data ?? [];
  const pendingExpenses = pendingExpensesQuery.data ?? [];
  const pendingTotal = pendingTimeEntries.length + pendingExpenses.length;
  const hasAnyData =
    monthEntries.length > 0 || (isApprover && pendingTotal > 0);

  const dataReady =
    !monthQuery.isLoading &&
    (!isApprover || (!pendingTimeEntriesQuery.isLoading && !pendingExpensesQuery.isLoading));

  const firstError =
    monthQuery.error ??
    (isApprover ? (pendingTimeEntriesQuery.error ?? pendingExpensesQuery.error) : null);

  const ctaLabel = isApprover ? "Review now" : "Log time";
  // /approvals lands in Plan 10-05; typed as the loose ToPathOption (the
  // sidebar convention) so the link compiles against the pre-10-05 route tree.
  const ctaTo: ToPathOption = isApprover ? "/approvals" : "/time-entries";
  const reviewTo: ToPathOption = "/approvals";

  const pendingPreview: PendingRow[] = [
    ...pendingTimeEntries.slice(0, 5).map((e) => ({
      id: e.id,
      date: e.entry_date,
      activityName: e.activity_name ?? "—",
      value: `${e.hours}`,
      status: e.status,
    })),
    ...pendingExpenses.slice(0, 5).map((e) => ({
      id: e.id,
      date: e.entry_date,
      activityName: e.activity_name ?? "—",
      value: `${e.amount.toFixed(2)}`,
      status: e.status,
    })),
  ].slice(0, 5);

  return (
    <>
      <Header>
        <h1 className="text-3xl font-semibold">Today</h1>
        <div className="ml-auto">
          <Button render={<Link to={ctaTo}>{ctaLabel}</Link>} />
        </div>
      </Header>
      <Body>
        <div className="h-full overflow-y-auto p-6">
          {firstError && !dataReady ? (
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <AlertTriangleIcon />
                  </EmptyMedia>
                  <EmptyTitle>We couldn't load Today.</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>{firstError.message}.</EmptyDescription>
                  <Button
                    variant="outline"
                    onClick={() => {
                      // invalidate() re-runs loaders and rebuilds the route
                      // match — the Phase 8 recovery convention.
                      void router.invalidate();
                    }}
                  >
                    Try again
                  </Button>
                </EmptyContent>
              </Empty>
            </div>
          ) : !dataReady ? (
            <div className="flex flex-col gap-6">
              {isApprover && <PendingSectionSkeleton />}
              <WeekSectionSkeleton />
            </div>
          ) : !hasAnyData ? (
            // New user: no entries at all (month query empty + nothing
            // pending as proxy) — locked copy, ADR-P-004 "never blank".
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <SparklesIcon />
                  </EmptyMedia>
                  <EmptyTitle>Welcome to Hourglass</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>
                    Start by logging time against an activity. Your week and
                    anything waiting on you will appear here.
                  </EmptyDescription>
                  <Button render={<Link to="/time-entries">Log time</Link>} />
                </EmptyContent>
              </Empty>
            </div>
          ) : pendingTotal === 0 && weekEntries.length === 0 ? (
            // Caught up: nothing pending and no week entries, but the user
            // has data (month query non-empty) — locked copy.
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <CheckCircle2Icon />
                  </EmptyMedia>
                  <EmptyTitle>You're all caught up</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>
                    Nothing is waiting on you. When drafts, rejections, or
                    approvals land, they&apos;ll show up here.
                  </EmptyDescription>
                </EmptyContent>
              </Empty>
            </div>
          ) : (
            <div className="flex flex-col gap-6">
              {isApprover && pendingTotal > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-xl font-semibold">
                      Waiting on you
                    </CardTitle>
                    <CardDescription>
                      {pendingTotal} waiting for your review
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <ul className="flex flex-col gap-2">
                      {pendingPreview.map((row) => (
                        <li
                          key={row.id}
                          className="flex items-center gap-3 rounded-md bg-muted/40 px-3 py-2"
                        >
                          <span className="w-24 shrink-0 text-xs text-muted-foreground">
                            {dateLabel(row.date)}
                          </span>
                          <span className="min-w-0 flex-1 truncate text-sm">
                            {row.activityName}
                          </span>
                          <span className="font-text text-sm tabular-nums">
                            {row.value}
                          </span>
                          <StatusBadge status={row.status} />
                        </li>
                      ))}
                    </ul>
                    <div className="mt-4">
                      <Button
                        variant="link"
                        className="px-0"
                        render={<Link to={reviewTo}>Review now</Link>}
                      />
                    </div>
                  </CardContent>
                </Card>
              )}
              <Card>
                <CardHeader>
                  <CardTitle className="text-xl font-semibold">
                    Your week
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {weekEntries.length === 0 ? (
                    <p className="text-sm text-muted-foreground">
                      No time entries this week.
                    </p>
                  ) : (
                    <ul className="flex flex-col gap-2">
                      {weekEntries.map((entry) => (
                        <li
                          key={entry.id}
                          className="flex items-center gap-3 rounded-md bg-muted/40 px-3 py-2"
                        >
                          <span className="w-24 shrink-0 text-xs text-muted-foreground">
                            {dateLabel(entry.entry_date)}
                          </span>
                          <span className="min-w-0 flex-1 truncate text-sm">
                            {entry.activity_name ?? "—"}
                          </span>
                          <span className="font-text text-sm tabular-nums">
                            {entry.hours}
                          </span>
                          <StatusBadge status={entry.status} />
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </Body>
    </>
  );
}

function PendingSectionSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-5 w-32" />
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-8 w-full" />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function WeekSectionSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-5 w-24" />
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-8 w-full" />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
