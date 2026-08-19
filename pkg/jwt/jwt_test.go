package jwt_test

import (
	"testing"
	"time"

	jwt_lib "github.com/golang-jwt/jwt/v5"
	"github.com/iqbaljlldn/nexus/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func TestJWT(t *testing.T) {
	// Set test environment variable
	t.Setenv("NEXUS_API_JWT_SECRET", "test_secret_key")

	t.Run("round-trip success", func(t *testing.T) {
		claims := jwt.BaseClaims{
			UserID: "user123",
			RegisteredClaims: jwt_lib.RegisteredClaims{
				IssuedAt:  jwt_lib.NewNumericDate(time.Now()),
				ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}

		// Sign
		tokenStr, err := jwt.Sign(claims)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenStr)

		// Verify
		verifiedClaims := &jwt.BaseClaims{}
		err = jwt.Verify(tokenStr, verifiedClaims)
		assert.NoError(t, err)
		assert.Equal(t, "user123", verifiedClaims.UserID)
	})

	t.Run("token expired", func(t *testing.T) {
		claims := jwt.BaseClaims{
			UserID: "user123",
			RegisteredClaims: jwt_lib.RegisteredClaims{
				IssuedAt:  jwt_lib.NewNumericDate(time.Now().Add(-30 * time.Minute)),
				ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(-15 * time.Minute)), // Expired 15 mins ago
			},
		}

		tokenStr, err := jwt.Sign(claims)
		assert.NoError(t, err)

		// Verify should fail
		verifiedClaims := &jwt.BaseClaims{}
		err = jwt.Verify(tokenStr, verifiedClaims)
		assert.ErrorIs(t, err, jwt.ErrExpiredToken)
	})

	t.Run("invalid signature", func(t *testing.T) {
		claims := jwt.BaseClaims{
			UserID: "user123",
			RegisteredClaims: jwt_lib.RegisteredClaims{
				ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}

		// Sign with a different secret directly
		token := jwt_lib.NewWithClaims(jwt_lib.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString([]byte("wrong_secret_key"))
		assert.NoError(t, err)

		// Verify should fail
		verifiedClaims := &jwt.BaseClaims{}
		err = jwt.Verify(tokenStr, verifiedClaims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature is invalid")
	})
}
