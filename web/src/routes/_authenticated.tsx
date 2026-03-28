import {createFileRoute, Outlet} from '@tanstack/react-router'
import {AppShell} from "@/components/layout/app-shell.tsx";
import {AuthApis} from "@/api/auth.ts";
import {LoaderIcon} from "lucide-react";


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
  pendingComponent: () => (
    <AppShell>
      <LoaderIcon className="animate-spin m-auto"/>
    </AppShell>
  ),
})