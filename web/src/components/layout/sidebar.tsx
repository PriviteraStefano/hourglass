import {
  ClockIcon,
  ReceiptIcon,
  CheckCircleIcon,
  FileTextIcon,
  FolderIcon,
  SettingsIcon,
  type LucideIcon
} from 'lucide-react'
import {Link} from '@tanstack/react-router'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
} from '@/src/components/ui/sidebar'
import {ThemeToggle} from '@/src/components/theme-toggle.tsx'

const navItems: Array<{ label: string; href: string; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Time', href: '/time-entries', icon: ClockIcon},
  {label: 'Expenses', href: '/expenses', icon: ReceiptIcon, disabled: true},
  {label: 'Approvals', href: '/approvals', icon: CheckCircleIcon, disabled: true},
]

const projectItems: Array<{ label: string; href: string; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Contracts', href: '/contracts', icon: FileTextIcon},
  {label: 'Projects', href: '/projects', icon: FolderIcon},
]

const settingsItems: Array<{ label: string; href: string; icon: LucideIcon; disabled?: boolean }> = [
  {label: 'Settings', href: '/settings', icon: SettingsIcon, disabled: true},
]

export function AppSidebar() {
  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
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
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
                    render={
                      <Link to={item.href}>
                        <item.icon/>
                        <span>{item.label}</span>
                      </Link>
                    }
                    disabled={item.disabled}
                    tooltip={item.label}
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
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
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
        <ThemeToggle />
      </SidebarFooter>
    </Sidebar>
  )
}