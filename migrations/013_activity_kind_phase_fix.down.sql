-- 013_activity_kind_phase_fix.down.sql — reverse the label fix: phase → task
--
-- Reverses 013 up for the same row set: parented activities carrying
-- kind = 'phase' are relabeled 'task'.
--
-- Best-effort per ADR-BE-004: a user-created 'phase'-kind child created
-- between up and down is also reverted. Acceptable: pre-deploy MVP seed
-- scope only (ADR-P-007 D-6) — no user-created phase children exist
-- before deploy.

UPDATE activities SET kind = 'task' WHERE kind = 'phase' AND parent_id IS NOT NULL;
