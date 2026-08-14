package infrastructure

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
)

type JWTTokenManager struct {
	secretKey string
	issuer    string
	audience  string
}

func NewJWTTokenManager(secretKey, issuer, audience string) domain.TokenManager {
	return &JWTTokenManager{
		secretKey: secretKey,
		issuer:    issuer,
		audience:  audience,
	}
}

func (m *JWTTokenManager) GenerateToken(userID, tokenType string, duration time.Duration, deviceInfo domain.DeviceInfo) (string, error) {
	now := time.Now()
	payload := &domain.Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	signedToken, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (m *JWTTokenManager) ParseToken(tokenStr, tokenType string) (*domain.Claims, error) {
	claims := &domain.Claims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) { return []byte(m.secretKey), nil },
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.TokenType != tokenType {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}
