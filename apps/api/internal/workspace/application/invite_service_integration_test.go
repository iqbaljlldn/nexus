package application_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	memberInfra "github.com/iqbaljlldn/nexus/apps/api/internal/member/infrastructure"
	roleInfra "github.com/iqbaljlldn/nexus/apps/api/internal/role/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpInfra "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestInviteService_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Clean up tables in correct order
	_, _ = db.Exec("TRUNCATE TABLE member_role_assignments CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE roles CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE members CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE invites CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE workspaces CASCADE")

	ownerID := uuid.New()
	joinerID := uuid.New()
	_, _ = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		ownerID, "owner_invite@test.com", "ownerinvite", "Owner Invite", "hash")
	_, _ = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		joinerID, "joiner_invite@test.com", "joinerinvite", "Joiner Invite", "hash")

	log := zaptest.NewLogger(t)

	wsRepo := wpInfra.NewPostgresWorkspaceRepository(db)
	inviteRepo := wpInfra.NewPostgresInviteRepository(db)
	memberRepo := memberInfra.NewPostgresMemberRepository(db)
	roleRepo := roleInfra.NewPostgresRoleRepository(db)
	txManager := wpInfra.NewPostgresTransactionManager(db)

	wsSvc := application.NewWorkspaceService(wsRepo, memberRepo, roleRepo, txManager, log)
	inviteSvc := application.NewInviteService(inviteRepo, memberRepo, roleRepo, txManager, log)

	ctx := context.Background()

	// Create workspace
	ws, err := wsSvc.Create(ctx, ownerID, "Invite Integration Workspace", "")
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	t.Run("redeem twice by same user is idempotent", func(t *testing.T) {
		createCtx := contextutil.WithUserID(ctx, ownerID)
		invite, err := inviteSvc.Create(createCtx, dto.CreateInviteReq{WorkspaceID: ws.ID})
		assert.NoError(t, err)

		// First redeem
		member1, err := inviteSvc.Redeem(ctx, invite.Code, joinerID)
		assert.NoError(t, err)
		assert.NotNil(t, member1)
		assert.Equal(t, ws.ID, member1.WorkspaceID)
		assert.Equal(t, joinerID, member1.UserID)

		// Second redeem by same user
		member2, err := inviteSvc.Redeem(ctx, invite.Code, joinerID)
		assert.NoError(t, err)
		assert.NotNil(t, member2)
		assert.Equal(t, member1.ID, member2.ID)

		// Verify DB member count for joiner in workspace is 1
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM members WHERE workspace_id = $1 AND user_id = $2", ws.ID, joinerID).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("redeem expired invite returns 422 BUSINESS_RULE_VIOLATION", func(t *testing.T) {
		past := time.Now().Add(-10 * time.Minute)
		createCtx := contextutil.WithUserID(ctx, ownerID)
		invite, err := inviteSvc.Create(createCtx, dto.CreateInviteReq{
			WorkspaceID: ws.ID,
			ExpiresAt:   &past,
		})
		assert.NoError(t, err)

		otherUser := uuid.New()
		_, _ = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
			otherUser, "other1@test.com", "otheruser1", "Other 1", "hash")

		member, err := inviteSvc.Redeem(ctx, invite.Code, otherUser)
		assert.Error(t, err)
		assert.Nil(t, member)

		var domainErr *pkgerrors.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)
	})

	t.Run("redeem max_uses reached invite returns 422 BUSINESS_RULE_VIOLATION", func(t *testing.T) {
		maxUses := 1
		createCtx := contextutil.WithUserID(ctx, ownerID)
		invite, err := inviteSvc.Create(createCtx, dto.CreateInviteReq{
			WorkspaceID: ws.ID,
			MaxUses:     &maxUses,
		})
		assert.NoError(t, err)

		userA := uuid.New()
		userB := uuid.New()
		_, _ = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
			userA, "usera@test.com", "user_a", "User A", "hash")
		_, _ = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
			userB, "userb@test.com", "user_b", "User B", "hash")

		// First redeem succeeds
		_, err = inviteSvc.Redeem(ctx, invite.Code, userA)
		assert.NoError(t, err)

		// Second redeem fails with max uses reached
		member, err := inviteSvc.Redeem(ctx, invite.Code, userB)
		assert.Error(t, err)
		assert.Nil(t, member)

		var domainErr *pkgerrors.DomainError
		assert.True(t, errors.As(err, &domainErr))
		assert.Equal(t, pkgerrors.CodeBusinessRuleViolation, domainErr.Code)
	})
}
