package repositories

import (
	"context"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type ChannelEventFilters struct {
	ChannelID int64
	OwnerID   int64
	ActorID   int64
	Source    string
	EventType string
	Status    string
	SessionID string
	Query     string
	DateFrom  *time.Time
	DateTo    *time.Time
	Limit     int
	Offset    int
}

type ChannelEventRepository struct {
	db *gorm.DB
}

func NewChannelEventRepository(db *gorm.DB) *ChannelEventRepository {
	return &ChannelEventRepository{db: db}
}

func (r *ChannelEventRepository) Create(ctx context.Context, event *models.ChannelEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *ChannelEventRepository) List(ctx context.Context, filters ChannelEventFilters) ([]models.ChannelEvent, int64, error) {
	var events []models.ChannelEvent
	var total int64

	query := r.applyFilters(r.db.WithContext(ctx).Model(&models.ChannelEvent{}), filters)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&events).Error
	return events, total, err
}

func (r *ChannelEventRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.ChannelEvent{})
	return result.RowsAffected, result.Error
}

func (r *ChannelEventRepository) applyFilters(query *gorm.DB, filters ChannelEventFilters) *gorm.DB {
	if filters.ChannelID != 0 {
		query = query.Where("channel_id = ?", filters.ChannelID)
	}
	if filters.OwnerID != 0 {
		query = query.Where("owner_id = ?", filters.OwnerID)
	}
	if filters.ActorID != 0 {
		query = query.Where("actor_id = ?", filters.ActorID)
	}
	if filters.Source != "" {
		query = query.Where("source = ?", filters.Source)
	}
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.SessionID != "" {
		query = query.Where("session_id = ?", filters.SessionID)
	}
	if filters.DateFrom != nil {
		query = query.Where("created_at >= ?", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		query = query.Where("created_at <= ?", *filters.DateTo)
	}
	if q := strings.TrimSpace(filters.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("channel_title LIKE ? OR error_message LIKE ? OR metadata LIKE ?", like, like, like)
	}
	return query
}
