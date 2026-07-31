import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/api.ts";

export interface Customer {
  id: string;
  organization_id: string;
  company_name: string;
  contact_name: string;
  email: string;
  phone: string;
  vat_number: string;
  address: string;
  is_active: boolean;
  is_internal: boolean;
  created_at: string;
}

export interface LinkedContract {
  id: string;
  name: string;
  km_rate: number;
  currency: string;
  governance_model: string;
  created_by_org_id: string;
  is_shared: boolean;
  is_active: boolean;
  created_at: string;
}

export interface CustomerWithContracts {
  customer: Customer;
  linked_contracts: LinkedContract[];
}

export interface CreateCustomerRequest {
  company_name: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  vat_number?: string;
  address?: string;
}

export interface UpdateCustomerRequest {
  company_name?: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  vat_number?: string;
  address?: string;
  is_active?: boolean;
}

function customersQueryKey() {
  return ["customers"] as const;
}

function customerQueryKey(id: string) {
  return ["customers", id] as const;
}

export function customersQueryOpts(search?: string) {
  const queryFn = () => {
    const path = search
      ? `/customers?search=${encodeURIComponent(search)}`
      : "/customers";
    return api<Customer[]>(path);
  };
  return queryOptions<Customer[]>({
    queryKey: search ? ["customers", search] : ["customers"],
    queryFn,
  });
}

export function customerQueryOpts(id: string) {
  return queryOptions({
    queryKey: customerQueryKey(id),
    queryFn: () => api<CustomerWithContracts>(`/customers/${id}`),
    enabled: !!id,
  });
}

export const createCustomerMutationOpts = mutationOptions({
  mutationFn: (data: CreateCustomerRequest) =>
    api<Customer>("/customers", { method: "POST", body: JSON.stringify(data) }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["customers"] });
  },
});

export const updateCustomerMutationOpts = mutationOptions({
  mutationFn: ({ id, data }: { id: string; data: UpdateCustomerRequest }) =>
    api<Customer>(`/customers/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, variables, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["customers"] });
    client.invalidateQueries({ queryKey: ["customers", variables.id] });
  },
});

export const deleteCustomerMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<void>(`/customers/${id}`, { method: "DELETE" }),
  onSuccess: (_, id, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["customers"] });
    client.invalidateQueries({ queryKey: ["customers", id] });
  },
});

export const CustomersApis = {
  customersQueryOpts,
  customerQueryOpts,
  createCustomerMutationOpts,
  updateCustomerMutationOpts,
  deleteCustomerMutationOpts,
};
