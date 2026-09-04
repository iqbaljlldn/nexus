package infrastructure

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
)

type PostgresChannelOverrideRepository struct {
	db *sql.DB
}

func NewPostgresChannelOverrideRepository(db *sql.DB) application.ChannelOverridePort {
	return &PostgresChannelOverrideRepository{db: db}
}

func (r *PostgresChannelOverrideRepository) FindMemberOverride(ctx context.Context, channelID, userID uuid.UUID) (*application.ChannelOverride, bool, error) {
	query := `
		SELECT o.allow_bitmask, o.deny_bitmask
		FROM channel_permission_overrides o
		JOIN members m ON o.member_id = m.id
		WHERE o.channel_id = $1 AND m.user_id = $2
	`
	var allow, deny int64
	err := r.db.QueryRowContext(ctx, query, channelID, userID).Scan(&allow, &deny)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &application.ChannelOverride{Allow: roleDomain.PermissionFlag(allow), Deny: roleDomain.PermissionFlag(deny)}, true, nil
}

func (r *PostgresChannelOverrideRepository) FindRoleOverride(ctx context.Context, channelID, roleID uuid.UUID) (*application.ChannelOverride, bool, error) {
	query := `
		SELECT allow_bitmask, deny_bitmask
		FROM channel_permission_overrides
		WHERE channel_id = $1 AND role_id = $2
	`
	var allow, deny int64
	err := r.db.QueryRowContext(ctx, query, channelID, roleID).Scan(&allow, &deny)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &application.ChannelOverride{Allow: roleDomain.PermissionFlag(allow), Deny: roleDomain.PermissionFlag(deny)}, true, nil
}
