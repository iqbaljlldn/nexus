package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func setupRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestCachedPermissionResolver_Resolve(t *testing.T) {
	mr, rdb := setupRedis(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	log := zaptest.NewLogger(t)
	mockOverride := new(MockChannelOverridePort)
	mockRole := new(MockRolePort2)
	mockWorkspace := new(MockPermResolverWorkspaceRepo)
	mockMember := new(MockPermResolverMemberPort)
	baseResolver := application.NewPermissionResolver(mockWorkspace, mockMember, mockOverride, mockRole)
	cachedResolver := application.NewCachedPermissionResolver(baseResolver, rdb, log)

	userID, workspaceID, channelID := uuid.New(), uuid.New(), uuid.New()
	memberID := uuid.New()
	reqFlag := roleDomain.PermSendMessages

	// Setup standard mock returns to avoid bypass
	mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil)
	mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil)

	// 1. Initial request - should call base resolver (cache miss)
	mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil).Once()
	mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{}, nil).Once()
	mockRole.On("GetEveryoneRole", mock.Anything, workspaceID).Return(&roleDomain.Role{
		PermissionBitmask: int64(reqFlag),
	}, nil).Once()

	allowed, err := cachedResolver.Resolve(context.Background(), userID, workspaceID, channelID, reqFlag)
	assert.NoError(t, err)
	assert.True(t, allowed)

	mockOverride.AssertExpectations(t)
	mockRole.AssertExpectations(t)
	mockWorkspace.AssertExpectations(t)
	mockMember.AssertExpectations(t)

	// 2. Second request - should NOT call base resolver (cache hit)
	allowedCached, err := cachedResolver.Resolve(context.Background(), userID, workspaceID, channelID, reqFlag)
	assert.NoError(t, err)
	assert.True(t, allowedCached)

	// 3. Fast-forward time to expire TTL
	mr.FastForward(61 * time.Second)

	// 4. Third request - should call base resolver again (cache expired)
	mockWorkspace.On("GetByID", mock.Anything, workspaceID).Return(&wpDomain.Workspace{OwnerID: uuid.New()}, nil).Once()
	mockMember.On("GetByWorkspaceAndUser", mock.Anything, workspaceID, userID).Return(&memberDomain.Member{ID: memberID}, nil).Once()
	mockOverride.On("FindMemberOverride", mock.Anything, channelID, memberID).Return(nil, false, nil).Once()
	mockRole.On("FindMemberRolesSortedByPosition", mock.Anything, memberID).Return([]*roleDomain.Role{}, nil).Once()
	mockRole.On("GetEveryoneRole", mock.Anything, workspaceID).Return(&roleDomain.Role{
		PermissionBitmask: int64(reqFlag),
	}, nil).Once()

	allowedAfterExpire, err := cachedResolver.Resolve(context.Background(), userID, workspaceID, channelID, reqFlag)
	assert.NoError(t, err)
	assert.True(t, allowedAfterExpire)

	mockOverride.AssertExpectations(t)
	mockRole.AssertExpectations(t)
}

func TestPermissionCacheInvalidator_InvalidateUserPermissions(t *testing.T) {
	mr, rdb := setupRedis(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	log := zaptest.NewLogger(t)
	invalidator := application.NewPermissionCacheInvalidator(rdb, log)

	workspaceID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	channelID := uuid.New()

	ctx := context.Background()

	// Seed some cache keys
	key1 := "perm:" + workspaceID.String() + ":" + userID1.String() + ":" + channelID.String() + ":1"
	key2 := "perm:" + workspaceID.String() + ":" + userID1.String() + ":" + channelID.String() + ":2"
	keyOtherUser := "perm:" + workspaceID.String() + ":" + userID2.String() + ":" + channelID.String() + ":1"
	keyOtherWorkspace := "perm:" + uuid.New().String() + ":" + userID1.String() + ":" + channelID.String() + ":1"

	rdb.Set(ctx, key1, "1", 0)
	rdb.Set(ctx, key2, "0", 0)
	rdb.Set(ctx, keyOtherUser, "1", 0)
	rdb.Set(ctx, keyOtherWorkspace, "1", 0)

	// Invalidate for userID1 in workspaceID
	err := invalidator.InvalidateUserPermissions(ctx, workspaceID, userID1)
	assert.NoError(t, err)

	// Check results
	assert.ErrorIs(t, rdb.Get(ctx, key1).Err(), redis.Nil, "key1 should be deleted")
	assert.ErrorIs(t, rdb.Get(ctx, key2).Err(), redis.Nil, "key2 should be deleted")

	val, err := rdb.Get(ctx, keyOtherUser).Result()
	assert.NoError(t, err, "keyOtherUser should not be deleted")
	assert.Equal(t, "1", val)

	val, err = rdb.Get(ctx, keyOtherWorkspace).Result()
	assert.NoError(t, err, "keyOtherWorkspace should not be deleted")
	assert.Equal(t, "1", val)
}
