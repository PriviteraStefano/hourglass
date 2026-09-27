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
import type { Ticket } from "@/types";

export interface DismissTicketDialogProps {
  /** The ticket awaiting confirmation; null keeps the dialog closed. */
  ticket: Ticket | null;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (id: string) => void;
}

/**
 * Dismissal is terminal and guard-checked server-side: a dismissal is refused
 * (409) while linked activities carry logged hours, and the toast surfaces that
 * reason verbatim.
 */
export function DismissTicketDialog({
  ticket,
  pending,
  onOpenChange,
  onConfirm,
}: DismissTicketDialogProps) {
  return (
    <AlertDialog open={Boolean(ticket)} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Dismiss this ticket?</AlertDialogTitle>
          <AlertDialogDescription>
            <strong>{ticket?.title}</strong> is rejected with the hours already
            logged against it snapshotted. Dismissal is terminal — the ticket
            cannot be reopened from here.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            onClick={() => ticket && onConfirm(ticket.id)}
          >
            {pending ? "Dismissing..." : "Dismiss"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
