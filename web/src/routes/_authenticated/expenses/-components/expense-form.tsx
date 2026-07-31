import { useState } from "react";
import { format } from "date-fns";
import { type ExpenseCategory, CATEGORY_LABELS } from "@/types/expense-types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ActivitiesApis } from "@/api/activities.ts";
import { UploadIcon } from "lucide-react";
import type { ChangeEvent } from "react";

const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  "mileage",
  "meal",
  "accommodation",
  "parking",
  "travel_tickets",
  "tolls",
  "taxi",
  "equipment",
  "other",
];

interface ExpenseFormProps {
  date: Date;
  onSaveDraft: (data: {
    activity_id: string;
    category: ExpenseCategory;
    amount: number;
    km_distance?: number;
    description?: string;
  }) => void;
  onSubmit: (data: {
    activity_id: string;
    category: ExpenseCategory;
    amount: number;
    km_distance?: number;
    description?: string;
  }) => void;
  isPending?: boolean;
}

export function ExpenseForm({
  date,
  onSaveDraft,
  onSubmit,
  isPending,
}: ExpenseFormProps) {
  const { data: activities } = useSuspenseQuery(
    ActivitiesApis.activitiesQueryOpts("all")
  );

  const [activityId, setActivityId] = useState("");
  const [category, setCategory] = useState<ExpenseCategory>("mileage");
  const [amount, setAmount] = useState(0);
  const [kmDistance, setKmDistance] = useState<number | undefined>(undefined);
  const [description, setDescription] = useState("");

  const formData = () => ({
    activity_id: activityId,
    category,
    amount,
    km_distance: category === "mileage" ? kmDistance : undefined,
    description: description || undefined,
  });

  const isFormValid = activityId !== "" && amount > 0;

  const handleFileChange = (_e: ChangeEvent<HTMLInputElement>) => {
    // Receipt upload handled at expense-detail level
  };

  return (
    <div className="flex flex-col gap-4 p-4 border rounded bg-muted/20">
      <div className="space-y-1">
        <Label className="text-sm">Date</Label>
        <p className="text-sm font-medium">{format(date, "MMMM d, yyyy")}</p>
      </div>

      <div className="space-y-1">
        <Label className="text-sm">Activity</Label>
        <Select
          value={activityId}
          onValueChange={(v) => v !== null && setActivityId(v)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select activity" />
          </SelectTrigger>
          <SelectContent>
            {activities?.map((a: { id: string; name: string }) => (
              <SelectItem key={a.id} value={a.id}>
                {a.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1">
        <Label className="text-sm">Category</Label>
        <Select
          value={category}
          onValueChange={(v) => setCategory(v as ExpenseCategory)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select category" />
          </SelectTrigger>
          <SelectContent>
            {EXPENSE_CATEGORIES.map((cat) => (
              <SelectItem key={cat} value={cat}>
                {CATEGORY_LABELS[cat]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1">
        <Label className="text-sm">Amount</Label>
        <Input
          type="number"
          step="0.01"
          min="0"
          value={amount || ""}
          onChange={(e) => setAmount(parseFloat(e.target.value) || 0)}
          placeholder="0.00"
        />
      </div>

      {category === "mileage" && (
        <div className="space-y-1">
          <Label className="text-sm">Km Distance</Label>
          <Input
            type="number"
            step="0.1"
            min="0"
            value={kmDistance ?? ""}
            onChange={(e) =>
              setKmDistance(
                e.target.value ? parseFloat(e.target.value) : undefined
              )
            }
            placeholder="Km"
          />
        </div>
      )}

      <div className="space-y-1">
        <Label className="text-sm">Receipt</Label>
        <label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          <UploadIcon className="w-4 h-4" />
          <span>Upload receipt (PDF, JPG, PNG)</span>
          <input
            type="file"
            accept=".pdf,.jpg,.jpeg,.png"
            className="hidden"
            onChange={handleFileChange}
          />
        </label>
      </div>

      <div className="space-y-1">
        <Label className="text-sm">Description</Label>
        <Textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Description (optional)"
          className="min-h-[60px]"
        />
      </div>

      <div className="flex gap-2 justify-end">
        <Button
          variant="outline"
          onClick={() => onSaveDraft(formData())}
          disabled={!isFormValid || isPending}
        >
          {isPending ? "Saving..." : "Save Draft"}
        </Button>
        <Button
          onClick={() => onSubmit(formData())}
          disabled={!isFormValid || isPending}
        >
          {isPending ? "Submitting..." : "Submit Expense"}
        </Button>
      </div>
    </div>
  );
}
