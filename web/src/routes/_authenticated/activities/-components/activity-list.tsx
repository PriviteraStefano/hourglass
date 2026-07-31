import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { GlobeIcon, LockIcon, PlusIcon, SearchIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ActivitiesApis } from "@/api/activities";
import { CreateActivityDialog } from "./create-activity-dialog";
import type { ActivityResponse } from "@/types/models";

interface ActivityListProps {
  initialTab?: "owned" | "adopted" | "all";
}

export function ActivityList({ initialTab = "owned" }: ActivityListProps) {
  const navigate = useNavigate();

  const [tab, setTab] = useState<"owned" | "adopted" | "all">(initialTab);
  const [searchQuery, setSearchQuery] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const { data: activities, isLoading } = useQuery(
    ActivitiesApis.activitiesQueryOpts(tab)
  );

  const filteredActivities = activities?.filter((a: ActivityResponse) =>
    a.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleTabChange = (newTab: string) => {
    const next = newTab as "owned" | "adopted" | "all";
    setTab(next);
    navigate({ to: "/activities", search: { tab: next } });
  };

  const handleRowClick = (activity: ActivityResponse) => {
    navigate({
      to: "/activities/$id",
      params: { id: activity.id },
      search: { from: tab },
    });
  };

  return (
    <>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-semibold">Activities</h1>
        <div className="flex items-center gap-4">
          <div className="relative">
            <SearchIcon className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search activities..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-64 pl-8"
            />
          </div>
          {tab === "owned" && (
            <Button onClick={() => setCreateDialogOpen(true)}>
              <PlusIcon className="w-4 h-4 mr-1" />
              New activity
            </Button>
          )}
        </div>
      </div>

      <Tabs value={tab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="owned">Owned</TabsTrigger>
          <TabsTrigger value="adopted">Adopted</TabsTrigger>
          <TabsTrigger value="all">All</TabsTrigger>
        </TabsList>

        <TabsContent value={tab} className="mt-4">
          {isLoading ? (
            <div className="text-center py-8 text-muted-foreground">
              Loading...
            </div>
          ) : filteredActivities?.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchQuery
                ? "No activities match your search"
                : `No ${tab} activities`}
            </div>
          ) : (
            <div className="border rounded-lg divide-y">
              {filteredActivities?.map((activity: ActivityResponse) => (
                <div
                  key={activity.id}
                  className="flex items-center justify-between p-4 hover:bg-muted/50 cursor-pointer"
                  onClick={() => handleRowClick(activity)}
                >
                  <div className="flex items-center gap-3">
                    {activity.is_shared ? (
                      <GlobeIcon className="w-4 h-4 text-muted-foreground" />
                    ) : (
                      <LockIcon className="w-4 h-4 text-muted-foreground" />
                    )}
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{activity.name}</span>
                        <Badge variant="secondary" className="text-xs">
                          {activity.kind}
                        </Badge>
                        {activity.is_shared && (
                          <Badge variant="outline" className="text-xs">
                            Shared
                          </Badge>
                        )}
                        {tab === "adopted" &&
                          activity.created_by_org_name && (
                            <span className="text-xs text-muted-foreground">
                              from {activity.created_by_org_name}
                            </span>
                          )}
                        {tab === "all" && activity.is_adopted && (
                          <Badge variant="outline" className="text-xs">
                            Already adopted
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <CreateActivityDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
    </>
  );
}
