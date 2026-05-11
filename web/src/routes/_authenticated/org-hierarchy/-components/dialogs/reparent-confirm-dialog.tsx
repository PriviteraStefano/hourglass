import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {useOrgHierarchyStore} from "@/routes/_authenticated/org-hierarchy/-context/org-hierarchy-context.tsx";
import {useMutation} from "@tanstack/react-query";
import {updateUnitMutationOpts} from "@/api/units.ts";
import {toast} from "sonner";

export function ReparentConfirmDialog() {
  const draggingUnit = useOrgHierarchyStore(s => s.draggingUnit)
  const reparentTarget = useOrgHierarchyStore(s => s.reparentTarget)
  const setDraggingUnit = useOrgHierarchyStore(s => s.setDraggingUnit)
  const setReparentTarget = useOrgHierarchyStore(s => s.setReparentTarget)

  const reparentOpen = draggingUnit !== null && reparentTarget !== null

  const {mutateAsync: updateUnit} = useMutation(updateUnitMutationOpts)

  if (!draggingUnit || !reparentTarget) return null

  const onOpenChange = (open: boolean) => {
    if (!open) {
      setDraggingUnit(null)
      setReparentTarget(null)
    }
  }

  const handleConfirm = async () => {
    try {
      await updateUnit({
        id: draggingUnit.id,
        body: {
          name: draggingUnit.name,
          code: draggingUnit.code,
          description: draggingUnit.description,
          parent_unit_id: reparentTarget.id
        }
      })
      toast.success("Unit moved successfully")
      onOpenChange(false)
    } catch {
      toast.error("Failed to move unit")
    }
  }

  return (
    <AlertDialog open={reparentOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Move Unit</AlertDialogTitle>
          <AlertDialogDescription>
            {reparentTarget
              ? `Move "${draggingUnit?.name}" under "${reparentTarget.name}"?`
              : `Move "${draggingUnit?.name}" to root level?`
            }
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={handleConfirm}>
            Move
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
