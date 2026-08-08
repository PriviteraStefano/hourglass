package coverage

import (
	"time"

	"github.com/google/uuid"
)

// CoverageAllocation is one row of the per-entry allocation ledger
// (ADR-P-012, ADR-BE-017): the decision layer that labels how an entry's
// hours are funded. The invariant Σ allocations = entry hours (COV-01) is
// enforced at the service/repo boundary (D-07 replace-set), never stored.
//
// source_type is a tagged union (D-01) mirroring the 015 origin encoding:
// the five funding sources (contract budget / support bucket / service
// request / internal absorption / cross-project transfer) are derived
// semantics of three row-level draws — 'contract' covers the first three
// (distinguished by the referenced contract's type + sold_hours), while
// 'absorption' requires unit_id + reason and 'transfer' requires contract_id
// + justification. Refs-to-type and mandatory-field CHECKs live in migration
// 019; the constants below mirror that vocabulary exactly (single source of
// truth, no drift).
type CoverageAllocation struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	EntryType     string     `json:"entry_type"`
	EntryID       uuid.UUID  `json:"entry_id"`
	SourceType    string     `json:"source_type"`
	ContractID    *uuid.UUID `json:"contract_id,omitempty"`
	UnitID        *uuid.UUID `json:"unit_id,omitempty"`
	Hours         float64    `json:"hours"`
	Reason        *string    `json:"reason,omitempty"`
	Justification *string    `json:"justification,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CoverageProposal is the computed-on-read default allocation for an entry
// (D-04/D-I): the system's one-click suggestion the manager confirms or edits
// at save time. Nothing is persisted — proposals exist only in the response.
//
// Flagged/FlagReason carry the D-06 "no eligible source" state: when the
// activity chain yields neither a contract nor a beneficiary unit, the row is
// still present (never an implicit gap) with Flagged=true and a self-explaining
// reason.
type CoverageProposal struct {
	EntryID       uuid.UUID  `json:"entry_id"`
	SourceType    string     `json:"source_type"`
	ContractID    *uuid.UUID `json:"contract_id,omitempty"`
	UnitID        *uuid.UUID `json:"unit_id,omitempty"`
	Hours         float64    `json:"hours"`
	Reason        *string    `json:"reason,omitempty"`
	Justification *string    `json:"justification,omitempty"`
	Flagged       bool       `json:"flagged"`
	FlagReason    string     `json:"flag_reason,omitempty"`
}

// ToCoverQueueRow is one row of the to-cover queue read-model (D-06, COV-01):
// every approved, non-deleted 'time' entry with Σ allocations < entry hours
// appears here — uncovered work is a visible state, never an implicit gap.
// Rows carry the raw covered/uncovered split; the Proposal (enriched via the
// D-04 default-source decision function) is attached service-side (D-06 — the
// repo only supplies raw uncovered rows).
type ToCoverQueueRow struct {
	EntryID         uuid.UUID         `json:"entry_id"`
	EmployeeID      uuid.UUID         `json:"employee_id"`
	EmployeeName    string            `json:"employee_name"`
	EntryDate       time.Time         `json:"entry_date"`
	ActivityID      uuid.UUID         `json:"activity_id"`
	ActivityName    string            `json:"activity_name"`
	Hours           float64           `json:"hours"`
	CoveredHours    float64           `json:"covered_hours"`
	UncoveredHours  float64           `json:"uncovered_hours"`
	Proposal        *CoverageProposal `json:"proposal,omitempty"`
}

// PeriodClose is the frozen snapshot of a closed reporting period
// (D-10/D-11/D-12, COV-04): an immutable copy of the period's allocation
// state written by the close operation. Reports (billing, bucket levels,
// per-unit) read the snapshot, never live rows — "a reported period never
// changes retroactively" holds by construction while allocations stay
// editable indefinitely (D-F snapshot-not-lock).
//
// DATE semantics: PeriodStart/PeriodEnd are DATE columns in the DB and the
// snapshot rows store the entry_date date part. Compare them via
// .Format("2006-01-02") or truncated days — never raw time.Time equality —
// because entry_date is TIMESTAMPTZ in time_entries (migration 000).
type PeriodClose struct {
	ID          uuid.UUID     `json:"id"`
	OrgID       uuid.UUID     `json:"org_id"`
	PeriodStart time.Time     `json:"period_start"`
	PeriodEnd   time.Time     `json:"period_end"`
	ClosedBy    uuid.UUID     `json:"closed_by"`
	ClosedAt    time.Time     `json:"closed_at"`
	Rows        []SnapshotRow `json:"rows"`
}

// SnapshotRow is one entry-level row of a period-close snapshot (D-11): the
// allocation state per entry/source as it stood at close, including the
// resolved contract/unit refs (the frozen "activity chain snapshot"). No
// aggregates are stored — bucket levels, billing totals, and per-unit report
// aggregates are computed from these rows on read when the Phase 17 surfaces
// land. Append-only: no UPDATE/DELETE paths exist (ticket precedent).
type SnapshotRow struct {
	ID            uuid.UUID  `json:"id"`
	CloseID       uuid.UUID  `json:"close_id"`
	EntryID       uuid.UUID  `json:"entry_id"`
	EmployeeID    uuid.UUID  `json:"employee_id"`
	EntryDate     time.Time  `json:"entry_date"`
	ActivityID    uuid.UUID  `json:"activity_id"`
	SourceType    string     `json:"source_type"`
	ContractID    *uuid.UUID `json:"contract_id,omitempty"`
	UnitID        *uuid.UUID `json:"unit_id,omitempty"`
	Hours         float64    `json:"hours"`
	Reason        *string    `json:"reason,omitempty"`
	Justification *string    `json:"justification,omitempty"`
}

// Closed entry-type vocabulary (D-K): 'time' is the only coverable entry type
// in v0.2. The schema CHECK (019 entry_type_check) mirrors this constant;
// the service branch rejecting non-'time' entries is the belt-and-braces pair
// (the costed D-K validation branch).
const (
	EntryTypeTime = "time"
)

// Closed source-type vocabulary (A1/D-01): three row-level draws; the five
// funding sources are derived from the referenced contract (contract_type +
// sold_hours). Mirrors the 019 source_type_check exactly.
const (
	SourceTypeContract   = "contract"
	SourceTypeAbsorption = "absorption"
	SourceTypeTransfer   = "transfer"
)

// Closed absorption-reason vocabulary (COV-02, A2): exactly three values;
// the Part-5 "plain internal" fourth value is superseded. Mirrors the 019
// reason_vocab_check exactly. reason is mandatory for absorption rows.
const (
	AbsorptionReasonWarrantyBug   = "WarrantyBug"
	AbsorptionReasonUnderEstimate = "UnderEstimate"
	AbsorptionReasonGoodwill      = "Goodwill"
)

// Audit vocabulary (A7, ADR-BE-017): coverage mutators hand audit rows to
// the repo with entity_type='coverage_allocation' and one of these actions.
// Exported so the repo and service can never drift from the pinned
// vocabulary (Phase 17 history reads rely on it).
const (
	AuditActionAllocationsSet  = "allocations-set"
	AuditActionCoverageClosed  = "coverage-closed"
	AuditEntityCoverageAllocation = "coverage_allocation"
)
