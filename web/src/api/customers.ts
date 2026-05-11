import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {api} from '@/lib/api.ts'

export interface Customer {
  id: string
  name: string
  created_at: string
}

function customersQueryKey() {
  return ['customers'] as const
}

export function customersQueryOpts() {
  return queryOptions({
    queryKey: customersQueryKey(),
    queryFn: () => api<Customer[]>(`/customers`),
  })
}

export function createCustomerMutationOpts() {
  return mutationOptions({
    mutationFn: (data: { name: string }) =>
      api<Customer>('/customers', {method: 'POST', body: JSON.stringify(data)}),
  })
}

export const CustomersApis = {
  customersQueryOpts,
  createCustomerMutationOpts,
}