import type { EntryStatus } from "@/types/models.ts";

export type ExpenseCategory =
  | "mileage"
  | "meal"
  | "accommodation"
  | "parking"
  | "travel_tickets"
  | "tolls"
  | "taxi"
  | "equipment"
  | "other";

export interface Expense {
  id: string;
  org_id: string;
  user_id: string;
  activity_id: string;
  activity_name?: string;
  activity_kind?: string;
  category: ExpenseCategory;
  amount: number;
  km_distance?: number;
  description: string;
  entry_date: string;
  status: EntryStatus;
  current_approver_role?: "manager" | "finance";
  submitted_at?: string;
  receipt_url?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateExpenseRequest {
  activity_id: string;
  category: ExpenseCategory;
  amount: number;
  km_distance?: number;
  description?: string;
  date: string;
}

export interface UpdateExpenseRequest {
  activity_id?: string;
  category?: ExpenseCategory;
  amount?: number;
  km_distance?: number;
  description?: string;
  date?: string;
}

export interface ExpenseApproval {
  id: string;
  expense_id: string;
  action: "approve" | "reject" | "submit";
  actor_user_id: string;
  actor_role: string;
  comment?: string;
  created_at: string;
}

export const CATEGORY_LABELS: Record<ExpenseCategory, string> = {
  mileage: "Mileage",
  meal: "Meal",
  accommodation: "Accommodation",
  parking: "Parking",
  travel_tickets: "Travel Tickets",
  tolls: "Tolls",
  taxi: "Taxi",
  equipment: "Equipment",
  other: "Other",
};
