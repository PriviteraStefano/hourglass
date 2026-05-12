import {createFileRoute, redirect} from '@tanstack/react-router'
import {AuthApis} from "@/api/auth.ts";
import {BootstrapForm} from "@/routes/_auth/bootstrap/-components/bootstrap-form.tsx";

export const Route = createFileRoute('/_auth/bootstrap/')({
  beforeLoad: async ({context: {client}}) => {
    try {
      const data = await client.fetchQuery(AuthApis.bootstrapCheckQueryOpts)
      if (!data.needs_bootstrap) {
        throw redirect({ to: '/login' })
      }
    } catch {
    }
  },
  component: () => (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <BootstrapForm />
    </div>
  ),
})