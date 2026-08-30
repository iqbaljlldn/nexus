package application_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	memberInfra "github.com/iqbaljlldn/nexus/apps/api/internal/member/infrastructure"
	roleInfra "github.com/iqbaljlldn/nexus/apps/api/internal/role/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpInfra "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap/zaptest"
)

func TestWorkspaceService_ListByUserID_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, _ = db.Exec("TRUNCATE TABLE member_role_assignments CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE roles CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE members CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE workspaces CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE users CASCADE")

	// Create a test user
	userID := uuid.New()
	_, err = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		userID, "listowner@test.com", "listowner", "List Owner", "hash")
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

	// Seed data (create 3 workspaces)
	names := []string{"Alpha", "Gamma", "Beta"} // not alphabetical so we can test sorting
	for _, name := range names {
		_, err := svc.Create(ctx, userID, name, "")
		if err != nil {
			t.Fatalf("Failed to seed workspace: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // ensure different created_at
	}

	t.Run("list default newest", func(t *testing.T) {
		req := &dto.ListWorkspacesRequest{Limit: 10}
		res, meta, err := svc.ListByUserID(ctx, userID, req)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(res.Workspaces) != 3 {
			t.Fatalf("Expected 3 workspaces, got %d", len(res.Workspaces))
		}
		if meta.Total != 3 {
			t.Fatalf("Expected total 3, got %d", meta.Total)
		}
		// newest first, so Beta should be index 0
		if res.Workspaces[0].Name != "Beta" {
			t.Errorf("Expected newest 'Beta' first, got '%s'", res.Workspaces[0].Name)
		}
	})

	t.Run("list search and name_asc", func(t *testing.T) {
		req := &dto.ListWorkspacesRequest{
			Limit:    10,
			SortMode: "name_asc",
			Search:   "a", // Alpha, Gamma, Beta all have 'a'
		}
		res, _, err := svc.ListByUserID(ctx, userID, req)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(res.Workspaces) != 3 {
			t.Fatalf("Expected 3 workspaces, got %d", len(res.Workspaces))
		}
		// alphabetical order
		if res.Workspaces[0].Name != "Alpha" || res.Workspaces[1].Name != "Beta" || res.Workspaces[2].Name != "Gamma" {
			t.Errorf("Expected Alpha, Beta, Gamma order. Got: %s, %s, %s",
				res.Workspaces[0].Name, res.Workspaces[1].Name, res.Workspaces[2].Name)
		}
	})

	t.Run("list pagination limit and cursor", func(t *testing.T) {
		// page 1
		req := &dto.ListWorkspacesRequest{
			Limit:    2,
			SortMode: "name_asc",
		}
		res1, meta1, err := svc.ListByUserID(ctx, userID, req)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if len(res1.Workspaces) != 2 {
			t.Fatalf("Expected 2 workspaces, got %d", len(res1.Workspaces))
		}
		if meta1.HasMore == false || meta1.Cursor == nil {
			t.Fatalf("Expected has_more=true and next_cursor")
		}

		// page 2
		req2 := &dto.ListWorkspacesRequest{
			Limit:    2,
			SortMode: "name_asc",
			Cursor:   *meta1.Cursor,
		}
		res2, meta2, err := svc.ListByUserID(ctx, userID, req2)
		if err != nil {
			t.Fatalf("Failed to list page 2: %v", err)
		}
		if len(res2.Workspaces) != 1 {
			t.Fatalf("Expected 1 workspace, got %d", len(res2.Workspaces))
		}
		if res2.Workspaces[0].Name != "Gamma" {
			t.Errorf("Expected 'Gamma', got '%s'", res2.Workspaces[0].Name)
		}
		if meta2.HasMore == true || meta2.Cursor != nil {
			t.Fatalf("Expected has_more=false and no next_cursor")
		}
	})
}
