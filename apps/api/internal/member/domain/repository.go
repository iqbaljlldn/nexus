package domain

import (
	"context"

	"github.com/google/uuid"
)

type MemberRepository interface {
	// Create persists a new member. The DB-generated ID is set on the
	// member pointer upon success.
	Create(ctx context.Context, member *Member) error
	GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*Member, error)
	ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID, limit int32) ([]Member, error)
}
