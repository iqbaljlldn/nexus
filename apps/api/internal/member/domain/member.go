package domain

import (
	"time"

	"github.com/google/uuid"
)

type Member struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Nickname    string
	JoinedAt    time.Time
	UpdatedAt   time.Time
}
