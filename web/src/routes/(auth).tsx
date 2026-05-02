import {Outlet, createFileRoute, redirect} from '@tanstack/react-router'
import {AuthApis} from "@/api/auth.ts";

export const Route = createFileRoute('/(auth)')({
  beforeLoad: async ({context: {client}, throw: throwRedirect}) => {
    try {
      await client.fetchQuery(AuthApis.profileQueryOpts)
      throw throwRedirect({ to: '/' })
    } catch {
    }
  },
  component: () => <Outlet />,
})