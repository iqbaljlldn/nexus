package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"github.com/iqbaljlldn/nexus/pkg/passwordhash"
	"go.uber.org/zap"
)

type AuthService struct {
	userRepo     domain.UserRepository
	sessionRepo  domain.SessionRepository
	tokenManager domain.TokenManager
	log          *zap.Logger
}

func NewAuthService(userRepo domain.UserRepository, sessionRepo domain.SessionRepository, tokenManager domain.TokenManager, log *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		tokenManager: tokenManager,
		log:          log,
	}
}

func (s *AuthService) Register(ctx context.Context, emailStr, usernameStr, displayName, password string) (*domain.User, error) {
	log := logger.FromContext(ctx, s.log)

	email, err := domain.NewEmail(emailStr)
	if err != nil {
		log.Warn("failed to validate email", zap.Error(err), zap.String("email", emailStr))
		return nil, err
	}

	if _, err := s.userRepo.FindByEmail(ctx, email.String()); err == nil {
		log.Warn("email already in use", zap.String("email", email.String()))
		return nil, domain.ErrDuplicateEmail
	}

	username, err := domain.NewUsername(usernameStr)
	if err != nil {
		log.Warn("failed to validate username", zap.Error(err), zap.String("username", usernameStr))
		return nil, err
	}

	if _, err := s.userRepo.FindByUsername(ctx, username.String()); err == nil {
		log.Warn("username already in use", zap.String("username", username.String()))
		return nil, domain.ErrDuplicateUsername
	}

	encodedHash, err := passwordhash.Hash(password)
	if err != nil {
		log.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	passwordHash, err := domain.NewPasswordHash(encodedHash)
	if err != nil {
		log.Error("failed to create password hash value object", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	user := domain.NewUser(email, username, displayName, passwordHash)

	if err := s.userRepo.Create(ctx, user); err != nil {
		log.Warn("failed to create user", zap.Error(err), zap.String("email", email.String()), zap.String("username", username.String()))
		return nil, err
	}

	log.Info("user registered successfully", zap.String("user_id", user.ID.String()), zap.String("email", email.String()))

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, identifier, password string, deviceInfo *domain.DeviceInfo) (*domain.TokenPair, *domain.User, error) {
	log := logger.FromContext(ctx, s.log)
	var user *domain.User
	var err error
	if strings.Contains(identifier, "@") {
		user, err = s.userRepo.FindByEmail(ctx, identifier)
	} else {
		user, err = s.userRepo.FindByUsername(ctx, identifier)
	}

	if err != nil {
		log.Warn("invalid credentials", zap.String("identifier", identifier))
		return nil, nil, domain.ErrInvalidCredentials
	}

	if matched, err := passwordhash.Verify(password, user.PasswordHash.String()); err != nil || !matched {
		log.Warn("invalid credentials", zap.String("identifier", identifier))
		return nil, nil, domain.ErrInvalidCredentials
	}

	accesToken, err := s.tokenManager.GenerateToken(user.ID.String(), "access", 15*time.Minute, *deviceInfo)
	if err != nil {
		log.Error("failed to generate access token", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("internal server error")
	}

	refreshToken, err := s.tokenManager.GenerateToken(user.ID.String(), "refresh", 24*time.Hour, *deviceInfo)
	if err != nil {
		log.Error("failed to generate refresh token", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("internal server error")
	}

	tokenPair := &domain.TokenPair{
		AccessToken:  accesToken,
		RefreshToken: refreshToken,
	}

	refreshTokenHash, err := passwordhash.Hash(refreshToken)
	if err != nil {
		log.Error("failed to hash refresh token", zap.Error(err))
		return nil, nil, fmt.Errorf("internal server error")
	}

	session := domain.NewSession(user.ID.String(), refreshTokenHash, *deviceInfo, 24*time.Hour)
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		log.Error("failed to create session", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("internal server error")
	}

	log.Info("user logged in successfully", zap.String("user_id", user.ID.String()), zap.String("session_id", session.ID), zap.String("email", user.Email.String()))

	return tokenPair, user, nil
}
