import * as React from "react";
import { createStore, useStore } from "zustand";
import type { Customer } from "@/api/customers";

/* oxlint-disable react-refresh-only-export-components */

export interface CustomersState {
  searchQuery: string;
  formOpen: boolean;
  formMode: "create" | "edit";
  editingCustomer: Customer | null;
  selectedCustomer: Customer | null;
  deleteOpen: boolean;
}

export interface CustomersActions {
  setSearchQuery: (query: string) => void;
  setFormOpen: (open: boolean) => void;
  setFormMode: (mode: "create" | "edit") => void;
  setEditingCustomer: (customer: Customer | null) => void;
  setSelectedCustomer: (customer: Customer | null) => void;
  setDeleteOpen: (open: boolean) => void;

  openCreate: () => void;
  openEdit: (customer: Customer) => void;
  openDelete: (customer: Customer) => void;
}

export type CustomersStore = CustomersState & CustomersActions;

export const createCustomersStore = () =>
  createStore<CustomersStore>()((set) => ({
    searchQuery: "",
    formOpen: false,
    formMode: "create",
    editingCustomer: null,
    selectedCustomer: null,
    deleteOpen: false,

    setSearchQuery: (query) => set({ searchQuery: query }),
    setFormOpen: (open) => set({ formOpen: open }),
    setFormMode: (mode) => set({ formMode: mode }),
    setEditingCustomer: (customer) => set({ editingCustomer: customer }),
    setSelectedCustomer: (customer) => set({ selectedCustomer: customer }),
    setDeleteOpen: (open) => set({ deleteOpen: open }),

    openCreate: () =>
      set({ formMode: "create", editingCustomer: null, formOpen: true }),
    openEdit: (customer) =>
      set({ formMode: "edit", editingCustomer: customer, formOpen: true }),
    openDelete: (customer) =>
      set({ selectedCustomer: customer, deleteOpen: true }),
  }));

const CustomersContext = React.createContext<ReturnType<
  typeof createCustomersStore
> | null>(null);

export function useCustomersStore<T>(
  selector: (state: CustomersStore) => T
): T {
  const store = React.useContext(CustomersContext);
  if (!store) {
    throw new Error("useCustomersStore must be used within CustomersProvider");
  }
  return useStore(store, selector);
}

export function CustomersProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [store] = React.useState(() => createCustomersStore());
  return (
    <CustomersContext.Provider value={store}>
      {children}
    </CustomersContext.Provider>
  );
}
