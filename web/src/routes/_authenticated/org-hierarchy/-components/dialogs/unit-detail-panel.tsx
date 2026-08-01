import { type ReactNode, useMemo, useState } from "react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  ChevronDown,
  ChevronRight,
  Edit,
  Plus,
  Trash2,
  Users,
  X as XIcon,
} from "lucide-react";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { useMutation, useQuery, useSuspenseQuery } from "@tanstack/react-query";
import {
  addUnitMemberMutationOpts,
  orgMembersQueryOpts,
  removeUnitMemberMutationOpts,
  unitMembersQueryOpts,
  unitTreeQueryOpts,
  updateUnitMemberMutationOpts,
} from "@/api/units.ts";
import {
  useOrgHierarchyStore,
} from "@/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx";
import { flattenTree } from "../utils/tree-utils";
import type { Unit, UnitMember, UnitTreeNode } from "@/types/unit.ts";
import { toast } from "sonner";

function useBreadcrumbs(unit: Unit | null) {
  const { data: tree } = useSuspenseQuery(unitTreeQueryOpts);
  const allUnits = useMemo(() => flattenTree(tree), [tree]);
  const unitsMap = useMemo(
    () => new Map(allUnits.map((u) => [u.id, u])),
    [allUnits]
  );

  return useMemo(() => {
    if (!unit) return [];
    const path: Unit[] = [unit];
    let current = unit.parent_unit_id;
    while (current) {
      const parent = unitsMap.get(current);
      if (!parent) break;
      path.unshift(parent);
      current = parent.parent_unit_id;
    }
    return path;
  }, [unit, unitsMap]);
}

function MemberRow({
  member,
  onRemove,
  onMakePrimary,
}: {
  member: UnitMember;
  onRemove?: () => void;
  onMakePrimary?: () => void;
}) {
  const initials = member.user_name
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className="flex items-center gap-2 py-1.5 group">
      <Avatar data-size="sm">
        <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium truncate">{member.user_name}</p>
        <p className="text-xs text-muted-foreground truncate">{member.role}</p>
      </div>
      <div className="flex items-center gap-1">
        {member.is_primary ? (
          <Badge variant="outline" className="text-[10px] px-1 py-0 shrink-0">
            Primary
          </Badge>
        ) : (
          onMakePrimary && (
            <Button
              variant="ghost"
              size="sm"
              className="h-6 text-[10px] px-1 opacity-0 group-hover:opacity-100 transition-opacity"
              onClick={onMakePrimary}
            >
              Make Primary
            </Button>
          )
        )}
        {onRemove && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={onRemove}
          >
            <XIcon className="h-3 w-3 text-muted-foreground" />
          </Button>
        )}
      </div>
    </div>
  );
}

function AddMemberPopover({ unitId }: { unitId: string }) {
  const [open, setOpen] = useState(false);
  const { data: orgMembers, isLoading: orgMembersLoading } =
    useQuery(orgMembersQueryOpts);
  const { data: unitMembers } = useQuery(unitMembersQueryOpts(unitId));
  const { mutateAsync: addMember } = useMutation(addUnitMemberMutationOpts);

  const existingUserIds = useMemo(() => {
    const ids = new Set<string>();
    unitMembers?.forEach((m) => ids.add(m.user_id));
    return ids;
  }, [unitMembers]);

  const availableMembers = useMemo(
    () =>
      orgMembers?.filter((m) => m.user_id && !existingUserIds.has(m.user_id)) ??
      [],
    [orgMembers, existingUserIds]
  );

  const handleAdd = async (userId: string, userName: string) => {
    try {
      toast.promise(
        addMember({
          unitId,
          body: { user_id: userId, role: "member", is_primary: false },
        }),
        {
          loading: "Adding member...",
          success: `Added "${userName}" to unit`,
          error: "Failed to add member",
        }
      );
    } catch {
      // toast.promise handles error
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-7 text-xs">
            <Plus className="h-3 w-3 mr-1" />
            Add Member
          </Button>
        }
      />
      <PopoverContent className="p-0 w-64" align="end">
        <Command>
          <CommandInput placeholder="Search members..." />
          <CommandList>
            <CommandEmpty>
              {orgMembersLoading ? "Loading..." : "No available members"}
            </CommandEmpty>
            <CommandGroup>
              {availableMembers.map((m) => (
                <CommandItem
                  key={m.user_id}
                  value={m.user_name}
                  onSelect={() => {
                    if (m.user_id) handleAdd(m.user_id, m.user_name ?? "");
                    setOpen(false);
                  }}
                >
                  <span>{m.user_name}</span>
                  <span className="ml-1 text-muted-foreground text-[10px]">
                    {m.user_email}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function SubtreeMembersSection({ unitId }: { unitId: string }) {
  const { data: tree } = useSuspenseQuery(unitTreeQueryOpts);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  const findNode = (
    nodes: UnitTreeNode[],
    id: string
  ): UnitTreeNode | undefined => {
    for (const n of nodes) {
      if (n.unit.id === id) return n;
      const found = n.children ? findNode(n.children, id) : undefined;
      if (found) return found;
    }
    return undefined;
  };

  const node = findNode(tree, unitId);
  if (!node?.children?.length) return null;

  return (
    <div className="mt-4">
      <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
        <Users className="h-3.5 w-3.5" />
        Sub-unit Members
      </h4>
      {node.children.map((child) => (
        <SubtreeGroup
          key={child.unit.id}
          node={child}
          expanded={expandedGroups.has(child.unit.id)}
          onToggle={() => {
            const next = new Set(expandedGroups);
            if (next.has(child.unit.id)) next.delete(child.unit.id);
            else next.add(child.unit.id);
            setExpandedGroups(next);
          }}
          depth={1}
        />
      ))}
    </div>
  );
}

function SubtreeGroup({
  node,
  expanded,
  onToggle,
  depth,
}: {
  node: UnitTreeNode;
  expanded: boolean;
  onToggle: () => void;
  depth: number;
}) {
  const { data: memberData } = useQuery({
    ...unitMembersQueryOpts(node.unit.id),
    enabled: expanded,
  });

  return (
    <div style={{ marginLeft: `${depth * 12}px` }} className="mb-1">
      <button
        onClick={onToggle}
        className="flex items-center gap-1 text-xs font-medium py-1 hover:text-foreground w-full text-left"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        {node.unit.name} ({memberData?.length ?? 0} members)
      </button>
      {expanded && memberData && (
        <div className="space-y-0.5">
          {memberData.map((m) => (
            <MemberRow key={m.id} member={m} />
          ))}
          {node.children?.map((child) => (
            <SubtreeGroup
              key={child.unit.id}
              node={child}
              expanded={false}
              onToggle={() => {}}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function BreadcrumbNav({
  breadcrumbs,
  onNavigate,
}: {
  breadcrumbs: Unit[];
  onNavigate: (unit: Unit) => void;
}) {
  if (breadcrumbs.length <= 1) return null;

  const items: ReactNode[] = [];
  breadcrumbs.forEach((b, i) => {
    const isLast = i === breadcrumbs.length - 1;
    if (i > 0) items.push(<BreadcrumbSeparator key={`sep-${i}`} />);
    items.push(
      <BreadcrumbItem key={b.id}>
        {isLast ? (
          <BreadcrumbPage>{b.name}</BreadcrumbPage>
        ) : (
          <button
            onClick={() => onNavigate(b)}
            className="transition-colors hover:text-foreground"
          >
            {b.name}
          </button>
        )}
      </BreadcrumbItem>
    );
  });

  return (
    <Breadcrumb className="mb-1">
      <BreadcrumbList>{items}</BreadcrumbList>
    </Breadcrumb>
  );
}

export function UnitDetailPanel() {
  const selectedUnit = useOrgHierarchyStore((s) => s.selectedUnit);
  const setSelectedUnit = useOrgHierarchyStore((s) => s.setSelectedUnit);
  const editUnit = useOrgHierarchyStore((s) => s.editUnit);
  const setDeleteOpen = useOrgHierarchyStore((s) => s.setDeleteOpen);

  const breadcrumbs = useBreadcrumbs(selectedUnit);

  const { data: unitMembers, isLoading: unitMembersLoading } = useQuery({
    ...unitMembersQueryOpts(selectedUnit?.id ?? ""),
    enabled: !!selectedUnit,
  });
  const { mutateAsync: removeMember } = useMutation(
    removeUnitMemberMutationOpts
  );
  const { mutateAsync: updateMemberPrimary } = useMutation(
    updateUnitMemberMutationOpts
  );

  const onEdit = () => {
    if (selectedUnit) editUnit(selectedUnit);
  };
  const onDelete = () => {
    setSelectedUnit(selectedUnit);
    setDeleteOpen(true);
  };

  const handleRemoveMember = async (member: UnitMember) => {
    if (!selectedUnit) return;
    toast.promise(
      removeMember({ unitId: selectedUnit.id, membershipId: member.id }),
      {
        loading: "Removing member...",
        success: `Removed "${member.user_name}"`,
        error: "Failed to remove member",
      }
    );
  };

  const handleMakePrimary = async (member: UnitMember) => {
    if (!selectedUnit) return;
    try {
      await updateMemberPrimary({
        unitId: selectedUnit.id,
        membershipId: member.id,
        is_primary: true,
      });
      toast.success(`Set "${member.user_name}" as primary member`);
    } catch {
      toast.error("Failed to update member");
    }
  };

  return (
    <Sheet
      open={selectedUnit !== null}
      onOpenChange={(o) => !o && setSelectedUnit(null)}
    >
      <SheetContent className="w-[400px] sm:max-w-[400px] overflow-y-auto">
        {selectedUnit && (
          <>
            <SheetHeader>
              <BreadcrumbNav
                breadcrumbs={breadcrumbs}
                onNavigate={setSelectedUnit}
              />
              <SheetTitle>{selectedUnit.name}</SheetTitle>
              <SheetDescription>
                <Badge variant="secondary">
                  {selectedUnit.hierarchy_level === 0
                    ? "Root"
                    : `Level ${selectedUnit.hierarchy_level}`}
                </Badge>
                <span className="ml-2 text-muted-foreground font-mono text-xs">
                  {selectedUnit.code}
                </span>
              </SheetDescription>
            </SheetHeader>

            <div className="mt-6 space-y-6 px-4">
              {selectedUnit.description && (
                <div>
                  <h4 className="text-sm font-medium mb-1">Description</h4>
                  <p className="text-sm text-muted-foreground">
                    {selectedUnit.description}
                  </p>
                </div>
              )}

              <Separator />

              <div>
                <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
                  <Users className="h-3.5 w-3.5" />
                  Members
                </h4>
                {unitMembersLoading ? (
                  <div className="space-y-2">
                    <Skeleton className="h-8 w-full" />
                    <Skeleton className="h-8 w-3/4" />
                  </div>
                ) : unitMembers && unitMembers.length > 0 ? (
                  <div className="space-y-0.5">
                    {unitMembers.map((m) => (
                      <MemberRow
                        key={m.id}
                        member={m}
                        onRemove={() => handleRemoveMember(m)}
                        onMakePrimary={
                          m.is_primary ? undefined : () => handleMakePrimary(m)
                        }
                      />
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground italic">
                    No members assigned
                  </p>
                )}
                <div className="mt-2">
                  <AddMemberPopover unitId={selectedUnit.id} />
                </div>
              </div>

              <SubtreeMembersSection unitId={selectedUnit.id} />

              <Separator />

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="text-sm font-medium mb-1">Created</h4>
                  <p className="text-sm text-muted-foreground">
                    {new Date(selectedUnit.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div>
                  <h4 className="text-sm font-medium mb-1">Updated</h4>
                  <p className="text-sm text-muted-foreground">
                    {new Date(selectedUnit.updated_at).toLocaleDateString()}
                  </p>
                </div>
              </div>

              <Separator />

              <div className="flex gap-2">
                <Button variant="outline" onClick={onEdit} className="flex-1">
                  <Edit className="h-4 w-4 mr-2" />
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  onClick={onDelete}
                  className="flex-1"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Delete
                </Button>
              </div>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
