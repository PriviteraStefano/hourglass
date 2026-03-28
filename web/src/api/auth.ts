import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {api} from "@/lib/api.ts";
import type {AuthResponse, LoginRequest, RegisterRequest, UserWithMembership} from "@/types";

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
    if (data) {
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

export const AuthApis = {
  profileQueryOpts,
  loginMutationOpts,
  registerMutationOpts,
  logoutMutationOpts,
  refreshMutationOpts,
}