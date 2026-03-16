CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    vk_id VARCHAR(255),
    yandex_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_vk_id_unique ON users(vk_id)
    WHERE vk_id IS NOT NULL AND vk_id != '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_yandex_id_unique ON users(yandex_id)
    WHERE yandex_id IS NOT NULL AND yandex_id != '';

CREATE INDEX IF NOT EXISTS idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_yandex_id ON users(yandex_id) WHERE yandex_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_vk_id ON users(vk_id) WHERE vk_id IS NOT NULL;
