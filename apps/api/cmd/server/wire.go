//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/iqbaljlldn/nexus/apps/api/internal/channel"
	channelhttp "github.com/iqbaljlldn/nexus/apps/api/internal/channel/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/health"
	healthhttp "github.com/iqbaljlldn/nexus/apps/api/internal/health/transport/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity"
	identityhttp "github.com/iqbaljlldn/nexus/apps/api/internal/identity/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/member"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/role"
	roleapp "github.com/iqbaljlldn/nexus/apps/api/internal/role/application"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	rolehttp "github.com/iqbaljlldn/nexus/apps/api/internal/role/interface/http"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace"
	workspaceapp "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/application"
	workspacehttp "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/http"
	"github.com/iqbaljlldn/nexus/pkg/router"
)

func provideRouters(
	healthRouter *healthhttp.Handler,
	identityRouter *identityhttp.AuthHandler,
	workspaceRouter *workspacehttp.WorkspaceHandler,
	roleRouter *rolehttp.RoleHandler,
	channelRouter *channelhttp.ChannelHandler,
) []router.ModuleRouter {
	return []router.ModuleRouter{
		healthRouter,
		identityRouter,
		workspaceRouter,
		roleRouter,
		channelRouter,
	}
}

func provideRoleTxManager(tm workspaceapp.TransactionManager) roleapp.TransactionManager {
	return tm
}

func provideRoleCacheInvalidator(r *workspaceapp.CachedPermissionResolver) roleapp.PermissionCacheInvalidator {
	return r
}

func provideRolePermResolver(r *workspaceapp.CachedPermissionResolver) rolehttp.PermissionResolver {
	return r
}

func provideChannelPermResolver(r *workspaceapp.CachedPermissionResolver) channelhttp.PermissionResolver {
	return r
}

func provideRolePort(repo roleDomain.RoleRepository) workspaceapp.RolePort {
	return repo
}

func provideBaseURL() string {
	return "http://localhost:3000"
}

func provideMemberPort(repo memberDomain.MemberRepository) workspaceapp.MemberPort {
	return repo
}

func InitializeRouter(log *zap.Logger, db *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	wire.Build(
		health.ProviderSet,
		identity.ProviderSet,
		member.ProviderSet,
		workspace.ProviderSet,
		role.ProviderSet,
		channel.ProviderSet,

		provideRoleTxManager,
		provideRoleCacheInvalidator,
		provideRolePermResolver,
		provideChannelPermResolver,
		provideRolePort,
		provideMemberPort,
		provideBaseURL,

		provideRouters,
		NewRouter,
	)
	return &gin.Engine{}
}
