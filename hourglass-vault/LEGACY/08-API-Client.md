# API Client Patterns

Best practices for building and consuming API clients in Hourglass.

---

## HTTP Client (lib/api.ts)

Frontend uses axios for HTTP requests with automatic authentication.

### Configuration

**File:** `web/src/lib/api.ts`

```typescript
import axios, { AxiosError } from 'axios'
import { useNavigate } from '@tanstack/react-router'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

export default api
```

### Request Interceptor (Auto Auth)

```typescript
// Add Bearer token to every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
}, (error) => {
  return Promise.reject(error)
})
```

### Response Interceptor (Error Handling)

```typescript
api.interceptors.response.use(
  // Success response
  (response) => response,
  
  // Error handling
  async (error: AxiosError) => {
    // 401: Token expired or invalid
    if (error.response?.status === 401) {
      const refreshToken = localStorage.getItem('refresh_token')
      
      if (refreshToken) {
        try {
          // Attempt refresh
          const { data } = await axios.post(
            `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/auth/refresh`,
            { refresh_token: refreshToken },
            { timeout: 5000 }
          )
          
          // Save new token
          localStorage.setItem('access_token', data.data.access_token)
          
          // Retry original request
          error.config!.headers.Authorization = `Bearer ${data.data.access_token}`
          return api(error.config!)
        } catch (refreshError) {
          // Refresh failed → redirect to login
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          localStorage.removeItem('current_org')
          window.location.href = '/login'
          return Promise.reject(refreshError)
        }
      } else {
        // No refresh token → redirect to login
        window.location.href = '/login'
      }
    }
    
    // Other errors: pass through
    return Promise.reject(error)
  }
)

export { api }
```

---

## React Query Integration

### Query Definition Pattern

**File:** `web/src/api/time-entries.ts`

```typescript
import { queryOptions, mutationOptions } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { TimeEntry, CreateTimeEntryRequest } from '@/types/api'

// Query Key factory (for easy invalidation)
export const timeEntriesKeys = {
  all: () => ['time-entries'] as const,
  lists: () => [...timeEntriesKeys.all(), 'list'] as const,
  list: (filters?: Record<string, unknown>) =>
    [...timeEntriesKeys.lists(), filters] as const,
  details: () => [...timeEntriesKeys.all(), 'detail'] as const,
  detail: (id: string) =>
    [...timeEntriesKeys.details(), id] as const,
}

// List query
export const timeEntriesQueryOptions = (filters?: {
  status?: string
  start_date?: string
  end_date?: string
}) =>
  queryOptions({
    queryKey: timeEntriesKeys.list(filters),
    queryFn: async () => {
      const { data } = await api.get<{ data: TimeEntry[] }>(
        '/time-entries',
        { params: filters }
      )
      return data.data
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 10,   // 10 minutes
  })

// Detail query
export const timeEntryQueryOptions = (id: string) =>
  queryOptions({
    queryKey: timeEntriesKeys.detail(id),
    queryFn: async () => {
      const { data } = await api.get<{ data: TimeEntry }>(
        `/time-entries/${id}`
      )
      return data.data
    },
    staleTime: 1000 * 60 * 5,
  })

// Create mutation
export const createTimeEntryMutation = () =>
  mutationOptions({
    mutationFn: async (data: CreateTimeEntryRequest) => {
      const response = await api.post<{ data: TimeEntry }>(
        '/time-entries',
        data
      )
      return response.data.data
    },
  })

// Submit mutation
export const submitTimeEntryMutation = () =>
  mutationOptions({
    mutationFn: async (id: string) => {
      const response = await api.post<{ data: TimeEntry }>(
        `/time-entries/${id}/submit`
      )
      return response.data.data
    },
  })

// Approve mutation
export const approveTimeEntryMutation = () =>
  mutationOptions({
    mutationFn: async (id: string, reason?: string) => {
      const response = await api.post<{ data: TimeEntry }>(
        `/time-entries/${id}/approve`,
        { reason }
      )
      return response.data.data
    },
  })
```

### Using Queries in Components

```typescript
import { useSuspenseQuery, useQuery } from '@tanstack/react-query'
import { timeEntriesQueryOptions } from '@/api/time-entries'

// With Suspense (loading handled by Suspense boundary)
function TimeEntriesList() {
  const { data: entries } = useSuspenseQuery(timeEntriesQueryOptions())
  
  return (
    <table>
      {entries.map(entry => (
        <tr key={entry.id}>
          <td>{entry.work_date}</td>
          <td>{entry.status}</td>
        </tr>
      ))}
    </table>
  )
}

// Without Suspense (handle loading)
function TimeEntriesListWithLoading() {
  const { data: entries, isLoading } = useQuery(timeEntriesQueryOptions())
  
  if (isLoading) return <div>Loading...</div>
  
  return (
    <table>
      {entries?.map(entry => (
        <tr key={entry.id}>...</tr>
      ))}
    </table>
  )
}
```

### Using Mutations in Forms

```typescript
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createTimeEntryMutation, timeEntriesKeys } from '@/api/time-entries'

function CreateTimeEntryForm() {
  const queryClient = useQueryClient()
  const { mutate, isPending } = useMutation({
    ...createTimeEntryMutation(),
    onSuccess: (newEntry) => {
      // Invalidate list queries
      queryClient.invalidateQueries({
        queryKey: timeEntriesKeys.lists(),
      })
      
      // Navigate to detail view
      navigate({ to: `/time-entries/${newEntry.id}` })
    },
    onError: (error) => {
      console.error('Failed to create entry:', error)
      // Show toast/snackbar error
    },
  })
  
  const handleSubmit = (formData) => {
    mutate(formData)
  }
  
  return (
    <form onSubmit={handleSubmit}>
      {/* form fields */}
      <button disabled={isPending} type=\"submit\">
        {isPending ? 'Creating...' : 'Create'}
      </button>
    </form>
  )
}
```

---

## Error Handling

### Typed Error Responses

```typescript
interface ErrorResponse {
  error: string
  details?: Record<string, string[]>  // Validation errors
}

// Use in error handler
const handleApiError = (error: AxiosError<ErrorResponse>) => {
  if (error.response?.status === 400) {
    const details = error.response.data.details
    // Show form field errors
    return details
  }
  
  if (error.response?.status === 401) {
    // Token expired or invalid
    redirectToLogin()
  }
  
  if (error.response?.status === 403) {
    // Not authorized
    showError('You do not have permission')
  }
  
  if (error.response?.status === 409) {
    // Conflict (e.g., duplicate)
    showError(error.response.data.error)
  }
  
  // 500+ errors
  showError('Something went wrong. Please try again.')
}
```

---

## Type Safety

### Shared Types

**File:** `web/src/types/api.ts`

```typescript
// Entities
export interface TimeEntry {
  id: string
  organization_id: string
  user_id: string
  status: 'draft' | 'submitted' | 'pending_manager' | 'pending_finance' | 'approved' | 'rejected'
  current_approver_role?: 'manager' | 'finance'
  work_date: string  // ISO date
  submitted_at?: string  // ISO datetime
  notes?: string
  items: TimeEntryItem[]
  created_at: string
  updated_at: string
}

export interface TimeEntryItem {
  id: string
  time_entry_id: string
  project_id: string
  hours: number
  notes?: string
}

// Requests
export interface CreateTimeEntryRequest {
  work_date: string
  items: Array<{
    project_id: string
    hours: number
    notes?: string
  }>
  notes?: string
}

export interface UpdateTimeEntryRequest {
  work_date: string
  items: Array<{
    id?: string
    project_id: string
    hours: number
    notes?: string
  }>
  notes?: string
}

// Generic API response
export interface ApiResponse<T> {
  data?: T
  error?: string
}
```

### Type-Safe Hooks

```typescript
// Hook with full type inference
export function useTimeEntry(id: string) {
  return useQuery(timeEntryQueryOptions(id))  // Infers data type
}

const { data: entry } = useTimeEntry(entryId)
// entry is TimeEntry | undefined
entry?.work_date  // ✓ TypeScript autocomplete
```

---

## Pagination

### Offset-Based Pagination

```typescript
export const timeEntriesQueryOptions = (filters?: {
  status?: string
  limit?: number
  offset?: number
}) => queryOptions({
  queryKey: timeEntriesKeys.list(filters),
  queryFn: async () => {
    const { data } = await api.get<{
      data: TimeEntry[]
      total: number
      offset: number
      limit: number
    }>('/time-entries', { params: filters })
    return data
  },
})

// Usage in paginated list
function PaginatedTimeEntries() {
  const [page, setPage] = useState(0)
  const limit = 25
  
  const { data } = useQuery(
    timeEntriesQueryOptions({
      limit,
      offset: page * limit,
    })
  )
  
  return (
    <>
      {/* entries */}
      <button onClick={() => setPage(page - 1)} disabled={page === 0}>
        Previous
      </button>
      <button onClick={() => setPage(page + 1)} disabled={!hasMore}>
        Next
      </button>
    </>
  )
}
```

---

## Caching Strategy

### Default Configuration

**File:** `web/src/lib/query-client.ts`

```typescript
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,          // Don't auto-retry
      staleTime: 5 * 60 * 1000,        // 5 minutes
      gcTime: 10 * 60 * 1000,          // 10 minutes (cache duration)
      refetchOnWindowFocus: false,      // Don't refetch on tab focus
      refetchOnReconnect: true,         // Refetch when reconnected
      refetchOnMount: false,
    },
    mutations: {
      retry: 0,              // Don't auto-retry mutations
    },
  },
})
```

### Query Invalidation

```typescript
// Invalidate specific query
queryClient.invalidateQueries({
  queryKey: ['time-entries', entryId],
})

// Invalidate all time entry queries
queryClient.invalidateQueries({
  queryKey: timeEntriesKeys.all(),
})

// Invalidate with predicate
queryClient.invalidateQueries({
  predicate: (query) =>
    query.queryKey[0] === 'time-entries' &&
    query.queryKey[1]?.status === 'draft',
})
```

---

## Request Cancellation

```typescript
import { CancelToken } from 'axios'

const cancelToken = new CancelToken((cancel) => {
  // Cancel when component unmounts
  return cancel('Operation cancelled')
})

const { data } = await api.get('/time-entries', { cancelToken })
```

---

## Optimistic Updates

```typescript
const { mutate } = useMutation({
  ...approveTimeEntryMutation(),
  onMutate: async (id) => {
    // Cancel outgoing queries
    await queryClient.cancelQueries({
      queryKey: timeEntriesKeys.detail(id),
    })
    
    // Snapshot previous data
    const previousEntry = queryClient.getQueryData(
      timeEntriesKeys.detail(id)
    )
    
    // Optimistically update cache
    queryClient.setQueryData(timeEntriesKeys.detail(id), (old: TimeEntry) => ({
      ...old,
      status: 'pending_finance',
    }))
    
    return { previousEntry }
  },
  onError: (error, id, context) => {
    // Rollback on error
    if (context?.previousEntry) {
      queryClient.setQueryData(
        timeEntriesKeys.detail(id),
        context.previousEntry
      )
    }
  },
})
```

---

**Next**: [[00-Index]] for all documentation, or continue with feature documentation.
