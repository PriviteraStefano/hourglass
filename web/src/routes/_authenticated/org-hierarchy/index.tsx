import { createFileRoute } from "@tanstack/react-router";
import { unitTreeQueryOpts } from "@/api/units.ts";
import { OrgHierarchyPage } from "./-components/org-hierarchy-page";
import "@xyflow/react/dist/style.css";

export const Route = createFileRoute("/_authenticated/org-hierarchy/")({
  beforeLoad: async ({ context: { client } }) => {
    await client.ensureQueryData(unitTreeQueryOpts);
  },
  component: OrgHierarchyPage,
});
