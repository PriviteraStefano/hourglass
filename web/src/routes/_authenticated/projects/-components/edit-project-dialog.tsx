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
import { ProjectsApis } from "@/api/projects";
import { ContractsApis } from "@/api/contracts";
import type { Project } from "@/types/models";

const GOVERNANCE_OPTIONS = [
  {
    value: "creator_controlled",
    label: "Creator Controlled",
    description: "Only your organization can approve changes to this project",
  },
  {
    value: "unanimous",
    label: "Unanimous",
    description: "All organizations using this project must approve changes",
  },
  {
    value: "majority",
    label: "Majority",
    description:
      "More than half of organizations using this project must approve changes",
  },
];

const PROJECT_TYPES = [
  { value: "billable", label: "Billable" },
  { value: "internal", label: "Internal" },
];

interface EditProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
  project: Project;
}

export function EditProjectDialog({
  open,
  onOpenChange,
  onSuccess,
  project,
}: EditProjectDialogProps) {
  const updateProject = useMutation(ProjectsApis.updateProjectMutationOpts);
  const queryClient = useQueryClient();
  const { data: contracts } = useQuery(ContractsApis.contractsQueryOpts("all"));

  const [name, setName] = useState(project.name);
  const [type, setType] = useState<"billable" | "internal">(project.type);
  const [contractId, setContractId] = useState(project.contract_id);
  const [governanceModel, setGovernanceModel] = useState<
    "creator_controlled" | "unanimous" | "majority"
  >(project.governance_model);
  const [isShared, setIsShared] = useState(project.is_shared);

  const handleSubmit = () => {
    if (!name.trim() || !contractId) return;

    updateProject.mutate(
      {
        id: project.id,
        data: {
          name: name.trim(),
          type,
          contract_id: contractId,
          governance_model: governanceModel,
          is_shared: isShared,
        },
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          queryClient.invalidateQueries({ queryKey: ["projects", project.id] });
          if (onSuccess) onSuccess();
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Project</DialogTitle>
          <DialogDescription>
            Update the details of {project.name}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Project name"
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Type</label>
            <Select
              value={type}
              onValueChange={(v) => setType(v as "billable" | "internal")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PROJECT_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Contract *</label>
            <Select
              value={contractId}
              onValueChange={(v) => setContractId(v ?? "")}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select a contract" />
              </SelectTrigger>
              <SelectContent>
                {contracts?.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
              id="project-is-shared"
              checked={isShared}
              onCheckedChange={(checked) => setIsShared(checked)}
            />
            <label htmlFor="project-is-shared" className="text-sm">
              Share with other organizations
            </label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!name.trim() || !contractId || updateProject.isPending}
          >
            {updateProject.isPending ? "Saving..." : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
