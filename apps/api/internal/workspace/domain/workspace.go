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
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	IconURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
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
