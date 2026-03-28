import {mutationOptions, queryOptions} from '@tanstack/react-query'
import {toast} from 'sonner'
import {api} from '@/src/lib/api.ts'
import type {Contract} from '@/src/types/models'
import type {CreateContractRequest} from "@/src/types";

function contractsQueryKey(scope: 'owned' | 'adopted' | 'all') {
  return ['contracts', scope] as const
}

function contractQueryKey(id: string) {
  return ['contracts', id] as const
}

function contractsQueryOpts(scope: 'owned' | 'adopted' | 'all' = 'owned') {
  return queryOptions({
    queryKey: contractsQueryKey(scope),
    queryFn: () => api<Contract[]>(`/contracts?scope=${scope}`),
  })
}

function contractQueryOpts(id: string) {
  return queryOptions({
    queryKey: contractQueryKey(id),
    queryFn: () => api<Contract>(`/contracts/${id}`),
    enabled: !!id,
  })
}

const createContractMutationOpts = mutationOptions({
  mutationFn: (data: CreateContractRequest) =>
    api<Contract>('/contracts', {method: 'POST', body: JSON.stringify(data)}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['contracts']})
    toast.success('Contract created')
  },
})

const adoptContractMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/contracts/${id}/adopt`, {method: 'POST'}),
  onSuccess: (_, __, {client}) => {
    client.invalidateQueries({queryKey: ['contracts']})
    toast.success('Contract added to your organization')
  },
})

export const ContractsApis = {
  contractsQueryOpts,
  contractQueryOpts,
  createContractMutationOpts,
  adoptContractMutationOpts,
}