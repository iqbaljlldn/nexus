package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
)

type PostgresInviteRepository struct {
	db *sql.DB
}

func NewPostgresInviteRepository(db *sql.DB) domain.InviteRepository {
	return &PostgresInviteRepository{db: db}
}

func (r *PostgresInviteRepository) Create(ctx context.Context, invite *domain.Invite) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	var maxUses sql.NullInt32
	if invite.MaxUses != nil {
		val := *invite.MaxUses
		if val < 0 || val > math.MaxInt32 {
			return fmt.Errorf("max_uses out of range: %d", val)
		}
		maxUses = sql.NullInt32{Int32: int32(val), Valid: true}
	}

	var expiresAt sql.NullTime
	if invite.ExpiresAt != nil {
		expiresAt = sql.NullTime{Time: *invite.ExpiresAt, Valid: true}
	}

	params := CreateInviteParams{
		WorkspaceID: invite.WorkspaceID,
		Code:        invite.Code,
		CreatedBy:   invite.CreatedBy,
		MaxUses:     maxUses,
		ExpiresAt:   expiresAt,
	}

	dbInvite, err := New(dbtx).CreateInvite(ctx, params)
	if err != nil {
		return err
	}

	invite.ID = dbInvite.ID
	invite.UseCount = int(dbInvite.UseCount)
	invite.CreatedAt = dbInvite.CreatedAt
	return nil
}

func (r *PostgresInviteRepository) GetByCode(ctx context.Context, code string) (*domain.Invite, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	dbInvite, err := New(dbtx).FindInviteByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInviteNotFound
		}
		return nil, err
	}

	var maxUses *int
	if dbInvite.MaxUses.Valid {
		m := int(dbInvite.MaxUses.Int32)
		maxUses = &m
	}

	var expiresAt *time.Time
	if dbInvite.ExpiresAt.Valid {
		e := dbInvite.ExpiresAt.Time
		expiresAt = &e
	}

	return &domain.Invite{
		ID:          dbInvite.ID,
		WorkspaceID: dbInvite.WorkspaceID,
		Code:        dbInvite.Code,
		CreatedBy:   dbInvite.CreatedBy,
		MaxUses:     maxUses,
		UseCount:    int(dbInvite.UseCount),
		ExpiresAt:   expiresAt,
		CreatedAt:   dbInvite.CreatedAt,
	}, nil
}

func (r *PostgresInviteRepository) IncrementUseCount(ctx context.Context, inviteID uuid.UUID) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	_, err := New(dbtx).IncrementInviteUseCount(ctx, inviteID)
	return err
}
