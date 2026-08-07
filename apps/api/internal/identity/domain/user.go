package domain

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEmail    = errors.New("invalid email address format")
	ErrInvalidUsername = errors.New("invalid username: must be 3-32 characters long and contain only alphanumeric characters and underscores")
	ErrInvalidHash     = errors.New("password hash cannot be empty")
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// Email represents a validated email address.
type Email string

// NewEmail validates and normalizes an email address.
func NewEmail(s string) (Email, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrInvalidEmail
	}

	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", ErrInvalidEmail
	}

	return Email(strings.ToLower(addr.Address)), nil
}

func (e Email) String() string {
	return string(e)
}

// Username represents a validated username.
type Username string

// NewUsername validates a username.
func NewUsername(s string) (Username, error) {
	if !usernameRegex.MatchString(s) {
		return "", ErrInvalidUsername
	}
	return Username(s), nil
}

func (u Username) String() string {
	return string(u)
}

// PasswordHash represents a validated password hash.
type PasswordHash string

func NewPasswordHash(s string) (PasswordHash, error) {
	if strings.TrimSpace(s) == "" {
		return "", ErrInvalidHash
	}
	return PasswordHash(s), nil
}

func (p PasswordHash) String() string {
	return string(p)
}

// User is the aggregate root for the Identity domain.
type User struct {
	ID           uuid.UUID
	Email        Email
	Username     Username
	DisplayName  string
	PasswordHash PasswordHash
	AvatarURL    *string
	IsSuspended  bool
	IsBanned     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// NewUser creates a new User entity.
func NewUser(email Email, username Username, displayName string, passwordHash PasswordHash) *User {
	now := time.Now().UTC()

	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}

	return &User{
		ID:           id,
		Email:        email,
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		IsSuspended:  false,
		IsBanned:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Suspend marks the user as suspended.
func (u *User) Suspend() {
	u.IsSuspended = true
	u.UpdatedAt = time.Now().UTC()
}

// Ban marks the user as banned.
func (u *User) Ban() {
	u.IsBanned = true
	u.UpdatedAt = time.Now().UTC()
}
