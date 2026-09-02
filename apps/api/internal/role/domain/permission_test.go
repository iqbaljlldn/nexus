package domain_test

import (
	"testing"

	"github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/stretchr/testify/assert"
)

func TestPermissionFlags_NoBitCollision(t *testing.T) {
	flags := []domain.PermissionFlag{
		domain.PermSendMessages,
		domain.PermManageWorkspace,
		domain.PermManageRoles,
		domain.PermManageChannels,
		domain.PermManageInvites,
		domain.PermManageMessages,
		domain.PermKickMembers,
		domain.PermBanMembers,
	}

	seen := make(map[domain.PermissionFlag]string)
	names := []string{
		"PermSendMessages",
		"PermManageWorkspace",
		"PermManageRoles",
		"PermManageChannels",
		"PermManageInvites",
		"PermManageMessages",
		"PermKickMembers",
		"PermBanMembers",
	}

	for i, flag := range flags {
		// Each flag must be a single bit (power of 2)
		assert.True(t, flag > 0, "flag %s must be positive", names[i])
		assert.Equal(t, domain.PermissionFlag(0), flag&(flag-1),
			"flag %s (value %d) must be a power of 2", names[i], flag)

		if existing, ok := seen[flag]; ok {
			t.Fatalf("bit collision: %s and %s both have value %d", existing, names[i], flag)
		}
		seen[flag] = names[i]
	}
}

func TestPermissionFlags_ExpectedValues(t *testing.T) {
	assert.Equal(t, domain.PermissionFlag(1), domain.PermSendMessages)
	assert.Equal(t, domain.PermissionFlag(2), domain.PermManageWorkspace)
	assert.Equal(t, domain.PermissionFlag(4), domain.PermManageRoles)
	assert.Equal(t, domain.PermissionFlag(8), domain.PermManageChannels)
	assert.Equal(t, domain.PermissionFlag(16), domain.PermManageInvites)
	assert.Equal(t, domain.PermissionFlag(32), domain.PermManageMessages)
	assert.Equal(t, domain.PermissionFlag(64), domain.PermKickMembers)
	assert.Equal(t, domain.PermissionFlag(128), domain.PermBanMembers)
}

func TestPermissionFlag_Has(t *testing.T) {
	bitmask := domain.PermSendMessages | domain.PermManageRoles // 1 + 4 = 5

	assert.True(t, bitmask.Has(domain.PermSendMessages))
	assert.True(t, bitmask.Has(domain.PermManageRoles))
	assert.False(t, bitmask.Has(domain.PermManageChannels))
	assert.False(t, bitmask.Has(domain.PermBanMembers))

	// Has with combined flags: must have ALL
	assert.True(t, bitmask.Has(domain.PermSendMessages|domain.PermManageRoles))
	assert.False(t, bitmask.Has(domain.PermSendMessages|domain.PermManageChannels))
}

func TestPermissionFlag_Add(t *testing.T) {
	bitmask := domain.PermSendMessages

	result := bitmask.Add(domain.PermManageRoles)
	assert.True(t, result.Has(domain.PermSendMessages))
	assert.True(t, result.Has(domain.PermManageRoles))

	// Original should be unchanged (value type)
	assert.False(t, bitmask.Has(domain.PermManageRoles))

	// Adding an already-set flag is idempotent
	result2 := result.Add(domain.PermSendMessages)
	assert.Equal(t, result, result2)
}

func TestPermissionFlag_Remove(t *testing.T) {
	bitmask := domain.PermSendMessages | domain.PermManageRoles | domain.PermManageChannels

	result := bitmask.Remove(domain.PermManageRoles)
	assert.True(t, result.Has(domain.PermSendMessages))
	assert.False(t, result.Has(domain.PermManageRoles))
	assert.True(t, result.Has(domain.PermManageChannels))

	// Removing an already-unset flag is idempotent
	result2 := result.Remove(domain.PermManageRoles)
	assert.Equal(t, result, result2)
}

func TestPermissionFlag_CombinedOperations(t *testing.T) {
	var bitmask domain.PermissionFlag

	// Build up a bitmask step by step
	bitmask = bitmask.Add(domain.PermSendMessages)
	bitmask = bitmask.Add(domain.PermManageRoles)
	bitmask = bitmask.Add(domain.PermKickMembers)

	assert.True(t, bitmask.Has(domain.PermSendMessages))
	assert.True(t, bitmask.Has(domain.PermManageRoles))
	assert.True(t, bitmask.Has(domain.PermKickMembers))
	assert.False(t, bitmask.Has(domain.PermBanMembers))

	// Remove one
	bitmask = bitmask.Remove(domain.PermManageRoles)
	assert.False(t, bitmask.Has(domain.PermManageRoles))
	assert.True(t, bitmask.Has(domain.PermSendMessages))
	assert.True(t, bitmask.Has(domain.PermKickMembers))
}

func TestDefaultEveryonePermissions(t *testing.T) {
	bitmask := domain.PermissionFlag(domain.DefaultEveryonePermissions)

	assert.True(t, bitmask.Has(domain.PermSendMessages))
	assert.False(t, bitmask.Has(domain.PermManageWorkspace))
	assert.False(t, bitmask.Has(domain.PermManageRoles))
	assert.False(t, bitmask.Has(domain.PermManageChannels))
	assert.False(t, bitmask.Has(domain.PermManageInvites))
	assert.False(t, bitmask.Has(domain.PermManageMessages))
	assert.False(t, bitmask.Has(domain.PermKickMembers))
	assert.False(t, bitmask.Has(domain.PermBanMembers))
}
