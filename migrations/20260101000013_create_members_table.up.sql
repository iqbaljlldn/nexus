CREATE TABLE members (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nickname      VARCHAR(64),
    joined_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, user_id)
);
CREATE INDEX idx_members_workspace_id ON members (workspace_id);
CREATE INDEX idx_members_user_id ON members (user_id);