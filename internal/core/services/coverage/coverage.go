package coveragesvc

import (
	"context"

	"github.com/google/uuid"
	activitydomain "github.com/stefanoprivitera/hourglass/internal/core/domain/activity"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	contractdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/contract"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/coverage"
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
