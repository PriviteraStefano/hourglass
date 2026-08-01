import { InvitationAcceptForm } from "./-components/invitation-accept-form";
import { createFileRoute } from "@tanstack/react-router";
import z from "zod";

export const Route = createFileRoute("/_auth/invite/")({
  component: () => (
    <div className="flex min-h-screen items-center justify-center p-4">
      <InvitationAcceptForm />
    </div>
  ),
  validateSearch: z.object({
    token: z.string().optional(),
    code: z.string().optional(),
  }),
});
