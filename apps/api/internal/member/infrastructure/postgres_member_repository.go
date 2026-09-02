package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
)

type PostgresMemberRepository struct {
	db *sql.DB
}

func NewPostgresMemberRepository(db *sql.DB) domain.MemberRepository {
	return &PostgresMemberRepository{db: db}
}

func (r *PostgresMemberRepository) Create(ctx context.Context, member *domain.Member) error {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	params := CreateMemberParams{
		WorkspaceID: member.WorkspaceID,
		UserID:      member.UserID,
		Nickname:    sql.NullString{String: member.Nickname, Valid: member.Nickname != ""},
	}

	dbMember, err := New(dbtx).CreateMember(ctx, params)
	if err != nil {
		return err
	}

	member.ID = dbMember.ID
	member.JoinedAt = dbMember.JoinedAt

	return nil
}

func (r *PostgresMemberRepository) GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*domain.Member, error) {
	dbtx := txcontext.ExtractDBTX(ctx, r.db)

	dbMember, err := New(dbtx).FindMemberByWorkspaceAndUser(ctx, FindMemberByWorkspaceAndUserParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMemberNotFound
		}
		return nil, err
	}

	var nickname string
	if dbMember.Nickname.Valid {
		nickname = dbMember.Nickname.String
	}

	return &domain.Member{
		ID:          dbMember.ID,
		WorkspaceID: dbMember.WorkspaceID,
		UserID:      dbMember.UserID,
		Nickname:    nickname,
		JoinedAt:    dbMember.JoinedAt,
	}, nil
}
