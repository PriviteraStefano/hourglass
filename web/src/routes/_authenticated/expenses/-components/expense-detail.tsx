import { useState, useCallback } from "react";
import { format } from "date-fns";
import {
  type Expense,
  type ExpenseCategory,
  CATEGORY_LABELS,
} from "@/types/expense-types";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { StatusBadge } from "@/routes/_authenticated/time-entries/-components/status-badge.tsx";
import { ExpenseRow } from "./expense-row.tsx";
import { ApprovalButtons } from "@/components/approval/approval-buttons.tsx";
import { ApprovalHistory } from "@/components/approval/approval-history.tsx";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import {
  useMutation,
  useSuspenseQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { ExpensesApis } from "@/api/expenses.ts";
import { ActivitiesApis } from "@/api/activities.ts";
import { useSearch } from "@tanstack/react-router";
import { api } from "@/lib/api.ts";
import { toast } from "sonner";
import { PlusIcon } from "lucide-react";
import { Label } from "@/components/ui/label";

const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  "mileage",
  "meal",
  "accommodation",
  "parking",
  "travel_tickets",
  "tolls",
  "taxi",
  "equipment",
  "other",
];

export function ExpenseDetail() {
  const queryClient = useQueryClient();
  const { date } = useSearch({ from: "/_authenticated/expenses/" });
  const { data: expenses } = useSuspenseQuery(
    ExpensesApis.expenseQueryOpts(date)
  );
  const { data: activities } = useSuspenseQuery(
    ActivitiesApis.activitiesQueryOpts("all")
  );

  const createExpense = useMutation(ExpensesApis.createExpenseMutationOpts);
  const updateExpense = useMutation(ExpensesApis.updateExpenseMutationOpts);
  const deleteExpense = useMutation(ExpensesApis.deleteExpenseMutationOpts);
  const submitExpense = useMutation(ExpensesApis.submitExpenseMutationOpts);
  const uploadReceipt = useMutation(ExpensesApis.uploadReceiptMutationOpts);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newCategory, setNewCategory] = useState<ExpenseCategory>("mileage");
  const [newAmount, setNewAmount] = useState(0);
  const [newKmDistance, setNewKmDistance] = useState<number | undefined>(
    undefined
  );
  const [newDescription, setNewDescription] = useState("");
  const [newActivityId, setNewActivityId] = useState("");

  const hasExpenses = expenses && expenses.length > 0;
  const totalAmount = expenses?.reduce((sum, e) => sum + e.amount, 0) ?? 0;

  const isEditable = (expense: Expense) =>
    expense.status === "draft" || expense.status === "rejected";

  const handleCreate = () => {
    setShowCreateForm(true);
  };

  const handleCreateSubmit = () => {
    createExpense.mutate(
      {
        date: format(date, "yyyy-MM-dd"),
        activity_id: newActivityId,
        category: newCategory,
        amount: newAmount,
        km_distance: newCategory === "mileage" ? newKmDistance : undefined,
        description: newDescription || undefined,
      },
      {
        onSuccess: () => {
          setShowCreateForm(false);
          setNewCategory("mileage");
          setNewAmount(0);
          setNewKmDistance(undefined);
          setNewDescription("");
          setNewActivityId("");
        },
      }
    );
  };

  const handleUpdate = useCallback(
    (id: string, field: string, value: string | number | null) => {
      updateExpense.mutate({ id, [field]: value } as any);
    },
    [updateExpense]
  );

  const handleDelete = (id: string) => {
    deleteExpense.mutate(id);
    setDeleteDialogOpen(false);
  };

  const handleSubmit = (id: string) => {
    submitExpense.mutate(id);
  };

  const handleApprove = (id: string) => {
    api<Expense>(`/expenses/${id}/approve`, { method: "POST" })
      .then(() => {
        queryClient.invalidateQueries({ queryKey: ["expenses"] });
        toast.success("Expense approved");
      })
      .catch(() => {
        toast.error("Failed to approve expense");
      });
  };

  const handleReject = (id: string, reason: string) => {
    api<Expense>(`/expenses/${id}/reject`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    })
      .then(() => {
        queryClient.invalidateQueries({ queryKey: ["expenses"] });
        toast.success("Expense rejected");
      })
      .catch(() => {
        toast.error("Failed to reject expense");
      });
  };

  const handleReceiptUpload = (id: string, file: File) => {
    uploadReceipt.mutate({ id, file });
  };

  if (!hasExpenses && !showCreateForm) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center rounded-lg p-8 gap-3">
        <p className="text-sm text-muted-foreground">
          No expenses for {format(date, "MMMM d, yyyy")}
        </p>
        <p className="text-xs text-muted-foreground">
          Add an expense to track costs for this day.
        </p>
        <Button onClick={handleCreate} disabled={createExpense.isPending}>
          {createExpense.isPending ? "Creating..." : "Create Expense"}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex-1 rounded-lg p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">
            {format(date, "EEEE, MMMM d, yyyy")}
          </h2>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-sm text-muted-foreground">
              Total: ${totalAmount.toFixed(2)}
            </span>
            <span className="text-sm text-muted-foreground">
              ({expenses?.length ?? 0} expense
              {(expenses?.length ?? 0) !== 1 ? "s" : ""})
            </span>
          </div>
        </div>
        {hasExpenses && (
          <Button
            onClick={handleCreate}
            size="sm"
            disabled={createExpense.isPending}
          >
            <PlusIcon className="w-4 h-4 mr-1" />
            Add Expense
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {showCreateForm && (
          <div className="flex flex-col gap-3 p-3 border rounded bg-muted/20 mb-3">
            <div className="flex items-center gap-2">
              <div className="w-36">
                <Select
                  value={newActivityId}
                  onValueChange={(v) => v !== null && setNewActivityId(v)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Activity" />
                  </SelectTrigger>
                  <SelectContent>
                    {activities?.map((a: { id: string; name: string }) => (
                      <SelectItem key={a.id} value={a.id}>
                        {a.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Select
                value={newCategory}
                onValueChange={(v) => setNewCategory(v as ExpenseCategory)}
              >
                <SelectTrigger className="w-36">
                  <SelectValue placeholder="Category" />
                </SelectTrigger>
                <SelectContent>
                  {EXPENSE_CATEGORIES.map((cat) => (
                    <SelectItem key={cat} value={cat}>
                      {CATEGORY_LABELS[cat]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                type="number"
                step="0.01"
                min="0"
                value={newAmount || ""}
                onChange={(e) => setNewAmount(parseFloat(e.target.value) || 0)}
                placeholder="Amount"
                className="w-24"
              />
              {newCategory === "mileage" && (
                <Input
                  type="number"
                  step="0.1"
                  min="0"
                  value={newKmDistance ?? ""}
                  onChange={(e) =>
                    setNewKmDistance(
                      e.target.value ? parseFloat(e.target.value) : undefined
                    )
                  }
                  placeholder="Km"
                  className="w-20"
                />
              )}
              <Input
                value={newDescription}
                onChange={(e) => setNewDescription(e.target.value)}
                placeholder="Description"
                className="flex-1"
              />
            </div>
            <div className="flex gap-2 justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowCreateForm(false)}
              >
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleCreateSubmit}
                disabled={!newActivityId || createExpense.isPending}
              >
                {createExpense.isPending ? "Saving..." : "Save Draft"}
              </Button>
              <Button
                size="sm"
                onClick={handleCreateSubmit}
                disabled={!newActivityId || createExpense.isPending}
                variant="default"
              >
                Submit Expense
              </Button>
            </div>
          </div>
        )}

        {expenses?.map((expense) => (
          <div key={expense.id}>
            <ExpenseRow
              expense={expense}
              editable={isEditable(expense)}
              onUpdate={(field, value) =>
                handleUpdate(expense.id, field, value)
              }
              onDelete={() => handleDelete(expense.id)}
              onSubmit={() => handleSubmit(expense.id)}
              onReceiptUpload={(file) => handleReceiptUpload(expense.id, file)}
            />
            {(expense.status === "pending_manager" ||
              expense.status === "pending_finance") && (
              <div className="mt-2 pl-2 space-y-2">
                <ApprovalButtons
                  status={expense.status}
                  currentApproverRole={expense.current_approver_role}
                  userRole="employee"
                  onApprove={() => handleApprove(expense.id)}
                  onReject={(reason) => handleReject(expense.id, reason)}
                  isPending={false}
                />
              </div>
            )}
            {(expense.status === "approved" ||
              expense.status === "rejected") && (
              <div className="mt-2 pl-2">
                <ApprovalHistory approvals={[]} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
