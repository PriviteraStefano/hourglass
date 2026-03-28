import {createFileRoute, Outlet, redirect} from '@tanstack/react-router'
import {AppShell} from "@/src/components/layout/app-shell.tsx";
import {AuthApis} from "@/src/api/auth.ts";


export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({context: {client}}) => {
    await client.fetchQuery(AuthApis.profileQueryOpts)
    const profile = await client.ensureQueryData(AuthApis.profileQueryOpts)
    return {
      profile: profile
    }
  },
  component: () => (
    <AppShell>
      <Outlet/>
    </AppShell>
  ),
})