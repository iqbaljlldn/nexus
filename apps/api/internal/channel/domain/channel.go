package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChannelType string

const (
	ChannelTypeText         ChannelType = "text"
	ChannelTypeVoice        ChannelType = "voice"
	ChannelTypeVideo        ChannelType = "video"
	ChannelTypeForum        ChannelType = "forum"
	ChannelTypeAnnouncement ChannelType = "announcement"
	ChannelTypeDM           ChannelType = "dm"
)

type Channel struct {
	ID             uuid.UUID
	WorkspaceID    *uuid.UUID // nil for DM
	CategoryID     *uuid.UUID
	Type           ChannelType
	Name           *string // nil for DM
	ParticipantKey *string // only for DM
	Position       int32
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

type ChannelPermissionOverride struct {
	ID           uuid.UUID
	ChannelID    uuid.UUID
	RoleID       *uuid.UUID // XOR with MemberID
	MemberID     *uuid.UUID
	AllowBitmask int64
	DenyBitmask  int64
}
