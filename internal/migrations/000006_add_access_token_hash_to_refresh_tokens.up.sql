ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS access_token_hash TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_access_token_hash
    ON refresh_tokens(access_token_hash)
    WHERE access_token_hash IS NOT NULL;
