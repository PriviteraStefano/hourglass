import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {format} from 'date-fns'
import {toast} from 'sonner'
import {api} from '@/lib/api.ts'
import type {CreateTimeEntryRequest, TimeEntry, UpdateTimeEntryRequest} from '@/types'

function timeEntriesForMonthQueryKey(month: number, year: number) {
  return ['time-entries', 'month', month, year] as const
}

function timeEntryQueryKey(date: Date) {
  return ['time-entries', 'date', format(date, 'yyyy-MM-dd')] as const
}

export function timeEntriesForMonthQueryOpts(month: number, year: number) {
  return queryOptions({
    queryKey: timeEntriesForMonthQueryKey(month, year),
    queryFn: () => api<TimeEntry[]>(`/time-entries?month=${month}&year=${year}`),
  })
}

function timeEntryQueryOpts(date: Date) {
  const formattedDate = format(date, 'yyyy-MM-dd')
  return queryOptions({
    queryKey: timeEntryQueryKey(date),
    queryFn: () => api<TimeEntry[]>(`/time-entries?date=${formattedDate}`),
    enabled: !!date,
  })
}

const createTimeEntryMutationOpts = mutationOptions({
  mutationFn: (data: CreateTimeEntryRequest) =>
    api<TimeEntry>('/time-entries', {method: 'POST', body: JSON.stringify(data)}),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Entry created')
  },
})
const updateTimeEntryMutationOpts = mutationOptions({
  mutationFn: ({id, ...data}: UpdateTimeEntryRequest & { id: string }) =>
    api<TimeEntry>(`/time-entries/${id}`, {method: 'PUT', body: JSON.stringify(data)}),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Time entry updated')
  },
})
const deleteTimeEntryMutationOpts = mutationOptions({
  mutationFn: (id: string) => api(`/time-entries/${id}`, {method: 'DELETE'}),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Time entry deleted')
  },
})
const submitTimeEntryMutationOpts = mutationOptions({
  mutationFn: (id: string) => api<TimeEntry>(`/time-entries/${id}/submit`, {method: 'POST'}),
  onSuccess: (_, __, ___, {client}) => {
    client.invalidateQueries({queryKey: ['time-entries']})
    toast.success('Entry submitted for approval')
  },
})

const approveTimeEntryMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<TimeEntry>(`/time-entries/${id}/approve`, {method: 'POST'}),
})

const rejectTimeEntryMutationOpts = mutationOptions({
  mutationFn: ({id, reason}: { id: string; reason: string }) =>
    api<TimeEntry>(`/time-entries/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({reason}),
    }),
})

const pendingTimeEntriesQueryOpts = queryOptions({
  queryKey: ['time-entries', 'pending'],
  queryFn: () => api<TimeEntry[]>('/time-entries/pending'),
})

export const TimeEntriesApis = {
  timeEntriesForMonthQueryOpts,
  timeEntryQueryOpts,
  createTimeEntryMutationOpts,
  updateTimeEntryMutationOpts,
  deleteTimeEntryMutationOpts,
  submitTimeEntryMutationOpts,
  approveTimeEntryMutationOpts,
  rejectTimeEntryMutationOpts,
  pendingTimeEntriesQueryOpts,
}
