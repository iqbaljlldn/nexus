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
	Status           string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

func NewSession(userID, refreshTokenHash string, deviceInfo DeviceInfo, duration time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:               uuid.New().String(),
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        deviceInfo.UserAgent,
		IPAddress:        deviceInfo.IPAddress,
		Status:           "active",
		CreatedAt:        now,
		ExpiresAt:        now.Add(duration),
	}
}
