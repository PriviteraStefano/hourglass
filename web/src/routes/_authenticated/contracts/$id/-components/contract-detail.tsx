import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import { ArrowLeftIcon, GlobeIcon, LockIcon } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Label } from "@/components/ui/label.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Checkbox } from "@/components/ui/checkbox.tsx";
import { toast } from "sonner";
import { ContractsApis } from "@/api/contracts.ts";
import { CustomersApis } from "@/api/customers.ts";
import { AuthApis } from "@/api/auth.ts";
import { Route } from "@/routes/_authenticated/contracts/$id";
import type { Contract } from "@/types/models.ts";

export function ContractDetail() {
  const { id } = Route.useParams();
  const { from: fromTab } = Route.useSearch();
  const navigate = useNavigate();
  const { data: c } = useSuspenseQuery(ContractsApis.contractQueryOpts(id));
  const { data: me } = useSuspenseQuery(AuthApis.profileQueryOpts);
  const { data: customers } = useSuspenseQuery(
    CustomersApis.customersQueryOpts()
  );

  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<Contract>>({});

  const updateContract = useMutation(ContractsApis.updateContractMutationOpts);
  const deleteContract = useMutation(ContractsApis.deleteContractMutationOpts);
  const recalculateMileage = useMutation(
    ContractsApis.recalculateMileageMutationOpts
  );

  const isFinance = me?.membership?.role === "finance";
  const hasTimeEntries = (c.time_entries_count ?? 0) > 0;
  const isAdopted = fromTab === "adopted";

  const handleEdit = () => {
    setFormData({
      name: c.name,
      km_rate: c.km_rate,
      currency: c.currency,
      governance_model: c.governance_model,
      is_shared: c.is_shared,
      is_active: c.is_active,
      customer_id: c.customer_id,
    });
    setIsEditing(true);
  };

  const handleSave = async () => {
    if (!formData.name || formData.km_rate === undefined) {
      toast.error("Name and KM rate are required");
      return;
    }
    try {
      await updateContract.mutateAsync({
        id,
        data: {
          name: formData.name!,
          km_rate: formData.km_rate!,
          currency: formData.currency!,
          governance_model: formData.governance_model!,
          is_shared: formData.is_shared!,
          is_active: formData.is_active!,
          customer_id: formData.customer_id,
        },
      });
      setIsEditing(false);
    } catch {
      toast.error("Failed to update contract");
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setFormData({});
  };

  const handleToggleActive = async () => {
    try {
      await updateContract.mutateAsync({
        id,
        data: {
          ...formData,
          is_active: !c.is_active,
          name: c.name,
          km_rate: c.km_rate,
          currency: c.currency,
          governance_model: c.governance_model,
          is_shared: c.is_shared,
        },
      });
      toast.success(
        c.is_active
          ? "Contract marked as inactive"
          : "Contract marked as active"
      );
    } catch {
      toast.error("Failed to update status");
    }
  };

  const handleRecalculate = async (fromDate: string) => {
    try {
      const result = await recalculateMileage.mutateAsync({ id, fromDate });
      toast.success(`Recalculated ${result.recalculated_count} expenses`);
    } catch {
      toast.error("Failed to recalculate mileage");
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this contract?")) return;
    try {
      await deleteContract.mutateAsync(id);
      navigate({ to: "/contracts", search: { tab: fromTab } });
    } catch (e: unknown) {
      const err = e as { status?: number };
      if (err.status === 409) {
        toast.error("Cannot delete contract with time entries");
      } else {
        toast.error("Failed to delete contract");
      }
    }
  };

  const getDefaultDate = () => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1)
      .toISOString()
      .split("T")[0];
  };

  return (
    <div className="space-y-4">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate({ to: "/contracts", search: { tab: fromTab } })}
      >
        <ArrowLeftIcon className="w-4 h-4 mr-1" />
        Back to Contracts
      </Button>

      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{c.name}</h1>
            {c.is_shared ? (
              <GlobeIcon className="w-5 h-5 text-muted-foreground" />
            ) : (
              <LockIcon className="w-5 h-5 text-muted-foreground" />
            )}
            {c.is_shared && <Badge variant="secondary">Shared</Badge>}
          </div>
          {isAdopted && c.created_by_org_name && (
            <p className="text-sm text-muted-foreground mt-1">
              Adopted from {c.created_by_org_name}
            </p>
          )}
        </div>
        {isFinance && !isEditing && (
          <Button variant="outline" onClick={handleEdit}>
            Edit
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {isEditing ? (
            <div className="space-y-4">
              <div className="grid gap-2">
                <Label>Name</Label>
                <Input
                  value={formData.name ?? ""}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>KM Rate</Label>
                <Input
                  type="number"
                  step="0.01"
                  value={formData.km_rate ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      km_rate: parseFloat(e.target.value),
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>Currency</Label>
                <Select
                  value={formData.currency ?? ""}
                  onValueChange={(v) =>
                    setFormData({ ...formData, currency: v })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="USD">USD</SelectItem>
                    <SelectItem value="EUR">EUR</SelectItem>
                    <SelectItem value="GBP">GBP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>Governance</Label>
                <Select
                  value={formData.governance_model}
                  onValueChange={(v) =>
                    setFormData({
                      ...formData,
                      governance_model: v as Contract["governance_model"],
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="creator_controlled">
                      Creator Controlled
                    </SelectItem>
                    <SelectItem value="unanimous">Unanimous</SelectItem>
                    <SelectItem value="majority">Majority</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="is_shared"
                  checked={formData.is_shared}
                  onCheckedChange={(v) =>
                    setFormData({ ...formData, is_shared: v })
                  }
                />
                <Label htmlFor="is_shared">Shared</Label>
              </div>
              <div className="grid gap-2">
                <Label>Customer</Label>
                <Select
                  value={formData.customer_id ?? ""}
                  onValueChange={(v) =>
                    setFormData({ ...formData, customer_id: v ? v : undefined })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select customer" />
                  </SelectTrigger>
                  <SelectContent>
                    {customers?.map((cust) => (
                      <SelectItem key={cust.id} value={cust.id}>
                        {cust.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex gap-2 pt-2">
                <Button
                  onClick={handleSave}
                  disabled={updateContract.isPending}
                >
                  Save
                </Button>
                <Button variant="outline" onClick={handleCancel}>
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <>
              {c.customer_name && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Customer</span>
                  <span>{c.customer_name}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-muted-foreground">KM Rate</span>
                <span>
                  {c.currency} {c.km_rate.toFixed(2)}/km
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Currency</span>
                <span>{c.currency}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Governance</span>
                <span className="capitalize">
                  {c.governance_model.replace("_", " ")}
                </span>
              </div>
              {c.is_shared && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Adoption Count</span>
                  <span>{c.adoption_count ?? 0}</span>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {isFinance && (
        <Card>
          <CardHeader>
            <CardTitle>Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span>Status</span>
                <Badge
                  variant={c.is_active ? "default" : "outline"}
                  className={c.is_active ? "bg-green-500" : ""}
                >
                  {c.is_active ? "Active" : "Inactive"}
                </Badge>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleToggleActive}
                disabled={updateContract.isPending}
              >
                Toggle
              </Button>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <span className="mr-2">Recalculate Mileage</span>
              </div>
              <div className="flex gap-2">
                <Input
                  type="date"
                  defaultValue={getDefaultDate()}
                  id="recalc-date"
                  className="w-[140px]"
                />
                <Button
                  onClick={() =>
                    handleRecalculate(
                      (
                        document.getElementById(
                          "recalc-date"
                        ) as HTMLInputElement
                      ).value
                    )
                  }
                  disabled={recalculateMileage.isPending}
                >
                  Recalculate
                </Button>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span>Delete Contract</span>
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDelete}
                disabled={deleteContract.isPending || hasTimeEntries}
              >
                Delete
              </Button>
              {hasTimeEntries && (
                <span className="text-xs text-muted-foreground ml-2">
                  Cannot delete - has time entries
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
