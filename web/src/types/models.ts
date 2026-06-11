export type Role = 'employee' | 'manager' | 'finance' | 'customer'

export type EntryStatus = 'draft' | 'submitted' | 'pending_manager' | 'pending_finance' | 'approved' | 'rejected'

export interface User {
  id: string
  email: string
  username: string
  name: string
  is_active: boolean
  created_at: string
}

export interface Organization {
  id: string
  name: string
  slug: string
  created_at: string
}

export interface OrganizationMembership {
  id: string
  user_id: string
  organization_id: string
  role: Role
  is_active: boolean
  invited_by?: string
  invited_at?: string
  activated_at?: string
}

export interface UserWithMembership  {
  user: User
  membership: OrganizationMembership
  organization: Organization
}

export interface Contract {
  id: string
  name: string
  km_rate: number
  currency: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
  is_active: boolean
  created_by_org_id: string
  created_by_org_name?: string
  adoption_count?: number
  is_adopted?: boolean
  customer_id?: string
  customer_name?: string
  time_entries_count?: number
  created_at: string
}

export interface Project {
  id: string
  name: string
  type: 'billable' | 'internal'
  contract_id: string
  contract_name?: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
  is_active: boolean
  created_by_org_id: string
  created_by_org_name?: string
  adoption_count?: number
  is_adopted?: boolean
  created_at: string
}

export interface Subproject {
  id: string
  project_id: string
  name: string
  description?: string
  sequence_order: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface TimeEntry {
  id: string
  user_id: string
  org_id: string
  project_id: string
  subproject_id: string
  wg_id: string
  unit_id: string
  hours: number
  description: string
  entry_date: string
  status: EntryStatus
  current_approver_role?: 'manager' | 'finance'
  submitted_at?: string
  created_at: string
  updated_at: string
}

export interface ApprovalRecord {
  id: string
  entry_id: string
  action: 'approve' | 'reject' | 'submit'
  actor_user_id: string
  actor_role: string
  comment?: string
  created_at: string
}

export type ExpenseCategory = 'mileage' | 'meal' | 'accommodation' | 'parking' | 'travel_tickets' | 'tolls' | 'taxi' | 'equipment' | 'other'