-- seed_demo.sql — Demo Seed Data (idempotent, time-relative)
--
-- Runs AFTER the full migration chain (000 → 013), i.e. against the post-011
-- activity-ontology schema and the post-012 staffing schema. It is deliberately
-- NOT a numbered migration: the demo data depends on constraints and tables
-- (two-stage statuses, approval columns, activities, the 'hr' role) that only
-- exist once every migration has been applied.
--
-- Apply with:  make seed   (or psql -f scripts/seed_demo.sql)
--
-- Demo goals:
--   * usernames are role-prefixed so each persona's role is legible mid-demo
--   * dates anchor to CURRENT_DATE so Today / Your-week / Approval queues are
--     always populated, no matter when the demo DB is re-seeded
--   * every approval lifecycle state is present (draft → submitted →
--     pending_manager → pending_finance → approved, plus rejected)
--   * a WG delegate (employee_emma) shows the non-manager approver stage
--   * hr_rachel shows the HR curator view (Economics visible, Review hidden)
--
-- Password for ALL users: demo123  (bcrypt $2a$12$ hash below)
-- ============================================================================
-- Persona                      | Username        | Org role | UUID
-- -----------------------------|-----------------|----------|--------------------------------------
-- Alex Rivera                  | manager_alex    | manager  | 019df6f5-ea95-735d-888b-158583ae4516
-- Sarah Chen                   | manager_sarah   | manager  | 019df6f6-8cdb-70b9-9d0b-ed032caf9f4b
-- Mike O'Brien                 | finance_mike    | finance  | 019df6f6-c6ca-7581-83f1-f50ab6c436cf
-- Emma Wilson (WG delegate)    | employee_emma   | employee | 019df6f7-3d87-734d-8a97-e93c89641c79
-- James Park                   | employee_james  | employee | 019df6f7-8ebf-779a-866d-46e196d4928d
-- Lisa Torres                  | employee_lisa   | employee | 019df6f7-c8c8-75c5-a6a7-224bbcd9cff0
-- Rachel Kim                   | hr_rachel       | hr       | 019df6f8-0001-7000-8000-000000000007
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Organization
-- ============================================================================
INSERT INTO organizations (id, name, slug, description, created_at, updated_at)
VALUES ('019df8b0-0001-7000-8000-000000000001',
        'Tech Consulting Group',
        'tech-consulting-group',
        'Tech Consulting Group — demo organization providing engineering, consulting, and operations services',
        NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 2. Users (7 — one per org role; usernames role-prefixed for the demo)
-- ============================================================================
INSERT INTO users (id, email, firstname, lastname, username, password_hash, is_active, created_at, updated_at)
VALUES ('019df6f5-ea95-735d-888b-158583ae4516', 'manager_alex@tcg.com', 'Alex', 'Rivera', 'manager_alex',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', 'manager_sarah@tcg.com', 'Sarah', 'Chen', 'manager_sarah',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f6-c6ca-7581-83f1-f50ab6c436cf', 'finance_mike@tcg.com', 'Mike', 'O''Brien', 'finance_mike',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f7-3d87-734d-8a97-e93c89641c79', 'employee_emma@tcg.com', 'Emma', 'Wilson', 'employee_emma',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f7-8ebf-779a-866d-46e196d4928d', 'employee_james@tcg.com', 'James', 'Park', 'employee_james',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'employee_lisa@tcg.com', 'Lisa', 'Torres', 'employee_lisa',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
       ('019df6f8-0001-7000-8000-000000000007', 'hr_rachel@tcg.com', 'Rachel', 'Kim', 'hr_rachel',
        '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username,
                               email    = EXCLUDED.email;

-- ============================================================================
-- 3. Customer
-- ============================================================================
INSERT INTO customers (id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at,
                       updated_at)
VALUES ('a71c678f-a767-5692-bd7b-0eeb10453073',
        '019df8b0-0001-7000-8000-000000000001',
        'NovaTech Industries', 'Dr. Elena Vasquez', 'elena.vasquez@novatech.com',
        '+1-555-0100', '1200 Innovation Drive, San Francisco, CA 94107', 'US-123456789',
        true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 4. Units (8)
-- ============================================================================
INSERT INTO units (id, org_id, name, description, code, hierarchy_level, created_at, updated_at)
VALUES ('2c7d6338-0862-5f71-8620-a36407caf226', '019df8b0-0001-7000-8000-000000000001', 'Engineering Unit',
        'Software engineering and platform development', 'ENG', 1, NOW(), NOW()),
       ('47b9341d-896e-58af-8ad8-506f06a0127a', '019df8b0-0001-7000-8000-000000000001', 'Consulting Unit',
        'Business consulting and data analytics services', 'CONS', 1, NOW(), NOW()),
       ('45a6c1b6-fd3c-5f55-8341-d02aa99841cb', '019df8b0-0001-7000-8000-000000000001', 'Operations Unit',
        'Operations, finance, and people management', 'OPS', 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO units (id, org_id, name, description, code, parent_unit_id, hierarchy_level, created_at, updated_at)
VALUES ('4365c0ce-9de2-50dc-9e63-3094923d08ae', '019df8b0-0001-7000-8000-000000000001', 'Platform Team',
        'Enterprise platform engineering and tooling', 'PLAT', '2c7d6338-0862-5f71-8620-a36407caf226', 2, NOW(), NOW()),
       ('1d0153d7-6708-5678-ac70-d2e900e32383', '019df8b0-0001-7000-8000-000000000001', 'Cloud Infrastructure',
        'Cloud infrastructure and DevOps', 'CLOUD', '2c7d6338-0862-5f71-8620-a36407caf226', 2, NOW(), NOW()),
       ('2c8a850a-6549-5cc7-90d4-227a0da64de9', '019df8b0-0001-7000-8000-000000000001', 'Data Analytics',
        'Data science and business intelligence', 'DATA', '47b9341d-896e-58af-8ad8-506f06a0127a', 2, NOW(), NOW()),
       ('d9d57882-fe66-5f75-96fa-b60c48eebde7', '019df8b0-0001-7000-8000-000000000001', 'Finance & Accounting',
        'Financial planning and accounting', 'FIN', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 2, NOW(), NOW()),
       ('fb69255e-f79c-5983-955f-b4ed68daf851', '019df8b0-0001-7000-8000-000000000001', 'Human Resources',
        'HR and people management', 'HR', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 2, NOW(),
        NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 5. Organization Memberships (7 — incl. hr, valid post-012)
-- ============================================================================
INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at, updated_at)
VALUES ('019df8b2-0001-7000-8000-000000000001', '019df6f5-ea95-735d-888b-158583ae4516',
        '019df8b0-0001-7000-8000-000000000001', 'manager', true, NOW(), NOW()),
       ('019df8b2-0002-7000-8000-000000000002', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b',
        '019df8b0-0001-7000-8000-000000000001', 'manager', true, NOW(), NOW()),
       ('019df8b2-0003-7000-8000-000000000003', '019df6f6-c6ca-7581-83f1-f50ab6c436cf',
        '019df8b0-0001-7000-8000-000000000001', 'finance', true, NOW(), NOW()),
       ('019df8b2-0004-7000-8000-000000000004', '019df6f7-3d87-734d-8a97-e93c89641c79',
        '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW()),
       ('019df8b2-0005-7000-8000-000000000005', '019df6f7-8ebf-779a-866d-46e196d4928d',
        '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW()),
       ('019df8b2-0006-7000-8000-000000000006', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0',
        '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW()),
       ('019df8b2-0007-7000-8000-000000000007', '019df6f8-0001-7000-8000-000000000007',
        '019df8b0-0001-7000-8000-000000000001', 'hr', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 6. Unit Memberships (7 — one primary per user)
-- ============================================================================
INSERT INTO unit_memberships (id, org_id, user_id, unit_id, is_primary, role, start_date, created_at)
VALUES ('1fd79fac-78a0-58de-9f45-b9ff237ea67e', '019df8b0-0001-7000-8000-000000000001',
        '019df6f5-ea95-735d-888b-158583ae4516', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'manager', NOW(), NOW()),
       ('68e7992a-516f-5378-b389-91f30aeeea27', '019df8b0-0001-7000-8000-000000000001',
        '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '47b9341d-896e-58af-8ad8-506f06a0127a', true, 'manager', NOW(), NOW()),
       ('c94c3236-fb9b-5849-9665-e0e075a0a1a1', '019df8b0-0001-7000-8000-000000000001',
        '019df6f6-c6ca-7581-83f1-f50ab6c436cf', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', true, 'finance', NOW(), NOW()),
       ('d3236d8a-0e70-516d-a0a1-6bd70bccdd3a', '019df8b0-0001-7000-8000-000000000001',
        '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'employee', NOW(), NOW()),
       ('ab5096c2-679e-50e9-bf7a-b23812b4d3ab', '019df8b0-0001-7000-8000-000000000001',
        '019df6f7-8ebf-779a-866d-46e196d4928d', '47b9341d-896e-58af-8ad8-506f06a0127a', true, 'employee', NOW(), NOW()),
       ('cb838783-7222-5c6a-8d1f-c7f0ab3e5ad8', '019df8b0-0001-7000-8000-000000000001',
        '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'employee', NOW(), NOW()),
       ('cb838783-7222-5c6a-8d1f-c7f0ab3e5ae9', '019df8b0-0001-7000-8000-000000000001',
        '019df6f8-0001-7000-8000-000000000007', 'fb69255e-f79c-5983-955f-b4ed68daf851', true, 'employee', NOW(),
        NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 7. Contracts (3)
-- ============================================================================
INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared,
                       is_active, created_at, updated_at)
VALUES ('019df8b1-0001-7000-8000-000000000001', 'Digital Transformation Program', 0.42, 'EUR',
        'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false,
        true, NOW(), NOW()),
       ('019df8b1-0002-7000-8000-000000000002', 'Cloud Infrastructure Migration', 0.35, 'EUR',
        'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false,
        true, NOW(), NOW()),
       ('019df8b1-0003-7000-8000-000000000003', 'Internal Operations', 0.00, 'EUR', NULL, 'creator_controlled',
        '019df8b0-0001-7000-8000-000000000001', true, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 8. Activities (post-011 recursive model)
--    6 engagements (contract-linked) + 6 phases (one child per engagement).
--    Children use kind='phase' per the 013 forward fix. The kind catalog is
--    ensured first so the seed is self-sufficient on any post-011 database.
-- ============================================================================
INSERT INTO activity_kinds (org_id, name, is_seed)
SELECT '019df8b0-0001-7000-8000-000000000001', k.name, TRUE
FROM (VALUES ('engagement'), ('phase'), ('task'), ('internal')) AS k(name)
ON CONFLICT (org_id, name) DO NOTHING;

INSERT INTO activities (id, org_id, parent_id, name, description, kind, contract_id, governance_model,
                        created_by_org_id, is_shared, billable, budget_amount, is_active, created_at, updated_at)
VALUES
    -- engagements (kind='engagement', parent_id NULL, contract set)
    ('81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '019df8b0-0001-7000-8000-000000000001', NULL, 'Platform Engineering',
     'Enterprise platform engineering and development', 'engagement', '019df8b1-0001-7000-8000-000000000001',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, 500000, true, NOW(), NOW()),
    ('6808945d-ad00-524b-a75c-c363c12ce57d', '019df8b0-0001-7000-8000-000000000001', NULL, 'Data Analytics',
     'Data analytics and business intelligence platform', 'engagement', '019df8b1-0001-7000-8000-000000000001',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, 350000, true, NOW(), NOW()),
    ('a7b26c54-d058-545f-bff4-c2836402f465', '019df8b0-0001-7000-8000-000000000001', NULL, 'Cloud Migration',
     'Cloud infrastructure migration from on-premise to AWS', 'engagement', '019df8b1-0002-7000-8000-000000000002',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, 400000, true, NOW(), NOW()),
    ('932d4cff-50b2-58da-bfc9-34748e6fa143', '019df8b0-0001-7000-8000-000000000001', NULL, 'DevOps Setup',
     'CI/CD pipeline and DevOps infrastructure setup', 'engagement', '019df8b1-0002-7000-8000-000000000002',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, 250000, true, NOW(), NOW()),
    ('42876f10-c005-57e3-9cc0-f22834d20d1a', '019df8b0-0001-7000-8000-000000000001', NULL, 'HR System',
     'Internal human resources management system', 'engagement', '019df8b1-0003-7000-8000-000000000003',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', true, false, 150000, true, NOW(), NOW()),
    ('1493583b-597d-551e-800f-906411ae5948', '019df8b0-0001-7000-8000-000000000001', NULL, 'Finance Tools',
     'Internal finance management and reporting tools', 'engagement', '019df8b1-0003-7000-8000-000000000003',
     'creator_controlled', '019df8b0-0001-7000-8000-000000000001', true, false, 120000, true, NOW(), NOW()),
    -- phases (kind='phase', child of engagement, contract NULL — inherits upward)
    ('586a89d4-c28a-5b2a-8db8-04c3098043b1', '019df8b0-0001-7000-8000-000000000001',
     '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', 'Platform Engineering — Phase 1', 'Core platform engineering workstream',
     'phase', NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(),
     NOW()),
    ('56c2ce54-51c2-5214-8590-5ce5f89128b8', '019df8b0-0001-7000-8000-000000000001',
     '6808945d-ad00-524b-a75c-c363c12ce57d', 'Data Analytics — Platform Build',
     'Data analytics platform implementation', 'phase', NULL, 'creator_controlled',
     '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(), NOW()),
    ('7ebf380c-ba3f-59cd-8576-a303c14a881e', '019df8b0-0001-7000-8000-000000000001',
     'a7b26c54-d058-545f-bff4-c2836402f465', 'Cloud Migration — Assessment & Plan',
     'Cloud migration assessment and planning phase', 'phase', NULL, 'creator_controlled',
     '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(), NOW()),
    ('e9aee293-f7fc-5589-bc87-6557f465ecfb', '019df8b0-0001-7000-8000-000000000001',
     '932d4cff-50b2-58da-bfc9-34748e6fa143', 'DevOps Setup — Pipeline Build', 'CI/CD pipeline implementation', 'phase',
     NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(), NOW()),
    ('589f8f21-e279-54f4-908b-147cae6e6a92', '019df8b0-0001-7000-8000-000000000001',
     '42876f10-c005-57e3-9cc0-f22834d20d1a', 'HR System — Core Features', 'Core HR system feature development', 'phase',
     NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(), NOW()),
    ('e516922a-3449-5511-916c-4869d5a7e8e8', '019df8b0-0001-7000-8000-000000000001',
     '1493583b-597d-551e-800f-906411ae5948', 'Finance Tools — Reporting', 'Finance reporting and dashboard features',
     'phase', NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, NULL, NULL, true, NOW(),
     NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 9. Activity Managers (governance)
-- ============================================================================
INSERT INTO activity_managers (id, activity_id, user_id, created_at)
VALUES ('019df8b3-0001-7000-8000-000000000001', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6',
        '019df6f5-ea95-735d-888b-158583ae4516', NOW()),
       ('019df8b3-0002-7000-8000-000000000002', '6808945d-ad00-524b-a75c-c363c12ce57d',
        '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', NOW()),
       ('019df8b3-0003-7000-8000-000000000003', 'a7b26c54-d058-545f-bff4-c2836402f465',
        '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', NOW()),
       ('019df8b3-0004-7000-8000-000000000004', '932d4cff-50b2-58da-bfc9-34748e6fa143',
        '019df6f5-ea95-735d-888b-158583ae4516', NOW()),
       ('019df8b3-0005-7000-8000-000000000005', '42876f10-c005-57e3-9cc0-f22834d20d1a',
        '019df6f5-ea95-735d-888b-158583ae4516', NOW()),
       ('019df8b3-0006-7000-8000-000000000006', '1493583b-597d-551e-800f-906411ae5948',
        '019df6f5-ea95-735d-888b-158583ae4516', NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 10. Working Groups (6 — one per task activity)
--     Emma (employee_emma) is a DELEGATE on Platform Eng & Data Analytics WGs so
--     the WG-derived manager approval stage shows for a non-manager.
-- ============================================================================
INSERT INTO working_groups (id, org_id, activity_id, name, description, unit_ids, manager_id, delegate_ids, is_active,
                            created_at, updated_at)
VALUES ('78e08f5c-b4b3-5927-a484-77df56d1f49a', '019df8b0-0001-7000-8000-000000000001',
        '586a89d4-c28a-5b2a-8db8-04c3098043b1', 'Platform Engineering WG',
        'Working group for platform engineering deliverables', ARRAY ['2c7d6338-0862-5f71-8620-a36407caf226',
            '4365c0ce-9de2-50dc-9e63-3094923d08ae']::UUID[], '019df6f5-ea95-735d-888b-158583ae4516',
        ARRAY ['019df6f7-3d87-734d-8a97-e93c89641c79']::UUID[], true, NOW(), NOW()),
       ('f097fef9-c784-5bc1-83ec-864271c9fab1', '019df8b0-0001-7000-8000-000000000001',
        '56c2ce54-51c2-5214-8590-5ce5f89128b8', 'Data Analytics WG', 'Working group for data analytics deliverables',
        ARRAY ['47b9341d-896e-58af-8ad8-506f06a0127a', '2c8a850a-6549-5cc7-90d4-227a0da64de9']::UUID[],
        '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', ARRAY ['019df6f7-3d87-734d-8a97-e93c89641c79']::UUID[], true, NOW(),
        NOW()),
       ('a89143d0-84d6-509a-a174-618db00b863a', '019df8b0-0001-7000-8000-000000000001',
        '7ebf380c-ba3f-59cd-8576-a303c14a881e', 'Cloud Migration WG', 'Working group for cloud migration deliverables',
        ARRAY ['2c7d6338-0862-5f71-8620-a36407caf226', '1d0153d7-6708-5678-ac70-d2e900e32383']::UUID[],
        '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', ARRAY []::UUID[], true, NOW(), NOW()),
       ('73291ba3-39b0-5113-b499-fe3f2c4ec190', '019df8b0-0001-7000-8000-000000000001',
        'e9aee293-f7fc-5589-bc87-6557f465ecfb', 'DevOps Setup WG', 'Working group for DevOps setup deliverables',
        ARRAY ['2c7d6338-0862-5f71-8620-a36407caf226', '1d0153d7-6708-5678-ac70-d2e900e32383']::UUID[],
        '019df6f5-ea95-735d-888b-158583ae4516', ARRAY []::UUID[], true, NOW(), NOW()),
       ('e9543bc3-a7fe-5a58-a25f-0803325dca0a', '019df8b0-0001-7000-8000-000000000001',
        '589f8f21-e279-54f4-908b-147cae6e6a92', 'HR System WG', 'Working group for HR system development',
        ARRAY ['45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'fb69255e-f79c-5983-955f-b4ed68daf851']::UUID[],
        '019df6f5-ea95-735d-888b-158583ae4516', ARRAY []::UUID[], true, NOW(), NOW()),
       ('bb3fd0a2-2927-5b20-b93f-2017add2dc2e', '019df8b0-0001-7000-8000-000000000001',
        'e516922a-3449-5511-916c-4869d5a7e8e8', 'Finance Tools WG', 'Working group for finance tools development',
        ARRAY ['45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'd9d57882-fe66-5f75-96fa-b60c48eebde7']::UUID[],
        '019df6f5-ea95-735d-888b-158583ae4516', ARRAY []::UUID[], true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 11. WG Members
-- ============================================================================
INSERT INTO wg_members (id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, created_at)
VALUES ('64cb0529-69f2-5356-a82f-8b2db0f0cf01', '78e08f5c-b4b3-5927-a484-77df56d1f49a',
        '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
       ('2032fa4f-ae95-512b-a2cd-564de777776b', 'f097fef9-c784-5bc1-83ec-864271c9fab1',
        '019df6f7-8ebf-779a-866d-46e196d4928d', '47b9341d-896e-58af-8ad8-506f06a0127a', 'member', true, NOW(), NOW()),
       ('a1cc3e09-0690-5dc7-b59d-0dffdb4d826a', 'a89143d0-84d6-509a-a174-618db00b863a',
        '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
       ('832671ee-e832-5be2-ad36-7f4509a9b85d', '73291ba3-39b0-5113-b499-fe3f2c4ec190',
        '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
       ('62340c91-72c5-5822-9257-1e2b2e3c77c2', 'e9543bc3-a7fe-5a58-a25f-0803325dca0a',
        '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'member', true, NOW(), NOW()),
       ('30e8f4e1-5633-57b9-a6e3-b89119c75a1a', 'bb3fd0a2-2927-5b20-b93f-2017add2dc2e',
        '019df6f7-8ebf-779a-866d-46e196d4928d', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'member', true, NOW(),
        NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 12. Time Entries — anchored to CURRENT_DATE; full approval lifecycle present.
--     activity_id points at the task (leaf) activity; unit_id as before.
-- ============================================================================
INSERT INTO time_entries (id, org_id, user_id, activity_id, unit_id, hours, description, entry_date, status,
                          current_approver_role, submitted_at, is_deleted, created_at, updated_at)
VALUES
    -- ── Emma Wilson (employee_emma) ─────────────────────────────────────────
    ('080becb7-f012-5b56-a093-b11e43f33b25', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '586a89d4-c28a-5b2a-8db8-04c3098043b1',
     '2c7d6338-0862-5f71-8620-a36407caf226', 7.5,
     'Frontend dashboard development — React components for time entry list view', (CURRENT_DATE - 1)::TIMESTAMPTZ,
     'submitted', 'manager', (CURRENT_DATE - 1)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('7715aa96-88a7-5ccd-b013-dc69693d384b', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '586a89d4-c28a-5b2a-8db8-04c3098043b1',
     '2c7d6338-0862-5f71-8620-a36407caf226', 6.0, 'API endpoint implementation for time entry CRUD operations',
     (CURRENT_DATE - 2)::TIMESTAMPTZ, 'pending_manager', 'manager', (CURRENT_DATE - 2)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    ('3bdabfd7-f0b8-566f-8d29-27541bfca284', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '7ebf380c-ba3f-59cd-8576-a303c14a881e',
     '2c7d6338-0862-5f71-8620-a36407caf226', 5.5, 'Cloud infrastructure assessment and migration planning',
     (CURRENT_DATE - 3)::TIMESTAMPTZ, 'pending_finance', 'finance', (CURRENT_DATE - 3)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    ('e11949af-59e7-51ff-8cea-44de121386ee', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '586a89d4-c28a-5b2a-8db8-04c3098043b1',
     '2c7d6338-0862-5f71-8620-a36407caf226', 4.5, 'Code review, sprint planning, and team sync',
     (CURRENT_DATE - 4)::TIMESTAMPTZ, 'approved', NULL, (CURRENT_DATE - 4)::TIMESTAMPTZ, false, NOW(), NOW()),
    -- ── James Park (employee_james) ─────────────────────────────────────────
    ('e00da174-5855-5c34-93a3-42b284488006', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', '56c2ce54-51c2-5214-8590-5ce5f89128b8',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 7.0, 'Data pipeline integration for analytics platform',
     (CURRENT_DATE - 1)::TIMESTAMPTZ, 'submitted', 'manager', (CURRENT_DATE - 1)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('68b40b8d-565e-5d2e-a7cd-26286c12d9a1', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', '56c2ce54-51c2-5214-8590-5ce5f89128b8',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 6.5, 'Client requirements analysis and specification document',
     (CURRENT_DATE - 2)::TIMESTAMPTZ, 'pending_manager', 'manager', (CURRENT_DATE - 2)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    ('53b22fa0-af15-5a0f-96c4-1a7f813888ae', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'e516922a-3449-5511-916c-4869d5a7e8e8',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 5.0, 'Financial reporting dashboard wireframes and data model',
     (CURRENT_DATE - 5)::TIMESTAMPTZ, 'approved', NULL, (CURRENT_DATE - 5)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('f74b7d6e-7451-5440-b056-b4846327cab6', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', '56c2ce54-51c2-5214-8590-5ce5f89128b8',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 4.5, 'Database optimization and query performance tuning',
     (CURRENT_DATE - 6)::TIMESTAMPTZ, 'rejected', NULL, (CURRENT_DATE - 6)::TIMESTAMPTZ, false, NOW(), NOW()),
    -- ── Lisa Torres (employee_lisa) ─────────────────────────────────────────
    ('eed9d22c-bbf5-51b7-b9a5-39a580909a2f', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'e9aee293-f7fc-5589-bc87-6557f465ecfb',
     '2c7d6338-0862-5f71-8620-a36407caf226', 7.0, 'CI/CD pipeline configuration and deployment automation',
     (CURRENT_DATE - 1)::TIMESTAMPTZ, 'submitted', 'manager', (CURRENT_DATE - 1)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('1ff2487a-0654-52d0-ad49-86541942336b', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'e9aee293-f7fc-5589-bc87-6557f465ecfb',
     '2c7d6338-0862-5f71-8620-a36407caf226', 6.0, 'Infrastructure automation with Terraform scripts',
     (CURRENT_DATE - 3)::TIMESTAMPTZ, 'pending_finance', 'finance', (CURRENT_DATE - 3)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    ('ea233fbc-b41f-505a-819e-01eafae5674a', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '589f8f21-e279-54f4-908b-147cae6e6a92',
     '2c7d6338-0862-5f71-8620-a36407caf226', 5.5, 'HR system employee onboarding feature development',
     (CURRENT_DATE - 4)::TIMESTAMPTZ, 'approved', NULL, (CURRENT_DATE - 4)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('98a858ad-ae3b-521b-b1ad-5c5bfeebf1ab', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'e9aee293-f7fc-5589-bc87-6557f465ecfb',
     '2c7d6338-0862-5f71-8620-a36407caf226', 4.0, 'Integration testing and technical documentation updates',
     CURRENT_DATE::TIMESTAMPTZ, 'draft', NULL, NULL, false, NOW(), NOW()),
    -- ── Rachel Kim (hr_rachel) — internal tracking ──────────────────────────
    ('019df8c0-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', '589f8f21-e279-54f4-908b-147cae6e6a92',
     '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 6.0, 'People operations — onboarding paperwork review',
     (CURRENT_DATE - 1)::TIMESTAMPTZ, 'submitted', 'manager', (CURRENT_DATE - 1)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('019df8c0-0002-7000-8000-000000000002', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', '589f8f21-e279-54f4-908b-147cae6e6a92',
     '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 4.0, 'HR policy documentation update', CURRENT_DATE::TIMESTAMPTZ, 'draft',
     NULL, NULL, false, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 13. Expenses — anchored to CURRENT_DATE; full lifecycle present.
--     activity_id is NOT NULL post-011 (non-project spend → 'General & Admin').
-- ============================================================================
INSERT INTO expenses (id, org_id, user_id, activity_id, unit_id, category, amount, currency, description, expense_date,
                      status, current_approver_role, submitted_at, is_deleted, created_at, updated_at)
VALUES
    -- ── Emma Wilson ─────────────────────────────────────────────────────────
    ('12bad46f-1658-5d33-a620-875246666532', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '586a89d4-c28a-5b2a-8db8-04c3098043b1',
     '2c7d6338-0862-5f71-8620-a36407caf226', 'mileage', 18.90, 'EUR', 'Client site visit mileage (45 km × 0.42/km)',
     (CURRENT_DATE - 2)::TIMESTAMPTZ, 'submitted', 'manager', (CURRENT_DATE - 2)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('ad8787e4-4048-56df-9229-93dfde5d888f', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6',
     '2c7d6338-0862-5f71-8620-a36407caf226', 'meal', 32.50, 'EUR', 'Team lunch — sprint retrospective catering',
     (CURRENT_DATE - 4)::TIMESTAMPTZ, 'pending_manager', 'manager', (CURRENT_DATE - 4)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    -- ── James Park ──────────────────────────────────────────────────────────
    ('49c7ea08-d0a9-5b06-8466-70f5ea579409', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 'mileage', 50.40, 'EUR', 'Travel to client office (120 km × 0.42/km)',
     (CURRENT_DATE - 3)::TIMESTAMPTZ, 'pending_finance', 'finance', (CURRENT_DATE - 3)::TIMESTAMPTZ, false, NOW(),
     NOW()),
    ('095b1d5b-3ee5-513d-8d20-a135aa920bc6', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d',
     '47b9341d-896e-58af-8ad8-506f06a0127a', 'meal', 68.00, 'EUR', 'Business dinner with NovaTech stakeholders',
     (CURRENT_DATE - 5)::TIMESTAMPTZ, 'approved', NULL, (CURRENT_DATE - 5)::TIMESTAMPTZ, false, NOW(), NOW()),
    -- ── Lisa Torres ─────────────────────────────────────────────────────────
    ('8e9ed22e-25dd-56b5-921b-40272ed6a91c', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'e9aee293-f7fc-5589-bc87-6557f465ecfb',
     '2c7d6338-0862-5f71-8620-a36407caf226', 'mileage', 15.00, 'EUR', 'Conference parking — DevOps Summit',
     (CURRENT_DATE - 6)::TIMESTAMPTZ, 'approved', NULL, (CURRENT_DATE - 6)::TIMESTAMPTZ, false, NOW(), NOW()),
    ('e4b5c6ac-321b-5c23-a0d9-402d8fdef10c', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '42876f10-c005-57e3-9cc0-f22834d20d1a',
     '2c7d6338-0862-5f71-8620-a36407caf226', 'other', 45.00, 'EUR',
     'Office supplies — ergonomic accessories and cables', CURRENT_DATE::TIMESTAMPTZ, 'draft', NULL, NULL, false, NOW(),
     NOW())
ON CONFLICT (id) DO NOTHING;

COMMIT;
