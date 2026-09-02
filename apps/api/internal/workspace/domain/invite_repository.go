package domain

import (
	"context"

	"github.com/google/uuid"
)

type InviteRepository interface {
	Create(ctx context.Context, invite *Invite) error
	GetByCode(ctx context.Context, code string) (*Invite, error)
	IncrementUseCount(ctx context.Context, inviteID uuid.UUID) error
}
