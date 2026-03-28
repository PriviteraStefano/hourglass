import {createRootRouteWithContext, Outlet} from '@tanstack/react-router'
import {Toaster} from "@/src/components/ui/sonner.tsx";
import type {QueryClient} from "@tanstack/react-query";

interface RouterContext {
  client: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => (
    <>
      <Outlet />
      <Toaster />
    </>
  ),
})