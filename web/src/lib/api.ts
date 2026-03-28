import {type ApiError, type ApiResponse} from "@/src/types";

const API_BASE = import.meta.env.VITE_API_URL || '/api'

export async function api<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: 'Request failed' })) as ApiError
    throw new Error(error.message || error.error || 'Request failed')
  }
  return (await res.json() as ApiResponse<T>).data
}