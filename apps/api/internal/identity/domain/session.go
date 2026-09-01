package domain

import (
	"time"
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

func NewSession(id, userID, refreshTokenHash string, deviceInfo DeviceInfo, duration time.Duration) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        deviceInfo.UserAgent,
		IPAddress:        deviceInfo.IPAddress,
		Status:           "active",
		CreatedAt:        now,
		ExpiresAt:        now.Add(duration),
	}
}

func (s *Session) IsValidForRefresh(deviceInfo DeviceInfo) error {
	if s.ExpiresAt.Before(time.Now()) {
		return ErrExpiredToken
	}

	if s.UserAgent == "" || s.IPAddress == "" {
		return ErrInvalidToken
	}

	// Strictly, we could also check if s.UserAgent == deviceInfo.UserAgent
	// and s.IPAddress == deviceInfo.IPAddress, but current logic just checks if session has them.
	// We'll keep the current leniency to match old logic, but it's now encapsulated here.
	return nil
}
