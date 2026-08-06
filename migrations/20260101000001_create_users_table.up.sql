-- ==============================================================================
-- Migration: Create users table (Identity Domain §2.1)
-- ==============================================================================
-- Ref: Database Design §2.1, Playbook §7.6 (UUID v7, timestamptz, soft delete)

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    email          CITEXT NOT NULL UNIQUE,
    username       CITEXT NOT NULL UNIQUE,
    display_name   VARCHAR(64) NOT NULL,
    password_hash  TEXT NOT NULL,
    avatar_url     TEXT,
    is_suspended   BOOLEAN NOT NULL DEFAULT FALSE,
    is_banned      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ
);

-- Partial index for soft-deleted rows (efficient queries filtering deleted users)
CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NOT NULL;
