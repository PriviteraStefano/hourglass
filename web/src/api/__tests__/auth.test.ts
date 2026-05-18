import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { AuthApis } from '../auth'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('AuthApis', () => {
  it('profileQueryOpts calls GET /auth/me and returns user with membership', async () => {
    const mockData = {
      user: {
        id: '1', email: 'test@test.com', username: 'testuser',
        name: 'Test User', is_active: true, created_at: '2025-01-01T00:00:00Z',
      },
      membership: {
        id: 'm1', user_id: '1', organization_id: 'o1',
        role: 'employee' as const, is_active: true,
      },
      organization: {
        id: 'o1', name: 'Test Org', slug: 'test-org', created_at: '2025-01-01T00:00:00Z',
      },
    }
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json({ data: mockData })),
    )

    const result = await AuthApis.profileQueryOpts.queryFn()
    expect(result).toEqual(mockData)
  })

  it('loginMutationOpts sends POST /auth/login with credentials', async () => {
    const creds = { identifier: 'user@test.com', password: 'password123' }
    const mockResponse = {
      user: {
        user: {
          id: '1', email: 'user@test.com', username: 'user',
          name: 'User', is_active: true, created_at: '2025-01-01T00:00:00Z',
        },
        membership: {
          id: 'm1', user_id: '1', organization_id: 'o1',
          role: 'employee' as const, is_active: true,
        },
        organization: {
          id: 'o1', name: 'Org', slug: 'org', created_at: '2025-01-01T00:00:00Z',
        },
      },
      token: 'abc123',
      refresh_token: 'def456',
      expires_at: '2026-01-01T00:00:00Z',
    }

    let capturedBody: unknown = null
    server.use(
      http.post('/api/auth/login', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ data: mockResponse })
      }),
    )

    const result = await AuthApis.loginMutationOpts.mutationFn(creds)
    expect(capturedBody).toEqual(creds)
    expect(result).toEqual(mockResponse)
  })

  it('registerMutationOpts sends POST /auth/register with registration data', async () => {
    const registerData = {
      email: 'new@test.com',
      password: 'password123',
      name: 'New User',
      organization_name: 'New Org',
    }
    const mockResponse = {
      user: {
        user: {
          id: '2', email: 'new@test.com', username: 'newuser',
          name: 'New User', is_active: true, created_at: '2025-01-01T00:00:00Z',
        },
        membership: {
          id: 'm2', user_id: '2', organization_id: 'o2',
          role: 'employee' as const, is_active: true,
        },
        organization: {
          id: 'o2', name: 'New Org', slug: 'new-org', created_at: '2025-01-01T00:00:00Z',
        },
      },
      token: 'abc',
      refresh_token: 'def',
      expires_at: '2026-01-01T00:00:00Z',
    }

    let capturedBody: unknown = null
    server.use(
      http.post('/api/auth/register', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ data: mockResponse })
      }),
    )

    const result = await AuthApis.registerMutationOpts.mutationFn(registerData)
    expect(capturedBody).toEqual(registerData)
    expect(result).toEqual(mockResponse)
  })

  it('logoutMutationOpts sends POST /auth/logout', async () => {
    server.use(
      http.post('/api/auth/logout', () =>
        HttpResponse.json({ data: { message: 'logged out' } }),
      ),
    )

    const result = await AuthApis.logoutMutationOpts.mutationFn()
    expect(result).toEqual({ message: 'logged out' })
  })

  it('refreshMutationOpts sends POST /auth/refresh and returns session', async () => {
    const mockResponse = {
      user: {
        user: {
          id: '1', email: 'user@test.com', username: 'user',
          name: 'User', is_active: true, created_at: '2025-01-01T00:00:00Z',
        },
        membership: {
          id: 'm1', user_id: '1', organization_id: 'o1',
          role: 'employee' as const, is_active: true,
        },
        organization: {
          id: 'o1', name: 'Org', slug: 'org', created_at: '2025-01-01T00:00:00Z',
        },
      },
      token: 'refreshed-token',
      refresh_token: 'new-refresh',
      expires_at: '2026-01-01T00:00:00Z',
    }

    server.use(
      http.post('/api/auth/refresh', () =>
        HttpResponse.json({ data: mockResponse }),
      ),
    )

    const result = await AuthApis.refreshMutationOpts.mutationFn()
    expect(result).toEqual(mockResponse)
  })
})
