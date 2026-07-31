import { useMutation } from "@tanstack/react-query";
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
import { toast } from "sonner";
import { WorkingGroupsApis } from "@/api/working-groups";
import type { WorkingGroup } from "@/types";

interface DeleteWorkingGroupDialogProps {
  wg: WorkingGroup | null;
  onClose: () => void;
}

/**
 * Destructive WG delete confirmation. Backend guard errors (e.g. "cannot
 * delete working group with members") surface verbatim via the mutation's
 * onError toast (T-10-06-3) — the client invents no cascade logic.
 */
export function DeleteWorkingGroupDialog({
  wg,
  onClose,
}: DeleteWorkingGroupDialogProps) {
  const deleteWg = useMutation({
    ...WorkingGroupsApis.deleteWorkingGroupMutationOpts,
    onError: (error: Error) => {
      toast.error(error.message || "Failed to delete working group");
    },
  });

  if (!wg) return null;

  const handleConfirm = async () => {
    await deleteWg.mutateAsync(wg.id);
    onClose();
  };

  return (
    <AlertDialog open={!!wg} onOpenChange={(open) => !open && onClose()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete working group</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete <strong>{wg.name}</strong>? This
            action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={deleteWg.isPending}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {deleteWg.isPending ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
