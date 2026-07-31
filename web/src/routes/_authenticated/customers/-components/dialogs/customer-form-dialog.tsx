import * as React from "react";
import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Field,
  FieldContent,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  CustomersApis,
  type CreateCustomerRequest,
  type UpdateCustomerRequest,
} from "@/api/customers";
import { useCustomersStore } from "../../-context/customers-context";

const customerFormSchema = z.object({
  company_name: z.string().min(1, "Company name is required"),
  contact_name: z.string().optional(),
  email: z.string().email("Invalid email").optional().or(z.literal("")),
  phone: z.string().optional(),
  vat_number: z.string().optional(),
  address: z.string().optional(),
});

type CustomerFormData = z.infer<typeof customerFormSchema>;

export function CustomerFormDialog() {
  const open = useCustomersStore((s) => s.formOpen);
  const onOpenChange = useCustomersStore((s) => s.setFormOpen);
  const mode = useCustomersStore((s) => s.formMode);
  const editingCustomer = useCustomersStore((s) => s.editingCustomer);

  const createCustomer = useMutation({
    ...CustomersApis.createCustomerMutationOpts,
    onError: (error: Error) => {
      toast.error(error.message || "Failed to create customer");
    },
  });

  const updateCustomer = useMutation({
    ...CustomersApis.updateCustomerMutationOpts,
    onError: (error: Error) => {
      toast.error(error.message || "Failed to update customer");
    },
  });

  const form = useForm<CustomerFormData>({
    resolver: zodResolver(customerFormSchema),
    defaultValues: {
      company_name: editingCustomer?.company_name || "",
      contact_name: editingCustomer?.contact_name || "",
      email: editingCustomer?.email || "",
      phone: editingCustomer?.phone || "",
      vat_number: editingCustomer?.vat_number || "",
      address: editingCustomer?.address || "",
    },
  });

  React.useEffect(() => {
    if (open) {
      if (mode === "edit" && editingCustomer) {
        form.reset({
          company_name: editingCustomer.company_name,
          contact_name: editingCustomer.contact_name || "",
          email: editingCustomer.email || "",
          phone: editingCustomer.phone || "",
          vat_number: editingCustomer.vat_number || "",
          address: editingCustomer.address || "",
        });
      } else {
        form.reset({
          company_name: "",
          contact_name: "",
          email: "",
          phone: "",
          vat_number: "",
          address: "",
        });
      }
    }
  }, [open, mode, editingCustomer, form]);

  const handleSubmit = async (data: CustomerFormData) => {
    try {
      if (mode === "edit" && editingCustomer) {
        const updateData: UpdateCustomerRequest = {
          company_name: data.company_name.trim(),
          contact_name: data.contact_name?.trim() || undefined,
          email: data.email?.trim() || undefined,
          phone: data.phone?.trim() || undefined,
          vat_number: data.vat_number?.trim() || undefined,
          address: data.address?.trim() || undefined,
        };
        await toast.promise(
          updateCustomer.mutateAsync({
            id: editingCustomer.id,
            data: updateData,
          }),
          {
            loading: "Updating customer...",
            success: "Customer updated",
            error: "Failed to update customer",
          }
        );
      } else {
        const createData: CreateCustomerRequest = {
          company_name: data.company_name.trim(),
          contact_name: data.contact_name?.trim() || undefined,
          email: data.email?.trim() || undefined,
          phone: data.phone?.trim() || undefined,
          vat_number: data.vat_number?.trim() || undefined,
          address: data.address?.trim() || undefined,
        };
        await toast.promise(createCustomer.mutateAsync(createData), {
          loading: "Creating customer...",
          success: "Customer created",
          error: "Failed to create customer",
        });
      }
      onOpenChange(false);
      form.reset();
    } catch {
      // error handled by toast.promise
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Add Customer" : "Edit Customer"}
          </DialogTitle>
          <DialogDescription>
            {mode === "create"
              ? "Add a new customer to your organization."
              : "Update customer details."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
          <Controller
            name="company_name"
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldContent>
                  <FieldLabel htmlFor="company_name">Company Name *</FieldLabel>
                  <Input
                    id="company_name"
                    placeholder="Acme Corp"
                    {...field}
                    disabled={mode === "edit" && !!editingCustomer?.is_internal}
                  />
                  {mode === "edit" && editingCustomer?.is_internal && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Company name is locked for internal customers
                    </p>
                  )}
                </FieldContent>
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />

          <Controller
            name="contact_name"
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldContent>
                  <FieldLabel htmlFor="contact_name">Contact Name</FieldLabel>
                  <Input id="contact_name" placeholder="John Doe" {...field} />
                </FieldContent>
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />

          <div className="grid grid-cols-2 gap-4">
            <Controller
              name="email"
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel htmlFor="email">Email</FieldLabel>
                    <Input
                      id="email"
                      type="email"
                      placeholder="john@example.com"
                      {...field}
                    />
                  </FieldContent>
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />

            <Controller
              name="phone"
              control={form.control}
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldContent>
                    <FieldLabel htmlFor="phone">Phone</FieldLabel>
                    <Input
                      id="phone"
                      placeholder="+1 234 567 8900"
                      {...field}
                    />
                  </FieldContent>
                  <FieldError errors={[fieldState.error]} />
                </Field>
              )}
            />
          </div>

          <Controller
            name="vat_number"
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldContent>
                  <FieldLabel htmlFor="vat_number">VAT Number</FieldLabel>
                  <Input id="vat_number" placeholder="US123456789" {...field} />
                </FieldContent>
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />

          <Controller
            name="address"
            control={form.control}
            render={({ field, fieldState }) => (
              <Field data-invalid={fieldState.invalid}>
                <FieldContent>
                  <FieldLabel htmlFor="address">Address</FieldLabel>
                  <Textarea
                    id="address"
                    placeholder="123 Main St, City, Country"
                    rows={2}
                    {...field}
                  />
                </FieldContent>
                <FieldError errors={[fieldState.error]} />
              </Field>
            )}
          />

          <DialogFooter className="pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createCustomer.isPending || updateCustomer.isPending}
            >
              {createCustomer.isPending || updateCustomer.isPending
                ? "Saving..."
                : mode === "create"
                  ? "Create"
                  : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
