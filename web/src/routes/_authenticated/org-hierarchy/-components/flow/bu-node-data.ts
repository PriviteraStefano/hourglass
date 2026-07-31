import type { Unit, UnitMember, UnitTreeNode } from "@/types/unit.ts";

export type BUNodeData = {
  unit: Unit;
  memberCount?: number;
  totalMemberCount?: number;
  childCount?: number;
  isCollapsed?: boolean;
  viewMode?: "tree" | "members";
  members?: UnitMember[];
  membersLoading?: boolean;
  isDragHovered?: boolean;
  onAddSubUnit?: (parentId: string) => void;
  onEdit?: (id: string) => void;
  onDelete?: (id: string) => void;
  onToggleCollapse?: (id: string) => void;
} & Record<string, unknown>;

export function treeNodeToNodeData(
  node: UnitTreeNode,
  handlers: {
    onAddSubUnit?: (parentId: string) => void;
    onEdit?: (id: string) => void;
    onDelete?: (id: string) => void;
    onToggleCollapse?: (id: string) => void;
  },
  collapsedIds: Set<string>,
  viewMode: "tree" | "members",
  membersMap?: Map<string, UnitMember[]>,
  membersLoadingSet?: Set<string>,
  dragHoverId?: string | null
): BUNodeData {
  const unitId = node.unit.id;
  return {
    unit: node.unit,
    memberCount: node.member_count,
    totalMemberCount: node.total_member_count,
    childCount: node.children?.length ?? 0,
    isCollapsed: collapsedIds.has(unitId),
    viewMode,
    members: membersMap?.get(unitId),
    membersLoading: membersLoadingSet?.has(unitId),
    isDragHovered: dragHoverId === unitId,
    ...handlers,
  };
}
