import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { TimeEntriesApis } from '../time-entries'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('TimeEntriesApis', () => {
  it('timeEntriesMonthlySummaryQueryOpts calls GET /time-entries/monthly-summary', async () => {
    const mockSummary = {
      days: [
        { date: '2026-05-01', has_draft: true, has_submitted: false, has_approved: false, has_rejected: false, total_hours: 8 },
      ],
      totals: { total_hours: 8, draft_count: 1, submitted_count: 0, approved_count: 0, rejected_count: 0 },
    }
    server.use(
      http.get('/api/time-entries/monthly-summary', ({ request }) => {
        const url = new URL(request.url)
        expect(url.searchParams.get('month')).toBe('5')
        expect(url.searchParams.get('year')).toBe('2026')
        return HttpResponse.json({ data: mockSummary })
      }),
    )

    const opts = TimeEntriesApis.timeEntriesMonthlySummaryQueryOpts(5, 2026)
    const result = await opts.queryFn()
    expect(result).toEqual(mockSummary)
  })

  it('createTimeEntryMutationOpts sends POST /time-entries with entry data', async () => {
    const entryData = {
      date: '2026-05-18',
      items: [{ project_id: 'p1', hours: 8, description: 'Work' }],
    }
    const mockEntry = {
      id: 'te1', user_id: 'u1', organization_id: 'o1',
      date: '2026-05-18', status: 'draft' as const,
      created_at: '2026-05-18T10:00:00Z', updated_at: '2026-05-18T10:00:00Z',
      items: [{ id: 'i1', time_entry_id: 'te1', project_id: 'p1', hours: 8, description: 'Work' }],
    }

    let capturedBody: unknown = null
    server.use(
      http.post('/api/time-entries', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ data: mockEntry })
      }),
    )

    const result = await TimeEntriesApis.createTimeEntryMutationOpts.mutationFn(entryData)
    expect(capturedBody).toEqual(entryData)
    expect(result).toEqual(mockEntry)
  })

  it('submitMonthMutationOpts sends POST /time-entries/submit-month', async () => {
    server.use(
      http.post('/api/time-entries/submit-month', async ({ request }) => {
        const body = await request.json()
        expect(body).toEqual({ month: 5, year: 2026 })
        return HttpResponse.json({ data: { message: 'submitted' } })
      }),
    )

    const result = await TimeEntriesApis.submitMonthMutationOpts.mutationFn({ month: 5, year: 2026 })
    expect(result).toEqual({ message: 'submitted' })
  })

  it('timeEntryQueryOpts calls GET /time-entries with date', async () => {
    const mockEntry = {
      id: 'te1', user_id: 'u1', organization_id: 'o1',
      date: '2026-05-18', status: 'draft' as const,
      created_at: '2026-05-18T10:00:00Z', updated_at: '2026-05-18T10:00:00Z',
      items: [],
    }
    server.use(
      http.get('/api/time-entries', ({ request }) => {
        const url = new URL(request.url)
        expect(url.searchParams.get('date')).toBe('2026-05-18')
        return HttpResponse.json({ data: mockEntry })
      }),
    )

    const opts = TimeEntriesApis.timeEntryQueryOpts(new Date('2026-05-18'))
    const result = await opts.queryFn()
    expect(result).toEqual(mockEntry)
  })
})
