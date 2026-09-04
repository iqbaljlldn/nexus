CREATE TABLE channels (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    workspace_id  UUID REFERENCES workspaces(id) ON DELETE CASCADE, -- NULL untuk tipe 'dm'
    category_id   UUID REFERENCES categories(id) ON DELETE SET NULL,
    type          VARCHAR(16) NOT NULL CHECK (type IN ('text','voice','video','forum','announcement','dm')),
    name          VARCHAR(100),           -- NULL diperbolehkan untuk channel dm (tidak butuh nama)
    participant_key CHAR(64),             -- hash SHA-256 partisipan ter-sort, HANYA untuk type='dm'
    position      INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT chk_workspace_scoped_or_dm CHECK (
        (type = 'dm' AND workspace_id IS NULL) OR (type != 'dm' AND workspace_id IS NOT NULL)
    )
);
CREATE INDEX idx_channels_workspace_id ON channels (workspace_id) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX uidx_channels_dm_participant_key ON channels (participant_key) WHERE type = 'dm' AND participant_key IS NOT NULL;
