package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/identity/domain"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
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
		return nil, fmt.Errorf("hash password: %w", err)
	}

	passwordHash, err := domain.NewPasswordHash(encodedHash)
	if err != nil {
		log.Error("failed to create password hash value object", zap.Error(err))
		return nil, fmt.Errorf("create password hash: %w", err)
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
	if domain.IsEmail(identifier) {
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

	sessionID, err := uuid.NewV7()
	if err != nil {
		sessionID = uuid.New()
	}

	accesToken, err := s.tokenManager.GenerateToken(user.ID.String(), sessionID.String(), "access", 15*time.Minute, *deviceInfo)
	if err != nil {
		log.Error("failed to generate access token", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateToken(user.ID.String(), sessionID.String(), "refresh", 24*time.Hour, *deviceInfo)
	if err != nil {
		log.Error("failed to generate refresh token", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("generate refresh token: %w", err)
	}

	tokenPair := &domain.TokenPair{
		AccessToken:  accesToken,
		RefreshToken: refreshToken,
	}

	refreshTokenHash, err := passwordhash.Hash(refreshToken)
	if err != nil {
		log.Error("failed to hash refresh token", zap.Error(err))
		return nil, nil, fmt.Errorf("hash refresh token: %w", err)
	}

	session := domain.NewSession(sessionID.String(), user.ID.String(), refreshTokenHash, *deviceInfo, 24*time.Hour)
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		log.Error("failed to create session", zap.Error(err), zap.String("user_id", user.ID.String()))
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	log.Info("user logged in successfully", zap.String("user_id", user.ID.String()), zap.String("session_id", session.ID), zap.String("email", user.Email.String()))

	return tokenPair, user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	log := logger.FromContext(ctx, s.log)

	claims, err := s.tokenManager.ParseToken(refreshToken, "refresh")
	if err != nil {
		log.Warn("invalid token signature or format", zap.Error(err))
		return nil, domain.ErrInvalidToken
	}

	sessionUUID, err := uuid.Parse(claims.ID)
	if err != nil {
		log.Warn("invalid session id in token", zap.Error(err))
		return nil, domain.ErrInvalidToken
	}

	session, err := s.sessionRepo.FindById(ctx, sessionUUID)
	if err != nil {
		log.Warn("session not found for token", zap.String("session_id", claims.ID))
		return nil, domain.ErrInvalidToken
	}

	if matched, _ := passwordhash.Verify(refreshToken, session.RefreshTokenHash); !matched {
		log.Warn("token hash mismatch", zap.String("session_id", claims.ID))
		return nil, domain.ErrInvalidToken
	}

	deviceInfo := domain.DeviceInfo{
		DeviceID:  "unknown",
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
	}

	if err := session.IsValidForRefresh(deviceInfo); err != nil {
		log.Warn("session invalid for refresh", zap.Error(err))
		return nil, err
	}

	newSessionID, err := uuid.NewV7()
	if err != nil {
		newSessionID = uuid.New()
	}

	accessToken, err := s.tokenManager.GenerateToken(session.UserID, newSessionID.String(), "access", 15*time.Minute, deviceInfo)
	if err != nil {
		log.Error("failed to generate access token", zap.Error(err), zap.String("user_id", session.UserID))
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenManager.GenerateToken(session.UserID, newSessionID.String(), "refresh", 24*time.Hour, deviceInfo)
	if err != nil {
		log.Error("failed to generate refresh token", zap.Error(err), zap.String("user_id", session.UserID))
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	newRefreshTokenHash, err := passwordhash.Hash(newRefreshToken)
	if err != nil {
		log.Error("failed to hash new refresh token", zap.Error(err))
		return nil, fmt.Errorf("hash refresh token: %w", err)
	}

	err = s.sessionRepo.RotateRefreshToken(ctx, session.ID, newSessionID.String(), newRefreshTokenHash)
	if err != nil {
		log.Error("failed to rotate refresh token", zap.Error(err), zap.String("user_id", session.UserID))
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	log := logger.FromContext(ctx, s.log)

	claims, err := s.tokenManager.ParseToken(refreshToken, "refresh")
	if err != nil {
		log.Warn("invalid token signature or format", zap.Error(err))
		return domain.ErrInvalidToken
	}

	sessionUUID, err := uuid.Parse(claims.ID)
	if err != nil {
		log.Warn("invalid session id in token", zap.Error(err))
		return domain.ErrInvalidToken
	}

	session, err := s.sessionRepo.FindById(ctx, sessionUUID)
	if err != nil {
		log.Warn("session not found for token", zap.String("session_id", claims.ID))
		return domain.ErrInvalidToken
	}

	if matched, _ := passwordhash.Verify(refreshToken, session.RefreshTokenHash); !matched {
		log.Warn("token hash mismatch", zap.String("session_id", claims.ID))
		return domain.ErrInvalidToken
	}

	if session.ExpiresAt.Before(time.Now()) {
		log.Warn("refresh token expired", zap.Error(err))
		return domain.ErrExpiredToken
	}

	_, err = s.sessionRepo.RevokeSessionById(ctx, sessionUUID)
	if err != nil {
		log.Error("failed to revoke session", zap.Error(err), zap.String("user_id", session.UserID))
		return fmt.Errorf("revoke session: %w", err)
	}

	log.Info("user logged out successfully", zap.String("user_id", session.UserID), zap.String("session_id", session.ID))

	return nil
}

func (s *AuthService) LogoutAll(ctx context.Context) error {
	log := logger.FromContext(ctx, s.log)

	if err := s.sessionRepo.RevokeAllSessions(ctx); err != nil {
		log.Error("failed to revoke all sessions", zap.Error(err))
		return fmt.Errorf("revoke all sessions: %w", err)
	}

	userID := ""
	if id, err := contextutil.UserID(ctx); err == nil {
		userID = id.String()
	}

	log.Info("user logged out from all devices successfully", zap.String("user_id", userID))

	return nil
}

func (s *AuthService) GetActiveSessions(ctx context.Context) ([]domain.Session, error) {
	log := logger.FromContext(ctx, s.log)

	sessions, err := s.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		log.Error("failed to find sessions", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	return sessions, nil
}

func (s *AuthService) RevokeSessionById(ctx context.Context, id string) (*domain.Session, error) {
	log := logger.FromContext(ctx, s.log)

	sessionID, err := uuid.Parse(id)
	if err != nil {
		log.Warn("failed to parse session id", zap.Error(err))
		return nil, fmt.Errorf("invalid session id")
	}

	// Fetch session by ID
	session, err := s.sessionRepo.FindById(ctx, sessionID)
	if err != nil {
		log.Warn("session not found", zap.Error(err), zap.String("session_id", id))
		return nil, domain.ErrInvalidToken
	}

	// Verify ownership
	userID := ""
	if id, err := contextutil.UserID(ctx); err == nil {
		userID = id.String()
	}

	if session.UserID != userID {
		log.Warn("unauthorized session revocation attempt", zap.String("session_id", id), zap.String("user_id", userID))
		return nil, domain.ErrUnauthorizedSession
	}

	session, err = s.sessionRepo.RevokeSessionById(ctx, sessionID)
	if err != nil {
		log.Error("failed to revoke session", zap.Error(err), zap.String("session_id", id))
		return nil, fmt.Errorf("revoke session: %w", err)
	}

	log.Info("session revoked successfully", zap.String("session_id", id))

	return session, nil
}
