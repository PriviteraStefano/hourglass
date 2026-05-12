import {InvitationAcceptForm} from "./-components/invitation-accept-form"
import {createFileRoute} from "@tanstack/react-router";

export const Route = createFileRoute('/_auth/invite/')({
  component: () => (
    <div className="flex min-h-screen items-center justify-center p-4">
      <InvitationAcceptForm />
    </div>
  ),
})

