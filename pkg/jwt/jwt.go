package jwt

import (
	"errors"
	"os"

	jwt_lib "github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token is expired")
)

type BaseClaims struct {
	UserID string `json:"uid"`
	jwt_lib.RegisteredClaims
}

func getSecret() []byte {
	secret := os.Getenv("NEXUS_API_JWT_SECRET")
	if secret == "" {
		secret = "default_secret_for_local_dev"
	}
	return []byte(secret)
}

// Sign creates a new JWT string from claims
func Sign(claims jwt_lib.Claims) (string, error) {
	token := jwt_lib.NewWithClaims(jwt_lib.SigningMethodHS256, claims)
	return token.SignedString(getSecret())
}

// Verify parses a JWT string and validates its signature and expiration
func Verify(tokenStr string, claims jwt_lib.Claims) error {
	token, err := jwt_lib.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt_lib.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt_lib.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return getSecret(), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt_lib.ErrTokenExpired) {
			return ErrExpiredToken
		}
		return err
	}

	if !token.Valid {
		return ErrInvalidToken
	}

	return nil
}
