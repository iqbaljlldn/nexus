package infrastructure_test

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sqlc-dev/pqtype"
)

func TestQueries(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	querier := infrastructure.New(db)

	ctx := context.Background()

	// Test CreateUser
	user, err := querier.CreateUser(ctx, infrastructure.CreateUserParams{
		Email:        "test@example.com",
		Username:     "testuser",
		DisplayName:  "Test User",
		PasswordHash: "hashed_password",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}

	// Test FindUserByEmailOrUsername
	foundUser, err := querier.FindUserByEmailOrUsername(ctx, infrastructure.FindUserByEmailOrUsernameParams{
		Email:    "test@example.com",
		Username: "wronguser", // testing OR condition
	})
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	if foundUser.ID != user.ID {
		t.Errorf("Expected user ID %v, got %v", user.ID, foundUser.ID)
	}

	_, ipnet, _ := net.ParseCIDR("127.0.0.1/32")

	// Test CreateSession
	session, err := querier.CreateSession(ctx, infrastructure.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: "refresh_hash",
		UserAgent:        sql.NullString{String: "Mozilla", Valid: true},
		IpAddress:        pqtype.Inet{IPNet: *ipnet, Valid: true},
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != user.ID {
		t.Errorf("Expected session user ID %v, got %v", user.ID, session.UserID)
	}
}
