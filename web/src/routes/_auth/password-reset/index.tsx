import { PasswordResetRequestForm } from "./-components/password-reset-request-form";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_auth/password-reset/")({
  component: () => (
    <div className="flex min-h-screen items-center justify-center p-4">
      <PasswordResetRequestForm />
    </div>
  ),
});
