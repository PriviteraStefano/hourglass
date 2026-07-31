import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  useNavigate,
  useRouteContext,
  useRouter,
  useSearch,
} from "@tanstack/react-router";
import { AlertTriangleIcon, InboxIcon } from "lucide-react";
import { toast } from "sonner";
import { ExpensesApis } from "@/api/expenses.ts";
import { TimeEntriesApis } from "@/api/time-entries.ts";
import { WorkingGroupsApis } from "@/api/working-groups.ts";
import { ApprovalButtons } from "@/components/approval/approval-buttons.tsx";
import { Body } from "@/components/layout/body.tsx";
import { Header } from "@/components/layout/header.tsx";
import { StatusBadge } from "@/components/shared/status-badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";
import { deriveApprovalStages } from "@/lib/role-visibility.ts";
import type { EntryStatus } from "@/types";
import type { Expense } from "@/types/expense-types.ts";

/**
 * Approvals queue (ADR-P-011 D-3 / UI-SPEC §Approvals).
 *
 * One page, stage-filtered tabs — Manager stage (WG manager/delegate queue,
 * submitted + pending_manager) and Finance stage (pending_finance). Rendered
 * only for users holding an approval stage (deriveApprovalStages); HR and
 * plain employees get a locked muted notice and never fire the pending
 * queries (no 403 spam — the backend stays authoritative, T-10-05-1).
 *
 * A query failure — including a 403 from the pending endpoints while tabs
 * render — shows the locked error state, never "Queue is clear" (T-10-05-2).
 */
type QueueRow = {
  kind: "time" | "expense";
  id: string;
  date: string;
  activityName: string;
  /** Numeric queue-row value: hours (time) or amount (expense), rendered .font-text. */
  value: string;
  status: EntryStatus;
  stage: "manager" | "finance";
  currentApproverRole?: "manager" | "finance" | null;
};

const STAGE_LABEL: Record<"manager" | "finance", string> = {
  manager: "Manager",
  finance: "Finance",
};

/** Row stage is derived from status — the two-stage chain per BE-014. */
function stageOf(status: EntryStatus): "manager" | "finance" {
  return status === "pending_finance" ? "finance" : "manager";
}

function dateLabel(dateStr: string): string {
  // entry_date arrives as RFC3339; the list-view convention normalizes on the
  // stored date part (yyyy-MM-dd) to stay timezone-stable.
  return dateStr.slice(0, 10);
}

export function ApprovalsPage() {
  const router = useRouter();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { profile } = useRouteContext({ from: "/_authenticated" });
  const search = useSearch({ from: "/_authenticated/approvals/" });
  const { data: workingGroups } = useQuery(WorkingGroupsApis.workingGroupsQueryOpts);

  const stages = deriveApprovalStages(profile, workingGroups);
  const isApprover = stages.length > 0;
  // URL-shareable stage (ADR-FE-017); single-stage users render their queue
  // directly, so the tab value defaults to their only stage.
  const activeStage: "manager" | "finance" | undefined =
    stages.length === 1
      ? stages[0]
      : (stages.includes(search.stage as "manager" | "finance")
          ? (search.stage as "manager" | "finance")
          : stages[0]);

  // Pending queries are gated on the approval stage — non-stage users never
  // touch /time-entries/pending or /expenses/pending (T-10-05-3 frontend
  // half; backend admission resolved in 10-05-01).
  const pendingTeQuery = useQuery({
    ...TimeEntriesApis.pendingTimeEntriesQueryOpts,
    enabled: isApprover,
  });
  const pendingExpensesQuery = useQuery({
    ...ExpensesApis.pendingExpensesQueryOpts,
    enabled: isApprover,
  });

  const rows: QueueRow[] = [
    ...(pendingTeQuery.data ?? []).map((e) => ({
      kind: "time" as const,
      id: e.id,
      date: e.entry_date,
      activityName: e.activity_name ?? "—",
      value: `${e.hours}`,
      status: e.status,
      stage: stageOf(e.status),
      currentApproverRole: e.current_approver_role,
    })),
    ...(pendingExpensesQuery.data ?? []).map((e: Expense) => ({
      kind: "expense" as const,
      id: e.id,
      date: e.entry_date,
      activityName: e.activity_name ?? "—",
      value: e.amount.toFixed(2),
      status: e.status,
      stage: stageOf(e.status),
      currentApproverRole: e.current_approver_role,
    })),
  ]
    .filter((r) => r.stage === activeStage)
    .sort((a, b) => a.date.localeCompare(b.date));

  const firstError =
    pendingTeQuery.error ?? pendingExpensesQuery.error ?? null;
  const loading =
    isApprover &&
    (pendingTeQuery.isLoading || pendingExpensesQuery.isLoading);

  const invalidateAfterAction = () => {
    // Pending queues + Today-relevant month keys stay fresh after an action.
    void queryClient.invalidateQueries({
      queryKey: ["time-entries", "pending"],
    });
    void queryClient.invalidateQueries({
      queryKey: ["expenses", "pending"],
    });
    void queryClient.invalidateQueries({ queryKey: ["time-entries"] });
    void queryClient.invalidateQueries({ queryKey: ["expenses"] });
  };

  const approveTimeEntry = useMutation({
    ...TimeEntriesApis.approveTimeEntryMutationOpts,
    onSuccess: () => {
      invalidateAfterAction();
      toast.success("Entry approved");
    },
  });
  const rejectTimeEntry = useMutation({
    ...TimeEntriesApis.rejectTimeEntryMutationOpts,
    onSuccess: () => {
      invalidateAfterAction();
      toast.success("Entry rejected");
    },
  });
  const approveExpense = useMutation({
    ...ExpensesApis.approveExpenseMutationOpts,
    onSuccess: () => {
      invalidateAfterAction();
      toast.success("Entry approved");
    },
  });
  const rejectExpense = useMutation({
    ...ExpensesApis.rejectExpenseMutationOpts,
    onSuccess: () => {
      invalidateAfterAction();
      toast.success("Entry rejected");
    },
  });

  const onApprove = (row: QueueRow) => {
    if (row.kind === "time") {
      approveTimeEntry.mutate(row.id);
    } else {
      approveExpense.mutate(row.id);
    }
  };
  const onReject = (row: QueueRow, reason: string) => {
    if (row.kind === "time") {
      rejectTimeEntry.mutate({ id: row.id, reason });
    } else {
      rejectExpense.mutate({ id: row.id, reason });
    }
  };
  const pendingAction =
    approveTimeEntry.isPending ||
    rejectTimeEntry.isPending ||
    approveExpense.isPending ||
    rejectExpense.isPending;

  return (
    <>
      <Header>
        <h1 className="text-xl font-semibold">Approvals</h1>
        {/* Tabs render only for dual-stage users; single-stage users skip the
            tab bar and see their queue directly (UI-SPEC §Approvals). */}
        {stages.length === 2 && activeStage && (
          <Tabs
            value={activeStage}
            onValueChange={(v) => {
              const next = v as "manager" | "finance";
              void navigate({
                to: "/approvals",
                search: { stage: next },
                replace: true,
              });
            }}
          >
            <TabsList>
              {stages.map((s) => (
                <TabsTrigger key={s} value={s}>
                  {STAGE_LABEL[s]}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        )}
      </Header>
      <Body>
        <div className="h-full overflow-y-auto p-6">
          {!isApprover ? (
            // Locked muted state for non-stage direct access (employee, HR,
            // customer): no pending queries fired, no 403 spam.
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <InboxIcon />
                  </EmptyMedia>
                  <EmptyTitle>Approvals are for manager and finance stages.</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>
                    Working-group managers and finance reviewers see submitted
                    entries here for review.
                  </EmptyDescription>
                </EmptyContent>
              </Empty>
            </div>
          ) : firstError ? (
            // 403 or any failure while tabs render → error state, NOT
            // "Queue is clear" (T-10-05-2).
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <AlertTriangleIcon />
                  </EmptyMedia>
                  <EmptyTitle>We couldn't load Approvals.</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>{firstError.message}.</EmptyDescription>
                  <Button
                    variant="outline"
                    onClick={() => {
                      void router.invalidate();
                    }}
                  >
                    Try again
                  </Button>
                </EmptyContent>
              </Empty>
            </div>
          ) : loading ? (
            <div className="flex flex-col gap-2">
              {[0, 1, 2].map((i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : rows.length === 0 ? (
            // Locked per-stage empty state.
            <div className="flex h-full items-center justify-center">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <InboxIcon />
                  </EmptyMedia>
                  <EmptyTitle>Queue is clear</EmptyTitle>
                </EmptyHeader>
                <EmptyContent>
                  <EmptyDescription>
                    There are no {activeStage} approvals waiting. Submitted
                    entries will appear here for review.
                  </EmptyDescription>
                </EmptyContent>
              </Empty>
            </div>
          ) : (
            <ul className="flex flex-col">
              {rows.map((row) => (
                <li
                  key={`${row.kind}-${row.id}`}
                  className="flex items-center gap-3 rounded-md px-3 py-3 odd:bg-muted/30"
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
                  <ApprovalButtons
                    status={row.status}
                    currentApproverRole={row.currentApproverRole}
                    userRole={profile.membership.role}
                    onApprove={() => onApprove(row)}
                    onReject={(reason) => onReject(row, reason)}
                    isPending={pendingAction}
                  />
                </li>
              ))}
            </ul>
          )}
        </div>
      </Body>
    </>
  );
}
