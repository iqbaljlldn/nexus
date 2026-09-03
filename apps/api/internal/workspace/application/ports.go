package application

import (
	"context"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
)

// TransactionManager executes fn within a single database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
// Repositories participating in the transaction extract the *sql.Tx from
// the context provided to fn (via the txcontext package).
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// MemberPort is the interface the workspace application layer uses to
// interact with the member domain without importing its infrastructure.
type MemberPort interface {
	// Create persists a new member. The DB-generated ID is set on the
	// member pointer upon success.
	Create(ctx context.Context, member *memberDomain.Member) error
	GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*memberDomain.Member, error)
}

// RolePort is the interface the workspace application layer uses to
// interact with the role domain without importing its infrastructure.
type RolePort interface {
	// Create persists a new role. The DB-generated ID is set on the
	// role pointer upon success.
	Create(ctx context.Context, role *roleDomain.Role) error
	// Assign creates a member-role assignment.
	Assign(ctx context.Context, memberID, roleID uuid.UUID) error
	GetEveryoneRole(ctx context.Context, workspaceID uuid.UUID) (*roleDomain.Role, error)
	FindMemberRolesSortedByPosition(ctx context.Context, memberID uuid.UUID) ([]*roleDomain.Role, error)
}

// ChannelOverride represents a channel-specific permission override.
type ChannelOverride struct {
	Allow roleDomain.PermissionFlag
	Deny  roleDomain.PermissionFlag
}

// ChannelOverridePort is the interface the workspace application layer uses
// to fetch channel overrides from the channel infrastructure.
type ChannelOverridePort interface {
	FindMemberOverride(ctx context.Context, channelID, userID uuid.UUID) (*ChannelOverride, bool, error)
	FindRoleOverride(ctx context.Context, channelID, roleID uuid.UUID) (*ChannelOverride, bool, error)
}
