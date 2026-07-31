import * as React from "react";
import { useCallback, useMemo } from "react";
import {
  Background,
  Controls,
  type Edge,
  MiniMap,
  type Node,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import { useQueries, useSuspenseQuery } from "@tanstack/react-query";
import { unitMembersQueryOpts, unitTreeQueryOpts } from "@/api/units.ts";
import type { Unit, UnitMember, UnitTreeNode } from "@/types/unit.ts";
import {
  findNode,
  flattenTree,
  getDescendantIds,
  getDescendants,
} from "./utils/tree-utils";
import { getLayoutElements } from "./flow/dagre-layout";
import { BUNode } from "./flow/bu-node";
import { type BUNodeData, treeNodeToNodeData } from "./flow/bu-node-data";
import { OrgChartToolbar } from "./org-chart-toolbar";
import { UnitDetailPanel } from "./dialogs/unit-detail-panel";
import { UnitFormDialog } from "./dialogs/unit-form-dialog";
import { DeleteConfirmDialog } from "./dialogs/delete-confirm-dialog";
import { ReparentConfirmDialog } from "./dialogs/reparent-confirm-dialog";
import { useTheme } from "@/components/theme-provider.tsx";
import { Header } from "@/components/layout/header.tsx";
import { Body } from "@/components/layout/body.tsx";
import {
  OrgHierarchyProvider,
  useOrgHierarchyStore,
} from "../-context/org-hierarchy-context";

const nodeTypes = { bu: BUNode } as const;

function buildNodes(
  tree: UnitTreeNode[],
  collapsedIds: Set<string>,
  viewMode: "tree" | "members",
  membersMap: Map<string, UnitMember[]>,
  membersLoadingSet: Set<string>,
  handlers: {
    onAddSubUnit: (parentId: string) => void;
    onEdit: (id: string) => void;
    onDelete: (id: string) => void;
    onToggleCollapse: (id: string) => void;
  }
): Node<BUNodeData>[] {
  const result: Node<BUNodeData>[] = [];

  function walk(nodes: UnitTreeNode[]) {
    for (const node of nodes) {
      result.push({
        id: node.unit.id,
        type: "bu",
        position: { x: 0, y: 0 },
        data: treeNodeToNodeData(
          node,
          handlers,
          collapsedIds,
          viewMode,
          membersMap,
          membersLoadingSet
        ),
        draggable: false,
      });
      if (!collapsedIds.has(node.unit.id) && node.children) {
        walk(node.children);
      }
    }
  }

  walk(tree);
  return result;
}

function buildEdges(tree: UnitTreeNode[], collapsedIds: Set<string>): Edge[] {
  const result: Edge[] = [];

  function walk(nodes: UnitTreeNode[]) {
    for (const node of nodes) {
      if (collapsedIds.has(node.unit.id)) continue;
      if (node.children) {
        for (const child of node.children) {
          result.push({
            id: `e-${node.unit.id}-${child.unit.id}`,
            source: node.unit.id,
            target: child.unit.id,
            type: "smoothstep" as const,
            animated: true,
            selectable: true,
          });
        }
        walk(node.children);
      }
    }
  }

  walk(tree);
  return result;
}

function computeVisibleIds(
  query: string,
  allUnits: Unit[],
  tree: UnitTreeNode[],
  allUnitsMap: Map<string, Unit>
): Set<string> | null {
  const lowerQuery = query.toLowerCase();
  if (!lowerQuery) return null;

  const matchingIds = new Set<string>();
  for (const u of allUnits) {
    if (
      u.name.toLowerCase().includes(lowerQuery) ||
      u.code.toLowerCase().includes(lowerQuery)
    ) {
      matchingIds.add(u.id);
      const nodeInTree = findNode(tree, u.id);
      if (nodeInTree) {
        for (const id of getDescendants(nodeInTree)) matchingIds.add(id);
      }
      let current = u.parent_unit_id;
      while (current) {
        matchingIds.add(current);
        const parent = allUnitsMap.get(current);
        current = parent?.parent_unit_id;
      }
    }
  }
  return matchingIds;
}

function OrgHierarchyRoot({ children }: { children: React.ReactNode }) {
  return <OrgHierarchyProvider>{children}</OrgHierarchyProvider>;
}

function OrgHierarchyFlow() {
  const { theme } = useTheme();
  const { data: tree } = useSuspenseQuery(unitTreeQueryOpts);

  const viewMode = useOrgHierarchyStore((s) => s.viewMode);
  const collapsedIds = useOrgHierarchyStore((s) => s.collapsedIds);
  const searchQuery = useOrgHierarchyStore((s) => s.searchQuery);

  const toggleCollapsed = useOrgHierarchyStore((s) => s.toggleCollapsed);
  const setSelectedUnit = useOrgHierarchyStore((s) => s.setSelectedUnit);
  const addUnit = useOrgHierarchyStore((s) => s.addUnit);
  const editUnit = useOrgHierarchyStore((s) => s.editUnit);
  const deleteUnit = useOrgHierarchyStore((s) => s.deleteUnit);
  const reparentUnit = useOrgHierarchyStore((s) => s.reparentUnit);

  const allUnits = useMemo(() => flattenTree(tree), [tree]);
  const allUnitsMap = useMemo(
    () => new Map(allUnits.map((u) => [u.id, u])),
    [allUnits]
  );

  const actions = useMemo(
    () => ({
      addUnit,
      editUnit: (id: string) => {
        const u = allUnitsMap.get(id);
        if (u) editUnit(u);
      },
      deleteUnit: (id: string) => {
        const u = allUnitsMap.get(id);
        if (u) deleteUnit(u);
      },
      reparentUnit: (unitId: string, newParentId: string | null) => {
        const dragUnit = allUnitsMap.get(unitId) ?? null;
        const targetUnit = newParentId
          ? (allUnitsMap.get(newParentId) ?? null)
          : null;
        reparentUnit(dragUnit, targetUnit);
      },
    }),
    [allUnitsMap, addUnit, editUnit, deleteUnit, reparentUnit]
  );

  const visibleUnitIds = useMemo(() => {
    const ids = new Set<string>();

    function walk(nodes: UnitTreeNode[]) {
      for (const node of nodes) {
        ids.add(node.unit.id);
        if (!collapsedIds.has(node.unit.id) && node.children) {
          walk(node.children);
        }
      }
    }

    walk(tree);
    return ids;
  }, [tree, collapsedIds]);

  const memberQueries = useQueries({
    queries:
      viewMode === "members"
        ? allUnits
            .filter((u) => visibleUnitIds.has(u.id))
            .map((u) => ({
              ...unitMembersQueryOpts(u.id),
              staleTime: 60_000,
            }))
        : [],
  });

  const membersMap = useMemo(() => {
    const map = new Map<string, UnitMember[]>();
    const visibleList =
      viewMode === "members"
        ? allUnits.filter((u) => visibleUnitIds.has(u.id))
        : [];
    visibleList.forEach((u, i) => {
      const q = memberQueries[i];
      if (q?.data) map.set(u.id, q.data);
    });
    return map;
  }, [viewMode, allUnits, visibleUnitIds, memberQueries]);

  const membersLoadingSet = useMemo(() => {
    const set = new Set<string>();
    const visibleList =
      viewMode === "members"
        ? allUnits.filter((u) => visibleUnitIds.has(u.id))
        : [];
    visibleList.forEach((u, i) => {
      const q = memberQueries[i];
      if (q?.isFetching) set.add(u.id);
    });
    return set;
  }, [viewMode, allUnits, visibleUnitIds, memberQueries]);

  const { nodes, edges } = useMemo(() => {
    const initialNodes = buildNodes(
      tree,
      collapsedIds,
      viewMode,
      membersMap,
      membersLoadingSet,
      {
        onAddSubUnit: actions.addUnit,
        onEdit: actions.editUnit,
        onDelete: actions.deleteUnit,
        onToggleCollapse: toggleCollapsed,
      }
    );
    const initialEdges = buildEdges(tree, collapsedIds);
    const visibleIds = computeVisibleIds(
      searchQuery,
      allUnits,
      tree,
      allUnitsMap
    );
    if (visibleIds) {
      for (const n of initialNodes) {
        if (!visibleIds.has(n.id)) n.hidden = true;
      }
    }
    return getLayoutElements(initialNodes, initialEdges);
  }, [
    tree,
    collapsedIds,
    viewMode,
    membersMap,
    membersLoadingSet,
    allUnits,
    allUnitsMap,
    searchQuery,
    actions.addUnit,
    actions.editUnit,
    actions.deleteUnit,
    toggleCollapsed,
  ]);

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const unit = allUnitsMap.get(node.id);
      if (unit) {
        setSelectedUnit(unit);
      }
    },
    [allUnitsMap, setSelectedUnit]
  );

  const onConnect = useCallback(
    (params: { source: string; target: string }) => {
      const sourceUnit = allUnitsMap.get(params.source);
      const targetUnit = allUnitsMap.get(params.target);
      if (sourceUnit && targetUnit) {
        reparentUnit(sourceUnit, targetUnit);
      }
    },
    [allUnitsMap, reparentUnit]
  );

  const isValidConnection = useCallback(
    (connection: { source: string; target: string }) => {
      const descendantIds = getDescendantIds(connection.source, tree);
      return !descendantIds.has(connection.target);
    },
    [tree]
  );

  return (
    <div className="flex-1 w-full relative">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodeClick={onNodeClick}
        onConnect={onConnect}
        isValidConnection={isValidConnection}
        nodeTypes={nodeTypes}
        colorMode={theme}
        fitView
        attributionPosition="bottom-left"
      >
        <Background />
        <MiniMap />
        <Controls />
      </ReactFlow>
    </div>
  );
}

function OrgHierarchyDialogs() {
  return (
    <>
      <UnitDetailPanel />
      <UnitFormDialog />
      <DeleteConfirmDialog />
      <ReparentConfirmDialog />
    </>
  );
}

const OrgHierarchy = {
  Root: OrgHierarchyRoot,
  Flow: OrgHierarchyFlow,
  Dialogs: OrgHierarchyDialogs,
  Toolbar: OrgChartToolbar,
};

export function OrgHierarchyPage() {
  return (
    <OrgHierarchy.Root>
      <ReactFlowProvider>
        <Header>
          <OrgHierarchy.Toolbar/>
        </Header>
        <Body>
          <div className="h-full flex flex-col relative w-full bg-background">
            <OrgHierarchy.Flow />
            <OrgHierarchy.Dialogs />
          </div>
        </Body>
      </ReactFlowProvider>
    </OrgHierarchy.Root>
  );
}
