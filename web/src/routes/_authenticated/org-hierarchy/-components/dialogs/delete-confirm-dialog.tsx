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
import { useOrgHierarchyStore } from "@/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteUnitMutationOpts } from "@/api/units.ts";
import { toast } from "sonner";

export function DeleteConfirmDialog() {
  const selectedUnit = useOrgHierarchyStore((s) => s.selectedUnit);
  const deleteOpen = useOrgHierarchyStore((s) => s.deleteOpen);
  const setDeleteOpen = useOrgHierarchyStore((s) => s.setDeleteOpen);
  const setSelectedUnit = useOrgHierarchyStore((s) => s.setSelectedUnit);

  const queryClient = useQueryClient();
  const { mutateAsync: deleteUnit } = useMutation(deleteUnitMutationOpts);

  if (!selectedUnit) return null;

  const handleConfirm = async () => {
    try {
      await deleteUnit(selectedUnit.id);
      toast.success("Unit deleted successfully");
      setDeleteOpen(false);
      setSelectedUnit(null);
      queryClient.invalidateQueries({ queryKey: ["units"] });
    } catch {
      toast.error("Failed to delete unit");
    }
  };

  return (
    <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Unit</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete "{selectedUnit?.name}"? This action
            cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
