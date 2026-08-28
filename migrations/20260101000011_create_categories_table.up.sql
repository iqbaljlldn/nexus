CREATE TABLE categories (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    position      INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_categories_workspace_id ON categories (workspace_id);