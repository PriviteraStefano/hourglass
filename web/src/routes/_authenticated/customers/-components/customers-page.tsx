import {useSuspenseQuery} from '@tanstack/react-query'
import {useNavigate} from '@tanstack/react-router'
import {SearchIcon, PlusIcon, Building2, Mail, Phone, MapPin} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {Badge} from '@/components/ui/badge'
import {InputGroup, InputGroupAddon, InputGroupInput} from '@/components/ui/input-group'
import {CustomersApis, type Customer} from '@/api/customers'
import {useCustomersStore, CustomersProvider} from '../-context/customers-context'
import {CustomerFormDialog} from './dialogs/customer-form-dialog'
import {DeleteConfirmDialog} from './dialogs/delete-confirm-dialog'

export function CustomersPage() {
  return (
    <CustomersProvider>
      <CustomersContent />
    </CustomersProvider>
  )
}

function CustomersContent() {
  const searchQuery = useCustomersStore(s => s.searchQuery)
  const setSearchQuery = useCustomersStore(s => s.setSearchQuery)
  const openCreate = useCustomersStore(s => s.openCreate)
  const openEdit = useCustomersStore(s => s.openEdit)

  const {data: customers} = useSuspenseQuery(CustomersApis.customersQueryOpts(searchQuery))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold">Customers</h1>
        <div className="flex items-center gap-4">
          <InputGroup>
            <InputGroupInput
              placeholder="Search customers..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            <InputGroupAddon>
              <SearchIcon className="h-4 w-4" />
            </InputGroupAddon>
          </InputGroup>
          <Button onClick={openCreate}>
            <PlusIcon className="w-4 h-4 mr-1" />
            Add Customer
          </Button>
        </div>
      </div>

      {customers.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {searchQuery ? 'No customers match your search' : 'No customers yet'}
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {customers.map((customer) => (
            <CustomerCard
              key={customer.id}
              customer={customer}
              onEdit={() => openEdit(customer)}
            />
          ))}
        </div>
      )}

      <CustomerFormDialog />
      <DeleteConfirmDialog />
    </div>
  )
}

function CustomerCard({
  customer,
  onEdit,
}: {
  customer: Customer
  onEdit: () => void
}) {
  const navigate = useNavigate()
  const openDelete = useCustomersStore(s => s.openDelete)

  return (
    <div
      className="border rounded-lg p-4 space-y-3 hover:bg-muted/50 transition-colors cursor-pointer"
      onClick={() => navigate({to: '/customers/$id', params: {id: customer.id}})}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <div className="bg-primary/10 p-2 rounded-lg">
            <Building2 className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-medium">
              {customer.company_name}
              {customer.is_internal && (
                <Badge variant="secondary" className="ml-2">Internal</Badge>
              )}
            </h3>
            {customer.contact_name && (
              <p className="text-sm text-muted-foreground">{customer.contact_name}</p>
            )}
          </div>
        </div>
        <Badge variant={customer.is_active ? 'default' : 'outline'} className={customer.is_active ? 'bg-green-500' : ''}>
          {customer.is_active ? 'Active' : 'Inactive'}
        </Badge>
      </div>

      <div className="space-y-1.5 text-sm">
        {customer.email && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Mail className="h-3.5 w-3.5" />
            <span className="truncate">{customer.email}</span>
          </div>
        )}
        {customer.phone && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Phone className="h-3.5 w-3.5" />
            <span>{customer.phone}</span>
          </div>
        )}
        {customer.address && (
          <div className="flex items-center gap-2 text-muted-foreground">
            <MapPin className="h-3.5 w-3.5" />
            <span className="truncate">{customer.address}</span>
          </div>
        )}
      </div>

      <div className="flex gap-2 pt-2">
        <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); onEdit(); }}>
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={(e) => { e.stopPropagation(); openDelete(customer); }}
          className="text-destructive hover:text-destructive"
        >
          Delete
        </Button>
      </div>
    </div>
  )
}