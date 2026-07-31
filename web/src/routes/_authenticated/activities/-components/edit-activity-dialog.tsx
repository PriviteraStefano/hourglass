import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
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
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  useComboboxAnchor,
} from "@/components/ui/combobox";
import { ActivitiesApis } from "@/api/activities";
import { ContractsApis } from "@/api/contracts";
import type { Activity } from "@/types/models";

const GOVERNANCE_OPTIONS = [
  {
    value: "creator_controlled",
    label: "Creator Controlled",
    description: "Only your organization can approve changes to this activity",
  },
  {
    value: "unanimous",
    label: "Unanimous",
    description: "All organizations using this activity must approve changes",
  },
  {
    value: "majority",
    label: "Majority",
    description:
      "More than half of organizations using this activity must approve changes",
  },
];

interface EditActivityDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
  activity: Activity;
}

export function EditActivityDialog({
  open,
  onOpenChange,
  onSuccess,
  activity,
}: EditActivityDialogProps) {
  const updateActivity = useMutation(ActivitiesApis.updateActivityMutationOpts);
  const queryClient = useQueryClient();
  const { data: contracts } = useQuery(ContractsApis.contractsQueryOpts("all"));
  const { data: kinds } = useQuery(ActivitiesApis.activityKindsQueryOpts);
  const { data: ownedActivities } = useQuery(
    ActivitiesApis.activitiesQueryOpts("owned")
  );
  const comboboxAnchor = useComboboxAnchor();

  const [name, setName] = useState(activity.name);
  const [kind, setKind] = useState(activity.kind);
  const [parentId, setParentId] = useState(activity.parent_id ?? "");
  const [contractId, setContractId] = useState(activity.contract_id ?? "");
  const [governanceModel, setGovernanceModel] = useState<
    "creator_controlled" | "unanimous" | "majority"
  >(activity.governance_model);
  const [isShared, setIsShared] = useState(activity.is_shared);
  const [billable, setBillable] = useState(activity.billable ?? false);
  const [isActive, setIsActive] = useState(activity.is_active);

  const handleSubmit = () => {
    if (!name.trim() || !kind) return;

    updateActivity.mutate(
      {
        id: activity.id,
        data: {
          name: name.trim(),
          kind,
          parent_id: parentId || undefined,
          contract_id: contractId || undefined,
          governance_model: governanceModel,
          is_shared: isShared,
          billable: billable ? true : undefined,
          is_active: isActive,
        },
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          queryClient.invalidateQueries({ queryKey: ["activities", activity.id] });
          if (onSuccess) onSuccess();
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Activity</DialogTitle>
          <DialogDescription>
            Update the details of {activity.name}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Activity name"
            />
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Kind *</label>
            <Combobox value={kind} onValueChange={(v) => setKind(v ?? "")}>
              <ComboboxInput
                placeholder="Select kind..."
                showTrigger
                showClear={!!kind}
              />
              <ComboboxContent>
                <ComboboxList>
                  {kinds?.map((k) => (
                    <ComboboxItem key={k} value={k}>
                      <span className="capitalize">{k}</span>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Parent activity</label>
            <Combobox
              value={parentId}
              onValueChange={(v) => setParentId(v ?? "")}
            >
              <ComboboxInput
                placeholder="Search activities..."
                showTrigger
                showClear={!!parentId}
              />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="">None (root activity)</ComboboxItem>
                  {ownedActivities
                    ?.filter((a) => a.id !== activity.id)
                    .map((a) => (
                      <ComboboxItem key={a.id} value={a.id}>
                        <span>{a.name}</span>
                      </ComboboxItem>
                    ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="space-y-2" ref={comboboxAnchor}>
            <label className="text-sm font-medium">Contract</label>
            <Combobox
              value={contractId}
              onValueChange={(v) => setContractId(v ?? "")}
            >
              <ComboboxInput
                placeholder="Search contracts..."
                showTrigger
                showClear={!!contractId}
              />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="">No contract</ComboboxItem>
                  {contracts?.map((c) => (
                    <ComboboxItem key={c.id} value={c.id}>
                      <span>{c.name}</span>
                    </ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Governance Model</label>
            <Select
              value={governanceModel}
              onValueChange={(v) =>
                setGovernanceModel(v as typeof governanceModel)
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {GOVERNANCE_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    <div>
                      <div className="font-medium">{opt.label}</div>
                      <div className="text-xs text-muted-foreground">
                        {opt.description}
                      </div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="activity-is-shared"
              checked={isShared}
              onCheckedChange={(checked) => setIsShared(checked)}
            />
            <label htmlFor="activity-is-shared" className="text-sm">
              Share with other organizations
            </label>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="activity-billable"
              checked={billable}
              onCheckedChange={(checked) => setBillable(checked)}
            />
            <label htmlFor="activity-billable" className="text-sm">
              Billable (unchecked = inherit from contract or parent)
            </label>
          </div>

          <div className="flex items-center space-x-2">
            <Checkbox
              id="activity-is-active"
              checked={isActive}
              onCheckedChange={(checked) => setIsActive(checked)}
            />
            <label htmlFor="activity-is-active" className="text-sm">
              Active
            </label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || !kind || updateActivity.isPending}
          >
            {updateActivity.isPending ? "Saving..." : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
