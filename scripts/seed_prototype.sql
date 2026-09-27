-- ============================================================================
-- PROTOTYPE FIXTURE — WIPE ME. NOT THE DEMO SEED.
-- ============================================================================
-- Rows for the screens being prototyped ahead of their job-cluster phase:
-- the availability windows and direction rows that the *live* read-models
-- (`GET /direction`, `GET /direction/coverage`) need in order to render
-- anything at all. `scripts/seed_demo.sql` predates migrations 019-022 and
-- therefore has no rows for these tables.
--
-- Apply against the local demo DB:
--   docker exec -i hourglass-postgres psql -U hourglass -d hourglass \
--     -v ON_ERROR_STOP=1 < scripts/seed_prototype.sql
--
-- Idempotent: fixed UUIDs + ON CONFLICT (id) DO NOTHING.
-- Delete everything this file owns:
--   DELETE FROM direction WHERE id::text LIKE '019df9%';
--   DELETE FROM availability_windows WHERE id::text LIKE '019df9%';
--   UPDATE organization_memberships SET valid_until = NULL
--     WHERE user_id = '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0';
--
-- Once the job-cluster phases land, fold the keepers into seed_demo.sql and
-- delete this file (see the prototype capture rule in `skill://prototype`).
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 1. Availability windows (E-08 / H-01)
-- ----------------------------------------------------------------------------
INSERT INTO availability_windows
    (id, org_id, user_id, kind, starts_on, ends_on, hours, certificate_ref, note, status, created_by)
VALUES
    -- Emma Wilson — confirmed holiday, declared half day, declared medical
    ('019df900-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', 'holiday', '2026-10-12', '2026-10-16',
     NULL, NULL, 'Booked leave, handover done', 'confirmed', '019df6f8-0001-7000-8000-000000000007'),
    ('019df900-0002-7000-8000-000000000002', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', 'unavailable', '2026-10-02', '2026-10-02',
     4.00, NULL, 'Afternoon only', 'declared', '019df6f7-3d87-734d-8a97-e93c89641c79'),
    ('019df900-0003-7000-8000-000000000003', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', 'medical', '2026-11-16', '2026-11-17',
     NULL, 'AU-2026-4712', NULL, 'declared', '019df6f7-3d87-734d-8a97-e93c89641c79'),

    -- James Park — medical (certificate on file), declared holiday, partial day
    ('019df900-0004-7000-8000-000000000004', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'medical', '2026-09-29', '2026-10-03',
     NULL, 'AU-2026-4471', 'Certificate in the HR folder', 'declared', '019df6f7-8ebf-779a-866d-46e196d4928d'),
    ('019df900-0005-7000-8000-000000000005', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'holiday', '2026-11-02', '2026-11-06',
     NULL, NULL, NULL, 'declared', '019df6f7-8ebf-779a-866d-46e196d4928d'),
    ('019df900-0006-7000-8000-000000000006', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'unavailable', '2026-10-21', '2026-10-21',
     6.00, NULL, 'Training day, back at 16:00', 'declared', '019df6f7-8ebf-779a-866d-46e196d4928d'),

    -- Lisa Torres — permit renewal window, confirmed holiday
    ('019df900-0007-7000-8000-000000000007', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'permit', '2026-10-05', '2026-10-20',
     NULL, NULL, 'Work permit renewal window', 'confirmed', '019df6f8-0001-7000-8000-000000000007'),
    ('019df900-0008-7000-8000-000000000008', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'holiday', '2026-12-21', '2026-12-31',
     NULL, NULL, NULL, 'confirmed', '019df6f8-0001-7000-8000-000000000007'),

    -- Alex Rivera — half day, confirmed medical in the past
    ('019df900-0009-7000-8000-000000000009', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', 'unavailable', '2026-10-08', '2026-10-08',
     4.00, NULL, 'Supplier workshop', 'declared', '019df6f5-ea95-735d-888b-158583ae4516'),
    ('019df900-000a-7000-8000-00000000000a', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', 'medical', '2026-09-24', '2026-09-25',
     NULL, 'AU-2026-4390', NULL, 'confirmed', '019df6f8-0001-7000-8000-000000000007'),

    -- Sarah Chen — declared holiday overlapping her planned week
    ('019df900-000b-7000-8000-00000000000b', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', 'holiday', '2026-10-26', '2026-10-30',
     NULL, NULL, NULL, 'declared', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b'),

    -- Mike O'Brien — month-end support only, confirmed
    ('019df900-000c-7000-8000-00000000000c', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-c6ca-7581-83f1-f50ab6c436cf', 'unavailable', '2026-10-08', '2026-10-09',
     4.00, NULL, 'Month-end support only', 'confirmed', '019df6f8-0001-7000-8000-000000000007'),

    -- Rachel Kim — declared half day
    ('019df900-000d-7000-8000-00000000000d', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', 'unavailable', '2026-10-15', '2026-10-15',
     2.00, NULL, NULL, 'declared', '019df6f8-0001-7000-8000-000000000007')
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2. Direction rows (M-02 set the plan · E-09 queue + claims · M-03 capacity)
--    Scheduled = planned_date + est_hours; WG rows are queued-only.
--    Only draft|active scheduled rows feed the coverage read-model.
-- ----------------------------------------------------------------------------
INSERT INTO direction
    (id, org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours,
     priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
VALUES
    -- Emma — a normal planned week, then her confirmed holiday
    ('019df910-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL,
     '586a89d4-c28a-5b2a-8db8-04c3098043b1', '2026-10-05', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-0002-7000-8000-000000000002', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL,
     '586a89d4-c28a-5b2a-8db8-04c3098043b1', '2026-10-06', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-0003-7000-8000-000000000003', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL,
     '586a89d4-c28a-5b2a-8db8-04c3098043b1', '2026-10-07', 6.00, NULL, NULL, 'draft', NULL, NULL, NULL, NOW(), NOW()),

    -- James — planned work on an away day (over-capacity) + the rest of the week
    ('019df910-0004-7000-8000-000000000004', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df6f7-8ebf-779a-866d-46e196d4928d', NULL,
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', '2026-10-01', 4.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-0005-7000-8000-000000000005', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df6f7-8ebf-779a-866d-46e196d4928d', NULL,
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', '2026-10-06', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-0006-7000-8000-000000000006', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df6f7-8ebf-779a-866d-46e196d4928d', NULL,
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', '2026-10-07', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),

    -- Lisa — back on the plan after her permit window closes
    ('019df910-0007-7000-8000-000000000007', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', NULL,
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', '2026-10-21', 6.00, NULL, NULL, 'draft', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-0008-7000-8000-000000000008', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', NULL,
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', '2026-10-22', 6.00, NULL, NULL, 'draft', NULL, NULL, NULL, NOW(), NOW()),

    -- Alex — a full planned week (his half day makes one day partial)
    ('019df910-0009-7000-8000-000000000009', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', '019df6f5-ea95-735d-888b-158583ae4516', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', '2026-10-12', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-000a-7000-8000-00000000000a', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', '019df6f5-ea95-735d-888b-158583ae4516', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', '2026-10-13', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-000b-7000-8000-00000000000b', '019df8b0-0001-7000-8000-000000000001',
     '019df6f8-0001-7000-8000-000000000007', '019df6f5-ea95-735d-888b-158583ae4516', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', '2026-10-14', 8.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),

    -- Sarah — planned work on her declared holiday week (over-capacity)
    ('019df910-000c-7000-8000-00000000000c', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', '2026-10-26', 6.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df910-000d-7000-8000-00000000000d', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', '2026-10-27', 6.00, NULL, NULL, 'active', NULL, NULL, NULL, NOW(), NOW()),

    -- Queued WG rows (E-09 queue: nobody owns them yet)
    ('019df920-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, 'a89143d0-84d6-509a-a174-618db00b863a',
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', NULL, 16.00, 1, '2026-10-30', 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df920-0002-7000-8000-000000000002', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, 'a89143d0-84d6-509a-a174-618db00b863a',
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', NULL, 8.00, 2, '2026-11-06', 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df920-0003-7000-8000-000000000003', '019df8b0-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', NULL, 'f097fef9-c784-5bc1-83ec-864271c9fab1',
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', NULL, 12.00, 1, '2026-11-13', 'active', NULL, NULL, NULL, NOW(), NOW()),

    -- One claimed queued row: Emma took 8 h off the Cloud Migration queue
    ('019df920-0004-7000-8000-000000000004', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', NULL, 8.00, 1, '2026-11-06', 'draft',
     NULL, '019df920-0002-7000-8000-000000000002', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2b. Queue depth (E-09): more WG rows so the claim surface has real choice,
--     plus one PARTIAL claim (6 h of a 16 h pool) so `partially_claimed` shows.
--     Each row sits inside its group's anchored activity.
-- ----------------------------------------------------------------------------
INSERT INTO direction
    (id, org_id, directed_by, directed_to, wg_id, activity_id, planned_date, est_hours,
     priority, due_date, status, supersedes_id, origin_direction_id, reason, created_at, updated_at)
VALUES
    ('019df920-0005-7000-8000-000000000005', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, 'a89143d0-84d6-509a-a174-618db00b863a',
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', NULL, 40.00, 3, '2026-11-27', 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df920-0006-7000-8000-000000000006', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, 'f097fef9-c784-5bc1-83ec-864271c9fab1',
     '56c2ce54-51c2-5214-8590-5ce5f89128b8', NULL, 20.00, 2, '2026-11-13', 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df920-0007-7000-8000-000000000007', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, '73291ba3-39b0-5113-b499-fe3f2c4ec190',
     'e9aee293-f7fc-5589-bc87-6557f465ecfb', NULL, 16.00, 1, '2026-11-30', 'active', NULL, NULL, NULL, NOW(), NOW()),
    -- Platform Engineering WG: the viewer's own group, freshly queued
    ('019df920-0008-7000-8000-000000000008', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, '78e08f5c-b4b3-5927-a484-77df56d1f49a',
     '586a89d4-c28a-5b2a-8db8-04c3098043b1', NULL, 24.00, 1, '2026-11-20', 'active', NULL, NULL, NULL, NOW(), NOW()),
    ('019df920-0009-7000-8000-000000000009', '019df8b0-0001-7000-8000-000000000001',
     '019df6f5-ea95-735d-888b-158583ae4516', NULL, '78e08f5c-b4b3-5927-a484-77df56d1f49a',
     '586a89d4-c28a-5b2a-8db8-04c3098043b1', NULL, 12.00, 2, '2026-12-04', 'active', NULL, NULL, NULL, NOW(), NOW()),

    -- Partial claim: Emma takes 6 h of the 16 h Cloud Migration pool
    ('019df920-000a-7000-8000-00000000000a', '019df8b0-0001-7000-8000-000000000001',
     '019df6f7-3d87-734d-8a97-e93c89641c79', '019df6f7-3d87-734d-8a97-e93c89641c79', NULL,
     '7ebf380c-ba3f-59cd-8576-a303c14a881e', NULL, 6.00, 1, '2026-10-30', 'draft',
     NULL, '019df920-0001-7000-8000-000000000001', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 2c. A manager who is ALSO a group member (the dimensions are independent):
--     the org-wide queue read is manager-only while the claim write is
--     membership-only, so without this row no single persona can both see the
--     pools and draw on them.
-- ----------------------------------------------------------------------------
INSERT INTO wg_members (id, wg_id, user_id, unit_id, role, is_default_subproject)
VALUES
    ('019df930-0001-7000-8000-000000000001', 'a89143d0-84d6-509a-a174-618db00b863a',
     '019df6f5-ea95-735d-888b-158583ae4516', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', false),
    ('019df930-0002-7000-8000-000000000002', '78e08f5c-b4b3-5927-a484-77df56d1f49a',
     '019df6f5-ea95-735d-888b-158583ae4516', '2c7d6338-0862-5f71-8620-a36407caf226', 'member', false)
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 4. Tickets (E-05 raise · M-08 triage) — the demo DB's own tickets came from
--    hand testing, so the prototype seeds its own set, one per lifecycle state,
--    plus comments. Ticket T6 is linked to a `customer_ticket` activity that
--    carries a submitted time entry: dismissing it must fail with
--    `dismissal_blocked` (ticket.go:288) — the guard the surface has to explain.
-- ----------------------------------------------------------------------------
INSERT INTO tickets (id, org_id, title, description, kind, status, requester_id, assignee_id, created_at, updated_at)
VALUES
    ('019df940-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
     'Export drops the last day of the range', 'Pick 1–31 Oct and the 31st is missing from the CSV.',
     'bug', 'open', '019df6f7-8ebf-779a-866d-46e196d4928d', NULL, NOW(), NOW()),
    ('019df940-0002-7000-8000-000000000002', '019df8b0-0001-7000-8000-000000000001',
     'Add a per-unit non-billed column to the export', 'Finance wants warranty and goodwill separated per unit.',
     'change', 'triage', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', NULL, NOW(), NOW()),
    ('019df940-0003-7000-8000-000000000003', '019df8b0-0001-7000-8000-000000000001',
     'Mileage amount rounds to two decimals', '0.435 €/km loses a cent per leg on long trips.',
     'bug', 'planned', '019df6f7-3d87-734d-8a97-e93c89641c79', '019df6f7-8ebf-779a-866d-46e196d4928d', NOW(), NOW()),
    ('019df940-0004-7000-8000-000000000004', '019df8b0-0001-7000-8000-000000000001',
     'Queue: show the remaining pool budget in the rail', 'I cannot see how much of a pool is left before I open it.',
     'evolution', 'in_progress', '019df6f7-3d87-734d-8a97-e93c89641c79', '019df6f7-3d87-734d-8a97-e93c89641c79', NOW(), NOW()),
    ('019df940-0005-7000-8000-000000000005', '019df8b0-0001-7000-8000-000000000001',
     'Where do I see my own coverage?', 'Answered: the entry detail shows how the hours were labeled.',
     'question', 'resolved', '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', '019df6f7-3d87-734d-8a97-e93c89641c79', NOW(), NOW()),
    ('019df940-0006-7000-8000-000000000006', '019df8b0-0001-7000-8000-000000000001',
     'Time entry loses its unit when a parent unit is reparented', 'After a reparent the entry shows no unit at all.',
     'bug', 'open', '019df6f7-8ebf-779a-866d-46e196d4928d', NULL, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO ticket_comments (id, ticket_id, author_id, body, created_at)
VALUES
    ('019df941-0001-7000-8000-000000000001', '019df940-0001-7000-8000-000000000001',
     '019df6f6-8cdb-70b9-9d0b-ed032caf9f4b', 'Reproduced on the October range — the query uses an exclusive upper bound.', NOW()),
    ('019df941-0002-7000-8000-000000000002', '019df940-0001-7000-8000-000000000001',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'Same for expenses.', NOW()),
    ('019df941-0003-7000-8000-000000000003', '019df940-0002-7000-8000-000000000002',
     '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0', 'Warranty hours should be non-billed, goodwill too.', NOW()),
    ('019df941-0004-7000-8000-000000000004', '019df940-0004-7000-8000-000000000004',
     '019df6f7-3d87-734d-8a97-e93c89641c79', 'Started on the rail change; the bar needs the pool budget from the read-model.', NOW()),
    ('019df941-0005-7000-8000-000000000005', '019df940-0006-7000-8000-000000000006',
     '019df6f7-8ebf-779a-866d-46e196d4928d', 'Workaround: re-pick the unit before submitting.', NOW())
ON CONFLICT (id) DO NOTHING;

-- The customer_ticket activity behind T6, plus a submitted 8 h entry on it.
INSERT INTO activities (id, org_id, name, description, kind, governance_model, created_by_org_id,
                        is_shared, is_active, origin_type, ticket_id, created_at, updated_at)
VALUES ('019df942-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
        'Unit reparent — bugfix', 'Work produced by ticket T6.', 'phase', 'creator_controlled',
        '019df8b0-0001-7000-8000-000000000001', false, true, 'customer_ticket',
        '019df940-0006-7000-8000-000000000006', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO time_entries (id, org_id, user_id, unit_id, hours, description, entry_date, status,
                          is_deleted, activity_id, created_at, updated_at)
VALUES ('019df943-0001-7000-8000-000000000001', '019df8b0-0001-7000-8000-000000000001',
        '019df6f7-8ebf-779a-866d-46e196d4928d', '2c7d6338-0862-5f71-8620-a36407caf226', 8.00,
        'Reparent bugfix', DATE '2026-09-20', 'submitted', false,
        '019df942-0001-7000-8000-000000000001', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- 3. Validity demo (H-02 / the `invalid` warning): Lisa's employment window
--    closes mid-period, so October drops her rows and surfaces `invalid`.
-- ----------------------------------------------------------------------------
UPDATE organization_memberships
   SET valid_from  = COALESCE(valid_from, '2026-01-01'),
       valid_until = '2026-10-15'
 WHERE user_id = '019df6f7-c8c8-75c5-a6a7-224bbcd9cff0';

COMMIT;

-- ----------------------------------------------------------------------------
-- Sanity check (run after applying):
--   SELECT count(*) FROM availability_windows;   -- >= 13
--   SELECT count(*) FROM direction;              -- >= 17
-- ----------------------------------------------------------------------------
