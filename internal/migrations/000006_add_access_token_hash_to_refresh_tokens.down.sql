DROP INDEX IF EXISTS idx_refresh_tokens_access_token_hash;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS access_token_hash;
