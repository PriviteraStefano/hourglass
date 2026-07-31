import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { format } from "date-fns";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type {
  Expense,
  CreateExpenseRequest,
  UpdateExpenseRequest,
} from "@/types";

function expensesForMonthQueryKey(month: number, year: number) {
  return ["expenses", "month", month, year] as const;
}

function expenseQueryKey(date: Date) {
  return ["expenses", "date", format(date, "yyyy-MM-dd")] as const;
}

export function expensesForMonthQueryOpts(month: number, year: number) {
  return queryOptions({
    queryKey: expensesForMonthQueryKey(month, year),
    queryFn: () => api<Expense[]>(`/expenses?month=${month}&year=${year}`),
  });
}

function expenseQueryOpts(date: Date) {
  const formattedDate = format(date, "yyyy-MM-dd");
  return queryOptions({
    queryKey: expenseQueryKey(date),
    queryFn: () => api<Expense[]>(`/expenses?date=${formattedDate}`),
    enabled: !!date,
  });
}

const createExpenseMutationOpts = mutationOptions({
  mutationFn: (data: CreateExpenseRequest) =>
    api<Expense>("/expenses", { method: "POST", body: JSON.stringify(data) }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["expenses"] });
    toast.success("Expense created");
  },
});

const updateExpenseMutationOpts = mutationOptions({
  mutationFn: ({ id, ...data }: UpdateExpenseRequest & { id: string }) =>
    api<Expense>(`/expenses/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["expenses"] });
    toast.success("Expense updated");
  },
});

const deleteExpenseMutationOpts = mutationOptions({
  mutationFn: (id: string) => api(`/expenses/${id}`, { method: "DELETE" }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["expenses"] });
    toast.success("Expense deleted");
  },
});

const submitExpenseMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<Expense>(`/expenses/${id}/submit`, { method: "POST" }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["expenses"] });
    toast.success("Expense submitted for approval");
  },
});

const approveExpenseMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<Expense>(`/expenses/${id}/approve`, { method: "POST" }),
});

const rejectExpenseMutationOpts = mutationOptions({
  mutationFn: ({ id, reason }: { id: string; reason: string }) =>
    api<Expense>(`/expenses/${id}/reject`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    }),
});

const uploadReceiptMutationOpts = mutationOptions({
  mutationFn: ({ id, file }: { id: string; file: File }) => {
    const formData = new FormData();
    formData.append("file", file);
    return api<Expense>(`/expenses/${id}/receipt`, {
      method: "POST",
      body: formData,
      headers: {}, // Let browser set Content-Type for multipart
    });
  },
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["expenses"] });
    toast.success("Receipt uploaded");
  },
});

const pendingExpensesQueryOpts = queryOptions({
  queryKey: ["expenses", "pending"],
  queryFn: () => api<Expense[]>("/expenses/pending"),
});

export const ExpensesApis = {
  expensesForMonthQueryOpts,
  expenseQueryOpts,
  createExpenseMutationOpts,
  updateExpenseMutationOpts,
  deleteExpenseMutationOpts,
  submitExpenseMutationOpts,
  approveExpenseMutationOpts,
  rejectExpenseMutationOpts,
  uploadReceiptMutationOpts,
  pendingExpensesQueryOpts,
};
