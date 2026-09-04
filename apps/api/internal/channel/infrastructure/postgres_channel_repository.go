package infrastructure

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
)

type PostgresChannelRepository struct {
	q *Queries
}

func NewPostgresChannelRepository(db DBTX) *PostgresChannelRepository {
	return &PostgresChannelRepository{
		q: New(db),
	}
}

func (r *PostgresChannelRepository) Create(ctx context.Context, ch *domain.Channel) error {
	var workspaceID uuid.NullUUID
	if ch.WorkspaceID != nil {
		workspaceID = uuid.NullUUID{UUID: *ch.WorkspaceID, Valid: true}
	}
	var categoryID uuid.NullUUID
	if ch.CategoryID != nil {
		categoryID = uuid.NullUUID{UUID: *ch.CategoryID, Valid: true}
	}
	var name sql.NullString
	if ch.Name != nil {
		name = sql.NullString{String: *ch.Name, Valid: true}
	}

	row, err := r.q.CreateChannel(ctx, CreateChannelParams{
		WorkspaceID: workspaceID,
		CategoryID:  categoryID,
		Type:        string(ch.Type),
		Name:        name,
		Position:    ch.Position,
	})
	if err != nil {
		return err
	}

	ch.ID = row.ID
	ch.CreatedAt = row.CreatedAt
	return nil
}

func (r *PostgresChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	row, err := r.q.GetChannelByID(ctx, id)
	if err != nil {
		if err.Error() == "no rows in result set" { // pgx.ErrNoRows or sql.ErrNoRows
			return nil, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeRecordNotFound,
				Message: "Channel tidak ditemukan.",
				Err:     err,
			}
		}
		return nil, err
	}

	return mapChannelRow(row), nil
}

func (r *PostgresChannelRepository) ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]domain.Channel, error) {
	rows, err := r.q.ListChannelsByWorkspaceID(ctx, uuid.NullUUID{UUID: workspaceID, Valid: true})
	if err != nil {
		return nil, err
	}

	var channels []domain.Channel
	for _, row := range rows {
		channels = append(channels, *mapChannelRow(row))
	}
	return channels, nil
}

func (r *PostgresChannelRepository) GetMaxPosition(ctx context.Context, workspaceID uuid.UUID) (int32, error) {
	pos, err := r.q.GetMaxChannelPosition(ctx, uuid.NullUUID{UUID: workspaceID, Valid: true})
	if err != nil {
		return 0, err
	}
	return pos, nil
}

func (r *PostgresChannelRepository) CreatePermissionOverride(ctx context.Context, override *domain.ChannelPermissionOverride) error {
	var roleID uuid.NullUUID
	if override.RoleID != nil {
		roleID = uuid.NullUUID{UUID: *override.RoleID, Valid: true}
	}
	var memberID uuid.NullUUID
	if override.MemberID != nil {
		memberID = uuid.NullUUID{UUID: *override.MemberID, Valid: true}
	}

	row, err := r.q.CreateChannelPermissionOverride(ctx, CreateChannelPermissionOverrideParams{
		ChannelID:    override.ChannelID,
		RoleID:       roleID,
		MemberID:     memberID,
		AllowBitmask: override.AllowBitmask,
		DenyBitmask:  override.DenyBitmask,
	})
	if err != nil {
		return err
	}

	override.ID = row.ID
	return nil
}

func (r *PostgresChannelRepository) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]domain.ChannelPermissionOverride, error) {
	rows, err := r.q.GetChannelPermissionOverrides(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var overrides []domain.ChannelPermissionOverride
	for _, row := range rows {
		ov := domain.ChannelPermissionOverride{
			ID:           row.ID,
			ChannelID:    row.ChannelID,
			AllowBitmask: row.AllowBitmask,
			DenyBitmask:  row.DenyBitmask,
		}
		if row.RoleID.Valid {
			id := row.RoleID.UUID
			ov.RoleID = &id
		}
		if row.MemberID.Valid {
			id := row.MemberID.UUID
			ov.MemberID = &id
		}
		overrides = append(overrides, ov)
	}
	return overrides, nil
}

func (r *PostgresChannelRepository) UpdatePermissionOverride(ctx context.Context, id uuid.UUID, allowBitmask, denyBitmask int64) error {
	_, err := r.q.UpdateChannelPermissionOverride(ctx, UpdateChannelPermissionOverrideParams{
		ID:           id,
		AllowBitmask: allowBitmask,
		DenyBitmask:  denyBitmask,
	})
	return err
}

func (r *PostgresChannelRepository) DeletePermissionOverride(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteChannelPermissionOverride(ctx, id)
}

func (r *PostgresChannelRepository) GetChannelPermissionOverrideByRole(ctx context.Context, channelID, roleID uuid.UUID) (*domain.ChannelPermissionOverride, error) {
	row, err := r.q.GetChannelPermissionOverrideByRole(ctx, GetChannelPermissionOverrideByRoleParams{
		ChannelID: channelID,
		RoleID:    uuid.NullUUID{UUID: roleID, Valid: true},
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Return nil, nil when not found, common pattern for "upsert" checks
		}
		return nil, err
	}
	id := row.RoleID.UUID
	return &domain.ChannelPermissionOverride{
		ID:           row.ID,
		ChannelID:    row.ChannelID,
		RoleID:       &id,
		AllowBitmask: row.AllowBitmask,
		DenyBitmask:  row.DenyBitmask,
	}, nil
}

func (r *PostgresChannelRepository) GetChannelPermissionOverrideByMember(ctx context.Context, channelID, memberID uuid.UUID) (*domain.ChannelPermissionOverride, error) {
	row, err := r.q.GetChannelPermissionOverrideByMember(ctx, GetChannelPermissionOverrideByMemberParams{
		ChannelID: channelID,
		MemberID:  uuid.NullUUID{UUID: memberID, Valid: true},
	})
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // Return nil, nil when not found
		}
		return nil, err
	}
	id := row.MemberID.UUID
	return &domain.ChannelPermissionOverride{
		ID:           row.ID,
		ChannelID:    row.ChannelID,
		MemberID:     &id,
		AllowBitmask: row.AllowBitmask,
		DenyBitmask:  row.DenyBitmask,
	}, nil
}

func (r *PostgresChannelRepository) GetCategoryWorkspaceID(ctx context.Context, categoryID uuid.UUID) (uuid.UUID, error) {
	return r.q.GetCategoryWorkspaceID(ctx, categoryID)
}

func mapChannelRow(row interface{}) *domain.Channel {
	switch r := row.(type) {
	case Channel:
		ch := &domain.Channel{
			ID:        r.ID,
			Type:      domain.ChannelType(r.Type),
			Position:  r.Position,
			CreatedAt: r.CreatedAt,
		}
		if r.WorkspaceID.Valid {
			id := r.WorkspaceID.UUID
			ch.WorkspaceID = &id
		}
		if r.CategoryID.Valid {
			id := r.CategoryID.UUID
			ch.CategoryID = &id
		}
		if r.Name.Valid {
			n := r.Name.String
			ch.Name = &n
		}
		if r.ParticipantKey.Valid {
			pk := r.ParticipantKey.String
			ch.ParticipantKey = &pk
		}
		if r.DeletedAt.Valid {
			da := r.DeletedAt.Time
			ch.DeletedAt = &da
		}
		return ch
	}
	return nil
}
