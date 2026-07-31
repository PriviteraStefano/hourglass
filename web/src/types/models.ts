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

// Mirrors the backend WorkingGroup payload (working_group.go). The frontend
// uses manager_id/delegate_ids to derive approval-stage visibility client-side
// (ADR-P-011 D-5, Plan 10-02); subproject_id is the legacy field name that
// anchors the group to an activity (D-5), kept as-is to match the API.
export interface WorkingGroup {
  id: string;
  org_id: string;
  subproject_id: string;
  name: string;
  description: string;
  unit_ids: string[];
  enforce_unit_tuple: boolean;
  manager_id: string;
  delegate_ids: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Mirrors the backend WorkingGroupMember payload (working_group.go). The API
// does NOT join user display fields — the client resolves names against
// `orgMembersQueryOpts` (GET /organizations/members) for pickers and badges.
export interface WorkingGroupMember {
  id: string;
  wg_id: string;
  user_id: string;
  unit_id: string;
  role: string;
  is_default_subproject: boolean;
  start_date: string;
  end_date?: string | null;
  created_at: string;
}

// HTTP request DTOs mirroring the handler payloads (working_group.go). The WG
// API keeps the legacy `subproject_id` field name that anchors the group to an
// activity (D-5) — the client sends the activity id in that field.
export interface CreateWorkingGroupRequest {
  org_id: string;
  subproject_id: string;
  name: string;
  description: string;
  unit_ids: string[];
  enforce_unit_tuple: boolean;
  manager_id: string;
  delegate_ids: string[];
}

export interface UpdateWorkingGroupRequest {
  name?: string;
  description?: string;
  unit_ids?: string[];
  enforce_unit_tuple?: boolean;
  manager_id?: string;
  delegate_ids?: string[];
}

export interface AddWorkingGroupMemberRequest {
  user_id: string;
  unit_id: string;
  role: string;
  is_default_subproject: boolean;
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
