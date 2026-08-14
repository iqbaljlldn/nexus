package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	IsRevoked        bool
	CreatedAt        time.Time
	ExpiresAt        time.Time
}

func NewSession(userID, refreshTokenHash string, deviceInfo DeviceInfo, duration time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:               uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        deviceInfo.UserAgent,
		IPAddress:        deviceInfo.IPAddress,
		IsRevoked:        false,
		CreatedAt:        now,
		ExpiresAt:        now.Add(duration),
	}
}
