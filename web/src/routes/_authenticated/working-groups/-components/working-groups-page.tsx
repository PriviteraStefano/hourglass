import { useState } from "react";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { PlusIcon, SearchIcon, UsersIcon } from "lucide-react";
import { ActivitiesApis } from "@/api/activities";
import { WorkingGroupsApis } from "@/api/working-groups";
import { orgMembersQueryOpts } from "@/api/units";
import { Header, Body } from "@/components/layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { WorkingGroup } from "@/types";
import { WorkingGroupFormDialog } from "./working-group-form-dialog";
import { DeleteWorkingGroupDialog } from "./delete-working-group-dialog";

/**
 * Working Groups surface (ADR-P-011 D-4 / UI-SPEC §Working Groups).
 *
 * Header carries the single h1 "Working Groups" + search + the sole accent
 * element (right-aligned "New working group" CTA). Cards render in muted/card
 * surfaces only. v0.1 scope = list + create/edit + members — no availability
 * or validity warnings (P-008 scope, explicitly deferred).
 */
export function WorkingGroupsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [editWg, setEditWg] = useState<WorkingGroup | null>(null);
  const [deleteWg, setDeleteWg] = useState<WorkingGroup | null>(null);

  const { data: workingGroups } = useSuspenseQuery(
    WorkingGroupsApis.workingGroupsQueryOpts
  );

  const filtered = workingGroups.filter((wg) =>
    wg.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <>
      <Header>
        <h1 className="text-xl font-semibold">Working Groups</h1>
        <div className="ml-auto flex items-center gap-4">
          <div className="relative">
            <SearchIcon className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search working groups..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-64 pl-8"
            />
          </div>
          <Button onClick={() => setCreateOpen(true)}>
            <PlusIcon className="w-4 h-4 mr-1" />
            New working group
          </Button>
        </div>
      </Header>

      <Body>
        <div className="h-full overflow-y-auto p-6">
          {workingGroups.length === 0 ? (
            <WorkingGroupsEmptyState onCreate={() => setCreateOpen(true)} />
          ) : filtered.length === 0 ? (
            <div className="py-16 text-center text-sm text-muted-foreground">
              No working groups match your search
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filtered.map((wg) => (
                <WorkingGroupCard
                  key={wg.id}
                  wg={wg}
                  onEdit={() => setEditWg(wg)}
                  onDelete={() => setDeleteWg(wg)}
                />
              ))}
            </div>
          )}

          <WorkingGroupFormDialog
            open={createOpen}
            onOpenChange={setCreateOpen}
            mode="create"
          />
          <WorkingGroupFormDialog
            open={!!editWg}
            onOpenChange={(open) => {
              if (!open) setEditWg(null);
            }}
            mode="edit"
            workingGroup={editWg}
          />
          <DeleteWorkingGroupDialog
            wg={deleteWg}
            onClose={() => setDeleteWg(null)}
          />
        </div>
      </Body>
    </>
  );
}

/** Locked UI-SPEC empty state copy + single accent CTA. */
function WorkingGroupsEmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
      <div className="bg-primary/10 p-3 rounded-full">
        <UsersIcon className="h-6 w-6 text-primary" />
      </div>
      <h2 className="text-xl font-semibold">No working groups yet</h2>
      <p className="max-w-sm text-sm text-muted-foreground">
        Working groups assign people to activities. Create one to start
        staffing work.
      </p>
      <Button onClick={onCreate} className="mt-2">
        <PlusIcon className="w-4 h-4 mr-1" />
        New working group
      </Button>
    </div>
  );
}

function WorkingGroupCard({
  wg,
  onEdit,
  onDelete,
}: {
  wg: WorkingGroup;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { data: activities } = useQuery(ActivitiesApis.activitiesQueryOpts("owned"));
  const { data: orgMembers } = useQuery(orgMembersQueryOpts);
  const { data: members } = useQuery(
    WorkingGroupsApis.workingGroupMembersQueryOpts(wg.id)
  );

  // WG payload carries no joined names — resolve activity/manager client-side
  // against the activities + org-members caches (same data the form pickers
  // use). subproject_id is the legacy field that anchors the WG to an activity.
  const activityName = activities?.find((a) => a.id === wg.subproject_id)?.name;
  const managerName = orgMembers?.find((m) => m.user_id === wg.manager_id)?.user_name;
  const memberCount = members?.length ?? 0;

  return (
    <div className="border rounded-lg bg-card p-4 space-y-3">
      <div>
        <h3 className="text-sm font-medium">{wg.name}</h3>
        <p className="text-sm text-muted-foreground">
          {activityName ?? "Activity not found"}
        </p>
      </div>

      <div className="space-y-1.5 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
          <span className="w-20 shrink-0">Manager</span>
          <span className="truncate">{managerName ?? "—"}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-20 shrink-0">Members</span>
          <span className="truncate">
            {memberCount} {memberCount === 1 ? "member" : "members"}
          </span>
        </div>
      </div>

      <div className="flex gap-2 pt-1">
        <Button variant="outline" size="sm" onClick={onEdit}>
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onDelete}
          className="text-destructive hover:text-destructive"
        >
          Delete
        </Button>
      </div>
    </div>
  );
}
