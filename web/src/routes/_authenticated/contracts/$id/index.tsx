import {createFileRoute} from '@tanstack/react-router'
import {ContractDetail} from './-components/contract-detail'
import {ContractsApis} from "@/api/contracts.ts";
import {tabsSchema} from "@/routes/_authenticated/contracts/-components/contract-list.tsx";

export const Route = createFileRoute('/_authenticated/contracts/$id/')({
  component: ContractDetail,
  validateSearch: (search: Record<string, unknown>) => ({
    from: tabsSchema.safeParse(search.from).data ?? 'owned',
  }),
  loader: ({context: {client}, params: {id}}) => (
    client.ensureQueryData(ContractsApis.contractQueryOpts(id))
  )
})
