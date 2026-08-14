package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"acces_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	UserID    string `json:"uid"`
	TokenType string `json:"typ"`

	jwt.RegisteredClaims
}

type TokenManager interface {
	GenerateToken(userID, tokenType string, duration time.Duration, deviceInfo DeviceInfo) (string, error)
	ParseToken(token, tokenType string) (*Claims, error)
}
