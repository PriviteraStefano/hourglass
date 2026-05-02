import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {api} from "@/lib/api.ts";
import type {AuthResponse, BootstrapRequest, InvitationResponse, LoginRequest, PasswordResetRequest, PasswordResetVerify, RegisterRequest, UserWithMembership} from "@/types";

const profileQueryOpts = queryOptions({
  queryKey: ['auth', 'me'],
  queryFn: async () => api<UserWithMembership>('/auth/me'),
  retry: false,
  staleTime: 5 * 60 * 1000,
})

const loginMutationOpts = mutationOptions({
  mutationFn: (creds: LoginRequest) =>
    api<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(creds),
    }),
  onSuccess: (data: AuthResponse, _, __, {client}) => {
    if (data?.user) {
      client.setQueryData(['auth', 'me'], data.user)
    }
  },
})

const registerMutationOpts = mutationOptions({
  mutationFn: (data: RegisterRequest) =>
    api<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  onSuccess: (result: AuthResponse, _, __, {client}) =>
    client.setQueryData(['auth', 'me'], result.user),
})

const bootstrapMutationOpts = mutationOptions({
  mutationFn: (data: BootstrapRequest) =>
    api<AuthResponse>('/auth/bootstrap', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  onSuccess: (result: AuthResponse, _, __, {client}) =>
    client.setQueryData(['auth', 'me'], result.user),
})

const logoutMutationOpts = mutationOptions({
  mutationFn: () =>
    api<{ message: string }>('/auth/logout', {
      method: 'POST',
    }),
  onSuccess: (_, __, ___, {client}) =>
    client.clear(),
})

const refreshMutationOpts = mutationOptions({
  mutationFn: () =>
    api<AuthResponse>('/auth/refresh', {
      method: 'POST',
    }),
  onSuccess: (data: AuthResponse, _, __, {client}) => {
    if (data) {
      client.setQueryData(['auth', 'me'], data.user)
    }
  },
})

const createInvitationMutationOpts = mutationOptions({
  mutationFn: (data: { organization_id: string; email?: string }) =>
    api<InvitationResponse>('/invitations', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})

const validateInvitationCodeQueryOpts = (code: string) => queryOptions({
  queryKey: ['invitations', 'code', code],
  queryFn: async () => api<InvitationResponse>(`/invitations/validate/code/${code}`),
  enabled: !!code,
})

const validateInvitationTokenQueryOpts = (token: string) => queryOptions({
  queryKey: ['invitations', 'token', token],
  queryFn: async () => api<InvitationResponse>(`/invitations/validate/token/${token}`),
  enabled: !!token,
})

const acceptInvitationMutationOpts = mutationOptions({
  mutationFn: (data: { token: string; email: string; username: string; password: string }) =>
    api<{ message: string }>('/invitations/accept', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})

const requestPasswordResetMutationOpts = mutationOptions({
  mutationFn: (data: PasswordResetRequest) =>
    api<{ message: string; code?: string }>('/auth/password-reset/request', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})

const verifyPasswordResetMutationOpts = mutationOptions({
  mutationFn: (data: PasswordResetVerify) =>
    api<{ message: string }>('/auth/password-reset/verify', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
})

const bootstrapCheckQueryOpts = queryOptions({
  queryKey: ['auth', 'bootstrap-check'],
  queryFn: async () => api<{ needs_bootstrap: boolean }>('/auth/bootstrap-check'),
  retry: false,
})

const switchOrganizationMutationOpts = mutationOptions({
  mutationFn: (data: { organization_id: string }) =>
    api<AuthResponse>('/auth/switch-organization', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  onSuccess: (data: AuthResponse, _, __, {client}) =>
    client.setQueryData(['auth', 'me'], data.user),
})

const membershipsQueryOpts = queryOptions({
  queryKey: ['auth', 'memberships'],
  queryFn: async () => api<{ memberships: Array<{ membership: UserWithMembership['membership']; organization: UserWithMembership['organization'] }> }>('/auth/memberships'),
  retry: false,
})

export const AuthApis = {
  profileQueryOpts,
  loginMutationOpts,
  registerMutationOpts,
  bootstrapMutationOpts,
  logoutMutationOpts,
  refreshMutationOpts,
  createInvitationMutationOpts,
  validateInvitationCodeQueryOpts,
  validateInvitationTokenQueryOpts,
  acceptInvitationMutationOpts,
  requestPasswordResetMutationOpts,
  verifyPasswordResetMutationOpts,
  bootstrapCheckQueryOpts,
  switchOrganizationMutationOpts,
  membershipsQueryOpts,
}