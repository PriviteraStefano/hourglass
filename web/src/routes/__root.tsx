import {createRootRouteWithContext, Outlet} from '@tanstack/react-router'
import {Toaster} from "@/components/ui/sonner.tsx";
import {TooltipProvider} from "@/components/ui/tooltip.tsx";
import {ThemeProvider} from "@/components/theme-provider.tsx";
import type {QueryClient} from "@tanstack/react-query";

interface RouterContext {
  client: QueryClient
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => (
    <ThemeProvider defaultTheme="system" storageKey="hourglass-theme" attribute="class">
      <TooltipProvider>
        <Outlet />
        <Toaster />
      </TooltipProvider>
    </ThemeProvider>
  ),
})