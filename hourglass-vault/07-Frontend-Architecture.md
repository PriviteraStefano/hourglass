# Frontend Architecture

Design patterns and conventions for React development in Hourglass.

## Overview

**Tech Stack:**
- React 19 — component library
- TanStack Router v1 — file-based routing with type-safe loaders
- TanStack React Query v5 — server state management
- Vite — bundler and dev server
- TypeScript — type safety
- Tailwind CSS + shadcn/ui — styling and components

**Key Pattern:** Controlled data flow with React Query, route-driven UI, composable components.

---

## File-Based Routing

TanStack Router automatically creates routes from file structure.

### Directory Structure

```
web/src/routes/
├── __root.tsx                    # Root layout (header, sidebar, etc)
├── (auth)
│   ├── login.tsx                # /login
│   └── register.tsx             # /register
├── _authenticated.tsx            # Protected route guard (beforeLoad)
├── _authenticated
│   ├── index.tsx                # /dashboard
│   ├── time-entries
│   │   ├── index.tsx           # /time-entries (list)
│   │   └── $id.tsx             # /time-entries/:id (detail)
│   ├── expenses
│   │   ├── index.tsx           # /expenses
│   │   └── $id.tsx             # /expenses/:id
│   ├── contracts
│   │   ├── index.tsx           # /contracts
│   │   └── $id.tsx             # /contracts/:id
│   └── reports
│       └── index.tsx            # /reports
└── catch.tsx                     # 404 page
```

### Route File Pattern

```typescript
// routes/time-entries/index.tsx
import { createFileRoute } from '@tanstack/react-router'
import TimeEntriesList from '@/components/time-entries-list'
import { useTimeEntriesQuery } from '@/hooks'

export const Route = createFileRoute('/_authenticated/time-entries/')({
  component: TimeEntriesPage,
  meta: () => ({
    title: 'Time Entries',
  }),
})

function TimeEntriesPage() {
  const { data: entries, isLoading } = useTimeEntriesQuery()
  
  if (isLoading) return <div>Loading...</div>
  return <TimeEntriesList entries={entries} />
}
```

### Parametrized Routes

```typescript
// routes/time-entries/$id.tsx
export const Route = createFileRoute('/_authenticated/time-entries/$id')({
  component: TimeEntryDetail,
})

function TimeEntryDetail() {
  const { id } = Route.useParams()  // Access URL param
  const { data: entry } = useTimeEntryQuery(id)
  
  return <div>{entry.id}</div>
}
```

---

## Protected Routes

Routes under `/_authenticated` require authentication.

**Pattern**: `beforeLoad` hook validates auth before rendering.

```typescript
// routes/_authenticated.tsx
import { createFileRoute, redirect } from '@tanstack/react-router'
import { AuthApis } from '@/api/auth'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context: { client } }) => {
    try {
      // Fetch current user profile
      const profile = await client.fetchQuery(AuthApis.profileQueryOpts)
      return { user: profile }
    } catch (error) {
      // Not authenticated, redirect to login
      throw redirect({ to: '/login' })
    }
  },
  component: ProtectedLayout,
})

function ProtectedLayout() {
  const { user } = Route.useRouterContext()
  const { outlet } = Route.useRouterContext()
  
  return (
    <div>
      <AppShell user={user}>
        {outlet}
      </AppShell>
    </div>
  )
}
```

---

## React Query (TanStack React Query)

All server state is managed via React Query.

### Query Pattern

**Define queries in `api/` files:**

```typescript
// api/time-entries.ts
import { queryOptions } from '@tanstack/react-query'
import { api } from '@/lib/api'

export const timeEntriesQueryOptions = () =>
  queryOptions({
    queryKey: ['time-entries'],
    queryFn: async () => {
      const { data } = await api.get('/time-entries')
      return data
    },
    staleTime: 1000 * 60 * 5,  // 5 minutes
  })

export const timeEntryQueryOptions = (id: string) =>
  queryOptions({
    queryKey: ['time-entries', id],
    queryFn: async () => {
      const { data } = await api.get(`/time-entries/${id}`)
      return data
    },
  })
```

**Use in components:**

```typescript
import { useSuspenseQuery } from '@tanstack/react-query'
import { timeEntriesQueryOptions } from '@/api/time-entries'

function TimeEntriesList() {
  const { data: entries } = useSuspenseQuery(timeEntriesQueryOptions())
  
  return (
    <ul>
      {entries.map(e => (
        <li key={e.id}>{e.id}</li>
      ))}
    </ul>
  )
}
```

### Mutation Pattern

**Define mutations:**

```typescript
// api/time-entries.ts
export const createTimeEntryMutation = () =>
  mutationOptions({
    mutationFn: async (data: CreateTimeEntryRequest) => {
      const { data: response } = await api.post('/time-entries', data)
      return response
    },
  })
```

**Use in forms:**

```typescript
import { useMutation } from '@tanstack/react-query'
import { createTimeEntryMutation } from '@/api/time-entries'

function CreateTimeEntryForm() {
  const { mutate, isPending } = useMutation({
    ...createTimeEntryMutation(),
    onSuccess: () => {
      // Invalidate list query to refetch
      queryClient.invalidateQueries({ queryKey: ['time-entries'] })
      // Navigate to new entry
      navigate({ to: '/time-entries' })
    },
  })
  
  const handleSubmit = (formData) => {
    mutate(formData)
  }
  
  return (
    <form onSubmit={handleSubmit}>
      {/* form fields */}
      <button disabled={isPending}>
        {isPending ? 'Creating...' : 'Create'}
      </button>
    </form>
  )
}
```

### Query Client Configuration

**File**: `lib/query-client.ts`

```typescript
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: 1000 * 60 * 5,  // 5 minutes
      gcTime: 1000 * 60 * 10,    // 10 minutes (cache duration)
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})
```

**Setup in main.tsx:**

```typescript
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { RouterProvider } from '@tanstack/react-router'

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <RouterProvider router={router} context={{ client: queryClient }} />
  </QueryClientProvider>
)
```

---

## API Client

**File**: `lib/api.ts`

Axios instance with automatic auth header injection and error handling.

```typescript
import axios from 'axios'

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  timeout: 10000,
})

// Add Bearer token to all requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle 401 by refreshing token
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Try to refresh token
      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        try {
          const { data } = await axios.post(
            `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}/auth/refresh`,
            { refresh_token: refreshToken }
          )
          localStorage.setItem('access_token', data.data.access_token)
          
          // Retry original request
          error.config.headers.Authorization = `Bearer ${data.data.access_token}`
          return api(error.config)
        } catch {
          // Refresh failed, redirect to login
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          window.location.href = '/login'
        }
      }
    }
    return Promise.reject(error)
  }
)
```

---

## Component Patterns

### Container vs Presentational

**Container** (smart, data fetching):
```typescript
function TimeEntriesList() {
  const { data: entries } = useSuspenseQuery(timeEntriesQueryOptions())
  
  return <TimeEntriesListUI entries={entries} />
}
```

**Presentational** (dumb, receives props):
```typescript
interface TimeEntriesListUIProps {
  entries: TimeEntry[]
}

export function TimeEntriesListUI({ entries }: TimeEntriesListUIProps) {
  return (
    <table>
      {entries.map(e => (
        <tr key={e.id}>
          <td>{e.id}</td>
          <td>{e.status}</td>
        </tr>
      ))}
    </table>
  )
}
```

---

## Form Handling

Use uncontrolled components with react-hook-form (recommended):

```typescript
import { useForm } from 'react-hook-form'
import { useMutation } from '@tanstack/react-query'

function TimeEntryForm() {
  const { register, handleSubmit, formState: { errors } } = useForm()
  const { mutate } = useMutation(createTimeEntryMutation())
  
  return (
    <form onSubmit={handleSubmit((data) => mutate(data))}>
      <input {...register('hours', { required: 'Hours required' })} />
      {errors.hours && <span>{errors.hours.message}</span>}
      
      <input {...register('projectId', { required: true })} />
      
      <button type="submit">Create</button>
    </form>
  )
}
```

---

## Type Safety

### API Response Types

**File**: `types/api.ts`

```typescript
export interface TimeEntry {
  id: string
  userId: string
  organizationId: string
  status: 'draft' | 'submitted' | 'pending_manager' | 'pending_finance' | 'approved' | 'rejected'
  currentApproverRole: 'manager' | 'finance' | null
  workDate: string  // ISO date
  notes?: string
  createdAt: string  // ISO datetime
  updatedAt: string
}

export interface CreateTimeEntryRequest {
  projectIds: string[]
  hours: number
  workDate: string
  notes?: string
}
```

### Route Context Types

```typescript
// routes/__root.tsx
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// In route loaders
const context = useRouterContext()  // Type-safe
```

---

## Styling

Uses **Tailwind CSS + shadcn/ui components**.

### Button Example
```typescript
import { Button } from '@/components/ui/button'

export function MyComponent() {
  return <Button variant="primary">Click me</Button>
}
```

### Form Example
```typescript
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function Form() {
  return (
    <div>
      <Label htmlFor="name">Name</Label>
      <Input id="name" placeholder="John Doe" />
    </div>
  )
}
```

---

## Naming Conventions

| Item | Convention | Example |
|------|-----------|---------|
| Components | PascalCase | `TimeEntryForm.tsx`, `StatusBadge.tsx` |
| Files | kebab-case | `time-entries-list.tsx` |
| Hooks | `useXxx` | `useTimeEntries`, `useOrgSettings` |
| State vars | camelCase | `isLoading`, `hasError` |
| Props interface | `{Component}Props` | `TimeEntryListProps` |
| Routes | kebab-case | `/time-entries`, `/org-settings` |

---

**Next**: [[08-API-Client]] for detailed API patterns, or [[07-Frontend-Architecture]] is complete! Head to [[10-Time-Entries]] for feature details.
