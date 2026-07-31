import { type TimeEntry } from "@/types";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { SaveIcon, Trash2Icon, SendIcon } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { ChangeEvent } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ActivitiesApis } from "@/api/activities.ts";
import { StatusBadge } from "./status-badge.tsx";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import { useState } from "react";

interface EntryRowProps {
  entry: TimeEntry;
  editable: boolean;
  onUpdate?: (field: string, value: string | number) => void;
  onDelete?: () => void;
  onSubmit?: () => void;
}

export function EntryRow({
  entry,
  editable,
  onUpdate,
  onDelete,
  onSubmit,
}: EntryRowProps) {
  const { data: activities } = useSuspenseQuery(
    ActivitiesApis.activitiesQueryOpts("all")
  );
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const selectedActivity = activities?.find(
    (a: { id: string }) => a.id === entry.activity_id
  );

  return (
    <div className="flex items-center gap-2 p-2 bg-muted/30 rounded">
      <Select
        value={entry.activity_id}
        onValueChange={(v) => v !== null && onUpdate?.("activity_id", v)}
        disabled={!editable}
      >
        <SelectTrigger className="w-48">
          <SelectValue
            placeholder={
              selectedActivity ? selectedActivity.name : "Select activity"
            }
          />
        </SelectTrigger>
        <SelectContent>
          {activities?.map((a: { id: string; name: string }) => (
            <SelectItem key={a.id} value={a.id}>
              {a.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Input
        type="number"
        step="0.25"
        min="0"
        max="24"
        value={entry.hours}
        onChange={(e: ChangeEvent<HTMLInputElement>) =>
          onUpdate?.("hours", parseFloat(e.target.value) || 0)
        }
        disabled={!editable}
        className="w-20"
      />
      <span className="text-sm">hours</span>

      <Input
        value={entry.description || ""}
        onChange={(e: ChangeEvent<HTMLInputElement>) =>
          onUpdate?.("description", e.target.value)
        }
        placeholder="Description (optional)"
        disabled={!editable}
        className="flex-1"
      />

      <StatusBadge status={entry.status} />

      <div className="flex gap-1">
        {editable && (
          <>
            <Button
              variant="ghost"
              size="sm"
              onClick={onSubmit}
              title="Submit Entry"
            >
              <SendIcon className="w-4 h-4" />
            </Button>
            <AlertDialog
              open={deleteDialogOpen}
              onOpenChange={setDeleteDialogOpen}
            >
              <AlertDialogTrigger
                render={
                  <Button variant="ghost" size="sm" title="Delete Entry">
                    <Trash2Icon className="w-4 h-4" />
                  </Button>
                }
              />
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Entry?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={onDelete}>
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </>
        )}
      </div>
    </div>
  );
}
