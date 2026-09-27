import { useState } from "react";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import type { OrgMember } from "@/api/units.ts";
import {
  TICKET_KINDS,
  TICKET_KIND_LABELS,
  type CreateTicketRequest,
  type TicketKind,
} from "@/types";

export interface CreateTicketDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  members: OrgMember[];
  pending: boolean;
  onCreate: (request: CreateTicketRequest) => void;
}

/** E-05: raise internal demand without leaving the surface. */
export function CreateTicketDialog({
  open,
  onOpenChange,
  members,
  pending,
  onCreate,
}: CreateTicketDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [kind, setKind] = useState<TicketKind>("bug");
  const [assigneeId, setAssigneeId] = useState("");

  const reset = () => {
    setTitle("");
    setDescription("");
    setKind("bug");
    setAssigneeId("");
  };

  const assignable = members.filter(
    (member): member is OrgMember & { user_id: string } =>
      Boolean(member.user_id) && member.is_active
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>New ticket</DialogTitle>
          <DialogDescription>
            Tickets are internal demand records — triage turns one into the
            activities the work belongs to.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label htmlFor="ticket-title" className="text-sm font-medium">
              Title *
            </label>
            <Input
              id="ticket-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="What is needed?"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="ticket-description" className="text-sm font-medium">
              Description
            </label>
            <Textarea
              id="ticket-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="Context, scope, what done looks like"
              rows={3}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="ticket-kind" className="text-sm font-medium">
              Kind
            </label>
            <NativeSelect
              id="ticket-kind"
              value={kind}
              onChange={(event) => setKind(event.target.value as TicketKind)}
              className="w-full"
            >
              {TICKET_KINDS.map((value) => (
                <NativeSelectOption key={value} value={value}>
                  {TICKET_KIND_LABELS[value]}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>

          <div className="space-y-2">
            <label htmlFor="ticket-assignee" className="text-sm font-medium">
              Assignee
            </label>
            <NativeSelect
              id="ticket-assignee"
              value={assigneeId}
              onChange={(event) => setAssigneeId(event.target.value)}
              className="w-full"
            >
              <NativeSelectOption value="">Unassigned</NativeSelectOption>
              {assignable.map((member) => (
                <NativeSelectOption key={member.user_id} value={member.user_id}>
                  {member.user_name ?? member.user_email ?? "Unnamed member"}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              reset();
              onOpenChange(false);
            }}
          >
            Cancel
          </Button>
          <Button
            disabled={pending || title.trim() === ""}
            onClick={() => {
              onCreate({
                title: title.trim(),
                description: description.trim(),
                kind,
                assignee_id: assigneeId || undefined,
              });
              reset();
              onOpenChange(false);
            }}
          >
            {pending ? "Raising..." : "Raise ticket"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
