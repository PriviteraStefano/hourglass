export type Role = "employee" | "manager" | "finance" | "hr" | "customer";

export type EntryStatus =
  | "draft"
  | "submitted"
  | "pending_manager"
  | "pending_finance"
  | "approved"
  | "rejected";

export interface User {
  id: string;
  email: string;
  username: string;
  name: string;
  is_active: boolean;
  created_at: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  created_at: string;
}

export interface OrganizationMembership {
  id: string;
  user_id: string;
  organization_id: string;
  role: Role;
  is_active: boolean;
  invited_by?: string;
  invited_at?: string;
  activated_at?: string;
}

export interface UserWithMembership {
  user: User;
  membership: OrganizationMembership;
  organization: Organization;
}

export interface Contract {
  id: string;
  name: string;
  km_rate: number;
  currency: string;
  governance_model: "creator_controlled" | "unanimous" | "majority";
  is_shared: boolean;
  is_active: boolean;
  created_by_org_id: string;
  created_by_org_name?: string;
  adoption_count?: number;
  is_adopted?: boolean;
  customer_id?: string;
  customer_name?: string;
  time_entries_count?: number;
  created_at: string;
}

export interface Activity {
  id: string;
  org_id: string;
  parent_id?: string;
  name: string;
  description: string;
  kind: string;
  contract_id?: string;
  governance_model: "creator_controlled" | "unanimous" | "majority";
  created_by_org_id: string;
  is_shared: boolean;
  billable?: boolean | null;
  budget_amount?: number | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ActivityResponse extends Activity {
  parent_name?: string;
  contract_name: string;
  created_by_org_name: string;
  adoption_count: number;
  is_adopted: boolean;
}

export interface ActivityDetail {
  activity: Activity;
  ancestry: Activity[];
  commercial_context: { contract_id?: string; customer_id?: string } | null;
  billable: boolean | null;
}

export interface CreateActivityRequest {
  parent_id?: string;
  name: string;
  description: string;
  kind: string;
  contract_id?: string;
  governance_model: "creator_controlled" | "unanimous" | "majority";
  is_shared: boolean;
  billable?: boolean;
  budget_amount?: number;
}

export interface UpdateActivityRequest {
  parent_id?: string;
  name?: string;
  description?: string;
  kind?: string;
  contract_id?: string;
  governance_model?: "creator_controlled" | "unanimous" | "majority";
  is_shared?: boolean;
  billable?: boolean;
  budget_amount?: number;
  is_active?: boolean;
}

export interface TimeEntry {
  id: string;
  user_id: string;
  org_id: string;
  activity_id: string;
  activity_name?: string;
  activity_kind?: string;
  unit_id: string;
  hours: number;
  description: string;
  entry_date: string;
  status: EntryStatus;
  current_approver_role?: "manager" | "finance";
  submitted_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalRecord {
  id: string;
  entry_id: string;
  action: "approve" | "reject" | "submit";
  actor_user_id: string;
  actor_role: string;
  comment?: string;
  created_at: string;
}

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
