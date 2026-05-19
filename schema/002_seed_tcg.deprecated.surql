-- ============================================================================
-- SEED DATA: Tech Consulting Organization Units
-- Uses existing organization: 8d152bac-1fc4-4f78-9a67-239198272c1c
-- ============================================================================

-- Users (one manager per unit)
CREATE users:u'019df6f5-ea95-735d-888b-158583ae4516' SET
    email = "maria.silva@tcg.com",
    name = "Maria Silva",
    firstname = "Maria",
    lastname = "Silva",
    username = "msilva",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b' SET
    email = "john.chen@tcg.com",
    name = "John Chen",
    firstname = "John",
    lastname = "Chen",
    username = "jchen",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf' SET
    email = "sophie.mueller@tcg.com",
    name = "Sophie Mueller",
    firstname = "Sophie",
    lastname = "Mueller",
    username = "smueller",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-3d87-734d-8a97-e93c89641c79' SET
    email = "anton.rossi@tcg.com",
    name = "Anton Rossi",
    firstname = "Anton",
    lastname = "Rossi",
    username = "arossi",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-8ebf-779a-866d-46e196d4928d' SET
    email = "lucy.brown@tcg.com",
    name = "Lucy Brown",
    firstname = "Lucy",
    lastname = "Brown",
    username = "lbrown",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0' SET
    email = "david.dubois@tcg.com",
    name = "David Dubois",
    firstname = "David",
    lastname = "Dubois",
    username = "dubois",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f7-fd27-7781-a62a-cc80974a5787' SET
    email = "emily.johnson@tcg.com",
    name = "Emily Johnson",
    firstname = "Emily",
    lastname = "Johnson",
    username = "ejohnson",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE users:u'019df6f8-44bb-775d-badb-fa2fabd5d366' SET
    email = "marc.weber@tcg.com",
    name = "Marc Weber",
    firstname = "Marc",
    lastname = "Weber",
    username = "mweber",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Units (8 business units)
CREATE units:jde SET
    id = "jde",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "JDE Unit",
    description = "Oracle JD Edwards consulting and implementation",
    hierarchy_level = 1,
    code = "JDE",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:netsuite SET
    id = "netsuite",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "NetSuite Unit",
    description = "Oracle NetSuite consulting and implementation",
    hierarchy_level = 1,
    code = "NS",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:aris SET
    id = "aris",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "ARIS Unit",
    description = "ARIS business process management consulting",
    hierarchy_level = 1,
    code = "ARIS",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:fusion SET
    id = "fusion",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "Fusion Unit",
    description = "Oracle Fusion Cloud consulting and implementation",
    hierarchy_level = 1,
    code = "FUSION",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:marketing SET
    id = "marketing",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "Marketing Unit",
    description = "Marketing strategy and digital marketing services",
    hierarchy_level = 1,
    code = "MKT",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:finance SET
    id = "finance",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "Finance Unit",
    description = "Financial planning and accounting consulting",
    hierarchy_level = 1,
    code = "FIN",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:hr SET
    id = "hr",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "HR Unit",
    description = "Human resources and people management consulting",
    hierarchy_level = 1,
    code = "HR",
    created_at = time::now(),
    updated_at = time::now()
;

CREATE units:technical SET
    id = "technical",
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    name = "Technical Unit",
    description = "Technical architecture and infrastructure consulting",
    hierarchy_level = 1,
    code = "TECH",
    created_at = time::now(),
    updated_at = time::now()
;

-- Organization Memberships
CREATE organization_memberships:u'019df6f9-2d41-75b3-98d2-df8854988d37' SET
    user_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6f9-5861-7152-bd6d-ebb963799a81' SET
    user_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6f9-77fc-756f-bd6e-28ce796d1478' SET
    user_id = users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6f9-9104-77d1-8999-1f54d595e811' SET
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6f9-c883-759a-9334-4a5886b8e589' SET
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6f9-eeda-73c0-bf68-ff56fc0d9b9f' SET
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6fa-1031-73fc-9012-adf65dda4a5e' SET
    user_id = users:u'019df6f7-fd27-7781-a62a-cc80974a5787',
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

CREATE organization_memberships:u'019df6fa-2476-721e-878b-0472bc9426f3' SET
    user_id = users:u108,
    organization_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;

-- Unit Memberships
CREATE unit_memberships:um001 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',
    unit_id = units:jde,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um002 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
    unit_id = units:netsuite,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um003 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f6-c6ca-7581-83f1-f50ab6c436cf',
    unit_id = units:aris,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um004 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f7-3d87-734d-8a97-e93c89641c79',
    unit_id = units:fusion,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um005 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f7-8ebf-779a-866d-46e196d4928d',
    unit_id = units:marketing,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um006 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
    unit_id = units:finance,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um007 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u'019df6f7-fd27-7781-a62a-cc80974a5787',
    unit_id = units:hr,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;

CREATE unit_memberships:um008 SET
    org_id = organizations:u'8d152bac-1fc4-4f78-9a67-239198272c1c',
    user_id = users:u108,
    unit_id = units:technical,
    is_primary = true,
    role = "manager",
    start_date = time::now(),
    created_at = time::now()
;