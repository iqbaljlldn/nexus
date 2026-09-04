package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/iqbaljlldn/nexus/apps/api/internal/channel/domain"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"go.uber.org/zap"
)

type ChannelService struct {
	repo domain.ChannelRepository
	log  *zap.Logger
}

func NewChannelService(repo domain.ChannelRepository, log *zap.Logger) *ChannelService {
	return &ChannelService{
		repo: repo,
		log:  log,
	}
}

// GetByID returns a channel by its ID.
func (s *ChannelService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByWorkspaceID returns all channels in a workspace.
func (s *ChannelService) ListByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]domain.Channel, error) {
	return s.repo.ListByWorkspaceID(ctx, workspaceID)
}

// CreateTextChannel creates a new text channel in a workspace.
func (s *ChannelService) CreateTextChannel(ctx context.Context, workspaceID uuid.UUID, name string, categoryID *uuid.UUID) (*domain.Channel, error) {
	log := logger.FromContext(ctx, s.log)

	if name == "" {
		return nil, &pkgerrors.DomainError{
			Code:    pkgerrors.CodeMissingRequiredField,
			Message: "Nama channel wajib diisi.",
			Err:     fmt.Errorf("channel name is required"),
		}
	}

	if categoryID != nil {
		catWsID, err := s.repo.GetCategoryWorkspaceID(ctx, *categoryID)
		if err != nil {
			log.Warn("category not found or error", zap.Error(err), zap.String("category_id", categoryID.String()))
			return nil, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeInvalidFieldFormat,
				Message: "Kategori tidak ditemukan.",
				Err:     fmt.Errorf("category %s not found: %w", categoryID, err),
			}
		}
		if catWsID != workspaceID {
			return nil, &pkgerrors.DomainError{
				Code:    pkgerrors.CodeBusinessRuleViolation,
				Message: "Kategori bukan milik workspace ini.",
				Err:     fmt.Errorf("category %s belongs to workspace %s, not %s", categoryID, catWsID, workspaceID),
			}
		}
	}

	maxPos, err := s.repo.GetMaxPosition(ctx, workspaceID)
	if err != nil {
		log.Error("failed to get max channel position", zap.Error(err), zap.String("workspace_id", workspaceID.String()))
		return nil, fmt.Errorf("get max channel position: %w", err)
	}

	channel := &domain.Channel{
		WorkspaceID: &workspaceID,
		CategoryID:  categoryID,
		Type:        domain.ChannelTypeText,
		Name:        &name,
		Position:    maxPos + 1,
	}

	if err := s.repo.Create(ctx, channel); err != nil {
		log.Error("failed to create channel", zap.Error(err), zap.String("workspace_id", workspaceID.String()))
		return nil, fmt.Errorf("create channel: %w", err)
	}

	log.Info("text channel created successfully",
		zap.String("channel_id", channel.ID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.String("name", name),
		zap.Int32("position", channel.Position),
	)

	return channel, nil
}

// PatchPermissionOverride handles the XOR logic and upserts a permission override.
func (s *ChannelService) PatchPermissionOverride(ctx context.Context, channelID uuid.UUID, roleID, memberID *uuid.UUID, allowBitmask, denyBitmask int64) error {
	log := logger.FromContext(ctx, s.log)

	if (roleID == nil && memberID == nil) || (roleID != nil && memberID != nil) {
		return &pkgerrors.DomainError{
			Code:    pkgerrors.CodeInvalidFieldFormat,
			Message: "Harus mengisi salah satu dari role_id atau member_id, tidak boleh keduanya.",
			Err:     fmt.Errorf("xor constraint failed: role_id and member_id both provided or both nil"),
		}
	}

	// Verify channel exists
	if _, err := s.repo.GetByID(ctx, channelID); err != nil {
		log.Warn("channel not found for permission override", zap.Error(err), zap.String("channel_id", channelID.String()))
		return &pkgerrors.DomainError{
			Code:    pkgerrors.CodeRecordNotFound,
			Message: "Channel tidak ditemukan.",
			Err:     err,
		}
	}

	var existingOverride *domain.ChannelPermissionOverride
	var err error

	if roleID != nil {
		existingOverride, err = s.repo.GetChannelPermissionOverrideByRole(ctx, channelID, *roleID)
	} else {
		existingOverride, err = s.repo.GetChannelPermissionOverrideByMember(ctx, channelID, *memberID)
	}

	if err != nil {
		log.Error("failed to check existing permission override", zap.Error(err))
		return fmt.Errorf("check existing override: %w", err)
	}

	if existingOverride != nil {
		// Update existing
		if err := s.repo.UpdatePermissionOverride(ctx, existingOverride.ID, allowBitmask, denyBitmask); err != nil {
			log.Error("failed to update permission override", zap.Error(err))
			return fmt.Errorf("update permission override: %w", err)
		}
		log.Info("permission override updated", zap.String("override_id", existingOverride.ID.String()))
		return nil
	}

	// Create new
	newOverride := &domain.ChannelPermissionOverride{
		ChannelID:    channelID,
		RoleID:       roleID,
		MemberID:     memberID,
		AllowBitmask: allowBitmask,
		DenyBitmask:  denyBitmask,
	}
	if err := s.repo.CreatePermissionOverride(ctx, newOverride); err != nil {
		log.Error("failed to create permission override", zap.Error(err))
		return fmt.Errorf("create permission override: %w", err)
	}
	log.Info("permission override created", zap.String("override_id", newOverride.ID.String()))
	return nil
}
