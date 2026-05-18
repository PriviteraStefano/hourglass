import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { CustomersApis } from '../customers'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('CustomersApis', () => {
  it('customersQueryOpts calls GET /api/customers', async () => {
    const mockCustomers = [
      {
        id: 'cust1', organization_id: 'o1', company_name: 'Acme Corp',
        contact_name: 'John', email: 'john@acme.com',
        phone: '+123', vat_number: 'VAT123', address: '123 Main St',
        is_active: true, created_at: '2025-01-01T00:00:00Z',
      },
    ]
    server.use(
      http.get('/api/customers', () =>
        HttpResponse.json({ data: mockCustomers }),
      ),
    )

    const result = await CustomersApis.customersQueryOpts().queryFn()
    expect(result).toEqual(mockCustomers)
  })

  it('createCustomerMutationOpts sends POST /api/customers', async () => {
    const customerData = {
      company_name: 'New Corp',
      contact_name: 'Jane',
      email: 'jane@newcorp.com',
    }
    const mockCustomer = {
      id: 'cust2', organization_id: 'o1', company_name: 'New Corp',
      contact_name: 'Jane', email: 'jane@newcorp.com',
      phone: '', vat_number: '', address: '',
      is_active: true, created_at: '2025-01-01T00:00:00Z',
    }

    let capturedBody: unknown = null
    server.use(
      http.post('/api/customers', async ({ request }) => {
        capturedBody = await request.json()
        return HttpResponse.json({ data: mockCustomer })
      }),
    )

    const result = await CustomersApis.createCustomerMutationOpts.mutationFn(customerData)
    expect(capturedBody).toEqual(customerData)
    expect(result).toEqual(mockCustomer)
  })

  it('customerQueryOpts calls GET /api/customers/:id', async () => {
    const mockCustomerWithContracts = {
      customer: {
        id: 'cust1', organization_id: 'o1', company_name: 'Acme Corp',
        contact_name: 'John', email: 'john@acme.com',
        phone: '+123', vat_number: 'VAT123', address: '123 Main St',
        is_active: true, created_at: '2025-01-01T00:00:00Z',
      },
      linked_contracts: [
        {
          id: 'c1', name: 'Contract A', km_rate: 0.5, currency: 'EUR',
          governance_model: 'creator_controlled' as const,
          created_by_org_id: 'o1', is_shared: false, is_active: true,
          created_at: '2025-01-01T00:00:00Z',
        },
      ],
    }
    server.use(
      http.get('/api/customers/cust1', () =>
        HttpResponse.json({ data: mockCustomerWithContracts }),
      ),
    )

    const opts = CustomersApis.customerQueryOpts('cust1')
    const result = await opts.queryFn()
    expect(result).toEqual(mockCustomerWithContracts)
  })
})
