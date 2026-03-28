import {createFileRoute} from '@tanstack/react-router'
import {ContractDetail} from './-components/contract-detail'
import {ContractsApis} from "@/api/contracts.ts";

export const Route = createFileRoute('/_authenticated/contracts/$id/')({
  component: ContractDetail,
  // validateSearch: (search: Record<string, unknown>) => ({
  //   from: (search.from as 'owned' | 'adopted' | 'all') || 'owned',
  // }),
  loader: ({context: {client}, params: {id}}) => (
    client.ensureQueryData(ContractsApis.contractQueryOpts(id))
  )
})
