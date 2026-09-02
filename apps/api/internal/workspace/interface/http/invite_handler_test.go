package http_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwt_lib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	memberInfra "github.com/iqbaljlldn/nexus/apps/api/internal/member/infrastructure"
	roleInfra "github.com/iqbaljlldn/nexus/apps/api/internal/role/infrastructure"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpInfra "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/infrastructure"
	wsHttp "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/http"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	pkgjwt "github.com/iqbaljlldn/nexus/pkg/jwt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func generateTestToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	claims := pkgjwt.BaseClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt_lib.RegisteredClaims{
			ExpiresAt: jwt_lib.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token, err := pkgjwt.Sign(claims)
	require.NoError(t, err)
	return token
}

func setupInviteTestEnvironment(t *testing.T) (*gin.Engine, *sql.DB, string, string, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping HTTP integration test")
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Clean up tables
	_, _ = db.Exec("TRUNCATE TABLE member_role_assignments CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE roles CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE members CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE invites CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE workspaces CASCADE")
	_, _ = db.Exec("TRUNCATE TABLE users CASCADE")

	ownerID := uuid.New()
	joinerID := uuid.New()

	_, err = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		ownerID, "owner_http@test.com", "ownerhttp", "Owner HTTP", "hash")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO users (id, email, username, display_name, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		joinerID, "joiner_http@test.com", "joinerhttp", "Joiner HTTP", "hash")
	require.NoError(t, err)

	_ = os.Setenv("NEXUS_API_JWT_SECRET", "test-integration-secret-key-that-is-long-enough")

	ownerToken := generateTestToken(t, ownerID)
	joinerToken := generateTestToken(t, joinerID)

	logger := zaptest.NewLogger(t)

	wsRepo := wpInfra.NewPostgresWorkspaceRepository(db)
	inviteRepo := wpInfra.NewPostgresInviteRepository(db)
	memberRepo := memberInfra.NewPostgresMemberRepository(db)
	roleRepo := roleInfra.NewPostgresRoleRepository(db)
	txManager := wpInfra.NewPostgresTransactionManager(db)

	wsSvc := application.NewWorkspaceService(wsRepo, memberRepo, roleRepo, txManager, logger)
	inviteSvc := application.NewInviteService(inviteRepo, memberRepo, roleRepo, txManager, logger)

	handler := wsHttp.NewWorkspaceHandler(wsSvc, inviteSvc, "http://localhost:3000")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			httpresponse.Error(c, err)
		}
	})

	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	return r, db, ownerToken, joinerToken, joinerID
}

func TestInviteHandler_CreateAndRedeemFlow(t *testing.T) {
	router, db, ownerToken, joinerToken, joinerID := setupInviteTestEnvironment(t)

	var workspaceID string
	var inviteCode string

	t.Run("1. Create Workspace", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Invite HTTP Workspace",
			"icon_url": "https://example.com/icon.png",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		workspaceID = data["id"].(string)
		require.NotEmpty(t, workspaceID)
	})

	t.Run("2. Create Invite - Invalid Workspace ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/invalid-uuid/invites", nil)
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("3. Create Invite - Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"max_uses":         5,
			"expires_in_hours": 24,
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/invites", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		inviteCode = data["code"].(string)
		url := data["url"].(string)
		assert.NotEmpty(t, inviteCode)
		assert.Contains(t, url, inviteCode)
	})

	t.Run("4. Redeem Invite - Missing Idempotency Key -> 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/invites/"+inviteCode+"/redeem", nil)
		req.Header.Set("Authorization", "Bearer "+joinerToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errResp httpresponse.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "MISSING_REQUIRED_FIELD", errResp.Error.Code)
	})

	t.Run("5. Redeem Invite - Success with Idempotency Key -> 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/invites/"+inviteCode+"/redeem", nil)
		req.Header.Set("Authorization", "Bearer "+joinerToken)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp httpresponse.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, workspaceID, data["workspace_id"])
		assert.NotEmpty(t, data["member_id"])

		// Verify member in DB
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM members WHERE workspace_id = $1 AND user_id = $2", workspaceID, joinerID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("6. Redeem Invite - Non-Existent Code -> 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/invites/nonexistentcode/redeem", nil)
		req.Header.Set("Authorization", "Bearer "+joinerToken)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
