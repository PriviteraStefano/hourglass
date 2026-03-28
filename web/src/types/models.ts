export type Role = 'employee' | 'manager' | 'finance' | 'customer'

export type EntryStatus = 'draft' | 'submitted' | 'pending_manager' | 'pending_finance' | 'approved' | 'rejected'

export interface User {
  id: string
  email: string
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

export interface UserWithMembership extends User {
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

export interface Contract {
  id: string
  name: string
  km_rate: number
  currency: string
  governance_model: 'creator_controlled' | 'unanimous' | 'majority'
  is_shared: boolean
  is_active: boolean
  created_at: string
}

export interface TimeEntryItem {
  id: string
  time_entry_id: string
  project_id: string
  project?: Project
  hours: number
  description?: string
}

export interface TimeEntry {
  id: string
  user_id: string
  organization_id: string
  date: string
  status: EntryStatus
  current_approver_role?: 'manager' | 'finance'
  submitted_at?: string
  created_at: string
  updated_at: string
  items: TimeEntryItem[]
}

export interface TimeEntryDaySummary {
  date: string
  has_draft: boolean
  has_submitted: boolean
  has_approved: boolean
  has_rejected: boolean
  total_hours: number
}

export interface TimeEntryMonthlySummary {
  days: TimeEntryDaySummary[]
  totals: {
    total_hours: number
    draft_count: number
    submitted_count: number
    approved_count: number
    rejected_count: number
  }
}