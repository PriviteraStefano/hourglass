package coveragesvc

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
	time_entrydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/time_entry"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	"github.com/stefanoprivitera/hourglass/internal/core/services/routing"
	"github.com/stefanoprivitera/hourglass/internal/models"
)

// Service implements the coverage plane's business semantics (ADR-P-012,
// ADR-BE-017): the D-04 chain-driven default-source decision function,
// the read paths (proposals, to-cover queue, bucket balance, snapshots,
// history), the atomic replace-set write with the Σ invariant and per-row
// vocabulary/ref validation, and the period-close orchestration.
//
// The repository (12-06) is a storage mechanism; every invariant a user can
// observe is decided here:
//
//   - D-04: proposals are computed on read from the activity chain (contract
//     draw vs support bucket vs service request vs absorption vs no-source
//     flag) — never persisted (D-I).
//   - D-08: the replace-set write is gated to the entry's own manager via
//     the shared routing.Service (BE-014) — ApproverIDs OR (RoleGated &&
//     role == "manager"); the entry owner is structurally forbidden.
//   - D-07/COV-01: Σ allocations = entry hours in cents; per-row vocabulary
//     and ref-pinning validation maps every violation to
//     coverage.ErrInvalidRequest (400) before any repo call — the 019
//     CHECKs stay the DB-backed last line of defense.
//   - BE-016: every mutator hands its audit row to the repo for an in-tx
//     write — never fire-and-forget.
//   - D-10/D-12: ClosePeriod orchestrates the frozen snapshot via the repo's
//     single-tx close with the coverage-closed audit row.
//
// Reads (Propose, ToCoverQueue, BucketBalance, GetSnapshot, ListHistory) are
// manager + finance only; finance is read-allowed but never writes (D-L).
type Service struct {
	repo         ports.CoverageRepository
	activityRepo ports.ActivityRepository
	contractRepo ports.ContractRepository
	unitRepo     ports.UnitRepository
	entryRepo    ports.TimeEntryRepository
	routing      *routing.Service
}

func NewService(repo ports.CoverageRepository, activityRepo ports.ActivityRepository, contractRepo ports.ContractRepository, unitRepo ports.UnitRepository, entryRepo ports.TimeEntryRepository, routingSvc *routing.Service) *Service {
	return &Service{
		repo:         repo,
		activityRepo: activityRepo,
		contractRepo: contractRepo,
		unitRepo:     unitRepo,
		entryRepo:    entryRepo,
		routing:      routingSvc,
	}
}

// ---------------------------------------------------------------------------
// D-04 — the pure chain-driven default-source decision function
// ---------------------------------------------------------------------------

// noSourceFlagReason is the D-06 self-explaining flag for entries whose
// activity chain yields neither a contract nor a beneficiary unit.
const noSourceFlagReason = "no eligible source — needs a unit or contract"

// DefaultSource maps the entry's funding chain to the default allocation
// source (D-04): the single chain-driven decision function, no
// billability-first branch (Pitfall 3).
//
// Decision order:
//  1. contract found:
//     - contract_type 'support' → contract draw (support bucket)
//     - contract_type 'project' with SoldHours IS NOT DISTINCT FROM 0
//     (nil or explicit 0, A3) → contract draw (service request)
//     - else → contract draw (contract budget)
//  2. no contract, BeneficiaryUnitID set → absorption draw with that unit
//     (COV-05 — the service attaches the resolved unit to the chain)
//  3. neither → flagged=true with the D-06 no-source reason — the row is
//     still present in the queue, never an implicit gap.
//
// D-05 extension seam: this function is the single switch point where
// ticket-kind → funding-eligibility rules will plug in (D-H). No kind →
// source matrix is implemented — the chain rules only.
//
// The helper is pure: no repos, no ctx — the caller resolves the chain.
func DefaultSource(chain *activitydomain.FundingContext) (sourceType string, contractID, unitID *uuid.UUID, flagged bool, flagReason string) {
	if chain == nil {
		return "", nil, nil, true, noSourceFlagReason
	}
	if chain.ContractID != nil {
		// Every contract found draws with source_type='contract'; which of
		// the three sources it IS derives from the contract's type + sold
		// hours (A1/D-04).
		if chain.ContractType != nil && *chain.ContractType == contractdomain.ContractTypeSupport {
			return coverage.SourceTypeContract, chain.ContractID, nil, false, ""
		}
		if chain.ContractType != nil && *chain.ContractType == contractdomain.ContractTypeProject && (chain.SoldHours == nil || *chain.SoldHours == 0) {
			return coverage.SourceTypeContract, chain.ContractID, nil, false, ""
		}
		return coverage.SourceTypeContract, chain.ContractID, nil, false, ""
	}
	if chain.BeneficiaryUnitID != nil {
		return coverage.SourceTypeAbsorption, nil, chain.BeneficiaryUnitID, false, ""
	}
	return "", nil, nil, true, noSourceFlagReason
}

// resolveChain assembles the D-04 decision input for an entry's activity:
// the funding context (nearest ancestor contract + type/sold hours) and —
// when the chain carries no contract — the beneficiary unit (COV-05
// absorption default). The service-side attach keeps DefaultSource pure.
func (s *Service) resolveChain(ctx context.Context, activityID uuid.UUID) (*activitydomain.FundingContext, error) {
	chain, err := s.activityRepo.ResolveFundingContext(ctx, activityID)
	if err != nil {
		return nil, err
	}
	if chain == nil {
		chain = &activitydomain.FundingContext{}
	}
	if chain.ContractID == nil {
		unitID, err := s.activityRepo.ResolveBeneficiaryUnit(ctx, activityID)
		if err != nil {
			return nil, err
		}
		chain.BeneficiaryUnitID = unitID
	}
	return chain, nil
}

// ---------------------------------------------------------------------------
// Read paths — manager + finance only (finance read-allowed, D-L)
// ---------------------------------------------------------------------------

// readAllowed is the read gate: manager and finance may read the coverage
// read-models; employee and customer are rejected (D-08 finance read-only).
func (s *Service) readAllowed(role string) bool {
	return role == string(models.RoleManager) || role == string(models.RoleFinance)
}

// Propose computes the D-04 default proposal for an entry (computed on
// read, never persisted — D-I) and returns it together with the entry's
// current confirmed allocations. The entry must belong to the org
// (entryRepo.GetByID is not org-scoped — the OrgID compare is the scope
// gate); the read gate is manager|finance.
func (s *Service) Propose(ctx context.Context, orgID, entryID uuid.UUID, role, userID string) (*coverage.CoverageProposal, []*coverage.CoverageAllocation, error) {
	e, err := s.entryRepo.GetByID(ctx, entryID)
	if err != nil {
		// The entry repo's sentinel normalizes to the coverage 404 (the
		// handler maps ErrEntryNotCoverable → 404; a raw ErrTimeEntryNotFound
		// would otherwise surface as 500).
		if errors.Is(err, time_entrydomain.ErrTimeEntryNotFound) {
			return nil, nil, coverage.ErrEntryNotCoverable
		}
		return nil, nil, err
	}
	if e == nil || e.OrgID != orgID {
		return nil, nil, coverage.ErrEntryNotCoverable
	}
	if !s.readAllowed(role) {
		return nil, nil, coverage.ErrForbidden
	}
	chain, err := s.resolveChain(ctx, e.ActivityID)
	if err != nil {
		return nil, nil, err
	}
	sourceType, contractID, unitID, flagged, flagReason := DefaultSource(chain)
	proposal := &coverage.CoverageProposal{
		EntryID:    entryID,
		SourceType: sourceType,
		ContractID: contractID,
		UnitID:     unitID,
		Hours:      e.Hours,
		Flagged:    flagged,
		FlagReason: flagReason,
	}
	allocs, err := s.repo.ListByEntry(ctx, orgID, entryID)
	if err != nil {
		return nil, nil, err
	}
	return proposal, allocs, nil
}

// ToCoverQueue returns the org's to-cover queue (D-06, COV-01): every
// approved, non-deleted 'time' entry with Σ allocations < entry hours,
// enriched service-side with the D-04 proposal (hours = the uncovered
// hours) or the flagged no-source proposal — no-source entries appear in
// the queue, never an implicit gap.
func (s *Service) ToCoverQueue(ctx context.Context, orgID uuid.UUID, role, userID string) ([]coverage.ToCoverQueueRow, error) {
	if !s.readAllowed(role) {
		return nil, coverage.ErrForbidden
	}
	rows, err := s.repo.ToCoverQueue(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		chain, err := s.resolveChain(ctx, rows[i].ActivityID)
		if err != nil {
			return nil, err
		}
		sourceType, contractID, unitID, flagged, flagReason := DefaultSource(chain)
		rows[i].Proposal = &coverage.CoverageProposal{
			EntryID:    rows[i].EntryID,
			SourceType: sourceType,
			ContractID: contractID,
			UnitID:     unitID,
			Hours:      rows[i].UncoveredHours,
			Flagged:    flagged,
			FlagReason: flagReason,
		}
	}
	return rows, nil
}

// BucketBalance passes through the derived support-bucket balance (D-02):
// sold_hours − Σ allocations drawn from the contract, computed on read,
// never stored. Negative balances are returned as-is (D-03 — overdraw is
// report-visible, never a gate).
func (s *Service) BucketBalance(ctx context.Context, orgID, contractID uuid.UUID, role, userID string) (float64, error) {
	if !s.readAllowed(role) {
		return 0, coverage.ErrForbidden
	}
	return s.repo.BucketBalance(ctx, orgID, contractID)
}

// GetSnapshot returns a frozen period-close snapshot by close id (D-10,
// COV-04). Read gate: manager|finance.
func (s *Service) GetSnapshot(ctx context.Context, orgID, closeID uuid.UUID, role, userID string) (*coverage.PeriodClose, error) {
	if !s.readAllowed(role) {
		return nil, coverage.ErrForbidden
	}
	return s.repo.GetSnapshot(ctx, orgID, closeID)
}

// ListHistory returns the audit stream behind an entry's allocations
// (A7 — entity_type='coverage_allocation'). Read gate: manager|finance.
func (s *Service) ListHistory(ctx context.Context, orgID, entryID uuid.UUID, role, userID string) ([]audit.AuditLog, error) {
	if !s.readAllowed(role) {
		return nil, coverage.ErrForbidden
	}
	return s.repo.ListHistory(ctx, orgID, entryID)
}

// ---------------------------------------------------------------------------
// ReplaceAllocations — the atomic replace-set write (D-07, COV-01)
// ---------------------------------------------------------------------------

// validAbsorptionReason reports whether the reason is in the closed
// absorption vocabulary (COV-02 — mirrors the 019 reason_vocab_check).
func validAbsorptionReason(reason string) bool {
	switch reason {
	case coverage.AbsorptionReasonWarrantyBug, coverage.AbsorptionReasonUnderEstimate, coverage.AbsorptionReasonGoodwill:
		return true
	}
	return false
}

// contains reports whether id is in ids (time_entry service helper analog).
func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// ReplaceAllocations atomically replaces the full allocation set for an
// entry (D-07): the service validates everything the user can observe and
// maps every violation to a sentinel BEFORE any repo call — the repo's
// in-tx re-validation under the FOR UPDATE entry lock (CR-01) is the
// authoritative last line, and the 019 CHECKs the DB-backed belt-and-braces.
//
// Step order (each gate maps to its sentinel — 400/403/404, never 500):
//
//  1. Entry fetch + scope: entryRepo.GetByID (not org-scoped) then
//     e.OrgID == orgID; must be approved and not deleted (D-09) →
//     coverage.ErrEntryNotCoverable.
//  2. D-K branch: every allocation's entry_type must be 'time' →
//     coverage.ErrInvalidRequest (the costed polymorphic-validation branch).
//  3. Σ fast-fail in cents (math.Round avoids float64 artifacts): the
//     request sum must equal the entry hours; an empty set never sums to
//     hours > 0 → coverage.ErrAllocationSumMismatch.
//  4. D-08 gate — mandatory, no optional role path (Pattern 6, A9):
//     allowed = actor ∈ ApproverIDs OR (res.RoleGated && role == "manager").
//     The owner is structurally forbidden before resolution; the
//     ApproverIDs path accepts WG manager/delegate in any role claim
//     (exactly as they may approve); the RoleGated terminal branch
//     (org root without a unit manager) requires the org role claim to be
//     exactly 'manager' — finance/employee/customer never pass on any path.
//  5. Per-row vocabulary + ref pinning (source_type ∈ {contract,absorption,
//     transfer}; absorption needs a vocabulary reason; exactly one pinned
//     ref per type) → coverage.ErrInvalidRequest.
//  6. Ref resolution + org-visibility: contract refs via contractRepo.Get
//     (not org-scoped — the service gate is
//     CreatedByOrgID == orgID || (IsShared && IsAdopted)); absorption unit
//     refs via unitRepo.GetByID + OrgID compare; transfer needs a
//     justification → coverage.ErrInvalidRequest.
//  7. Audit: full-set payload (A7) handed to the repo for the in-tx write
//     (BE-016 — never fire-and-forget), then the repo call.
func (s *Service) ReplaceAllocations(ctx context.Context, orgID, entryID uuid.UUID, req []*coverage.CoverageAllocation, userID, role string) ([]*coverage.CoverageAllocation, error) {
	// 1. Entry fetch + scope (GetByID is not org-scoped — the compare is).
	e, err := s.entryRepo.GetByID(ctx, entryID)
	if err != nil {
		// Normalize the entry repo's sentinel to the coverage 404 (see
		// Propose — a raw ErrTimeEntryNotFound must not surface as 500).
		if errors.Is(err, time_entrydomain.ErrTimeEntryNotFound) {
			return nil, coverage.ErrEntryNotCoverable
		}
		return nil, err
	}
	if e == nil || e.OrgID != orgID {
		return nil, coverage.ErrEntryNotCoverable
	}
	if e.Status != time_entrydomain.StatusApproved || e.IsDeleted {
		return nil, coverage.ErrEntryNotCoverable
	}

	actor, err := uuid.Parse(userID)
	if err != nil {
		return nil, coverage.ErrInvalidRequest
	}

	// 2. D-K branch: 'time' is the only coverable entry type in v0.2.
	for _, a := range req {
		if a.EntryType != coverage.EntryTypeTime {
			return nil, coverage.ErrInvalidRequest
		}
	}

	// 3. Σ fast-fail in cents; the empty set can never equal hours > 0.
	var sum int64
	for _, a := range req {
		sum += int64(math.Round(a.Hours * 100))
	}
	if len(req) == 0 || sum != int64(math.Round(e.Hours*100)) {
		return nil, coverage.ErrAllocationSumMismatch
	}

	// 4. D-08 gate — mandatory (Pattern 6). Structural self-barrier first:
	// the owner can never allocate their own coverage (A9).
	if e.UserID == actor {
		return nil, coverage.ErrForbidden
	}
	res, err := s.routing.ResolveManagerStage(ctx, e.OrgID, e.ActivityID, e.UnitID, e.UserID)
	if err != nil {
		if errors.Is(err, activitydomain.ErrActivityNotLoggable) {
			// Commercial activity without an anchored WG — no legitimate
			// manager-stage writer exists (mirrors Approve).
			return nil, coverage.ErrForbidden
		}
		return nil, err
	}
	if !res.RoleGated && !contains(res.ApproverIDs, actor) {
		return nil, coverage.ErrForbidden
	}
	// RoleGated terminal state (org root without a unit manager): the org
	// role claim MUST be exactly 'manager' — no optional role path.
	if res.RoleGated && role != string(models.RoleManager) {
		return nil, coverage.ErrForbidden
	}

	// 5. Per-row vocabulary + ref pinning — every row checked before the
	// repo call (violations → 400; check_violation 23514 is unmapped in
	// wrapPGError and would otherwise surface as 500).
	for _, a := range req {
		if a.Hours <= 0 {
			return nil, coverage.ErrInvalidRequest
		}
		switch a.SourceType {
		case coverage.SourceTypeContract:
			if a.ContractID == nil || a.UnitID != nil {
				return nil, coverage.ErrInvalidRequest
			}
		case coverage.SourceTypeAbsorption:
			if a.UnitID == nil || a.ContractID != nil {
				return nil, coverage.ErrInvalidRequest
			}
			if a.Reason == nil || !validAbsorptionReason(*a.Reason) {
				return nil, coverage.ErrInvalidRequest
			}
		case coverage.SourceTypeTransfer:
			if a.ContractID == nil || a.UnitID != nil {
				return nil, coverage.ErrInvalidRequest
			}
			if a.Justification == nil || *a.Justification == "" {
				return nil, coverage.ErrInvalidRequest
			}
		default:
			return nil, coverage.ErrInvalidRequest
		}
	}

	// 6. Ref validation: existence + org-visibility on every ref before the
	// repo call (T-12-10).
	for _, a := range req {
		switch a.SourceType {
		case coverage.SourceTypeContract, coverage.SourceTypeTransfer:
			c, err := s.contractRepo.Get(ctx, orgID, *a.ContractID)
			if err != nil || c == nil {
				return nil, coverage.ErrInvalidRequest
			}
			// Get filters only by c.id (orgID feeds the is_adopted
			// subquery) — this predicate is the actual visibility gate,
			// matching BucketBalance's pre-check in 12-06.
			if c.CreatedByOrgID != orgID && !(c.IsShared && c.IsAdopted) {
				return nil, coverage.ErrInvalidRequest
			}
		case coverage.SourceTypeAbsorption:
			u, err := s.unitRepo.GetByID(ctx, a.UnitID.String())
			if err != nil || u == nil || u.OrgID != orgID {
				return nil, coverage.ErrInvalidRequest
			}
		}
	}

	// 7. Audit + repo call: the full allocation set in the payload (A7);
	// the repo writes the row IN THE SAME TX as the replace (BE-016).
	allocsPayload := make([]map[string]any, 0, len(req))
	for _, a := range req {
		row := map[string]any{
			"source_type": a.SourceType,
			"hours":       a.Hours,
		}
		if a.ContractID != nil {
			row["contract_id"] = a.ContractID.String()
		}
		if a.UnitID != nil {
			row["unit_id"] = a.UnitID.String()
		}
		if a.Reason != nil {
			row["reason"] = *a.Reason
		}
		if a.Justification != nil {
			row["justification"] = *a.Justification
		}
		allocsPayload = append(allocsPayload, row)
	}
	return s.repo.ReplaceAllocations(ctx, orgID, entryID, req, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: coverage.AuditEntityCoverageAllocation,
		EntityID:   entryID,
		Action:     coverage.AuditActionAllocationsSet,
		ActorID:    &actor,
		Payload:    map[string]any{"allocations": allocsPayload},
		CreatedAt:  time.Now().UTC(),
	})
}

// ---------------------------------------------------------------------------
// ClosePeriod — the frozen snapshot orchestration (D-10/D-12, COV-04)
// ---------------------------------------------------------------------------

// ClosePeriod freezes the period's allocation state into an immutable
// snapshot (D-10/D-11): the service generates the close id and the
// coverage-closed audit log, then hands both to the repo for the single-tx
// close (BE-016) — header insert + frozen rows + audit row commit together.
// The repo's in-tx overlap re-check is authoritative (A6): an overlapping
// close surfaces coverage.ErrPeriodAlreadyClosed (409 at the handler).
//
// Write gate: manager only. The period is org-scoped with no entry chain to
// resolve, so the gate is purely the org role claim — finance/employee/
// customer never close (no optional/approver-resolution branch). The frozen
// snapshot is the only coverage write that does not touch live allocations:
// they stay editable indefinitely (D-F snapshot-not-lock).
func (s *Service) ClosePeriod(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time, userID uuid.UUID, role string) (*coverage.PeriodClose, error) {
	if role != string(models.RoleManager) {
		return nil, coverage.ErrForbidden
	}
	closeID := uuid.New()
	actor := userID
	return s.repo.ClosePeriod(ctx, orgID, periodStart, periodEnd, closeID, userID, &audit.AuditLog{
		OrgID:      orgID,
		EntityType: coverage.AuditEntityCoverageAllocation,
		EntityID:   closeID,
		Action:     coverage.AuditActionCoverageClosed,
		ActorID:    &actor,
		Payload: map[string]any{
			"period_start": periodStart.Format("2006-01-02"),
			"period_end":   periodEnd.Format("2006-01-02"),
		},
		CreatedAt: time.Now().UTC(),
	})
}
