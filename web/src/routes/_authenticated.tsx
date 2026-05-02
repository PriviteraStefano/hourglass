import {createFileRoute, redirect, Outlet} from '@tanstack/react-router'
import {AppShell} from "@/components/layout/app-shell.tsx";
import {AuthApis} from "@/api/auth.ts";
import {LoaderIcon} from "lucide-react";


export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({context: {client}}) => {
    try {
      const profile = await client.fetchQuery(AuthApis.profileQueryOpts)
      return { profile }
    } catch {
      throw redirect({ to: '/login' })
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