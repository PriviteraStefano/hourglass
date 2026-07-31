import { createFileRoute } from "@tanstack/react-router";
import { CustomersPage } from "./-components/customers-page";
import { CustomersApis } from "@/api/customers";
import { RouteError } from "@/components/layout/route-error";

export const Route = createFileRoute("/_authenticated/customers/")({
  component: CustomersPage,
  loader: ({ context: { client } }) =>
    client.ensureQueryData(CustomersApis.customersQueryOpts()),
  // Leaf-level boundary (P0-4): the error attaches to THIS match, which is
  // rebuilt on navigation — unlike the layout-level fallback, whose match
  // persists and can keep the panel tripped after recovery.
  errorComponent: RouteError,
  pendingMs: 50,
});
