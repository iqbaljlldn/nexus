CREATE TABLE invites (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code          VARCHAR(16) NOT NULL UNIQUE,
    created_by    UUID NOT NULL REFERENCES users(id),
    max_uses      INT,
    use_count     INT NOT NULL DEFAULT 0,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);