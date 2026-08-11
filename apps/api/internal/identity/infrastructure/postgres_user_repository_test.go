package infrastructure_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/infrastructure"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresUserRepository_CreateAndFind(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Clean up table before testing
	_, _ = db.Exec("TRUNCATE TABLE users CASCADE")

	querier := infrastructure.New(db)
	repo := infrastructure.NewPostgresUserRepository(querier)

	ctx := context.Background()

	email, _ := domain.NewEmail("repo@example.com")
	username, _ := domain.NewUsername("repouser")
	passwordHash, _ := domain.NewPasswordHash("some_hash")

	user := domain.NewUser(email, username, "Repo User", passwordHash)

	// 1. Test Create User
	err = repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// 2. Test Duplicate Email
	userDupEmail := domain.NewUser(email, "differentuser", "Repo User 2", passwordHash)
	err = repo.Create(ctx, userDupEmail)
	if err != domain.ErrDuplicateEmail {
		t.Errorf("Expected ErrDuplicateEmail, got %v", err)
	}

	// 3. Test Duplicate Username
	diffEmail, _ := domain.NewEmail("different@example.com")
	userDupUsername := domain.NewUser(diffEmail, username, "Repo User 3", passwordHash)
	err = repo.Create(ctx, userDupUsername)
	if err != domain.ErrDuplicateUsername {
		t.Errorf("Expected ErrDuplicateUsername, got %v", err)
	}

	// 4. Test Find By Email
	foundByEmail, err := repo.FindByEmailOrUsername(ctx, "repo@example.com")
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}
	if foundByEmail.ID != user.ID {
		t.Errorf("Expected found user ID %v, got %v", user.ID, foundByEmail.ID)
	}

	// 5. Test Find By Username
	foundByUsername, err := repo.FindByEmailOrUsername(ctx, "repouser")
	if err != nil {
		t.Fatalf("Failed to find user by username: %v", err)
	}
	if foundByUsername.ID != user.ID {
		t.Errorf("Expected found user ID %v, got %v", user.ID, foundByUsername.ID)
	}

	// 6. Test Find Non-Existent User
	_, err = repo.FindByEmailOrUsername(ctx, "nonexistent@example.com")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
