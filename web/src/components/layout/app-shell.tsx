import { AppSidebar } from "./sidebar.tsx";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className={"overflow-clip"}>{children}</SidebarInset>
    </SidebarProvider>
  );
}
