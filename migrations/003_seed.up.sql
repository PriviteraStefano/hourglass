-- 003_seed.up.sql — MVP Demo Seed Data (idempotent)
--
-- UUID Mapping Table:
-- ============================================================================
-- Entity                       | UUID
-- -----------------------------|----------------------------------------------
-- Organization                 | 019df8b0-0001-7000-8000-000000000000001
-- Alex Rivera                  | 019df6f5-ea95-735d-888b-158583ae4516
-- Sarah Chen                   | 019df6f6-8cdb-70b9-9d0b-ed032caf9f4b
-- Mike O'Brien                 | 019df6f6-c6ca-7581-83f1-f50ab6c436cf
-- Emma Wilson                  | 019df6f7-3d87-734d-8a97-e93c89641c79
-- James Park                   | 019df6f7-8ebf-779a-866d-46e196d4928d
-- Lisa Torres                  | 019df6f7-c8c8-75c5-a6a7-224bbcd9cff0
-- NovaTech (customer)          | a71c678f-a767-5692-bd7b-0eeb10453073
-- Engineering Unit             | 2c7d6338-0862-5f71-8620-a36407caf226
-- Consulting Unit              | 47b9341d-896e-58af-8ad8-506f06a0127a
-- Operations Unit              | 45a6c1b6-fd3c-5f55-8341-d02aa99841cb
-- Platform Team                | 4365c0ce-9de2-50dc-9e63-3094923d08ae
-- Cloud Infrastructure         | 1d0153d7-6708-5678-ac70-d2e900e32383
-- Data Analytics               | 2c8a850a-6549-5cc7-90d4-227a0da64de9
-- Finance & Accounting         | d9d57882-fe66-5f75-96fa-b60c48eebde7
-- Human Resources              | fb69255e-f79c-5983-955f-b4ed68daf851
-- Contract 1 (Digital Transf.) | 019df8b1-0001-7000-8000-000000000001
-- Contract 2 (Cloud Infra.)    | 019df8b1-0002-7000-8000-000000000002
-- Contract 3 (Internal Ops)    | 019df8b1-0003-7000-8000-000000000003
-- Platform Engineering (proj)  | 81828b6c-75ef-58a9-bdb4-d081ee3e99e6
-- Data Analytics (proj)        | 6808945d-ad00-524b-a75c-c363c12ce57d
-- Cloud Migration (proj)       | a7b26c54-d058-545f-bff4-c2836402f465
-- DevOps Setup (proj)          | 932d4cff-50b2-58da-bfc9-34748e6fa143
-- HR System (proj)             | 42876f10-c005-57e3-9cc0-f22834d20d1a
-- Finance Tools (proj)         | 1493583b-597d-551e-800f-906411ae5948
-- Subproj Platform Eng         | 586a89d4-c28a-5b2a-8db8-04c3098043b1
-- Subproj Data Analytics       | 56c2ce54-51c2-5214-8590-5ce5f89128b8
-- Subproj Cloud Migration      | 7ebf380c-ba3f-59cd-8576-a303c14a881e
-- Subproj DevOps Setup         | e9aee293-f7fc-5589-bc87-6557f465ecfb
-- Subproj HR System            | 589f8f21-e279-54f4-908b-147cae6e6a92
-- Subproj Finance Tools        | e516922a-3449-5511-916c-4869d5a7e8e8
-- WG Platform Eng              | 78e08f5c-b4b3-5927-a484-77df56d1f49a
-- WG Data Analytics            | f097fef9-c784-5bc1-83ec-864271c9fab1
-- WG Cloud Migration           | a89143d0-84d6-509a-a174-618db00b863a
-- WG DevOps Setup              | 73291ba3-39b0-5113-b499-fe3f2c4ec190
-- WG HR System                 | e9543bc3-a7fe-5a58-a25f-0803325dca0a
-- WG Finance Tools             | bb3fd0a2-2927-5b20-b93f-2017add2dc2e
-- ============================================================================

-- ============================================================================
-- 1. Organization
-- ============================================================================
INSERT INTO organizations (id, name, slug, description, created_at, updated_at)
VALUES (
    '019df8b0-0001-7000-8000-000000000001',
    'Tech Consulting Group',
    'tech-consulting-group',
    'Tech Consulting Group — MVP demo organization providing engineering, consulting, and operations services',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 2. Users (6)
-- ============================================================================
INSERT INTO users (id, email, firstname, lastname, username, password_hash, is_active, created_at, updated_at)
VALUES
    ('019df6f5-ea95-735d-888b-158583ae4516', 'alex.rivera@tcg.com', 'Alex', 'Rivera', 'arivera', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
    ('019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', 'sarah.chen@tcg.com', 'Sarah', 'Chen', 'schen', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
    ('019df6f6-c6ca-7581-83f1-f50ab6c436cf', 'mike.obrien@tcg.com', 'Mike', 'O''Brien', 'mobrien', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
    ('019df6f7-3d87-734d-8a97-e93c89641c79', 'emma.wilson@tcg.com', 'Emma', 'Wilson', 'ewilson', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
    ('019df6f7-8ebf-779a-866d-46e196d4928d', 'james.park@tcg.com', 'James', 'Park', 'jpark', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW()),
    ('019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'lisa.torres@tcg.com', 'Lisa', 'Torres', 'ltorres', '$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 3. Customer
-- ============================================================================
INSERT INTO customers (id, org_id, name, contact_name, email, phone, address, vat_number, is_active, created_at, updated_at)
VALUES (
    'a71c678f-a767-5692-bd7b-0eeb10453073',
    '019df8b0-0001-7000-8000-000000000001',
    'NovaTech Industries',
    'Dr. Elena Vasquez',
    'elena.vasquez@novatech.com',
    '+1-555-0100',
    '1200 Innovation Drive, San Francisco, CA 94107',
    'US-123456789',
    true,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 4. Units (8)
-- ============================================================================
-- Level 1: Departments
INSERT INTO units (id, org_id, name, description, code, hierarchy_level, created_at, updated_at)
VALUES
    ('2c7d6338-0862-5f71-8620-a36407caf226', '019df8b0-0001-7000-8000-000000000001', 'Engineering Unit', 'Software engineering and platform development', 'ENG', 1, NOW(), NOW()),
    ('47b9341d-896e-58af-8ad8-506f06a0127a', '019df8b0-0001-7000-8000-000000000001', 'Consulting Unit', 'Business consulting and data analytics services', 'CONS', 1, NOW(), NOW()),
    ('45a6c1b6-fd3c-5f55-8341-d02aa99841cb', '019df8b0-0001-7000-8000-000000000001', 'Operations Unit', 'Operations, finance, and people management', 'OPS', 1, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Level 2: Sub-units
INSERT INTO units (id, org_id, name, description, code, parent_unit_id, hierarchy_level, created_at, updated_at)
VALUES
    ('4365c0ce-9de2-50dc-9e63-3094923d08ae', '019df8b0-0001-7000-8000-000000000001', 'Platform Team', 'Enterprise platform engineering and tooling', 'PLAT', '2c7d6338-0862-5f71-8620-a36407caf226', 2, NOW(), NOW()),
    ('1d0153d7-6708-5678-ac70-d2e900e32383', '019df8b0-0001-7000-8000-000000000001', 'Cloud Infrastructure', 'Cloud infrastructure and DevOps', 'CLOUD', '2c7d6338-0862-5f71-8620-a36407caf226', 2, NOW(), NOW()),
    ('2c8a850a-6549-5cc7-90d4-227a0da64de9', '019df8b0-0001-7000-8000-000000000001', 'Data Analytics', 'Data science and business intelligence', 'DATA', '47b9341d-896e-58af-8ad8-506f06a0127a', 2, NOW(), NOW()),
    ('d9d57882-fe66-5f75-96fa-b60c48eebde7', '019df8b0-0001-7000-8000-000000000001', 'Finance & Accounting', 'Financial planning and accounting', 'FIN', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 2, NOW(), NOW()),
    ('fb69255e-f79c-5983-955f-b4ed68daf851', '019df8b0-0001-7000-8000-000000000001', 'Human Resources', 'HR and people management', 'HR', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 2, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 5. Organization Memberships (6)
-- ============================================================================
INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, created_at, updated_at)
VALUES
    ('019df8b2-0001-7000-8000-000000000001', '019df6f5-ea95-735d-888b-158583ae4516', '019df8b0-0001-7000-8000-000000000001', 'manager', true, NOW(), NOW()),
    ('019df8b2-0002-7000-8000-000000000002', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df8b0-0001-7000-8000-000000000001', 'manager', true, NOW(), NOW()),
    ('019df8b2-0003-7000-8000-000000000003', '019df6f6-c6ca-7581-83f1-f50ab6c436cf', '019df8b0-0001-7000-8000-000000000001', 'finance', true, NOW(), NOW()),
    ('019df8b2-0004-7000-8000-000000000004', '019df6f7-3d87-734d-8a97-e93c89641c79', '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW()),
    ('019df8b2-0005-7000-8000-000000000005', '019df6f7-8ebf-779a-866d-46e196d4928d', '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW()),
    ('019df8b2-0006-7000-8000-000000000006', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '019df8b0-0001-7000-8000-000000000001', 'employee', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 6. Unit Memberships (6 — one per user, primary)
-- ============================================================================
INSERT INTO unit_memberships (id, org_id, user_id, unit_id, is_primary, role, start_date, created_at)
VALUES
    ('1fd79fac-78a0-58de-9f45-b9ff237ea67e', '019df8b0-0001-7000-8000-000000000001', '019df6f5-ea95-735d-888b-158583ae4516', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'manager', NOW(), NOW()),
    ('68e7992a-516f-5378-b389-91f30aeeea27', '019df8b0-0001-7000-8000-000000000001', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '47b9341d-896e-58af-8ad8-506f06a0127a', true, 'manager', NOW(), NOW()),
    ('c94c3236-fb9b-5849-9665-e0e075a0a1a1', '019df8b0-0001-7000-8000-000000000001', '019df6f6-c6ca-7581-83f1-f50ab6c436cf', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', true, 'finance', NOW(), NOW()),
    ('d3236d8a-0e70-516d-a0a1-6bd70bccdd3a', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'employee', NOW(), NOW()),
    ('ab5096c2-679e-50e9-bf7a-b23812b4d3ab', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '47b9341d-896e-58af-8ad8-506f06a0127a', true, 'employee', NOW(), NOW()),
    ('cb838783-7222-5c6a-8d1f-c7f0ab3e5ad8', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '2c7d6338-0862-5f71-8620-a36407caf226', true, 'employee', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 7. Contracts (3)
-- ============================================================================
INSERT INTO contracts (id, name, km_rate, currency, customer_id, governance_model, created_by_org_id, is_shared, is_active, created_at, updated_at)
VALUES
    ('019df8b1-0001-7000-8000-000000000001', 'Digital Transformation Program', 0.42, 'EUR', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, NOW(), NOW()),
    ('019df8b1-0002-7000-8000-000000000002', 'Cloud Infrastructure Migration', 0.35, 'EUR', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, true, NOW(), NOW()),
    ('019df8b1-0003-7000-8000-000000000003', 'Internal Operations', 0.00, 'EUR', NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', true, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 9. Projects (6)
-- ============================================================================
INSERT INTO projects (id, org_id, name, description, project_type, type, contract_id, customer_id, governance_model, created_by_org_id, is_shared, budget_amount, is_active, created_at, updated_at)
VALUES
    ('81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '019df8b0-0001-7000-8000-000000000001', 'Platform Engineering', 'Enterprise platform engineering and development', 'billable', 'billable', '019df8b1-0001-7000-8000-000000000001', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, 500000, true, NOW(), NOW()),
    ('6808945d-ad00-524b-a75c-c363c12ce57d', '019df8b0-0001-7000-8000-000000000001', 'Data Analytics', 'Data analytics and business intelligence platform', 'billable', 'billable', '019df8b1-0001-7000-8000-000000000001', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, 350000, true, NOW(), NOW()),
    ('a7b26c54-d058-545f-bff4-c2836402f465', '019df8b0-0001-7000-8000-000000000001', 'Cloud Migration', 'Cloud infrastructure migration from on-premise to AWS', 'billable', 'billable', '019df8b1-0002-7000-8000-000000000002', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, 400000, true, NOW(), NOW()),
    ('932d4cff-50b2-58da-bfc9-34748e6fa143', '019df8b0-0001-7000-8000-000000000001', 'DevOps Setup', 'CI/CD pipeline and DevOps infrastructure setup', 'billable', 'billable', '019df8b1-0002-7000-8000-000000000002', 'a71c678f-a767-5692-bd7b-0eeb10453073', 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', false, 250000, true, NOW(), NOW()),
    ('42876f10-c005-57e3-9cc0-f22834d20d1a', '019df8b0-0001-7000-8000-000000000001', 'HR System', 'Internal human resources management system', 'internal', 'internal', '019df8b1-0003-7000-8000-000000000003', NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', true, 150000, true, NOW(), NOW()),
    ('1493583b-597d-551e-800f-906411ae5948', '019df8b0-0001-7000-8000-000000000001', 'Finance Tools', 'Internal finance management and reporting tools', 'internal', 'internal', '019df8b1-0003-7000-8000-000000000003', NULL, 'creator_controlled', '019df8b0-0001-7000-8000-000000000001', true, 120000, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 12. Subprojects (6 — one per project)
-- ============================================================================
INSERT INTO subprojects (id, project_id, name, description, sequence_order, is_active, created_at, updated_at)
VALUES
    ('586a89d4-c28a-5b2a-8db8-04c3098043b1', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', 'Platform Engineering — Phase 1', 'Core platform engineering workstream', 1, true, NOW(), NOW()),
    ('56c2ce54-51c2-5214-8590-5ce5f89128b8', '6808945d-ad00-524b-a75c-c363c12ce57d', 'Data Analytics — Platform Build', 'Data analytics platform implementation', 2, true, NOW(), NOW()),
    ('7ebf380c-ba3f-59cd-8576-a303c14a881e', 'a7b26c54-d058-545f-bff4-c2836402f465', 'Cloud Migration — Assessment & Plan', 'Cloud migration assessment and planning phase', 3, true, NOW(), NOW()),
    ('e9aee293-f7fc-5589-bc87-6557f465ecfb', '932d4cff-50b2-58da-bfc9-34748e6fa143', 'DevOps Setup — Pipeline Build', 'CI/CD pipeline implementation', 4, true, NOW(), NOW()),
    ('589f8f21-e279-54f4-908b-147cae6e6a92', '42876f10-c005-57e3-9cc0-f22834d20d1a', 'HR System — Core Features', 'Core HR system feature development', 5, true, NOW(), NOW()),
    ('e516922a-3449-5511-916c-4869d5a7e8e8', '1493583b-597d-551e-800f-906411ae5948', 'Finance Tools — Reporting', 'Finance reporting and dashboard features', 6, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 13. Working Groups (6 — one per subproject)
-- ============================================================================
INSERT INTO working_groups (id, org_id, subproject_id, name, description, unit_ids, enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at)
VALUES
    ('78e08f5c-b4b3-5927-a484-77df56d1f49a', '019df8b0-0001-7000-8000-000000000001', '586a89d4-c28a-5b2a-8db8-04c3098043b1', 'Platform Engineering WG', 'Working group for platform engineering deliverables', ARRAY['2c7d6338-0862-5f71-8620-a36407caf226', '4365c0ce-9de2-50dc-9e63-3094923d08ae']::UUID[], true, '019df6f5-ea95-735d-888b-158583ae4516', ARRAY[]::UUID[], true, NOW(), NOW()),
    ('f097fef9-c784-5bc1-83ec-864271c9fab1', '019df8b0-0001-7000-8000-000000000001', '56c2ce54-51c2-5214-8590-5ce5f89128b8', 'Data Analytics WG', 'Working group for data analytics deliverables', ARRAY['47b9341d-896e-58af-8ad8-506f06a0127a', '2c8a850a-6549-5cc7-90d4-227a0da64de9']::UUID[], true, '019df6f5-ea95-735d-888b-158583ae4516', ARRAY[]::UUID[], true, NOW(), NOW()),
    ('a89143d0-84d6-509a-a174-618db00b863a', '019df8b0-0001-7000-8000-000000000001', '7ebf380c-ba3f-59cd-8576-a303c14a881e', 'Cloud Migration WG', 'Working group for cloud migration deliverables', ARRAY['2c7d6338-0862-5f71-8620-a36407caf226', '1d0153d7-6708-5678-ac70-d2e900e32383']::UUID[], true, '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', ARRAY[]::UUID[], true, NOW(), NOW()),
    ('73291ba3-39b0-5113-b499-fe3f2c4ec190', '019df8b0-0001-7000-8000-000000000001', 'e9aee293-f7fc-5589-bc87-6557f465ecfb', 'DevOps Setup WG', 'Working group for DevOps setup deliverables', ARRAY['2c7d6338-0862-5f71-8620-a36407caf226', '1d0153d7-6708-5678-ac70-d2e900e32383']::UUID[], true, '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', ARRAY[]::UUID[], true, NOW(), NOW()),
    ('e9543bc3-a7fe-5a58-a25f-0803325dca0a', '019df8b0-0001-7000-8000-000000000001', '589f8f21-e279-54f4-908b-147cae6e6a92', 'HR System WG', 'Working group for HR system development', ARRAY['45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'fb69255e-f79c-5983-955f-b4ed68daf851']::UUID[], true, '019df6f5-ea95-735d-888b-158583ae4516', ARRAY[]::UUID[], true, NOW(), NOW()),
    ('bb3fd0a2-2927-5b20-b93f-2017add2dc2e', '019df8b0-0001-7000-8000-000000000001', 'e516922a-3449-5511-916c-4869d5a7e8e8', 'Finance Tools WG', 'Working group for finance tools development', ARRAY['45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'd9d57882-fe66-5f75-96fa-b60c48eebde7']::UUID[], true, '019df6f5-ea95-735d-888b-158583ae4516', ARRAY[]::UUID[], true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 14. WG Members (6 — one per working group)
-- ============================================================================
INSERT INTO wg_members (id, wg_id, user_id, unit_id, role, is_default_subproject, start_date, created_at)
VALUES
    ('64cb0529-69f2-5356-a82f-8b2db0f0cf01', '78e08f5c-b4b3-5927-a484-77df56d1f49a', '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
    ('2032fa4f-ae95-512b-a2cd-564de777776b', 'f097fef9-c784-5bc1-83ec-864271c9fab1', '019df6f7-8ebf-779a-866d-46e196d4928d', '47b9341d-896e-58af-8ad8-506f06a0127a', 'member', true, NOW(), NOW()),
    ('a1cc3e09-0690-5dc7-b59d-0dffdb4d826a', 'a89143d0-84d6-509a-a174-618db00b863a', '019df6f7-3d87-734d-8a97-e93c89641c79', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
    ('832671ee-e832-5be2-ad36-7f4509a9b85d', '73291ba3-39b0-5113-b499-fe3f2c4ec190', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', true, NOW(), NOW()),
    ('62340c91-72c5-5822-9257-1e2b2e3c77c2', 'e9543bc3-a7fe-5a58-a25f-0803325dca0a', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'member', true, NOW(), NOW()),
    ('30e8f4e1-5633-57b9-a6e3-b89119c75a1a', 'bb3fd0a2-2927-5b20-b93f-2017add2dc2e', '019df6f7-8ebf-779a-866d-46e196d4928d', '45a6c1b6-fd3c-5f55-8341-d02aa99841cb', 'member', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 15. Time Entries (12 — 4 per employee)
-- ============================================================================
INSERT INTO time_entries (id, org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status, is_deleted, created_at, updated_at)
VALUES
    -- Emma Wilson — Entry 1: Mon 18 May — Platform Engineering
    ('080becb7-f012-5b56-a093-b11e43f33b25', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '586a89d4-c28a-5b2a-8db8-04c3098043b1', '78e08f5c-b4b3-5927-a484-77df56d1f49a', '2c7d6338-0862-5f71-8620-a36407caf226', 7.5, 'Frontend dashboard development — React components for time entry list view', '2026-05-18 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Emma Wilson — Entry 2: Fri 15 May — Platform Engineering
    ('7715aa96-88a7-5ccd-b013-dc69693d384b', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '586a89d4-c28a-5b2a-8db8-04c3098043b1', '78e08f5c-b4b3-5927-a484-77df56d1f49a', '2c7d6338-0862-5f71-8620-a36407caf226', 6.0, 'API endpoint implementation for time entry CRUD operations', '2026-05-15 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Emma Wilson — Entry 3: Thu 14 May — Cloud Migration
    ('3bdabfd7-f0b8-566f-8d29-27541bfca284', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', 'a7b26c54-d058-545f-bff4-c2836402f465', '7ebf380c-ba3f-59cd-8576-a303c14a881e', 'a89143d0-84d6-509a-a174-618db00b863a', '2c7d6338-0862-5f71-8620-a36407caf226', 5.5, 'Cloud infrastructure assessment and migration planning', '2026-05-14 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Emma Wilson — Entry 4: Wed 13 May — Platform Engineering
    ('e11949af-59e7-51ff-8cea-44de121386ee', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '586a89d4-c28a-5b2a-8db8-04c3098043b1', '78e08f5c-b4b3-5927-a484-77df56d1f49a', '2c7d6338-0862-5f71-8620-a36407caf226', 4.5, 'Code review, sprint planning, and team sync', '2026-05-13 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Entry 5: Mon 18 May — Data Analytics
    ('e00da174-5855-5c34-93a3-42b284488006', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d', '56c2ce54-51c2-5214-8590-5ce5f89128b8', 'f097fef9-c784-5bc1-83ec-864271c9fab1', '47b9341d-896e-58af-8ad8-506f06a0127a', 7.0, 'Data pipeline integration for analytics platform', '2026-05-18 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Entry 6: Fri 15 May — Data Analytics
    ('68b40b8d-565e-5d2e-a7cd-26286c12d9a1', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d', '56c2ce54-51c2-5214-8590-5ce5f89128b8', 'f097fef9-c784-5bc1-83ec-864271c9fab1', '47b9341d-896e-58af-8ad8-506f06a0127a', 6.5, 'Client requirements analysis and specification document', '2026-05-15 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Entry 7: Thu 14 May — Finance Tools
    ('53b22fa0-af15-5a0f-96c4-1a7f813888ae', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '1493583b-597d-551e-800f-906411ae5948', 'e516922a-3449-5511-916c-4869d5a7e8e8', 'bb3fd0a2-2927-5b20-b93f-2017add2dc2e', '47b9341d-896e-58af-8ad8-506f06a0127a', 5.0, 'Financial reporting dashboard wireframes and data model', '2026-05-14 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Entry 8: Tue 12 May — Data Analytics
    ('f74b7d6e-7451-5440-b056-b4846327cab6', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d', '56c2ce54-51c2-5214-8590-5ce5f89128b8', 'f097fef9-c784-5bc1-83ec-864271c9fab1', '47b9341d-896e-58af-8ad8-506f06a0127a', 4.5, 'Database optimization and query performance tuning', '2026-05-12 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Entry 9: Mon 18 May — DevOps Setup
    ('eed9d22c-bbf5-51b7-b9a5-39a580909a2f', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '932d4cff-50b2-58da-bfc9-34748e6fa143', 'e9aee293-f7fc-5589-bc87-6557f465ecfb', '73291ba3-39b0-5113-b499-fe3f2c4ec190', '2c7d6338-0862-5f71-8620-a36407caf226', 7.0, 'CI/CD pipeline configuration and deployment automation', '2026-05-18 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Entry 10: Fri 15 May — DevOps Setup
    ('1ff2487a-0654-52d0-ad49-86541942336b', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '932d4cff-50b2-58da-bfc9-34748e6fa143', 'e9aee293-f7fc-5589-bc87-6557f465ecfb', '73291ba3-39b0-5113-b499-fe3f2c4ec190', '2c7d6338-0862-5f71-8620-a36407caf226', 6.0, 'Infrastructure automation with Terraform scripts', '2026-05-15 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Entry 11: Thu 14 May — HR System
    ('ea233fbc-b41f-505a-819e-01eafae5674a', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '42876f10-c005-57e3-9cc0-f22834d20d1a', '589f8f21-e279-54f4-908b-147cae6e6a92', 'e9543bc3-a7fe-5a58-a25f-0803325dca0a', '2c7d6338-0862-5f71-8620-a36407caf226', 5.5, 'HR system employee onboarding feature development', '2026-05-14 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Entry 12: Wed 13 May — DevOps Setup
    ('98a858ad-ae3b-521b-b1ad-5c5bfeebf1ab', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '932d4cff-50b2-58da-bfc9-34748e6fa143', 'e9aee293-f7fc-5589-bc87-6557f465ecfb', '73291ba3-39b0-5113-b499-fe3f2c4ec190', '2c7d6338-0862-5f71-8620-a36407caf226', 4.0, 'Integration testing and technical documentation updates', '2026-05-13 09:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- 17. Expenses (6 — 2 per employee)
-- ============================================================================
INSERT INTO expenses (id, org_id, user_id, project_id, unit_id, category, amount, currency, description, expense_date, status, is_deleted, created_at, updated_at)
VALUES
    -- Emma Wilson — Expense 1: Client mileage
    ('12bad46f-1658-5d33-a620-875246666532', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', '81828b6c-75ef-58a9-bdb4-d081ee3e99e6', '2c7d6338-0862-5f71-8620-a36407caf226', 'mileage', 18.90, 'EUR', 'Client site visit mileage (45 km × 0.42/km)', '2026-05-15 12:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Emma Wilson — Expense 2: Team lunch
    ('ad8787e4-4048-56df-9229-93dfde5d888f', '019df8b0-0001-7000-8000-000000000001', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL, '2c7d6338-0862-5f71-8620-a36407caf226', 'meal', 32.50, 'EUR', 'Team lunch meeting — sprint retrospective catering', '2026-05-13 12:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Expense 3: Travel mileage
    ('49c7ea08-d0a9-5b06-8466-70f5ea579409', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d', '47b9341d-896e-58af-8ad8-506f06a0127a', 'mileage', 50.40, 'EUR', 'Travel to client office (120 km × 0.42/km)', '2026-05-14 08:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- James Park — Expense 4: Business dinner
    ('095b1d5b-3ee5-513d-8d20-a135aa920bc6', '019df8b0-0001-7000-8000-000000000001', '019df6f7-8ebf-779a-866d-46e196d4928d', '6808945d-ad00-524b-a75c-c363c12ce57d', '47b9341d-896e-58af-8ad8-506f06a0127a', 'meal', 68.00, 'EUR', 'Business dinner with NovaTech stakeholders', '2026-05-18 19:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Expense 5: Conference parking
    ('8e9ed22e-25dd-56b5-921b-40272ed6a91c', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '932d4cff-50b2-58da-bfc9-34748e6fa143', '2c7d6338-0862-5f71-8620-a36407caf226', 'mileage', 15.00, 'EUR', 'Conference parking — DevOps Summit 2026', '2026-05-13 08:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW()),
    -- Lisa Torres — Expense 6: Office supplies
    ('e4b5c6ac-321b-5c23-a0d9-402d8fdef10c', '019df8b0-0001-7000-8000-000000000001', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', NULL, '2c7d6338-0862-5f71-8620-a36407caf226', 'other', 45.00, 'EUR', 'Office supplies — ergonomic accessories and cables', '2026-05-12 10:00:00+00'::TIMESTAMPTZ, 'submitted', false, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
