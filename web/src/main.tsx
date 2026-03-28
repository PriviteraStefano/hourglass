import {StrictMode} from 'react'
import {createRoot} from 'react-dom/client'
import {createRouter, RouterProvider} from '@tanstack/react-router'
import {routeTree} from './routeTree.gen'
import './index.css'
import {QueryClientProvider} from "@tanstack/react-query";
import {queryClient} from "./lib/query-client";


const router = createRouter({
  routeTree,
  context: {
    client: queryClient,
  },
})
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)