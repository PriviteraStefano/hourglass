# Hourglass Roadmap

## Current Milestone: Contract Management

## Phase 0: Testing foundation

**Status:** Not started

**Goal:** Backend & frontend testing foundation — catch bugs before building new features. Establish test infrastructure, patterns, and baseline coverage.

**Depends on:** None (prerequisite for all future phases)

### Plans

- [ ] 00-CONTEXT.md
- [ ] 00-DISCUSSION-LOG.md

---

## Phase 1: Org hierarchy edge-driven

**Status:** Not started

**Goal:** Define organization hierarchy with edge-driven relationships

**Depends on:** 0

### Plans

- [ ] 01-CONTEXT.md
- [ ] 01-DISCUSSION-LOG.md

---

## Phase 2: Customers management page

**Goal:** Create a new customers page with full CRUD operations. Customers can be internal (organization itself) or external.

**Depends on:** 0, 1

### Success Criteria

- [x] Customers list page at /customers
- [x] Create new customer (name, contact, email, phone, VAT, address)
- [x] Edit existing customer
- [x] Delete customer (handle contracts assignment)
- [x] Edge case: cannot delete customer with active contracts (409 from API)
- [ ] Edge case: "internal customer" represented specially

---

## Phase 3: Contracts - add projects display

**Goal:** Add projects list display to contract detail page. Show related projects per contract.

**Depends on:** 0, 1

### Success Criteria

- [x] Projects list displayed on contract detail page
- [x] Projects filterable by contract (via contract_id filter)
- [x] Edge case: contract with no projects handled
- [ ] Edge case: contract with adopted projects handled

---

## Phase 4: Integrate customers into contracts

**Goal:** Update contract creation dialog to include customer selection. Add "internal customer" option.

**Depends on:** 0, 2

### Success Criteria

- [ ] Customer dropdown in create contract dialog
- [ ] "Internal customer" option available
- [ ] Customer required for new contracts
- [ ] Existing contracts without customer still work