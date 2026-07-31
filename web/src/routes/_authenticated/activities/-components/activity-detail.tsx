import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeftIcon, GlobeIcon, LockIcon, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { ActivitiesApis } from "@/api/activities";
import { EditActivityDialog } from "./edit-activity-dialog";
import { Header, Body } from "@/components/layout";
import type { ActivityDetail as ActivityDetailType } from "@/types/models";

interface ActivityDetailProps {
  id: string;
  fromTab?: "owned" | "adopted" | "all";
}

export function ActivityDetail({ id, fromTab = "owned" }: ActivityDetailProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: detail, isLoading } = useQuery(
    ActivitiesApis.activityQueryOpts(id)
  );

  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const deleteActivity = useMutation({
    mutationFn: ActivitiesApis.deleteActivityMutationOpts.mutationFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["activities"] });
      toast.success("Activity deleted");
      navigate({ to: "/activities", search: { tab: fromTab } });
    },
    onError: (error: Error) => {
      setDeleteError(error.message);
    },
  });

  if (isLoading) {
    return (
      <>
        <Header>
          <h1 className="text-xl font-semibold">Activity</h1>
        </Header>
        <Body>
          <div className="h-full overflow-y-auto p-6">
            <div className="text-center py-8 text-muted-foreground">
              Loading...
            </div>
          </div>
        </Body>
      </>
    );
  }

  if (!detail) {
    return (
      <>
        <Header>
          <h1 className="text-xl font-semibold">Activity</h1>
        </Header>
        <Body>
          <div className="h-full overflow-y-auto p-6">
            <div className="text-center py-8 text-muted-foreground">
              Activity not found
            </div>
          </div>
        </Body>
      </>
    );
  }

  const d: ActivityDetailType = detail;
  const a = d.activity;
  const isAdopted = fromTab === "adopted";
  const contractName = d.commercial_context?.contract_id
    ? "Contract linked"
    : "None (internal work)";

  return (
    <>
      <Header>
        <Button
          variant="ghost"
          size="sm"
          onClick={() =>
            navigate({ to: "/activities", search: { tab: fromTab } })
          }
        >
          <ArrowLeftIcon className="w-4 h-4 mr-1" />
          Back to Activities
        </Button>

        {d.ancestry.length > 0 && (
          <nav className="flex items-center gap-1 text-sm text-muted-foreground flex-wrap">
            {d.ancestry
              .slice()
              .reverse()
              .map((ancestor) => (
                <span key={ancestor.id} className="flex items-center gap-1">
                  <span>{ancestor.name}</span>
                  <span>/</span>
                </span>
              ))}
            <span className="font-medium text-foreground">{a.name}</span>
          </nav>
        )}

        <div className="flex items-center gap-3">
          <h1 className="text-xl font-semibold">{a.name}</h1>
          {a.is_shared ? (
            <GlobeIcon className="w-5 h-5 text-muted-foreground" />
          ) : (
            <LockIcon className="w-5 h-5 text-muted-foreground" />
          )}
          <Badge variant="secondary">{a.kind}</Badge>
          {a.is_shared && <Badge variant="outline">Shared</Badge>}
        </div>

        <div className="ml-auto flex gap-2">
          <Button variant="outline" onClick={() => setEditOpen(true)}>
            Edit
          </Button>
          <Button
            variant="destructive"
            onClick={() => {
              setDeleteOpen(true);
              setDeleteError(null);
            }}
          >
            <Trash2 className="w-4 h-4 mr-1" />
            Delete
          </Button>
        </div>
      </Header>

      <Body>
        <div className="h-full overflow-y-auto p-6 space-y-4">
          {isAdopted && a.created_by_org_id && (
            <p className="text-sm text-muted-foreground">Adopted</p>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Contract</span>
                <span>{contractName}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Kind</span>
                <span className="capitalize">{a.kind}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Billable</span>
                <span>
                  {d.billable === true
                    ? "Billable"
                    : d.billable === false
                      ? "Non-billable"
                      : "Inherited (unset)"}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Governance</span>
                <span className="capitalize">
                  {a.governance_model.replace("_", " ")}
                </span>
              </div>
              {a.is_shared && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Adoption Count</span>
                  <span>—</span>
                </div>
              )}
              {a.description && (
                <p className="text-sm text-muted-foreground pt-2">
                  {a.description}
                </p>
              )}
            </CardContent>
          </Card>

          <Accordion>
            <AccordionItem value="children">
              <AccordionTrigger>Children</AccordionTrigger>
              <AccordionContent>
                <ChildrenSection id={id} />
              </AccordionContent>
            </AccordionItem>
          </Accordion>

          <EditActivityDialog
            open={editOpen}
            onOpenChange={setEditOpen}
            activity={a}
            onSuccess={() => {
              queryClient.invalidateQueries({ queryKey: ["activities", id] });
            }}
          />

          <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete "{a.name}"?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete this activity and cannot be
                  undone. If this activity or its children have active time
                  entries or expenses, deletion will be blocked.
                </AlertDialogDescription>
              </AlertDialogHeader>
              {deleteError && (
                <div className="text-sm text-destructive bg-destructive/10 rounded-md p-3">
                  {deleteError}
                </div>
              )}
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  disabled={deleteActivity.isPending}
                  onClick={(e) => {
                    e.preventDefault();
                    setDeleteError(null);
                    deleteActivity.mutate(a.id);
                  }}
                >
                  {deleteActivity.isPending ? "Deleting..." : "Delete"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </Body>
    </>
  );
}

function ChildrenSection({ id }: { id: string }) {
  const { data, isLoading, isError, refetch } = useQuery(
    ActivitiesApis.activityChildrenQueryOpts(id)
  );
  const children = data;

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="text-center py-4">
        <p className="text-sm text-destructive">Failed to load children.</p>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => refetch()}
          className="mt-2"
        >
          Retry
        </Button>
      </div>
    );
  }

  if (!children?.length) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        <p className="text-sm">No children</p>
        <p className="text-xs mt-1">
          Child activities can be created from the activities list.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {children.map((child) => (
        <div
          key={child.id}
          className="flex items-center justify-between py-2 border-b last:border-0"
        >
          <div>
            <span className="font-medium">{child.name}</span>
            <Badge variant="secondary" className="text-xs ml-2">
              {child.kind}
            </Badge>
            {child.description && (
              <p className="text-sm text-muted-foreground">
                {child.description}
              </p>
            )}
          </div>
          <Badge variant={child.is_active ? "default" : "secondary"}>
            {child.is_active ? "Active" : "Inactive"}
          </Badge>
        </div>
      ))}
    </div>
  );
}
