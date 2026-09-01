package infrastructure

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	jwtutil "github.com/iqbaljlldn/nexus/pkg/jwt"
)

type JWTTokenManager struct {
	issuer   string
	audience string
}

func NewJWTTokenManager(issuer, audience string) domain.TokenManager {
	return &JWTTokenManager{
		issuer:   issuer,
		audience: audience,
	}
}

func (m *JWTTokenManager) GenerateToken(userID, sessionID, tokenType string, duration time.Duration, deviceInfo domain.DeviceInfo) (string, error) {
	now := time.Now()
	payload := &domain.Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sessionID,
			Subject:   userID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	return jwtutil.Sign(payload)
}

func (m *JWTTokenManager) ParseToken(tokenStr, tokenType string) (*domain.Claims, error) {
	claims := &domain.Claims{}

	err := jwtutil.Verify(tokenStr, claims)
	if err != nil {
		if errors.Is(err, jwtutil.ErrExpiredToken) {
			return nil, domain.ErrInvalidToken // Or some specific expired error
		}
		return nil, domain.ErrInvalidToken
	}

	if claims.TokenType != tokenType || claims.Issuer != m.issuer || len(claims.Audience) == 0 || claims.Audience[0] != m.audience {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}
