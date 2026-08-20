package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrDuplicateEmail      = errors.New("email already in use")
	ErrDuplicateUsername   = errors.New("username already in use")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidToken        = errors.New("invalid token")
	ErrExpiredToken        = errors.New("expired token")
	ErrUnauthorizedSession = errors.New("unauthorized session")
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, identifier string) (*User, error)
	FindByUsername(ctx context.Context, identifier string) (*User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string) error
	FindByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)
	RevokeSession(ctx context.Context, refreshToken string) error
	RevokeAllSessions(ctx context.Context) error
	GetActiveSessions(ctx context.Context) ([]Session, error)
	RevokeSessionById(ctx context.Context, id uuid.UUID) (*Session, error)
	FindById(ctx context.Context, id uuid.UUID) (*Session, error)
}
