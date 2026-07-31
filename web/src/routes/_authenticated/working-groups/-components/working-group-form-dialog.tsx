import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRouteContext } from "@tanstack/react-router";
import { ActivitiesApis } from "@/api/activities";
import { WorkingGroupsApis } from "@/api/working-groups";
import { orgMembersQueryOpts } from "@/api/units";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
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
import { Input } from "@/components/ui/input";
import type { WorkingGroup } from "@/types";

interface WorkingGroupFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  workingGroup?: WorkingGroup | null;
}

/**
 * Create/edit working group in one dialog (activity + manager + delegates).
 *
 * Org-scoping (threat T-10-06-1): create sends org_id sourced from the
 * route-context profile (the user's own org), never from user input — a client
 * cannot target another org's data. The backend remains authoritative on
 * membership; no client-side org isolation guards are invented.
 *
 * Membership (add/remove members) is a separate concern managed via the
 * members dialog — the approver set (manager/delegate) lives here only.
 */
export function WorkingGroupFormDialog({
  open,
  onOpenChange,
  mode,
  workingGroup,
}: WorkingGroupFormDialogProps) {
  const routeContext = useRouteContext({ from: "/_authenticated" });
  const comboboxAnchor = useComboboxAnchor();

  const [name, setName] = useState("");
  const [activityId, setActivityId] = useState("");
  const [managerId, setManagerId] = useState("");
  const [delegateIds, setDelegateIds] = useState<string[]>([]);

  const createWg = useMutation(WorkingGroupsApis.createWorkingGroupMutationOpts);
  const updateWg = useMutation(WorkingGroupsApis.updateWorkingGroupMutationOpts);

  const { data: activities } = useQuery(ActivitiesApis.activitiesQueryOpts("owned"));
  const { data: orgMembers } = useQuery(orgMembersQueryOpts);

  useEffect(() => {
    if (!open) return;
    if (mode === "edit" && workingGroup) {
      setName(workingGroup.name);
      setActivityId(workingGroup.subproject_id);
      setManagerId(workingGroup.manager_id);
      setDelegateIds(workingGroup.delegate_ids);
    } else {
      setName("");
      setActivityId("");
      setManagerId("");
      setDelegateIds([]);
    }
  }, [open, mode, workingGroup]);

  const memberOptions =
    orgMembers
      ?.filter((m) => m.user_id)
      .map((m) => ({
        id: m.user_id as string,
        label: m.user_name || m.user_email || m.id,
      })) ?? [];

  const canSubmit = name.trim().length > 0 && !!activityId && !!managerId;

  const handleSubmit = () => {
    if (!canSubmit) return;
    if (mode === "edit" && workingGroup) {
      updateWg.mutate(
        {
          id: workingGroup.id,
          data: {
            name: name.trim(),
            description: workingGroup.description,
            unit_ids: workingGroup.unit_ids,
            manager_id: managerId,
            delegate_ids: delegateIds,
          },
        },
        { onSuccess: () => onOpenChange(false) }
      );
    } else {
      createWg.mutate(
        {
          org_id: routeContext.profile.membership.organization_id,
          subproject_id: activityId,
          name: name.trim(),
          description: "",
          unit_ids: [],
          enforce_unit_tuple: false,
          manager_id: managerId,
          delegate_ids: delegateIds,
        },
        { onSuccess: () => onOpenChange(false) }
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "New working group" : "Edit working group"}
          </DialogTitle>
          <DialogDescription>
            Working groups assign people to activities. Choose the activity and
            the people who run it.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Working group name"
            />
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Activity *</label>
            <Combobox value={activityId} onValueChange={(v) => setActivityId(v ?? "")}>
              <ComboboxInput placeholder="Select activity..." showTrigger showClear={!!activityId} />
              <ComboboxContent anchor={comboboxAnchor}>
                <ComboboxList>
                  {activities?.map((a) => (
                    <ComboboxItem key={a.id} value={a.id}>
                      <span>{a.name}</span>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Manager *</label>
            <Combobox value={managerId} onValueChange={(v) => setManagerId(v ?? "")}>
              <ComboboxInput placeholder="Select manager..." showTrigger showClear={!!managerId} />
              <ComboboxContent anchor={comboboxAnchor}>
                <ComboboxList>
                  {memberOptions.map((m) => (
                    <ComboboxItem key={m.id} value={m.id}>
                      <span>{m.label}</span>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Delegates</label>
            <Combobox multiple value={delegateIds} onValueChange={(v) => setDelegateIds(v)}>
              <ComboboxChips>
                {delegateIds.map((id) => {
                  const member = memberOptions.find((m) => m.id === id);
                  return <ComboboxChip key={id}>{member?.label ?? id}</ComboboxChip>;
                })}
                <ComboboxChipsInput placeholder="Select delegates..." />
              </ComboboxChips>
              <ComboboxContent anchor={comboboxAnchor}>
                <ComboboxList>
                  {memberOptions.map((m) => (
                    <ComboboxItem key={m.id} value={m.id}>
                      <span>{m.label}</span>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!canSubmit || createWg.isPending || updateWg.isPending}
          >
            {createWg.isPending || updateWg.isPending
              ? "Saving..."
              : mode === "create"
                ? "Create"
                : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
