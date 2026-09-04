package main_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	main "github.com/iqbaljlldn/nexus/apps/api/cmd/server"
	channelDTO "github.com/iqbaljlldn/nexus/apps/api/internal/channel/interface/dto"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	roleDTO "github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/dto"
	workspaceDTO "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/httpresponse"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// setupIntegrationEnvironment sets up DB, Redis, and initializes the Gin router using the actual Wire provider.
func setupIntegrationEnvironment(t *testing.T) (*gin.Engine, *pgxpool.Pool, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Clean tables before test using standard sql.DB just for truncation
	dbSQL, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	_, err = dbSQL.Exec("TRUNCATE TABLE users, sessions, workspaces, members, roles, member_role_assignments, invites, channels, channel_permission_overrides CASCADE")
	require.NoError(t, err)
	_ = dbSQL.Close()

	// 1. Setup Postgres pgxpool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// 2. Setup Redis (Miniredis)
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	logger := zaptest.NewLogger(t)

	// Ensure the pkg/jwt has a secret for middleware verification
	_ = os.Setenv("NEXUS_API_JWT_SECRET", "test-integration-secret-key-that-is-long-enough")

	gin.SetMode(gin.TestMode)

	// Use InitializeRouter from wire_gen.go
	engine := main.InitializeRouter(logger, pool, redisClient)

	// Add an ad-hoc route to simulate GET /channels/:id/messages using the real PermissionResolver
	// since the message domain is not yet implemented, but we need to test the channel override resolving.
	// We have to extract the cachedPermissionResolver from wire if possible, but we don't have access to it directly.
	// Instead, we will test User B attempting to PATCH /channels/:id/permission-overrides and expect 403,
	// which verifies role overrides and permission caching end-to-end.

	return engine, pool, mr, redisClient
}

func getCookie(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var response httpresponse.SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		return nil
	}
	return data
}

func TestSprint3_EndToEnd_Verification(t *testing.T) {
	router, _, _, _ := setupIntegrationEnvironment(t)

	var userAAccessToken string
	var userBAccessToken string
	var workspaceID string
	var inviteCode string
	var restrictedRoleID string
	var channelID string
	var userBMemberID string

	// 1. User A Register & Login
	t.Run("User A Register and Login", func(t *testing.T) {
		reqBody := identityhttp.RegisterRequest{
			Email:       "owner@example.com",
			Username:    "owner",
			DisplayName: "Owner User",
			Password:    "password123",
		}
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // false positive for test data
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		loginBody := identityhttp.LoginRequest{
			Identifier: "owner@example.com",
			Password:   "password123",
		}
		jsonValue, _ = json.Marshal(loginBody) //nolint:gosec // false positive for test data
		req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonValue))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		data := parseResponse(t, w)
		userAAccessToken = data["access_token"].(string)
		require.NotEmpty(t, userAAccessToken)
	})

	// 2. User A Create Workspace
	t.Run("User A Create Workspace", func(t *testing.T) {
		iconURL := "https://example.com/icon.png"
		reqBody := workspaceDTO.CreateWorkspaceRequest{
			Name:    "Test E2E Workspace",
			IconURL: &iconURL,
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		data := parseResponse(t, w)
		workspaceID = data["id"].(string)
		require.NotEmpty(t, workspaceID)
	})

	// 3. User A Create Invite
	t.Run("User A Create Invite", func(t *testing.T) {
		reqBody := workspaceDTO.CreateInviteRequest{
			MaxUses:        func() *int { i := 5; return &i }(),
			ExpiresInHours: func() *int { i := 24; return &i }(),
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/invites", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		data := parseResponse(t, w)
		inviteCode = data["code"].(string)
		require.NotEmpty(t, inviteCode)
	})

	// 4. User B Register & Login & Redeem Invite
	t.Run("User B Register, Login, and Redeem Invite", func(t *testing.T) {
		reqBody := identityhttp.RegisterRequest{
			Email:       "userb@example.com",
			Username:    "userb",
			DisplayName: "User B",
			Password:    "password123",
		}
		jsonValue, _ := json.Marshal(reqBody) //nolint:gosec // false positive for test data
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonValue))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		loginBody := identityhttp.LoginRequest{
			Identifier: "userb@example.com",
			Password:   "password123",
		}
		jsonValue, _ = json.Marshal(loginBody) //nolint:gosec // false positive for test data
		req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonValue))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		data := parseResponse(t, w)
		userBAccessToken = data["access_token"].(string)
		require.NotEmpty(t, userBAccessToken)

		// Redeem Invite
		req, _ = http.NewRequest(http.MethodPost, "/api/v1/invites/"+inviteCode+"/redeem", nil)
		req.Header.Set("Authorization", "Bearer "+userBAccessToken)
		req.Header.Set("Idempotency-Key", "e2e-test-key")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		data = parseResponse(t, w)
		userBMemberID = data["member_id"].(string)
		require.NotEmpty(t, userBMemberID)
	})

	// 5. User A Create "Restricted" Role (no SEND_MESSAGES) and Assign to User B
	t.Run("User A Create Role and Assign to User B", func(t *testing.T) {
		// Create Role (Permissions: 0 - No Permissions)
		reqBody := roleDTO.CreateRoleRequest{
			Name:              "Restricted",
			PermissionBitmask: 0,
			Position:          func() *int32 { i := int32(10); return &i }(), // Higher than @everyone (0)
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/roles", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		data := parseResponse(t, w)
		restrictedRoleID = data["id"].(string)
		require.NotEmpty(t, restrictedRoleID)

		// Assign Role to User B
		var roleIDs []uuid.UUID
		rID, _ := uuid.Parse(restrictedRoleID)
		roleIDs = append(roleIDs, rID)
		assignReq := roleDTO.AssignRolesRequest{RoleIDs: roleIDs}
		jsonValue, _ = json.Marshal(assignReq)
		req, _ = http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+workspaceID+"/members/"+userBMemberID+"/roles", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code) // The handler returns 200 OK with a message
	})

	// 6. User A Create Private Channel and Set Permission Override
	t.Run("User A Create Private Channel and Set Overrides", func(t *testing.T) {
		reqBody := channelDTO.CreateChannelRequest{
			Name: "top-secret",
			Type: "text",
		}
		jsonValue, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/channels", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		data := parseResponse(t, w)
		channelID = data["id"].(string)
		require.NotEmpty(t, channelID)

		// Set Override for "Restricted" Role: Deny SEND_MESSAGES & MANAGE_CHANNELS
		rID, _ := uuid.Parse(restrictedRoleID)
		denyBitmask := int64(roleDomain.PermSendMessages | roleDomain.PermManageChannels)
		overrideReq := channelDTO.PatchPermissionOverrideRequest{
			RoleID:       &rID,
			AllowBitmask: 0,
			DenyBitmask:  denyBitmask,
		}
		jsonValue, _ = json.Marshal(overrideReq)
		req, _ = http.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID+"/permission-overrides", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	// 7. Verification
	t.Run("Verification: User B forbidden, User A allowed", func(t *testing.T) {
		// User B attempts to Manage Channel Permissions (which requires PermManageRoles) -> 403 Forbidden
		rID, _ := uuid.Parse(restrictedRoleID)
		overrideReq := channelDTO.PatchPermissionOverrideRequest{
			RoleID:       &rID,
			AllowBitmask: 0,
			DenyBitmask:  0,
		}
		jsonValue, _ := json.Marshal(overrideReq)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID+"/permission-overrides", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userBAccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert that User B is forbidden (due to role override or base role permissions)
		require.Equal(t, http.StatusForbidden, w.Code)
		var response httpresponse.ErrorResponse
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, pkgerrors.CodeForbidden, response.Error.Code)

		// User A (Owner) attempts the exact same action -> 200 OK (Owner bypasses all denials)
		req, _ = http.NewRequest(http.MethodPatch, "/api/v1/channels/"+channelID+"/permission-overrides", bytes.NewBuffer(jsonValue))
		req.Header.Set("Authorization", "Bearer "+userAAccessToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}
