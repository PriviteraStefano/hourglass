import * as React from "react";
import { createStore, useStore } from "zustand";
import type { Unit } from "@/types/unit.ts";

/* oxlint-disable react-refresh-only-export-components */

export interface OrgHierarchyState {
  viewMode: "tree" | "members";
  collapsedIds: Set<string>;
  searchQuery: string;
  selectedUnit: Unit | null;
  formOpen: boolean;
  formMode: "create" | "edit";
  editingUnit: Unit | null;
  deleteOpen: boolean;
  reparentTarget: Unit | null;
  draggingUnit: Unit | null;
}

export interface OrgHierarchyActions {
  setViewMode: (mode: "tree" | "members") => void;
  toggleCollapsed: (id: string) => void;
  setSearchQuery: (query: string) => void;
  setSelectedUnit: (unit: Unit | null) => void;
  setFormOpen: (open: boolean) => void;
  setFormMode: (mode: "create" | "edit") => void;
  setEditingUnit: (unit: Unit | null) => void;
  setDeleteOpen: (open: boolean) => void;
  setReparentTarget: (unit: Unit | null) => void;
  setDraggingUnit: (unit: Unit | null) => void;

  addUnit: (parentId?: string | null) => void;
  editUnit: (unit: Unit) => void;
  deleteUnit: (unit: Unit) => void;
  reparentUnit: (dragUnit: Unit | null, targetUnit: Unit | null) => void;
}

export type OrgHierarchyStore = OrgHierarchyState & OrgHierarchyActions;

export const createOrgHierarchyStore = () =>
  createStore<OrgHierarchyStore>()((set) => ({
    viewMode: "tree",
    collapsedIds: new Set(),
    searchQuery: "",
    selectedUnit: null,
    formOpen: false,
    formMode: "create",
    editingUnit: null,
    deleteOpen: false,
    reparentTarget: null,
    draggingUnit: null,

    setViewMode: (mode) => set({ viewMode: mode }),
    toggleCollapsed: (id) =>
      set((state) => {
        const next = new Set(state.collapsedIds);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return { collapsedIds: next };
      }),
    setSearchQuery: (query) => set({ searchQuery: query }),
    setSelectedUnit: (unit) => set({ selectedUnit: unit }),
    setFormOpen: (open) => set({ formOpen: open }),
    setFormMode: (mode) => set({ formMode: mode }),
    setEditingUnit: (unit) => set({ editingUnit: unit }),
    setDeleteOpen: (open) => set({ deleteOpen: open }),
    setReparentTarget: (unit) => set({ reparentTarget: unit }),
    setDraggingUnit: (unit) => set({ draggingUnit: unit }),

    addUnit: (parentId = null) =>
      set({
        formMode: "create",
        editingUnit: parentId ? ({ parent_unit_id: parentId } as Unit) : null,
        formOpen: true,
      }),
    editUnit: (unit) =>
      set({
        formMode: "edit",
        editingUnit: unit,
        formOpen: true,
      }),
    deleteUnit: (unit) =>
      set({
        selectedUnit: unit,
        deleteOpen: true,
      }),
    reparentUnit: (dragUnit, targetUnit) =>
      set({
        draggingUnit: dragUnit,
        reparentTarget: targetUnit,
      }),
  }));

const OrgHierarchyContext = React.createContext<ReturnType<
  typeof createOrgHierarchyStore
> | null>(null);

export function useOrgHierarchyStore<T>(
  selector: (state: OrgHierarchyStore) => T
): T {
  const store = React.useContext(OrgHierarchyContext);
  if (!store) {
    throw new Error(
      "useOrgHierarchyStore must be used within OrgHierarchy.Root"
    );
  }
  return useStore(store, selector);
}

export function OrgHierarchyProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [store] = React.useState(() => createOrgHierarchyStore());
  return (
    <OrgHierarchyContext.Provider value={store}>
      {children}
    </OrgHierarchyContext.Provider>
  );
}
