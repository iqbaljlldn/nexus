CREATE TABLE roles (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    permission_bitmask BIGINT NOT NULL DEFAULT 0,
    position      INT NOT NULL DEFAULT 0,
    is_everyone   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_roles_workspace_id_position ON roles (workspace_id, position DESC);