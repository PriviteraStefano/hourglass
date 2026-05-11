import {
  CheckCircleIcon,
  ClockIcon,
  FileTextIcon,
  FolderIcon,
  type LucideIcon,
  NetworkIcon,
  ReceiptIcon,
  SettingsIcon,
  SidebarCloseIcon,
  SidebarOpenIcon
} from 'lucide-react'
import {Link, ToPathOption, useMatchRoute} from '@tanstack/react-router'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from '@/components/ui/sidebar'
import {ThemeToggle} from '@/components/theme-toggle.tsx'
import {ProfileMenu} from "@/components/app/profile-menu.tsx";
import {OrgSwitcher} from "@/components/app/org-switcher.tsx";


const navItems: Array<{ label: string; href: ToPathOption; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Time', href: '/time-entries', icon: ClockIcon},
  {label: 'Expenses', href: '/expenses', icon: ReceiptIcon, disabled: true},
  {label: 'Approvals', href: '/approvals', icon: CheckCircleIcon, disabled: true},
]

const projectItems: Array<{ label: string; href: ToPathOption; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Contracts', href: '/contracts', icon: FileTextIcon},
  {label: 'Projects', href: '/projects', icon: FolderIcon},
  {label: 'Org Hierarchy', href: '/org-hierarchy', icon: NetworkIcon},
]

const settingsItems: Array<{ label: string; href: ToPathOption; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Settings', href: '/settings', icon: SettingsIcon, disabled: true},
]

export function AppSidebar() {
  const matchRoute = useMatchRoute()
  const {state, toggleSidebar} = useSidebar()


  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>
        <OrgSwitcher organizations={[]}/>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuButton className={"border"} onClick={toggleSidebar}>
                {state === "collapsed"
                  ? (
                    <>
                      <SidebarOpenIcon/>
                      <span>Expand</span>
                    </>
                  ) : (
                    <>
                      <SidebarCloseIcon/>
                      <span>Collapse</span>
                    </>
                  )}
              </SidebarMenuButton>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={String(item.href)}>
                  <SidebarMenuButton
                    isActive={!!matchRoute({to: item.href})}
                    tooltip={item.label}
                    render={
                      <Link to={item.href} disabled={item.disabled}>
                        <item.icon/>
                        <span>{item.label}</span>
                      </Link>
                    }
                  />
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarSeparator/>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {projectItems.map((item) => (
                <SidebarMenuItem key={String(item.href)}>
                  <SidebarMenuButton
                    isActive={!!matchRoute({to: item.href})}
                    tooltip={item.label}
                    render={
                      <Link to={item.href} disabled={item.disabled}>
                        <item.icon/>
                        <span>{item.label}</span>
                      </Link>
                    }
                  />
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarSeparator/>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {settingsItems.map((item) => (
                <SidebarMenuItem key={String(item.href)}>
                  <SidebarMenuButton
                    isActive={!!matchRoute({to: item.href})}
                    tooltip={item.label}
                    render={
                      <Link to={item.href} disabled={item.disabled}>
                        <item.icon/>
                        <span>{item.label}</span>
                      </Link>
                    }
                  />
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <ProfileMenu/>
        <ThemeToggle/>
      </SidebarFooter>
    </Sidebar>
  )
}
