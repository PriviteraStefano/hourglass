import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { CheckIcon, XIcon } from "lucide-react";
import type { EntryStatus, Role } from "@/types";

interface ApprovalButtonsProps {
  status: EntryStatus;
  currentApproverRole?: "manager" | "finance" | null;
  userRole: Role;
  onApprove: () => void;
  onReject: (reason: string) => void;
  isPending?: boolean;
}

export function ApprovalButtons({
  status,
  currentApproverRole,
  userRole,
  onApprove,
  onReject,
  isPending,
}: ApprovalButtonsProps) {
  const [showRejectReason, setShowRejectReason] = useState(false);
  const [rejectReason, setRejectReason] = useState("");

  // Visibility matrix per UI-SPEC:
  // employee -> none
  // manager -> approve/reject at pending_manager, submitted
  // finance -> approve/reject at pending_finance
  const canApprove =
    (userRole === "manager" &&
      (status === "pending_manager" || status === "submitted")) ||
    (userRole === "finance" && status === "pending_finance");

  const canReject =
    (userRole === "manager" &&
      (status === "pending_manager" || status === "submitted")) ||
    (userRole === "finance" && status === "pending_finance");

  if (!canApprove && !canReject) return null;

  const handleRejectConfirm = () => {
    if (rejectReason.trim().length < 10) return;
    onReject(rejectReason.trim());
    setShowRejectReason(false);
    setRejectReason("");
  };

  return (
    <div className="flex gap-2">
      {canApprove && (
        <Button
          onClick={onApprove}
          disabled={isPending}
          size="sm"
          className="min-w-[44px] min-h-[44px]"
          aria-label="Approve"
        >
          <CheckIcon className="w-4 h-4 mr-1" />
          {isPending ? "Approving..." : "Approve"}
        </Button>
      )}
      {canReject && !showRejectReason && (
        <Button
          variant="destructive"
          onClick={() => setShowRejectReason(true)}
          disabled={isPending}
          size="sm"
          className="min-w-[44px] min-h-[44px]"
          aria-label="Reject"
        >
          <XIcon className="w-4 h-4 mr-1" />
          {isPending ? "Rejecting..." : "Reject"}
        </Button>
      )}
      {canReject && showRejectReason && (
        <div className="flex flex-col gap-2 p-2 border rounded bg-muted/30">
          <Label htmlFor="reject-reason" className="text-xs">
            Reason for rejection (required)
          </Label>
          <Textarea
            id="reject-reason"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            placeholder="Explain why this entry is being rejected"
            className="min-h-[60px] text-sm"
          />
          <div className="flex gap-2">
            <Button
              variant="destructive"
              size="sm"
              onClick={handleRejectConfirm}
              disabled={rejectReason.trim().length < 10 || isPending}
            >
              {isPending ? "Rejecting..." : "Reject"}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setShowRejectReason(false);
                setRejectReason("");
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
