package domain

import (
	"context"
	"errors"
)

var (
	ErrDuplicateEmail     = errors.New("email already in use")
	ErrDuplicateUsername  = errors.New("username already in use")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, identifier string) (*User, error)
	FindByUsername(ctx context.Context, identifier string) (*User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string) error
}
