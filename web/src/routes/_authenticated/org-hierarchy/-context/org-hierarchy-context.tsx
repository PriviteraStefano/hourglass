import * as React from 'react'
import {Unit} from "@/types/unit.ts";

export interface OrgHierarchyContextValue {
  searchQuery: string
  setSearchQuery: (query: string) => void
  selectedUnit: Unit | null
  setSelectedUnit: (unit: Unit | null) => void
  showFormDialog: boolean
  setShowFormDialog: (show: boolean) => void
  formMode: 'create' | 'edit'
  setFormMode: (mode: 'create' | 'edit') => void
  editingUnit: Unit | null
  setEditingUnit: (unit: Unit | null) => void
  showDeleteConfirm: boolean
  setShowDeleteConfirm: (show: boolean) => void

  handleAddUnit: (parentId?: string | null) => void
  handleEditDirect: (id: string) => void
  handleDeleteDirect: (id: string) => void
  handleEdit: () => void
}

const OrgHierarchyContext = React.createContext<OrgHierarchyContextValue | null>(null)

export function useOrgHierarchy() {
  const context = React.useContext(OrgHierarchyContext)
  if (!context) {
    throw new Error('useOrgHierarchy must be used within OrgHierarchy.Root')
  }
  return context
}

export const OrgHierarchyProvider = OrgHierarchyContext.Provider

