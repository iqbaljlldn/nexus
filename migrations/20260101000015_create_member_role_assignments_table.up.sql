CREATE TABLE member_role_assignments (
    member_id  UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (member_id, role_id)
);
CREATE INDEX idx_mra_role_id ON member_role_assignments (role_id);