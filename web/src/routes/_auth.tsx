import {createFileRoute, Outlet} from '@tanstack/react-router'
import {AuthApis} from "@/api/auth.ts";

export const Route = createFileRoute('/_auth')({
  beforeLoad: async ({context: {client}}) => {
    await client.fetchQuery(AuthApis.profileQueryOpts)
  },
  component: () => <Outlet/>,
})