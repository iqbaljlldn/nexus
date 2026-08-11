package domain

import (
	"context"
	"errors"
)

var (
	ErrDuplicateEmail    = errors.New("email already in use")
	ErrDuplicateUsername = errors.New("username already in use")
	ErrUserNotFound      = errors.New("user not found")
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmailOrUsername(ctx context.Context, identifier string) (*User, error)
}
