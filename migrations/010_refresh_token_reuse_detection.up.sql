-- 010_refresh_token_reuse_detection.up.sql
-- Adds refresh-token family tracking for reuse detection (audit P0-5).
--
-- family_id groups a chain of rotated tokens: every rotation issues a
-- successor that inherits the family. If a rotated (or revoked) token is
-- ever presented again, the whole family is revoked — an attacker replaying
-- a stolen token kills every session in that family.
--
-- rotated_at marks the moment a token was rotated. revoked_at (already in the
-- base schema) remains the tombstone used by RevokeByHash/RevokeAllByUser.

ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS family_id UUID;
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS rotated_at TIMESTAMPTZ;

-- Backfill: each existing row becomes the root of its own family.
UPDATE refresh_tokens SET family_id = gen_random_uuid() WHERE family_id IS NULL;

-- New rows always belong to a fresh family (evaluated per-row for INSERT).
ALTER TABLE refresh_tokens ALTER COLUMN family_id SET DEFAULT gen_random_uuid();

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens(family_id);
