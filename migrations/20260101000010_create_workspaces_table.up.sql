CREATE TABLE workspaces (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_id    UUID NOT NULL REFERENCES users(id),
    name        VARCHAR(100) NOT NULL,
    icon_url    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
