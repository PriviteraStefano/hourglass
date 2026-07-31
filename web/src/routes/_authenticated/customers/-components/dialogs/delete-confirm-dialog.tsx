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
import { useCustomersStore } from "../../-context/customers-context";
import { useMutation } from "@tanstack/react-query";
import { CustomersApis } from "@/api/customers";
import { toast } from "sonner";

export function DeleteConfirmDialog() {
  const selectedCustomer = useCustomersStore((s) => s.selectedCustomer);
  const deleteOpen = useCustomersStore((s) => s.deleteOpen);
  const setDeleteOpen = useCustomersStore((s) => s.setDeleteOpen);
  const setSelectedCustomer = useCustomersStore((s) => s.setSelectedCustomer);

  const deleteCustomer = useMutation({
    ...CustomersApis.deleteCustomerMutationOpts,
    onError: (error: Error) => {
      if (error.message.includes("409")) {
        toast.error("Cannot delete customer with linked contracts");
      } else {
        toast.error(error.message || "Failed to delete customer");
      }
    },
  });

  if (!selectedCustomer) return null;

  const handleConfirm = async () => {
    await deleteCustomer.mutateAsync(selectedCustomer.id);
    setDeleteOpen(false);
    setSelectedCustomer(null);
  };

  return (
    <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Customer</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete{" "}
            <strong>{selectedCustomer.company_name}</strong>? This action cannot
            be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={deleteCustomer.isPending}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {deleteCustomer.isPending ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
