import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {useOrgHierarchy} from "@/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx";
import {useMutation} from "@tanstack/react-query";
import {deleteUnitMutationOpts} from "@/api/units.ts";

export function DeleteConfirmDialog() {
  const {selectedUnit, showDeleteConfirm, setShowDeleteConfirm, setSelectedUnit} = useOrgHierarchy()
  const {mutateAsync: deleteUnit} = useMutation(deleteUnitMutationOpts)

  return (
    <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Unit</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete "{selectedUnit?.name}"? This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={async () => {
              if (selectedUnit) {
                try {
                  await deleteUnit(selectedUnit.id)
                  setShowDeleteConfirm(false)
                  setSelectedUnit(null)
                } catch {
                  toast.error("Failed to delete unit")
                }
              }
            }}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}