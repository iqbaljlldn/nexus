CREATE TABLE channel_permission_overrides (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    channel_id   UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    role_id      UUID REFERENCES roles(id) ON DELETE CASCADE,   -- salah satu dari role_id/member_id diisi (XOR, enforced service layer)
    member_id    UUID REFERENCES members(id) ON DELETE CASCADE,
    allow_bitmask BIGINT NOT NULL DEFAULT 0,
    deny_bitmask  BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT chk_override_target CHECK (
        (role_id IS NOT NULL AND member_id IS NULL) OR (role_id IS NULL AND member_id IS NOT NULL)
    )
);
CREATE INDEX idx_cpo_channel_role ON channel_permission_overrides (channel_id, role_id) WHERE role_id IS NOT NULL;
CREATE INDEX idx_cpo_channel_member ON channel_permission_overrides (channel_id, member_id) WHERE member_id IS NOT NULL;
