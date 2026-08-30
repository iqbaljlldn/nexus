package application_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	memberInfra "github.com/iqbaljlldn/nexus/apps/api/internal/member/infrastructure"
	roleInfra "github.com/iqbaljlldn/nexus/apps/api/internal/role/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpInfra "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/infrastructure"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap/zaptest"
)

func TestWorkspaceService_Create_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Clean up tables in correct order (respecting FK constraints)
	_, _ = db.Exec("TRUNCATE TABLE member_role_assignments CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE roles CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE members CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE workspaces CASCADE")

	// Create a test user to be the workspace owner
	ownerID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		ownerID, "owner@test.com", "owneruser", "Owner", "hash_placeholder")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	log := zaptest.NewLogger(t)

	wsRepo := wpInfra.NewPostgresWorkspaceRepository(db)
	memberRepo := memberInfra.NewPostgresMemberRepository(db)
	roleRepo := roleInfra.NewPostgresRoleRepository(db)
	txManager := wpInfra.NewPostgresTransactionManager(db)

	svc := application.NewWorkspaceService(wsRepo, memberRepo, roleRepo, txManager, log)

	ctx := context.Background()

	t.Run("create workspace with auto-owner and @everyone role", func(t *testing.T) {
		ws, err := svc.Create(ctx, ownerID, "Integration Test Workspace", "https://example.com/icon.png")
		if err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		if ws.ID == uuid.Nil {
			t.Fatal("Workspace ID should not be nil")
		}
		if ws.Name != "Integration Test Workspace" {
			t.Errorf("Expected name 'Integration Test Workspace', got '%s'", ws.Name)
		}
		if ws.OwnerID != ownerID {
			t.Errorf("Expected owner ID %v, got %v", ownerID, ws.OwnerID)
		}
		if ws.IconURL != "https://example.com/icon.png" {
			t.Errorf("Expected icon URL 'https://example.com/icon.png', got '%s'", ws.IconURL)
		}

		// Verify member was created
		var memberID uuid.UUID
		var memberUserID uuid.UUID
		err = db.QueryRowContext(ctx,
			"SELECT id, user_id FROM members WHERE workspace_id = $1 AND user_id = $2",
			ws.ID, ownerID).Scan(&memberID, &memberUserID)
		if err != nil {
			t.Fatalf("Member not found after workspace creation: %v", err)
		}
		if memberUserID != ownerID {
			t.Errorf("Member user_id mismatch: expected %v, got %v", ownerID, memberUserID)
		}

		// Verify @everyone role was created
		var roleID uuid.UUID
		var roleName string
		var isEveryone bool
		var permBitmask int64
		err = db.QueryRowContext(ctx,
			"SELECT id, name, is_everyone, permission_bitmask FROM roles WHERE workspace_id = $1 AND is_everyone = true",
			ws.ID).Scan(&roleID, &roleName, &isEveryone, &permBitmask)
		if err != nil {
			t.Fatalf("@everyone role not found after workspace creation: %v", err)
		}
		if roleName != "@everyone" {
			t.Errorf("Expected role name '@everyone', got '%s'", roleName)
		}
		if !isEveryone {
			t.Error("Expected is_everyone to be true")
		}
		if permBitmask != 1 { // SEND_MESSAGES = 1 << 0 = 1
			t.Errorf("Expected permission bitmask 1 (SEND_MESSAGES), got %d", permBitmask)
		}

		// Verify role assignment exists
		var assignmentCount int
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM member_role_assignments WHERE member_id = $1 AND role_id = $2",
			memberID, roleID).Scan(&assignmentCount)
		if err != nil {
			t.Fatalf("Failed to query role assignments: %v", err)
		}
		if assignmentCount != 1 {
			t.Errorf("Expected 1 role assignment, got %d", assignmentCount)
		}
	})

	t.Run("rollback on invalid owner_id (FK violation)", func(t *testing.T) {
		fakeOwnerID := uuid.New() // non-existent user

		ws, err := svc.Create(ctx, fakeOwnerID, "Should Rollback", "")
		if err == nil {
			t.Fatal("Expected error for non-existent owner_id, got nil")
		}
		if ws != nil {
			t.Error("Expected nil workspace on error")
		}

		// Verify nothing was committed
		var count int
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM workspaces WHERE name = 'Should Rollback'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query workspaces: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 workspaces with name 'Should Rollback' (rollback should have removed it), got %d", count)
		}
	})
}
