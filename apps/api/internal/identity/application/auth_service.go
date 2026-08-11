package application

import (
	"context"
	"fmt"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/pkg/passwordhash"
	"go.uber.org/zap"
)

type AuthService struct {
	userRepo domain.UserRepository
	log      *zap.Logger
}

func NewAuthService(userRepo domain.UserRepository, log *zap.Logger) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		log:      log,
	}
}

func (s *AuthService) Register(ctx context.Context, emailStr, usernameStr, displayName, password string) (*domain.User, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		s.log.Warn("failed to validate email", zap.Error(err), zap.String("email", emailStr))
		return nil, err
	}

	username, err := domain.NewUsername(usernameStr)
	if err != nil {
		s.log.Warn("failed to validate username", zap.Error(err), zap.String("username", usernameStr))
		return nil, err
	}

	encodedHash, err := passwordhash.Hash(password)
	if err != nil {
		s.log.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	passwordHash, err := domain.NewPasswordHash(encodedHash)
	if err != nil {
		s.log.Error("failed to create password hash value object", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	user := domain.NewUser(email, username, displayName, passwordHash)

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Warn("failed to create user", zap.Error(err), zap.String("email", email.String()), zap.String("username", username.String()))
		return nil, err
	}

	s.log.Info("user registered successfully", zap.String("user_id", user.ID.String()), zap.String("email", email.String()))

	return user, nil
}
