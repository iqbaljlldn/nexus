package application

import (
	"context"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
)

type PermissionResolver struct {
	workspaceRepo       wpDomain.WorkspaceRepository
	memberPort          MemberPort
	channelOverrideRepo ChannelOverridePort
	roleRepo            RolePort
}

func NewPermissionResolver(
	workspaceRepo wpDomain.WorkspaceRepository,
	memberPort MemberPort,
	channelOverrideRepo ChannelOverridePort,
	roleRepo RolePort,
) *PermissionResolver {
	return &PermissionResolver{
		workspaceRepo:       workspaceRepo,
		memberPort:          memberPort,
		channelOverrideRepo: channelOverrideRepo,
		roleRepo:            roleRepo,
	}
}

// Resolve executes the 4-level permission resolution algorithm as specified in LLD 2.1.
// Resolves in the following order:
// 1. Channel-specific member override
// 2. Channel-specific role override (iterated highest position to lowest)
// 3. Role default (iterated highest position to lowest)
// 4. @everyone role fallback
func (r *PermissionResolver) Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required roleDomain.PermissionFlag) (bool, error) {
	// Pengecualian: Jika Member adalah Owner dari Workspace, langsung return true (LLD 2.1)
	workspace, err := r.workspaceRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	if workspace.OwnerID == userID {
		return true, nil
	}

	// Dapatkan member ID
	member, err := r.memberPort.GetByWorkspaceAndUser(ctx, workspaceID, userID)
	if err != nil {
		return false, err
	}
	memberID := member.ID

	// 1. Channel-specific MEMBER override (prioritas tertinggi)
	override, ok, err := r.channelOverrideRepo.FindMemberOverride(ctx, channelID, memberID)
	if err != nil {
		return false, err
	}
	if ok {
		if override.Deny.Has(required) {
			return false, nil // deny eksplisit selalu menang di level ini
		}
		if override.Allow.Has(required) {
			return true, nil
		}
		// Tidak diatur di level ini -> lanjut ke level berikutnya
	}

	// Fetch member roles (sorted by position DESC) for step 2 and 3
	roles, err := r.roleRepo.FindMemberRolesSortedByPosition(ctx, memberID)
	if err != nil {
		return false, err
	}

	// 2. Channel-specific ROLE override, diiterasi dari role dengan `position` tertinggi
	for _, role := range roles {
		roleOverride, ok, err := r.channelOverrideRepo.FindRoleOverride(ctx, channelID, role.ID)
		if err != nil {
			return false, err
		}
		if ok {
			if roleOverride.Deny.Has(required) {
				return false, nil // deny di role tertinggi
			}
			if roleOverride.Allow.Has(required) {
				return true, nil // allow di role tertinggi
			}
			// Tidak diatur di role ini -> lanjut ke role berikutnya
		}
	}

	// 3. Role default permission (bitmask), role tertinggi menang
	for _, role := range roles {
		if roleDomain.PermissionFlag(role.PermissionBitmask).Has(required) {
			return true, nil
		}
	}

	// 4. @everyone fallback
	everyone, err := r.roleRepo.GetEveryoneRole(ctx, workspaceID)
	if err != nil {
		return false, err
	}
	return roleDomain.PermissionFlag(everyone.PermissionBitmask).Has(required), nil
}
