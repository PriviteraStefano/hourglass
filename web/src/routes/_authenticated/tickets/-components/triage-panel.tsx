import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { PlusIcon, TrashIcon } from "lucide-react";
import { ActivitiesApis } from "@/api/activities.ts";
import { ContractsApis } from "@/api/contracts.ts";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select.tsx";
import { GOVERNANCE_OPTIONS } from "@/lib/governance.ts";
import {
  TICKET_KINDS,
  TICKET_KIND_LABELS,
  type Ticket,
  type TicketKind,
  type TriageActivityPlan,
} from "@/types";

function blankPlan(name = ""): TriageActivityPlan {
  return {
    name,
    kind: "",
    parent_id: "",
    contract_id: "",
    governance_model: "creator_controlled",
  };
}

export interface TriagePanelProps {
  ticket: Ticket;
  busy: boolean;
  onTriage: (kind: TicketKind, plans: TriageActivityPlan[]) => void;
}

/**
 * M-08 triage: each row becomes an activity the ticket's work belongs to, and
 * the ticket flips to `planned`. The server requires a name and a kind in the
 * org's activity-kind catalog per row, so the submit stays disabled until every
 * row has both — an incomplete row would fail the whole request.
 */
export function TriagePanel({ ticket, busy, onTriage }: TriagePanelProps) {
  const { data: kinds } = useQuery(ActivitiesApis.activityKindsQueryOpts);
  const { data: activities } = useQuery(
    ActivitiesApis.activitiesQueryOpts("all")
  );
  const { data: contracts } = useQuery(ContractsApis.contractsQueryOpts("all"));

  const [kind, setKind] = useState<TicketKind>(ticket.kind);
  const [plans, setPlans] = useState<TriageActivityPlan[]>(() => [
    blankPlan(ticket.title),
  ]);

  const complete = plans.every(
    (plan) => plan.name.trim() !== "" && plan.kind !== ""
  );

  const patchPlan = (index: number, patch: Partial<TriageActivityPlan>) =>
    setPlans((prev) =>
      prev.map((plan, i) => (i === index ? { ...plan, ...patch } : plan))
    );

  return (
    <section className="space-y-3 rounded border p-3">
      <div>
        <p className="text-xs font-medium">Triage into planned work</p>
        <p className="text-[11px] text-muted-foreground">
          Each row becomes an activity linked to this ticket; the ticket moves
          to Planned.
        </p>
      </div>

      {plans.map((plan, index) => (
        <div key={index} className="space-y-2 rounded border p-2">
          <div className="flex items-center gap-2">
            <Input
              value={plan.name}
              onChange={(event) => patchPlan(index, { name: event.target.value })}
              placeholder="Activity name"
              aria-label={`Activity ${index + 1} name`}
              className="h-8 text-xs"
            />
            {plans.length > 1 && (
              <Button
                size="icon-sm"
                variant="ghost"
                aria-label={`Remove activity ${index + 1}`}
                onClick={() =>
                  setPlans((prev) => prev.filter((_, i) => i !== index))
                }
              >
                <TrashIcon />
              </Button>
            )}
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="space-y-1 text-[11px]">
              Kind
              <NativeSelect
                value={plan.kind}
                onChange={(event) =>
                  patchPlan(index, { kind: event.target.value })
                }
                aria-label={`Activity ${index + 1} kind`}
                className="w-full"
              >
                <NativeSelectOption value="">Select kind…</NativeSelectOption>
                {(kinds ?? []).map((value) => (
                  <NativeSelectOption key={value} value={value}>
                    {value.charAt(0).toUpperCase() + value.slice(1)}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
            <label className="space-y-1 text-[11px]">
              Parent
              <NativeSelect
                value={plan.parent_id ?? ""}
                onChange={(event) =>
                  patchPlan(index, { parent_id: event.target.value })
                }
                aria-label={`Activity ${index + 1} parent`}
                className="w-full"
              >
                <NativeSelectOption value="">No parent</NativeSelectOption>
                {(activities ?? []).map((activity) => (
                  <NativeSelectOption key={activity.id} value={activity.id}>
                    {activity.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
            <label className="space-y-1 text-[11px]">
              Contract
              <NativeSelect
                value={plan.contract_id ?? ""}
                onChange={(event) =>
                  patchPlan(index, { contract_id: event.target.value })
                }
                aria-label={`Activity ${index + 1} contract`}
                className="w-full"
              >
                <NativeSelectOption value="">No contract</NativeSelectOption>
                {(contracts ?? []).map((contract) => (
                  <NativeSelectOption key={contract.id} value={contract.id}>
                    {contract.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
            <label className="space-y-1 text-[11px]">
              Governance
              <NativeSelect
                value={plan.governance_model}
                onChange={(event) =>
                  patchPlan(index, {
                    governance_model: event.target
                      .value as TriageActivityPlan["governance_model"],
                  })
                }
                aria-label={`Activity ${index + 1} governance model`}
                className="w-full"
              >
                {GOVERNANCE_OPTIONS.map((option) => (
                  <NativeSelectOption key={option.value} value={option.value}>
                    {option.label}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </label>
          </div>
        </div>
      ))}

      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setPlans((prev) => [...prev, blankPlan()])}
        >
          <PlusIcon /> Another activity
        </Button>
        <label className="ml-auto flex items-center gap-2 text-[11px]">
          Ticket kind
          <NativeSelect
            value={kind}
            onChange={(event) => setKind(event.target.value as TicketKind)}
            aria-label="Ticket kind after triage"
          >
            {TICKET_KINDS.map((value) => (
              <NativeSelectOption key={value} value={value}>
                {TICKET_KIND_LABELS[value]}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </label>
        <Button
          size="sm"
          disabled={busy || !complete}
          onClick={() => onTriage(kind, plans)}
        >
          Plan it
        </Button>
      </div>

      {!complete && (
        <p className="text-[11px] text-muted-foreground">
          Every row needs a name and a kind before the ticket can be planned.
        </p>
      )}
    </section>
  );
}
