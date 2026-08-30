package domain

import "context"

type WorkspaceRepository interface {
	// Create persists a new workspace. The DB-generated ID is set on the
	// workspace pointer upon success.
	Create(ctx context.Context, workspace *Workspace) error
}
