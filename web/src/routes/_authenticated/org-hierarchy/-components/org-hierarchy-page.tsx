import * as React from 'react'
import {useCallback, useMemo, useState} from 'react'
import {Background, Controls, type Edge, MiniMap, type Node, ReactFlow, ReactFlowProvider,} from '@xyflow/react'
import {useSuspenseQuery} from '@tanstack/react-query'
import {unitTreeQueryOpts,} from '@/api/units.ts'
import type {Unit, UnitTreeNode} from '@/types/unit.ts'
import {findNode, flattenTree, getDescendants} from './utils/tree-utils'
import {type BUNodeData, getLayoutElements} from './flow/dagre-layout'
import {BUNode} from './flow/bu-node'
import {OrgChartToolbar} from './org-chart-toolbar'
import {UnitDetailPanel} from './dialogs/unit-detail-panel'
import {UnitFormDialog} from './dialogs/unit-form-dialog'
import {DeleteConfirmDialog} from './dialogs/delete-confirm-dialog'
import {useTheme} from "@/components/theme-provider.tsx";
import {Header} from "@/components/layout/header.tsx";
import {Body} from "@/components/layout/body.tsx";
import {OrgHierarchyProvider, useOrgHierarchy} from '../-context/org-hierarchy-context'

const nodeTypes = {bu: BUNode}

function buildNodes(units: Unit[], handlers: {
  onAddSubUnit: (parentId: string) => void
  onEdit: (id: string) => void
  onDelete: (id: string) => void
}): Node<BUNodeData>[] {
  return units.map((unit) => ({
    id: unit.id,
    type: 'bu',
    position: {x: 0, y: 0},
    data: {
      unit,
      onAddSubUnit: handlers.onAddSubUnit,
      onEdit: handlers.onEdit,
      onDelete: handlers.onDelete
    },
    draggable: true,
  }))
}

function buildEdges(units: Unit[]): Edge[] {
  const nodeIds = new Set(units.map((u) => u.id))
  return units
    .filter((u) => u.parent_unit_id && nodeIds.has(u.parent_unit_id))
    .map((u) => ({
      id: `e-${u.parent_unit_id}-${u.id}`,
      source: u.parent_unit_id!,
      target: u.id,
      type: 'smoothstep' as const,
      animated: true,
    }))
}

function computeVisibleIds(
  query: string,
  allUnits: Unit[],
  tree: UnitTreeNode[],
  allUnitsMap: Map<string, Unit>
): Set<string> | null {
  const lowerQuery = query.toLowerCase()
  if (!lowerQuery) return null

  const matchingIds = new Set<string>()
  for (const u of allUnits) {
    if (u.name.toLowerCase().includes(lowerQuery) || u.code.toLowerCase().includes(lowerQuery)) {
      matchingIds.add(u.id)
      const nodeInTree = findNode(tree, u.id)
      if (nodeInTree) {
        for (const id of getDescendants(nodeInTree)) matchingIds.add(id)
      }
      let current = u.parent_unit_id
      while (current) {
        matchingIds.add(current)
        const parent = allUnitsMap.get(current)
        current = parent?.parent_unit_id
      }
    }
  }
  return matchingIds
}

function OrgHierarchyRoot({children}: { children: React.ReactNode }) {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedUnit, setSelectedUnit] = useState<Unit | null>(null)
  const [showFormDialog, setShowFormDialog] = useState(false)
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create')
  const [editingUnit, setEditingUnit] = useState<Unit | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const {data: tree} = useSuspenseQuery(unitTreeQueryOpts)

  const allUnits = useMemo(() => flattenTree(tree), [tree])
  const allUnitsMap = useMemo(() => new Map(allUnits.map((u) => [u.id, u])), [allUnits])

  const handleAddUnit = useCallback((parentId: string | null = null) => {
    setFormMode('create')
    setEditingUnit(parentId ? {parent_unit_id: parentId} as Unit : null)
    setShowFormDialog(true)
  }, [])

  const handleEditDirect = useCallback((id: string) => {
    const unit = allUnitsMap.get(id)
    if (!unit) return
    setFormMode('edit')
    setEditingUnit(unit)
    setShowFormDialog(true)
  }, [allUnitsMap])

  const handleDeleteDirect = useCallback((id: string) => {
    const unit = allUnitsMap.get(id)
    if (!unit) return
    setSelectedUnit(unit)
    setShowDeleteConfirm(true)
  }, [allUnitsMap])

  const handleEdit = useCallback(() => {
    if (!selectedUnit) return
    setFormMode('edit')
    setEditingUnit(selectedUnit)
    setShowFormDialog(true)
  }, [selectedUnit])

  const value = useMemo(() => ({
    searchQuery, setSearchQuery: setSearchQuery, selectedUnit, setSelectedUnit,
    showFormDialog, setShowFormDialog,
    formMode, setFormMode, editingUnit, setEditingUnit,
    showDeleteConfirm, setShowDeleteConfirm, handleAddUnit,
    handleEditDirect, handleDeleteDirect, handleEdit
  }), [searchQuery, setSearchQuery, selectedUnit, setSelectedUnit, showFormDialog, setShowFormDialog, formMode, setFormMode, editingUnit, setEditingUnit, showDeleteConfirm, setShowDeleteConfirm, handleAddUnit, handleEditDirect, handleDeleteDirect, handleEdit])

  return <OrgHierarchyProvider value={value}>{children}</OrgHierarchyProvider>
}

function OrgHierarchyFlow() {
  const {theme} = useTheme()
  const {data: tree} = useSuspenseQuery(unitTreeQueryOpts)

  const {
    searchQuery,
    setSelectedUnit,
    handleAddUnit,
    handleEditDirect,
    handleDeleteDirect,
  } = useOrgHierarchy()

  const allUnits = useMemo(() => flattenTree(tree), [tree])
  const allUnitsMap = useMemo(() => new Map(allUnits.map((u) => [u.id, u])), [allUnits])

  const {nodes, edges} = useMemo(() => {
    const initialNodes = buildNodes(allUnits, {
      onAddSubUnit: handleAddUnit,
      onEdit: handleEditDirect,
      onDelete: handleDeleteDirect
    })
    const initialEdges = buildEdges(allUnits)
    const visibleIds = computeVisibleIds(searchQuery, allUnits, tree, allUnitsMap)
    if (visibleIds) {
      for (const n of initialNodes) {
        if (!visibleIds.has(n.id)) n.hidden = true
      }
    }
    return getLayoutElements(initialNodes, initialEdges)
  }, [allUnits, tree, allUnitsMap, searchQuery, handleAddUnit, handleEditDirect, handleDeleteDirect])

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node<BUNodeData>) => {
      const unit = allUnitsMap.get(node.id)
      if (unit) {
        setSelectedUnit(unit)
      }
    },
    [allUnitsMap, setSelectedUnit]
  )

  return (
    <div className="flex-1 w-full relative">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodeClick={onNodeClick}
        nodeTypes={nodeTypes}
        colorMode={theme}
        fitView
        attributionPosition="bottom-left"
      >
        <Background/>
        <MiniMap/>
        <Controls/>
      </ReactFlow>
    </div>
  )
}

function OrgHierarchyDialogs() {
  return (
    <>
      <UnitDetailPanel/>
      <UnitFormDialog/>
      <DeleteConfirmDialog/>
    </>
  )
}

const OrgHierarchy = {
  Root: OrgHierarchyRoot,
  Flow: OrgHierarchyFlow,
  Dialogs: OrgHierarchyDialogs,
  Toolbar: OrgChartToolbar
}

export function OrgHierarchyPage() {
  return (
    <OrgHierarchy.Root>
      <ReactFlowProvider>
        <Header className="border-">
          <OrgHierarchy.Toolbar/>
        </Header>
        <Body>
          <div className="h-full flex flex-col relative w-full bg-background">
            <OrgHierarchy.Flow/>
            <OrgHierarchy.Dialogs/>
          </div>
        </Body>
      </ReactFlowProvider>
    </OrgHierarchy.Root>
  )
}