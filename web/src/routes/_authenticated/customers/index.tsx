import { createFileRoute } from "@tanstack/react-router";
import { CustomersPage } from "./-components/customers-page";
import { CustomersApis } from "@/api/customers";

export const Route = createFileRoute("/_authenticated/customers/")({
  component: CustomersPage,
  loader: ({ context: { client } }) =>
    client.ensureQueryData(CustomersApis.customersQueryOpts()),
  pendingMs: 50,
});
