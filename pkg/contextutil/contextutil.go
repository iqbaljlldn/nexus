package contextutil

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "user_id"

func WithUserID(ctx context.Context, UserID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, UserID)
}

func UserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("user id not found in context")
	}

	return userID, nil
}
