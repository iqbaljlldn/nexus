package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInviteNotFound       = errors.New("invite not found")
	ErrInviteExpired        = errors.New("invite has expired")
	ErrInviteMaxUsesReached = errors.New("invite max uses reached")
)

type Invite struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Code        string
	CreatedBy   uuid.UUID
	MaxUses     *int
	UseCount    int
	ExpiresAt   *time.Time
	CreatedAt   time.Time
}

func NewInvite(workspaceID, createdBy uuid.UUID, code string, maxUses *int, expiresAt *time.Time) *Invite {
	return &Invite{
		WorkspaceID: workspaceID,
		Code:        code,
		CreatedBy:   createdBy,
		MaxUses:     maxUses,
		ExpiresAt:   expiresAt,
		UseCount:    0,
	}
}

func (i *Invite) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}

	return time.Now().After(*i.ExpiresAt)
}

func (i *Invite) IsMaxUsesReached() bool {
	if i.MaxUses == nil {
		return false
	}

	return i.UseCount >= *i.MaxUses
}
