import {
  CheckCircleIcon,
  ClockIcon,
  FileTextIcon,
  FolderIcon, HourglassIcon,
  type LucideIcon,
  ReceiptIcon,
  SettingsIcon
} from 'lucide-react'
import {Link} from '@tanstack/react-router'
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
} from '@/components/ui/sidebar'
import {ThemeToggle} from '@/components/theme-toggle.tsx'
import {ProfileMenu} from "@/components/app/profile-menu.tsx";

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
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size={"lg"}
              render={
                <Link to={"/"}>
                  <HourglassIcon className={"size-5! ml-1.5"}/>
                  <span className="text-base font-semibold">Hourglass</span>
                </Link>
              }
            />
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
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
        <ProfileMenu/>
        <ThemeToggle/>
      </SidebarFooter>
    </Sidebar>
  )
}