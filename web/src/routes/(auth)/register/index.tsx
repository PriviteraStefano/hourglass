import {createFileRoute} from '@tanstack/react-router'
import {BootstrapOrgForm} from "@/routes/(auth)/register/-components/bootstrap-org-form.tsx";

export const Route = createFileRoute('/(auth)/register/')({
  component: () => (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <BootstrapOrgForm />
    </div>
  ),
})
