import {createFileRoute} from '@tanstack/react-router'
import {CustomerDetail} from './-components/customer-detail'
import {CustomersApis} from '@/api/customers'

export const Route = createFileRoute('/_authenticated/customers/$id')({
  component: CustomerDetail,
  loader: ({context: {client}, params: {id}}) => (
    client.ensureQueryData(CustomersApis.customerQueryOpts(id))
  ),
})
