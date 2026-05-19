-- ============================================================================
-- MVP DEMO SEED DATA — Foundation Entities
-- ============================================================================
-- This file provides idempotent demo seed data for the Hourglass MVP.
-- It populates all core structural entities: org, users, units, customer,
-- memberships, contracts, projects, subprojects, working groups, and WG members.
--
-- Entity dependency order (must be followed when adding new entities):
--   organizations → users → units → customers → organization_memberships →
--   unit_memberships → contracts → projects → subprojects →
--   working_groups → wg_members
--
-- Idempotency: This file uses CREATE (not DEFINE) statements. Re-running
-- cmd/schema will attempt to re-CREATE records with the same IDs. SurrealDB
-- returns "already exists" errors for duplicate CREATE — this is expected and
-- harmless. A future version may wrap these in conditionals.
--
-- Demo login credentials: All 6 users have password "demo123".
-- ============================================================================

-- ============================================================================
-- ORGANIZATION
-- ============================================================================

CREATE organizations:u'019df8b0-0001-7000-8000-000000000001' SET
    name = "Tech Consulting Group",
    slug = "tech-consulting-group",
    description = "Tech Consulting Group — MVP demo organization providing engineering, consulting, and operations services",
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- USERS (6 users: 2 managers, 1 finance, 3 employees)
-- All password hashes are bcrypt cost 12 for "demo123"
-- ============================================================================

CREATE users:u'019df6f5-ea95-735d-888b-158583ae4516' SET
    email = "alex.rivera@tcg.com",
    name = "Alex Rivera",
    firstname = "Alex",
    lastname = "Rivera",
    username = "arivera",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b' SET
    email = "sarah.chen@tcg.com",
    name = "Sarah Chen",
    firstname = "Sarah",
    lastname = "Chen",
    username = "schen",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf' SET
    email = "mike.obrien@tcg.com",
    name = "Mike O'Brien",
    firstname = "Mike",
    lastname = "O'Brien",
    username = "mobrien",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-3d87-734d-8a97-e93c89641c79' SET
    email = "emma.wilson@tcg.com",
    name = "Emma Wilson",
    firstname = "Emma",
    lastname = "Wilson",
    username = "ewilson",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-8ebf-779a-866d-46e196d4928d' SET
    email = "james.park@tcg.com",
    name = "James Park",
    firstname = "James",
    lastname = "Park",
    username = "jpark",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0' SET
    email = "lisa.torres@tcg.com",
    name = "Lisa Torres",
    firstname = "Lisa",
    lastname = "Torres",
    username = "ltorres",
    password_hash = "$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- UNITS (8 units: 3 departments + 5 sub-units with hierarchy)
-- ============================================================================

-- Level 1: Departments
CREATE units:engineering SET
    id = "engineering",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Engineering Unit",
    description = "Software engineering and platform development",
    code = "ENG",
    hierarchy_level = 1,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:consulting SET
    id = "consulting",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Consulting Unit",
    description = "Business consulting and data analytics services",
    code = "CONS",
    hierarchy_level = 1,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:operations SET
    id = "operations",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Operations Unit",
    description = "Operations, finance, and people management",
    code = "OPS",
    hierarchy_level = 1,
    created_at = time::now(),
    updated_at = time::now()
;

-- Level 2: Sub-units under Engineering
CREATE units:platform SET
    id = "platform",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Platform Team",
    description = "Enterprise platform engineering and tooling",
    code = "PLAT",
    parent_unit_id = units:engineering,
    hierarchy_level = 2,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:cloud SET
    id = "cloud",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Cloud Infrastructure",
    description = "Cloud infrastructure and DevOps",
    code = "CLOUD",
    parent_unit_id = units:engineering,
    hierarchy_level = 2,
    created_at = time::now(),
    updated_at = time::now()
;

-- Level 2: Sub-units under Consulting
CREATE units:data SET
    id = "data",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Data Analytics",
    description = "Data science and business intelligence",
    code = "DATA",
    parent_unit_id = units:consulting,
    hierarchy_level = 2,
    created_at = time::now(),
    updated_at = time::now()
;

-- Level 2: Sub-units under Operations
CREATE units:finance SET
    id = "finance",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Finance & Accounting",
    description = "Financial planning and accounting",
    code = "FIN",
    parent_unit_id = units:operations,
    hierarchy_level = 2,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:hr SET
    id = "hr",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Human Resources",
    description = "HR and people management",
    code = "HR",
    parent_unit_id = units:operations,
    hierarchy_level = 2,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- CUSTOMER
-- ============================================================================

CREATE customers:novatech SET
    id = "novatech",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "NovaTech Industries",
    contact_name = "Dr. Elena Vasquez",
    email = "elena.vasquez@novatech.com",
    phone = "+1-555-0100",
    address = "1200 Innovation Drive, San Francisco, CA 94107",
    vat_number = "US-123456789",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- ORGANIZATION MEMBERSHIPS (6 — one per user)
-- ============================================================================

CREATE organization_memberships:u'019df8b2-0001-7000-8000-000000000001' SET
    user_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df8b2-0002-7000-8000-000000000002' SET
    user_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df8b2-0003-7000-8000-000000000003' SET
    user_id = users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "finance",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df8b2-0004-7000-8000-000000000004' SET
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "employee",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df8b2-0005-7000-8000-000000000005' SET
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "employee",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df8b2-0006-7000-8000-000000000006' SET
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    organization_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    role = "employee",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- UNIT MEMBERSHIPS (6 — one per user, primary)
-- ============================================================================

CREATE unit_memberships:um_001 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_id = units:engineering,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um_002 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    unit_id = units:consulting,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um_003 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf',
    unit_id = units:operations,
    is_primary = true,
    role = "finance",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um_004 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    unit_id = units:engineering,
    is_primary = true,
    role = "employee",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um_005 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    unit_id = units:consulting,
    is_primary = true,
    role = "employee",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um_006 SET
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    unit_id = units:engineering,
    is_primary = true,
    role = "employee",
    start_date = time::now(),
    created_at = time::now()
;

-- ============================================================================
-- CONTRACTS (3 contracts — 2 billable + 1 internal)
-- Contracts table is SCHEMALESS, use UUID-type record IDs
-- ============================================================================

CREATE contracts:u'019df8b1-0001-7000-8000-000000000001' SET
    name = "Digital Transformation Program",
    km_rate = 0.42,
    currency = "EUR",
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    is_active = true,
    created_at = time::now()
;

CREATE contracts:u'019df8b1-0002-7000-8000-000000000002' SET
    name = "Cloud Infrastructure Migration",
    km_rate = 0.35,
    currency = "EUR",
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    is_active = true,
    created_at = time::now()
;

CREATE contracts:u'019df8b1-0003-7000-8000-000000000003' SET
    name = "Internal Operations",
    km_rate = 0.00,
    currency = "EUR",
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = true,
    is_active = true,
    created_at = time::now()
;

-- ============================================================================
-- PROJECTS (6 projects — 2 per contract)
-- Each project includes both `project_type` (schema) and `type` (Go code) fields
-- ============================================================================

-- Contract 1: Digital Transformation Program → Platform Engineering + Data Analytics
CREATE projects:proj_platform_eng SET
    id = "proj_platform_eng",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Platform Engineering",
    description = "Enterprise platform engineering and development",
    project_type = "billable",
    type = "billable",
    contract_id = contracts:u'019df8b1-0001-7000-8000-000000000001',
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    budget_amount = 500000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE projects:proj_data_analytics SET
    id = "proj_data_analytics",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Data Analytics",
    description = "Data analytics and business intelligence platform",
    project_type = "billable",
    type = "billable",
    contract_id = contracts:u'019df8b1-0001-7000-8000-000000000001',
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    budget_amount = 350000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Contract 2: Cloud Infrastructure Migration → Cloud Migration + DevOps Setup
CREATE projects:proj_cloud_migration SET
    id = "proj_cloud_migration",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Cloud Migration",
    description = "Cloud infrastructure migration from on-premise to AWS",
    project_type = "billable",
    type = "billable",
    contract_id = contracts:u'019df8b1-0002-7000-8000-000000000002',
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    budget_amount = 400000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE projects:proj_devops_setup SET
    id = "proj_devops_setup",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "DevOps Setup",
    description = "CI/CD pipeline and DevOps infrastructure setup",
    project_type = "billable",
    type = "billable",
    contract_id = contracts:u'019df8b1-0002-7000-8000-000000000002',
    customer_id = customers:novatech,
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = false,
    budget_amount = 250000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Contract 3: Internal Operations → HR System + Finance Tools (internal)
CREATE projects:proj_hr_system SET
    id = "proj_hr_system",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "HR System",
    description = "Internal human resources management system",
    project_type = "internal",
    type = "internal",
    contract_id = contracts:u'019df8b1-0003-7000-8000-000000000003',
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = true,
    budget_amount = 150000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE projects:proj_finance_tools SET
    id = "proj_finance_tools",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    name = "Finance Tools",
    description = "Internal finance management and reporting tools",
    project_type = "internal",
    type = "internal",
    contract_id = contracts:u'019df8b1-0003-7000-8000-000000000003',
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    is_shared = true,
    budget_amount = 120000,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- SUBPROJECTS (6 — one per project)
-- ============================================================================

CREATE subprojects:subproj_platform_eng SET
    id = "subproj_platform_eng",
    project_id = projects:proj_platform_eng,
    name = "Platform Engineering — Phase 1",
    description = "Core platform engineering workstream",
    sequence_order = 1,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE subprojects:subproj_data_analytics SET
    id = "subproj_data_analytics",
    project_id = projects:proj_data_analytics,
    name = "Data Analytics — Platform Build",
    description = "Data analytics platform implementation",
    sequence_order = 2,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE subprojects:subproj_cloud_migration SET
    id = "subproj_cloud_migration",
    project_id = projects:proj_cloud_migration,
    name = "Cloud Migration — Assessment & Plan",
    description = "Cloud migration assessment and planning phase",
    sequence_order = 3,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE subprojects:subproj_devops_setup SET
    id = "subproj_devops_setup",
    project_id = projects:proj_devops_setup,
    name = "DevOps Setup — Pipeline Build",
    description = "CI/CD pipeline implementation",
    sequence_order = 4,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE subprojects:subproj_hr_system SET
    id = "subproj_hr_system",
    project_id = projects:proj_hr_system,
    name = "HR System — Core Features",
    description = "Core HR system feature development",
    sequence_order = 5,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE subprojects:subproj_finance_tools SET
    id = "subproj_finance_tools",
    project_id = projects:proj_finance_tools,
    name = "Finance Tools — Reporting",
    description = "Finance reporting and dashboard features",
    sequence_order = 6,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- WORKING GROUPS (6 — one per subproject)
-- ============================================================================

-- Contract 1 projects → Alex Rivera (engineering)
CREATE working_groups:wg_platform_eng SET
    id = "wg_platform_eng",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_platform_eng,
    name = "Platform Engineering WG",
    description = "Working group for platform engineering deliverables",
    manager_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_ids = [units:engineering, units:platform],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE working_groups:wg_data_analytics SET
    id = "wg_data_analytics",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_data_analytics,
    name = "Data Analytics WG",
    description = "Working group for data analytics deliverables",
    manager_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_ids = [units:consulting, units:data],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Contract 2 projects → Sarah Chen
CREATE working_groups:wg_cloud_migration SET
    id = "wg_cloud_migration",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_cloud_migration,
    name = "Cloud Migration WG",
    description = "Working group for cloud migration deliverables",
    manager_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    unit_ids = [units:engineering, units:cloud],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE working_groups:wg_devops_setup SET
    id = "wg_devops_setup",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_devops_setup,
    name = "DevOps Setup WG",
    description = "Working group for DevOps setup deliverables",
    manager_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    unit_ids = [units:engineering, units:cloud],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Contract 3 projects (internal) → Alex Rivera
CREATE working_groups:wg_hr_system SET
    id = "wg_hr_system",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_hr_system,
    name = "HR System WG",
    description = "Working group for HR system development",
    manager_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_ids = [units:operations, units:hr],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE working_groups:wg_finance_tools SET
    id = "wg_finance_tools",
    org_id = organizations:u'019df8b0-0001-7000-8000-000000000001',
    subproject_id = subprojects:subproj_finance_tools,
    name = "Finance Tools WG",
    description = "Working group for finance tools development",
    manager_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_ids = [units:operations, units:finance],
    enforce_unit_tuple = true,
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- ============================================================================
-- WG MEMBERS (6 — one per working group)
-- ============================================================================

CREATE wg_members:wgm_001 SET
    id = "wgm_001",
    wg_id = working_groups:wg_platform_eng,
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    unit_id = units:engineering,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;

CREATE wg_members:wgm_002 SET
    id = "wgm_002",
    wg_id = working_groups:wg_data_analytics,
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    unit_id = units:consulting,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;

CREATE wg_members:wgm_003 SET
    id = "wgm_003",
    wg_id = working_groups:wg_cloud_migration,
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    unit_id = units:engineering,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;

CREATE wg_members:wgm_004 SET
    id = "wgm_004",
    wg_id = working_groups:wg_devops_setup,
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    unit_id = units:engineering,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;

CREATE wg_members:wgm_005 SET
    id = "wgm_005",
    wg_id = working_groups:wg_hr_system,
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    unit_id = units:operations,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;

CREATE wg_members:wgm_006 SET
    id = "wgm_006",
    wg_id = working_groups:wg_finance_tools,
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    unit_id = units:operations,
    role = "member",
    is_default_subproject = true,
    start_date = time::now(),
    created_at = time::now()
;
