package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	mbDomain "github.com/iqbaljlldn/nexus/apps/api/internal/member/domain"
	wpDomain "github.com/iqbaljlldn/nexus/apps/api/internal/workspace/domain"
	"github.com/iqbaljlldn/nexus/apps/api/internal/workspace/interface/dto"
	"github.com/iqbaljlldn/nexus/pkg/contextutil"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"go.uber.org/zap"
)

type InviteService struct {
	inviteRepo wpDomain.InviteRepository
	memberPort MemberPort
	rolePort   RolePort
	txManager  TransactionManager
	log        *zap.Logger
}

func NewInviteService(
	inviteRepo wpDomain.InviteRepository,
	memberPort MemberPort,
	rolePort RolePort,
	txManager TransactionManager,
	log *zap.Logger,
) *InviteService {
	return &InviteService{
		inviteRepo: inviteRepo,
		memberPort: memberPort,
		rolePort:   rolePort,
		txManager:  txManager,
		log:        log,
	}
}

func (s *InviteService) Create(ctx context.Context, req dto.CreateInviteReq) (*wpDomain.Invite, error) {
	log := logger.FromContext(ctx, s.log)

	userID, err := contextutil.UserID(ctx)
	if err != nil {
		log.Warn("unauthorized invite creation attempt", zap.Error(err))
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeUserUnauthorized,
			Message: "User tidak terautentikasi.",
			Err:     err,
		}
	}

	code, err := uuid.NewRandom()
	if err != nil {
		log.Error("failed to generate invite code", zap.Error(err))
		return nil, fmt.Errorf("generate invite code: %w", err)
	}

	invite := wpDomain.NewInvite(
		req.WorkspaceID,
		userID,
		code.String(),
		req.MaxUses,
		req.ExpiresAt,
	)

	err = s.inviteRepo.Create(ctx, invite)
	if err != nil {
		log.Error("failed to create invite in database", zap.Error(err), zap.String("workspace_id", req.WorkspaceID.String()))
		return nil, fmt.Errorf("create invite: %w", err)
	}

	log.Info("invite code created successfully",
		zap.String("invite_id", invite.ID.String()),
		zap.String("code", invite.Code),
		zap.String("workspace_id", req.WorkspaceID.String()),
	)

	return invite, nil
}

func (s *InviteService) Redeem(ctx context.Context, code string, userID uuid.UUID) (*mbDomain.Member, error) {
	log := logger.FromContext(ctx, s.log)

	// 1. Get invite by code
	invite, err := s.inviteRepo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, wpDomain.ErrInviteNotFound) {
			log.Warn("invite code not found", zap.String("code", code))
			return nil, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeRecordNotFound,
				Message: "Undangan tidak ditemukan.",
				Err:     err,
			}
		}
		log.Error("failed to get invite by code", zap.Error(err), zap.String("code", code))
		return nil, fmt.Errorf("get invite: %w", err)
	}

	// 2. Validate expiration
	if invite.IsExpired() {
		log.Warn("invite has expired", zap.String("code", code))
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeBusinessRuleViolation,
			Message: "Kode undangan telah kedaluwarsa.",
			Err:     wpDomain.ErrInviteExpired,
		}
	}

	// 3. Validate max uses
	if invite.IsMaxUsesReached() {
		log.Warn("invite max uses reached", zap.String("code", code))
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeBusinessRuleViolation,
			Message: "Batas penggunaan kode undangan telah tercapai.",
			Err:     wpDomain.ErrInviteMaxUsesReached,
		}
	}

	// 4. Check if user is already a member (Idempotency by design)
	existingMember, err := s.memberPort.GetByWorkspaceAndUser(ctx, invite.WorkspaceID, userID)
	if err == nil && existingMember != nil {
		log.Info("user is already a member, returning existing membership",
			zap.String("workspace_id", invite.WorkspaceID.String()),
			zap.String("user_id", userID.String()),
			zap.String("member_id", existingMember.ID.String()),
		)
		return existingMember, nil
	}

	// 5. Create new member & assign @everyone role in a single transaction
	var createdMember *mbDomain.Member
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Increment use count in DB
		if err := s.inviteRepo.IncrementUseCount(txCtx, invite.ID); err != nil {
			log.Error("failed to increment invite use count", zap.Error(err), zap.String("invite_id", invite.ID.String()))
			return fmt.Errorf("increment invite use count: %w", err)
		}

		// Create member
		member := &mbDomain.Member{
			WorkspaceID: invite.WorkspaceID,
			UserID:      userID,
		}
		if err := s.memberPort.Create(txCtx, member); err != nil {
			log.Error("failed to create member", zap.Error(err), zap.String("workspace_id", invite.WorkspaceID.String()))
			return fmt.Errorf("create member: %w", err)
		}

		// Fetch @everyone role for workspace
		everyoneRole, err := s.rolePort.GetEveryoneRole(txCtx, invite.WorkspaceID)
		if err != nil {
			log.Error("failed to get @everyone role", zap.Error(err), zap.String("workspace_id", invite.WorkspaceID.String()))
			return fmt.Errorf("get @everyone role: %w", err)
		}

		// Assign @everyone role
		if err := s.rolePort.Assign(txCtx, member.ID, everyoneRole.ID); err != nil {
			log.Error("failed to assign @everyone role", zap.Error(err), zap.String("member_id", member.ID.String()))
			return fmt.Errorf("assign @everyone role: %w", err)
		}

		createdMember = member
		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Info("invite redeemed successfully",
		zap.String("code", code),
		zap.String("workspace_id", invite.WorkspaceID.String()),
		zap.String("user_id", userID.String()),
		zap.String("member_id", createdMember.ID.String()),
	)

	return createdMember, nil
}
