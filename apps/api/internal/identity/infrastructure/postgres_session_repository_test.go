package infrastructure_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
)

func TestPostgresSessionRepository_RotateRefreshToken(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Clean up tables
	_, _ = db.Exec("TRUNCATE TABLE sessions CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE users CASCADE")

	ctx := context.Background()

	// Need a user to satisfy foreign key constraint on sessions
	q := infrastructure.New(db)
	userRepo := infrastructure.NewPostgresUserRepository(q)

	email, _ := domain.NewEmail("sessionuser@example.com")
	username, _ := domain.NewUsername("sessionuser")
	passwordHash, _ := domain.NewPasswordHash("hash")
	user := domain.NewUser(email, username, "Session User", passwordHash)

	err = userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	repo := infrastructure.NewPostgresSessionRepository(db)

	oldTokenHash := "old_hash_123"
	newTokenHash := "new_hash_456"

	session := &domain.Session{
		ID:               uuid.New().String(),
		UserID:           user.ID.String(),
		RefreshTokenHash: oldTokenHash,
		UserAgent:        "TestAgent",
		IPAddress:        "127.0.0.1",
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}

	err = repo.Create(ctx, session)
	assert.NoError(t, err)
	assert.NotEmpty(t, session.ID)

	// Verify session exists
	dbSession, err := q.FindSessionByTokenHash(ctx, oldTokenHash)
	assert.NoError(t, err)
	assert.Equal(t, "active", dbSession.Status)

	// 2. Rotate the refresh token
	newSessionID := uuid.New().String()
	err = repo.RotateRefreshToken(ctx, session.ID, newSessionID, newTokenHash)
	assert.NoError(t, err)

	// 3. Verify old token is revoked
	_, err = q.FindSessionByTokenHash(ctx, oldTokenHash)
	assert.Error(t, err) // Should error because query checks `deleted_at IS NULL` AND wait, Revoke sets deleted_at?

	// Let's check status manually in DB
	var oldStatus string
	var deletedAt sql.NullTime
	err = db.QueryRow("SELECT status, deleted_at FROM sessions WHERE refresh_token_hash = $1", oldTokenHash).Scan(&oldStatus, &deletedAt)
	assert.NoError(t, err)
	assert.Equal(t, "revoked", oldStatus)
	assert.True(t, deletedAt.Valid)

	// 4. Verify new token exists and is active
	newSession, err := q.FindSessionByTokenHash(ctx, newTokenHash)
	assert.NoError(t, err)
	assert.Equal(t, "active", newSession.Status)
	assert.Equal(t, user.ID.String(), newSession.UserID.String())
	assert.Equal(t, "TestAgent", newSession.UserAgent.String)
}
