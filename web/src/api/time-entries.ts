import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {format} from 'date-fns'
import {toast} from 'sonner'
import {api} from '@/src/lib/api.ts'
import type {
  ApiResponse,
  CreateTimeEntryRequest,
  TimeEntry,
  TimeEntryMonthlySummary,
  UpdateTimeEntryRequest,
} from '@/src/types'
function timeEntriesQueryKey(month: number, year: number) {
  return ['time-entries', 'monthly', month, year] as const
}
function timeEntryQueryKey(date: Date) {
  return ['time-entries', format(date, 'yyyy-MM-dd')] as const
}
function timeEntriesMonthlySummaryQueryOpts(month: number, year: number) {
  return queryOptions({
    queryKey: timeEntriesQueryKey(month, year),
    queryFn: () => api<ApiResponse<TimeEntryMonthlySummary>>(`/time-entries/monthly-summary?month=${month}&year=${year}`),
  })
}
function timeEntryQueryOpts(date: Date) {
  const formattedDate = format(date, 'yyyy-MM-dd')
  return queryOptions({
    queryKey: timeEntryQueryKey(date),
    queryFn: () => api<ApiResponse<TimeEntry>>(`/time-entries?date=${formattedDate}`),
    enabled: !!date,
  })
}
const createTimeEntryMutationOpts = mutationOptions({
  mutationFn: (data: CreateTimeEntryRequest) =>
    api<ApiResponse<TimeEntry>>('/time-entries', { method: 'POST', body: JSON.stringify(data) }),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})

  },
})
const updateTimeEntryMutationOpts = mutationOptions({
  mutationFn: ({id, ...data}: UpdateTimeEntryRequest & { id: string }) =>
    api<ApiResponse<TimeEntry>>(`/time-entries/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Time entry updated')
  },
})
const deleteTimeEntryMutationOpts = mutationOptions({
  mutationFn: (id: string) => api(`/time-entries/${id}`, { method: 'DELETE' }),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Time entry deleted')
  },
})
const submitTimeEntryMutationOpts = mutationOptions({
  mutationFn: (id: string) => api<ApiResponse<TimeEntry>>(`/time-entries/${id}/submit`, { method: 'POST' }),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Entry submitted for approval')
  },
})
const submitMonthMutationOpts = mutationOptions({
  mutationFn: ({month, year}: { month: number; year: number }) =>
    api(`/time-entries/submit-month`, { method: 'POST', body: JSON.stringify({month, year}) }),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('All drafts submitted')
  },
})
export const TimeEntriesApis = {
  timeEntriesMonthlySummaryQueryOpts,
  timeEntryQueryOpts,
  createTimeEntryMutationOpts,
  updateTimeEntryMutationOpts,
  deleteTimeEntryMutationOpts,
  submitTimeEntryMutationOpts,
  submitMonthMutationOpts,
}
