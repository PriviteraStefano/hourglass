import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { PlusIcon, UserRoundIcon } from "lucide-react";
import { WorkingGroupsApis } from "@/api/working-groups";
import { orgMembersQueryOpts, unitTreeQueryOpts } from "@/api/units";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  useComboboxAnchor,
} from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import type { WorkingGroup, WorkingGroupMember } from "@/types";
import type { UnitTreeNode } from "@/types/unit";

// WG member roles are a free-form VARCHAR on wg_members with no CHECK
// constraint; the codebase uses 'member' (seed 003, unit-detail-panel) and
// 'lead' (wg_member_repository_test). Offer exactly those two.
const MEMBER_ROLES = ["member", "lead"] as const;

interface WorkingGroupMembersDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  wg: WorkingGroup;
}

/**
 * WG member management (list / add / remove). Distinct from the approver set:
 * manager_id/delegate_ids are edited in the form dialog only — membership and
 * approver-set stay two separate concerns in the UI (plan 10-06 task 3).
 *
 * Backend org-scopes all member endpoints (T-10-06-1); the dialog never sends
 * org ids and adds no client-side org-isolation guard. AddMember REQUIRES
 * user_id + unit_id (handler 400s otherwise), so the add form carries a unit
 * picker sourced from the unit tree.
 */
export function WorkingGroupMembersDialog({
  open,
  onOpenChange,
  wg,
}: WorkingGroupMembersDialogProps) {
  const comboboxAnchor = useComboboxAnchor();

  const { data: members } = useQuery(
    WorkingGroupsApis.workingGroupMembersQueryOpts(wg.id)
  );
  const { data: orgMembers } = useQuery(orgMembersQueryOpts);
  const { data: unitTree } = useQuery(unitTreeQueryOpts);

  const [userId, setUserId] = useState("");
  const [unitId, setUnitId] = useState("");
  const [role, setRole] = useState<string>("member");
  const [removeTarget, setRemoveTarget] = useState<WorkingGroupMember | null>(
    null
  );

  const addMember = useMutation(WorkingGroupsApis.addWorkingGroupMemberMutationOpts);
  const removeMember = useMutation(
    WorkingGroupsApis.removeWorkingGroupMemberMutationOpts
  );

  const units = useMemo(() => {
    const flat: Array<{ id: string; name: string }> = [];
    const walk = (nodes: UnitTreeNode[] | undefined) => {
      nodes?.forEach((node) => {
        flat.push({ id: node.unit.id, name: node.unit.name });
        walk(node.children);
      });
    };
    walk(unitTree);
    return flat;
  }, [unitTree]);

  const memberIds = new Set((members ?? []).map((m) => m.user_id));

  // Candidate users = org members minus users already on the WG. user_id is
  // nullable on OrgMember (invited-but-not-activated) — only real users are
  // selectable.
  const candidateUsers =
    orgMembers?.filter((m) => m.user_id && !memberIds.has(m.user_id)) ?? [];
  const userOptions = candidateUsers.map((m) => ({
    id: m.user_id as string,
    label: m.user_name || m.user_email || m.id,
  }));

  const nameFor = (userId: string) =>
    orgMembers?.find((m) => m.user_id === userId)?.user_name ?? userId;

  const canAdd = !!userId && !!unitId;

  const handleAdd = () => {
    if (!canAdd) return;
    addMember.mutate(
      {
        wgId: wg.id,
        data: {
          user_id: userId,
          unit_id: unitId,
          role,
          is_default_subproject: false,
        },
      },
      {
        onSuccess: () => {
          setUserId("");
          setUnitId("");
          setRole("member");
        },
      }
    );
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Members — {wg.name}</DialogTitle>
            <DialogDescription>
              People assigned to this working group. Managers and delegates are
              edited separately via Edit.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              {members?.length === 0 ? (
                <p className="text-sm text-muted-foreground py-3 text-center">
                  No members yet.
                </p>
              ) : (
                members?.map((m) => (
                  <div
                    key={m.id}
                    className="flex items-center justify-between rounded-lg border p-2.5"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <UserRoundIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span className="truncate text-sm">
                        {nameFor(m.user_id)}
                      </span>
                      <Badge variant="secondary" className="capitalize">
                        {m.role}
                      </Badge>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setRemoveTarget(m)}
                    >
                      Remove
                    </Button>
                  </div>
                ))
              )}
            </div>

            <div className="space-y-3 rounded-lg border p-3">
              <p className="text-xs font-medium text-muted-foreground">
                Add member
              </p>
              <div className="space-y-2" ref={comboboxAnchor}>
                <Combobox value={userId} onValueChange={(v) => setUserId(v ?? "")}>
                  <ComboboxInput
                    placeholder="Select user..."
                    showTrigger
                    showClear={!!userId}
                  />
                  <ComboboxContent anchor={comboboxAnchor}>
                    <ComboboxList>
                      {userOptions.map((u) => (
                        <ComboboxItem key={u.id} value={u.id}>
                          <span>{u.label}</span>
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-2" ref={comboboxAnchor}>
                  <Combobox value={unitId} onValueChange={(v) => setUnitId(v ?? "")}>
                    <ComboboxInput
                      placeholder="Unit..."
                      showTrigger
                      showClear={!!unitId}
                    />
                    <ComboboxContent anchor={comboboxAnchor}>
                      <ComboboxList>
                        {units.map((u) => (
                          <ComboboxItem key={u.id} value={u.id}>
                            <span>{u.name}</span>
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                </div>
                <Select value={role} onValueChange={(v) => setRole(v ?? "member")}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {MEMBER_ROLES.map((r) => (
                      <SelectItem key={r} value={r} className="capitalize">
                        {r}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button
                onClick={handleAdd}
                disabled={!canAdd || addMember.isPending}
                className="w-full"
                size="sm"
              >
                <PlusIcon className="w-4 h-4 mr-1" />
                {addMember.isPending ? "Adding..." : "Add member"}
              </Button>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!removeTarget}
        onOpenChange={(open) => !open && setRemoveTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove member</AlertDialogTitle>
            <AlertDialogDescription>
              Remove {removeTarget ? nameFor(removeTarget.user_id) : ""} from{" "}
              <strong>{wg.name}</strong>? They will no longer be assigned to
              this working group.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                if (!removeTarget) return;
                await removeMember.mutateAsync({
                  wgId: wg.id,
                  memberId: removeTarget.id,
                });
                setRemoveTarget(null);
              }}
              disabled={removeMember.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeMember.isPending ? "Removing..." : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
