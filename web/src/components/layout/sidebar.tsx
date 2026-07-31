import {
  Building2,
  CalendarClockIcon,
  CheckCircleIcon,
  ClockIcon,
  DownloadIcon,
  FileTextIcon,
  FolderIcon,
  HouseIcon,
  type LucideIcon,
  NetworkIcon,
  ReceiptIcon,
  SettingsIcon,
  SidebarCloseIcon,
  SidebarOpenIcon,
  TicketIcon,
  UsersIcon,
} from "lucide-react";
import { Fragment } from "react";
import {
  Link,
  ToPathOption,
  useMatchRoute,
  useRouteContext,
} from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/theme-toggle.tsx";
import { ProfileMenu } from "@/components/app/profile-menu.tsx";
import { OrgSwitcher } from "@/components/app/org-switcher.tsx";
import { Suspense } from "react";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { WorkingGroupsApis } from "@/api/working-groups.ts";
import {
  deriveApprovalStages,
  isAdminVisible,
  isEconomicsVisible,
  isReviewVisible,
} from "@/lib/role-visibility.ts";

type NavItem = {
  label: string;
  href: ToPathOption;
  icon: LucideIcon;
  disabled?: boolean;
  /** Tooltip copy for disabled placeholders (locked by UI-SPEC Copywriting Contract). */
  tooltip?: string;
};

// D-1 pillar-mapped group structure (render order + exact casing locked by
// ADR-P-011 D-1). group null = ungrouped top item (Today).
const navStructure: Array<{ group: string | null; items: NavItem[] }> = [
  {
    group: null,
    items: [{ label: "Today", href: "/", icon: HouseIcon }],
  },
  {
    group: "Track",
    items: [
      { label: "Time", href: "/time-entries", icon: ClockIcon },
      { label: "Expenses", href: "/expenses", icon: ReceiptIcon },
      {
        label: "Tickets",
        href: "/tickets",
        icon: TicketIcon,
        disabled: true,
        tooltip: "Tickets arrive in v0.2",
      },
    ],
  },
  {
    group: "Work",
    items: [
      { label: "Activities", href: "/activities", icon: FolderIcon },
      { label: "Working Groups", href: "/working-groups", icon: UsersIcon },
    ],
  },
  {
    group: "People",
    items: [
      { label: "Org", href: "/org-hierarchy", icon: NetworkIcon },
      {
        label: "Availability",
        href: "/availability",
        icon: CalendarClockIcon,
        disabled: true,
        tooltip: "Availability lands with the staffing schema",
      },
    ],
  },
  {
    group: "Economics",
    items: [
      { label: "Contracts", href: "/contracts", icon: FileTextIcon },
      { label: "Customers", href: "/customers", icon: Building2 },
    ],
  },
  {
    group: "Review",
    items: [{ label: "Approvals", href: "/approvals", icon: CheckCircleIcon }],
  },
  {
    group: "Reports",
    items: [{ label: "Exports", href: "/exports", icon: DownloadIcon }],
  },
  {
    group: "Admin",
    items: [
      {
        label: "Settings",
        href: "/settings",
        icon: SettingsIcon,
        disabled: true,
      },
    ],
  },
];

export function AppSidebar() {
  const matchRoute = useMatchRoute();
  const { state, toggleSidebar } = useSidebar();
  // Profile is hydrated by the _authenticated layout's beforeLoad — read it
  // from the route context (same source ProfileMenu renders from).
  const { profile } = useRouteContext({ from: "/_authenticated" });
  // Non-suspense: an unloaded WG list simply means "no WG-derived stages yet"
  // (deriveApprovalStages treats undefined as org-role only, threat T-10-02-2).
  const { data: workingGroups } = useQuery(
    WorkingGroupsApis.workingGroupsQueryOpts,
  );

  const stages = deriveApprovalStages(profile, workingGroups);
  const role = profile.membership.role;

  // ADR-P-011 D-5 role matrix — hiding is UX scoping only; every role-
  // restricted surface stays backend-gated (threat T-10-02-1). No client-side
  // route guards are invented here.
  const groups = navStructure.filter(({ group }) => {
    switch (group) {
      case "Economics":
        return isEconomicsVisible(role);
      case "Review":
        return isReviewVisible(role, stages);
      case "Admin":
        return isAdminVisible(role); // false for every v0.1 role
      default:
        return true;
    }
  });

  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>
        <Suspense fallback={<Skeleton className="h-6 w-32" />}>
          <OrgSwitcher />
        </Suspense>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuButton
              className={"border z-10"}
              onClick={toggleSidebar}
            >
              {state === "collapsed" ? (
                <>
                  <SidebarOpenIcon />
                  <span>Expand</span>
                </>
              ) : (
                <>
                  <SidebarCloseIcon />
                  <span>Collapse</span>
                </>
              )}
            </SidebarMenuButton>
          </SidebarMenu>
        </SidebarGroup>

        {groups.map(({ group, items }) => (
          <Fragment key={group ?? "today"}>
            <SidebarSeparator />
            <SidebarGroup>
              {group && <SidebarGroupLabel>{group}</SidebarGroupLabel>}
              <SidebarGroupContent>
                <SidebarMenu>
                  {items.map((item) => (
                    <SidebarMenuItem key={String(item.href)}>
                      <SidebarMenuButton
                        isActive={!!matchRoute({ to: item.href, fuzzy: item.href !== "/" })}
                        tooltip={item.tooltip ?? item.label}
                        render={
                          <Link to={item.href} disabled={item.disabled}>
                            <item.icon />
                            <span>{item.label}</span>
                          </Link>
                        }
                      />
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </Fragment>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <ProfileMenu />
        <ThemeToggle />
      </SidebarFooter>
    </Sidebar>
  );
}
