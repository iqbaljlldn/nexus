package domain

import "context"

type MemberRepository interface {
	// Create persists a new member. The DB-generated ID is set on the
	// member pointer upon success.
	Create(ctx context.Context, member *Member) error
}
