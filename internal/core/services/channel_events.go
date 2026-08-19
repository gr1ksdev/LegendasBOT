package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

const (
	ChannelEventSourceChannelPost = "channel_post"
	ChannelEventSourcePostBuilder = "post_builder"

	ChannelEventStatusSuccess = "success"
	ChannelEventStatusError   = "error"
	ChannelEventStatusSkipped = "skipped"
	ChannelEventStatusInfo    = "info"

	ChannelEventRetentionDays  = 90
	maxChannelEventMetadataLen = 12000
)

type ChannelEventRecordInput struct {
	ChannelID         int64
	ChannelTitle      string
	OwnerID           int64
	ActorID           int64
	Source            string
	EventType         string
	Status            string
	MessageType       string
	TelegramMessageID int
	SessionID         string
	Error             error
	ErrorMessage      string
	Metadata          map[string]any
}

type ChannelEventListFilters = repositories.ChannelEventFilters

type ChannelEventListResult struct {
	Events []models.ChannelEvent `json:"events"`
	Total  int64                 `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

type ChannelEventService struct {
	repo *repositories.ChannelEventRepository
}

func NewChannelEventService(repo *repositories.ChannelEventRepository) *ChannelEventService {
	return &ChannelEventService{repo: repo}
}

func (s *ChannelEventService) Record(ctx context.Context, input ChannelEventRecordInput) {
	if s == nil || s.repo == nil {
		return
	}
	if strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.EventType) == "" {
		return
	}
	if input.Status == "" {
		input.Status = ChannelEventStatusInfo
	}

	metadata := ""
	if len(input.Metadata) > 0 {
		payload, err := json.Marshal(input.Metadata)
		if err != nil {
			payload, _ = json.Marshal(map[string]any{"metadata_error": err.Error()})
		}
		metadata = string(payload)
		if len(metadata) > maxChannelEventMetadataLen {
			metadata = metadata[:maxChannelEventMetadataLen]
		}
	}

	errorMessage := input.ErrorMessage
	if input.Error != nil {
		errorMessage = input.Error.Error()
	}
	if len(errorMessage) > 4000 {
		errorMessage = errorMessage[:4000]
	}

	event := &models.ChannelEvent{
		ID:                uuid.NewString(),
		ChannelID:         input.ChannelID,
		ChannelTitle:      input.ChannelTitle,
		OwnerID:           input.OwnerID,
		ActorID:           input.ActorID,
		Source:            input.Source,
		EventType:         input.EventType,
		Status:            input.Status,
		MessageType:       input.MessageType,
		TelegramMessageID: input.TelegramMessageID,
		SessionID:         input.SessionID,
		ErrorMessage:      errorMessage,
		Metadata:          metadata,
	}

	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	logCtx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
	defer cancel()
	if err := s.repo.Create(logCtx, event); err != nil {
		logger.Warn("CHANNEL_EVENTS", "Falha ao registrar evento %s/%s: %v", input.Source, input.EventType, err)
	}
}

func (s *ChannelEventService) ListAdmin(ctx context.Context, filters ChannelEventListFilters) (*ChannelEventListResult, error) {
	events, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}
	return &ChannelEventListResult{Events: events, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *ChannelEventService) CleanupOld(ctx context.Context, retentionDays int) {
	if s == nil || s.repo == nil {
		return
	}
	if retentionDays <= 0 {
		retentionDays = ChannelEventRetentionDays
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		logger.Warn("CHANNEL_EVENTS", "Falha ao limpar eventos antigos: %v", err)
		return
	}
	if deleted > 0 {
		logger.Info("CHANNEL_EVENTS", "Eventos antigos removidos: %d", deleted)
	}
}
