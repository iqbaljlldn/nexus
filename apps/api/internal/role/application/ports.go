package application

import (
	"context"

	"github.com/google/uuid"
	memberDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
)

// TransactionManager executes fn within a single database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// MemberPort is the interface the role application layer uses to
// verify member existence without importing member infrastructure.
type MemberPort interface {
	GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID uuid.UUID) (*memberDomain.Member, error)
}

// PermissionCacheInvalidator is the interface used to invalidate cached permissions
// when role assignments change.
type PermissionCacheInvalidator interface {
	InvalidateUserPermissions(ctx context.Context, workspaceID, userID uuid.UUID) error
}
