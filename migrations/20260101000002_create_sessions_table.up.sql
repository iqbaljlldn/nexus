-- ==============================================================================
-- Migration: Create sessions table (Identity Domain §2.1)
-- ==============================================================================
-- Ref: Database Design §2.1, SRS §3.8 (refresh token hash, not plaintext)

CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE,
    user_agent          TEXT,
    ip_address          INET,
    status              VARCHAR(16) NOT NULL DEFAULT 'active',
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Composite index for querying active sessions per user (login check, session list)
CREATE INDEX idx_sessions_user_id_status ON sessions (user_id, status);
