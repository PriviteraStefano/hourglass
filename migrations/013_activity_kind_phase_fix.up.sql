-- 013_activity_kind_phase_fix.up.sql — forward label fix: subprojects → kind 'phase'
--
-- SPEC acceptance #6 (locked in the 09-SPEC interview log) requires
-- subproject-derived activities to carry kind = 'phase'. Migration 011
-- (line 115) hardcoded kind = 'task' for the subproject INSERT.
--
-- 011 is applied history and immutable per ADR-BE-004 (append-only
-- migrations; schema drift is fixed forward). The correction therefore
-- lands as a NEW forward migration instead of an edit to 011.
--
-- The predicate targets ONLY subproject-derived rows: every subproject
-- child has a parent (parent_id IS NOT NULL), and in the pre-deploy MVP
-- seed scope (ADR-P-007 D-6) no user-created 'task'-kind rows exist.
-- Root rows ('engagement'/'internal') and any other kind are untouched.
--
-- Expected post-fix distribution for the MVP seed (13 activities total):
--   engagement 6, phase 6, task 0, internal 1.

UPDATE activities SET kind = 'phase' WHERE kind = 'task' AND parent_id IS NOT NULL;
