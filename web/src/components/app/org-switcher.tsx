import {SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar} from "@/components/ui/sidebar.tsx";
import {useMutation, useQueryClient, useSuspenseQuery} from "@tanstack/react-query";
import {AuthApis} from "@/api/auth.ts";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu.tsx";
import {ChevronsUpDown, HourglassIcon, Plus} from "lucide-react";
import {useNavigate} from "@tanstack/react-router";

export function OrgSwitcher() {
  const {isMobile} = useSidebar()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const {data: profileData} = useSuspenseQuery(AuthApis.profileQueryOpts)
  const {data: membershipsData} = useSuspenseQuery(AuthApis.membershipsQueryOpts)
  const organization = profileData?.organization
  const {mutateAsync: switchOrg} = useMutation(AuthApis.switchOrganizationMutationOpts)

  const handleSwitch = async (orgId: string) => {
    await switchOrg({organization_id: orgId})
    queryClient.clear()
    await queryClient.invalidateQueries({queryKey: ['auth', 'me']})
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger render={
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            />
          }>
            <div
              className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
              <HourglassIcon className="size-4"/>
            </div>
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{organization.name}</span>
              <span className="truncate text-xs">{organization.slug}</span>
            </div>
            <ChevronsUpDown className="ml-auto"/>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            align="start"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="text-xs text-muted-foreground">
                Organizations
              </DropdownMenuLabel>
              {membershipsData.memberships.map((membership) => (
                <DropdownMenuItem
                  key={membership.organization.id}
                  onClick={() => handleSwitch(membership.organization.id)}
                  className="gap-2 p-2"
                >
                  <div className="flex flex-col">
                    <span>{membership.organization.name}</span>
                    <span className="text-xs text-muted-foreground">{membership.membership.role}</span>
                  </div>
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator/>
              <DropdownMenuItem className="gap-2 p-2" disabled>
                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                  <Plus className="size-4"/>
                </div>
                <div className="font-medium text-muted-foreground">Add organization</div>
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
