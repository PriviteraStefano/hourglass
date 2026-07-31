import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
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
import type { ActivityResponse } from "@/types/models";

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

interface CreateActivityDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (activity: ActivityResponse) => void;
}

export function CreateActivityDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateActivityDialogProps) {
  const navigate = useNavigate();
  const createActivity = useMutation(ActivitiesApis.createActivityMutationOpts);
  const { data: contracts } = useQuery(ContractsApis.contractsQueryOpts("all"));
  const { data: kinds } = useQuery(ActivitiesApis.activityKindsQueryOpts);
  const { data: ownedActivities } = useQuery(
    ActivitiesApis.activitiesQueryOpts("owned")
  );
  const comboboxAnchor = useComboboxAnchor();

  const [name, setName] = useState("");
  const [kind, setKind] = useState("");
  const [parentId, setParentId] = useState("");
  const [contractId, setContractId] = useState("");
  const [governanceModel, setGovernanceModel] = useState<
    "creator_controlled" | "unanimous" | "majority"
  >("creator_controlled");
  const [isShared, setIsShared] = useState(false);
  const [billable, setBillable] = useState(false);

  const handleSubmit = () => {
    if (!name.trim() || !kind) return;

    createActivity.mutate(
      {
        name: name.trim(),
        description: "",
        kind,
        parent_id: parentId || undefined,
        contract_id: contractId || undefined,
        governance_model: governanceModel,
        is_shared: isShared,
        billable: billable ? true : undefined,
      },
      {
        onSuccess: (data) => {
          onOpenChange(false);
          resetForm();
          if (onSuccess) {
            onSuccess(data);
          } else {
            navigate({
              to: "/activities/$id",
              params: { id: data.id },
              search: { from: "owned" },
            });
          }
        },
      }
    );
  };

  const resetForm = () => {
    setName("");
    setKind("");
    setParentId("");
    setContractId("");
    setGovernanceModel("creator_controlled");
    setIsShared(false);
    setBillable(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create Activity</DialogTitle>
          <DialogDescription>
            Add a new activity for your organization.
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
                  {ownedActivities?.map((a) => (
                    <ComboboxItem key={a.id} value={a.id}>
                      <span>{a.name}</span>
                      {a.parent_name && (
                        <span className="text-xs text-muted-foreground">
                          ({a.parent_name})
                        </span>
                      )}
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
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || !kind || createActivity.isPending}
          >
            {createActivity.isPending ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
