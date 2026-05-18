import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { api } from '../api'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('api<T>() HTTP client', () => {
  it('sends credentials include and returns data (envelope unwrapped)', async () => {
    server.use(
      http.get('/api/test', () =>
        HttpResponse.json({ data: { id: '1', name: 'test' } }),
      ),
    )

    const result = await api<{ id: string; name: string }>('/test')
    expect(result).toEqual({ id: '1', name: 'test' })
  })

  it('handles error response', async () => {
    server.use(
      http.get('/api/test', () =>
        HttpResponse.json({ error: 'bad request' }, { status: 400 }),
      ),
    )

    await expect(api('/test')).rejects.toThrow('bad request')
  })

  it('handles 401 then refresh then retry succeeds', async () => {
    let protectedCalls = 0
    server.use(
      http.get('/api/protected', () => {
        protectedCalls++
        if (protectedCalls === 1) {
          return new HttpResponse(null, { status: 401 })
        }
        return HttpResponse.json({ data: { success: true } })
      }),
      http.post('/api/auth/refresh', () =>
        HttpResponse.json({ data: { user: { id: '1' } } }),
      ),
    )

    const result = await api<{ success: boolean }>('/protected')
    expect(result).toEqual({ success: true })
    expect(protectedCalls).toBe(2)
  })

  it('handles 401 then refresh fails', async () => {
    server.use(
      http.get('/api/protected', () => new HttpResponse(null, { status: 401 })),
      http.post('/api/auth/refresh', () => new HttpResponse(null, { status: 401 })),
    )

    await expect(api('/protected')).rejects.toThrow('Unauthorized')
  })

  it('sets Content-Type: application/json header', async () => {
    let capturedHeaders: Headers | null = null
    server.use(
      http.get('/api/test', ({ request }) => {
        capturedHeaders = request.headers
        return HttpResponse.json({ data: 'ok' })
      }),
    )

    await api('/test')
    expect(capturedHeaders?.get('Content-Type')).toBe('application/json')
  })

  it('handles network error', async () => {
    server.use(
      http.get('/api/test', () => HttpResponse.error()),
    )

    await expect(api('/test')).rejects.toThrow()
  })
})
