import type {
  Organization,
  OrganizationMembership,
  User,
} from "@/types/models.ts";

export interface ApiResponse<T> {
  data: T;
}

export interface AuthResponse {
  user: User;
  membership: OrganizationMembership;
  organization: Organization;
  token: string;
  refresh_token: string;
  expires_at: string;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name?: string;
  organization_name?: string;
  invite_code?: string;
  username?: string;
  firstname?: string;
  lastname?: string;
}

export interface BootstrapRequest {
  org_name: string;
  email: string;
  username: string;
  firstname: string;
  lastname: string;
  password: string;
}

export interface InvitationResponse {
  id: string;
  code: string;
  token: string;
  link: string;
  email?: string;
  status: string;
  expires_at: string;
  organization_id: string;
}

export interface PasswordResetRequest {
  identifier: string;
}

export interface PasswordResetVerify {
  identifier: string;
  code: string;
  password: string;
}

export interface CreateTimeEntryRequest {
  activity_id: string;
  unit_id: string;
  hours: number;
  description?: string;
  date: string;
}

export interface UpdateTimeEntryRequest {
  activity_id?: string;
  unit_id?: string;
  hours?: number;
  description?: string;
  date?: string;
}

export interface ApiError {
  message: string;
  error?: string;
}

export interface CreateContractRequest {
  name: string;
  km_rate: number;
  currency: string;
  governance_model: "creator_controlled" | "unanimous" | "majority";
  is_shared: boolean;
  customer_id?: string;
}
