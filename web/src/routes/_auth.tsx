import {createFileRoute, isRedirect, Outlet, redirect} from '@tanstack/react-router'
import {AuthApis} from "@/api/auth.ts";

export const Route = createFileRoute('/_auth')({
  beforeLoad: async ({context: {client}}) => {
    // Best-effort profile check: if the user IS logged in, redirect to /.
    // If they're NOT logged in (401), swallow the error and render the outlet
    // (login/register/password-reset pages). This is the key fix that breaks
    // the infinite redirect loop — previously this threw on 401, which
    // triggered another redirect to /login, which re-fetched /auth/me, etc.
    try {
      await client.fetchQuery(AuthApis.profileQueryOpts)
      // Profile fetch succeeded → user is already authenticated, shouldn't see login
      throw redirect({to: '/'})
    } catch (err) {
      // If it's a genuine TanStack redirect, let it propagate
      if (isRedirect(err)) throw err
      // Otherwise it's a 401 or network error — user is not authenticated,
      // which is exactly what we want on the login page. Swallow it.
    }
  },
  component: () => <Outlet/>,
})