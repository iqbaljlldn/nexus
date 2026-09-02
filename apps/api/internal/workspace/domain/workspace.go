package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidName       = errors.New("name must be 3-100 characters")
	ErrEmptyName         = errors.New("name cannot be empty")
	ErrDuplicateName     = errors.New("name already exists")
	ErrWorkspaceNotFound = errors.New("workspace not found")
)

type Workspace struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	IconURL   string    `json:"icon_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewWorkspace(name string) (*Workspace, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	if len(name) < 3 || len(name) > 100 {
		return nil, ErrInvalidName
	}

	return &Workspace{
		Name:      name,
		IconURL:   "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}
