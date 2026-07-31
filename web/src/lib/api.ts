import { type ApiError, type ApiResponse } from "@/types";

const API_BASE = import.meta.env.VITE_API_URL || "/api";

/**
 * Thrown when the access token is expired AND the refresh attempt also fails.
 * Route guards catch this to redirect to /login via TanStack Router's redirect().
 * The HTTP client itself never performs navigation — that was the root cause
 * of the original infinite loop (hard window.location.href fighting the router).
 */
export class UnauthorizedError extends Error {
  constructor(message = "Unauthorized") {
    super(message);
    this.name = "UnauthorizedError";
  }
}

/**
 * Auth paths that should NOT trigger a refresh attempt.
 * If we're already on /login, /register, etc., a 401 means "you're not logged in"
 * — trying to refresh would be pointless and risks recursion.
 */
const AUTH_PATHS = [
  "/auth/login",
  "/auth/register",
  "/auth/refresh",
  "/auth/password-reset",
];

function isAuthPath(path: string): boolean {
  return AUTH_PATHS.some((p) => path.startsWith(p));
}

let refreshPromise: Promise<void> | null = null;

export async function api<T>(path: string, options?: RequestInit): Promise<T> {
  let res = await fetch(`${API_BASE}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (res.status === 401) {
    // Don't attempt refresh for auth endpoints themselves — prevents recursion
    if (isAuthPath(path)) {
      throw new UnauthorizedError();
    }

    if (!refreshPromise) {
      refreshPromise = fetch(`${API_BASE}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      })
        .then(async (refresh) => {
          if (!refresh.ok) {
            throw new UnauthorizedError();
          }
        })
        .finally(() => {
          refreshPromise = null;
        });
    }
    try {
      await refreshPromise;
    } catch {
      throw new UnauthorizedError();
    }

    res = await fetch(`${API_BASE}${path}`, {
      ...options,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });
  }

  if (!res.ok) {
    const error = (await res
      .json()
      .catch(() => ({ message: "Request failed" }))) as ApiError;
    throw new Error(error.message || error.error || "Request failed");
  }
  return ((await res.json()) as ApiResponse<T>).data;
}
