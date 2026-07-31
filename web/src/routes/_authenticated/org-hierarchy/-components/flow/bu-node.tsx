import { memo } from "react";
import { Handle, type NodeProps, Position } from "@xyflow/react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  ChevronDown,
  ChevronRight,
  Edit2,
  MoreVertical,
  Plus,
  Trash2,
  Users,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { BUNodeData } from "./bu-node-data";
import type { UnitMember } from "@/types/unit.ts";

function MemberRow({ member }: { member: UnitMember }) {
  const initials = member.user_name
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className="flex items-center gap-2 py-1">
      <Avatar data-size="sm">
        <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-medium truncate">{member.user_name}</p>
        <p className="text-[10px] text-muted-foreground truncate">
          {member.role}
        </p>
      </div>
      {member.is_primary && (
        <Badge variant="outline" className="text-[10px] px-1 py-0 shrink-0">
          Primary
        </Badge>
      )}
    </div>
  );
}

function BUNodeComponent(props: NodeProps) {
  const {
    unit,
    memberCount,
    totalMemberCount,
    childCount,
    isCollapsed,
    viewMode,
    members,
    membersLoading,
    isDragHovered,
    onAddSubUnit,
    onEdit,
    onDelete,
    onToggleCollapse,
  } = props.data as BUNodeData;

  const levelLabel =
    unit.hierarchy_level === 0 ? "Root" : `Level ${unit.hierarchy_level}`;
  const hasChildren = (childCount ?? 0) > 0;
  const isMembersView = viewMode === "members";

  return (
    <Card
      className={`w-[250px] cursor-pointer transition-all relative ${
        props.selected ? "ring-2 ring-primary ring-offset-2" : ""
      } ${
        isDragHovered
          ? "border-blue-500 border-2 shadow-lg shadow-blue-500/20"
          : ""
      }`}
    >
      <div
        className="absolute top-2 right-2 z-10"
        onClick={(e) => e.stopPropagation()}
      >
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="ghost" className="h-6 w-6 p-0 hover:bg-muted" />
            }
          >
            <MoreVertical className="h-4 w-4 text-muted-foreground" />
            <span className="sr-only">Open menu</span>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-[160px]">
            <DropdownMenuItem onClick={() => onAddSubUnit?.(unit.id)}>
              <Plus className="mr-2 h-4 w-4" />
              <span>Add Sub-unit</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onEdit?.(unit.id)}>
              <Edit2 className="mr-2 h-4 w-4" />
              <span>Edit</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => onDelete?.(unit.id)}
              className="text-destructive focus:text-destructive focus:bg-destructive/10"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              <span>Delete</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Handle
        type="target"
        position={Position.Top}
        className="!bg-muted-foreground"
      />
      <CardContent className="p-3 space-y-1">
        <div className="flex items-center justify-between gap-2 pr-6">
          <div className="flex items-center gap-1 min-w-0">
            {hasChildren && onToggleCollapse && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onToggleCollapse(unit.id);
                }}
                className="shrink-0 p-0.5 hover:bg-muted rounded"
              >
                {isCollapsed ? (
                  <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                ) : (
                  <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                )}
              </button>
            )}
            <h3 className="font-semibold text-sm truncate">{unit.name}</h3>
          </div>
        </div>
        <div className="flex items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground font-mono">{unit.code}</p>
          <div className="flex items-center gap-1">
            {(memberCount ?? 0) > 0 && (
              <Badge
                variant="outline"
                className="text-xs shrink-0 gap-1 px-1.5"
              >
                <Users className="h-3 w-3" />
                {isCollapsed ? (totalMemberCount ?? memberCount) : memberCount}
              </Badge>
            )}
            <Badge variant="secondary" className="text-xs shrink-0">
              {levelLabel}
            </Badge>
          </div>
        </div>

        {isCollapsed && hasChildren && (
          <div className="flex items-center gap-1 mt-0.5">
            <Badge variant="outline" className="text-xs px-1.5">
              +{childCount} sub-unit{childCount !== 1 ? "s" : ""}
            </Badge>
            {(totalMemberCount ?? 0) > (memberCount ?? 0) && (
              <Badge variant="outline" className="text-xs px-1.5">
                +{(totalMemberCount ?? 0) - (memberCount ?? 0)} members
              </Badge>
            )}
          </div>
        )}

        {isMembersView && !isCollapsed && (
          <div className="mt-2 pt-2 border-t">
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1">
              Members ({memberCount ?? 0})
            </p>
            {membersLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-3/4" />
              </div>
            ) : members && members.length > 0 ? (
              <div className="space-y-0.5 max-h-32 overflow-y-auto">
                {members.map((m) => (
                  <MemberRow key={m.id} member={m} />
                ))}
              </div>
            ) : (
              <p className="text-[10px] text-muted-foreground italic">
                No members
              </p>
            )}
          </div>
        )}
      </CardContent>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-muted-foreground"
      />
    </Card>
  );
}

export const BUNode = memo(BUNodeComponent);
