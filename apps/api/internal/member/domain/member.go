package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrMemberNotFound = errors.New("member not found")

type Member struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Nickname    string
	JoinedAt    time.Time
	UpdatedAt   time.Time
}
