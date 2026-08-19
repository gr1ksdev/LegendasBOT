package services

import (
	"context"
	"fmt"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

type UserCaptionTemplateService struct {
	repo *repositories.UserCaptionTemplateRepository
}

func NewUserCaptionTemplateService(repo *repositories.UserCaptionTemplateRepository) *UserCaptionTemplateService {
	return &UserCaptionTemplateService{repo: repo}
}

func (s *UserCaptionTemplateService) Create(ctx context.Context, userID int64, code, caption, reactions string) (*models.UserCaptionTemplate, error) {
	id := fmt.Sprintf("uct_%d_%d", time.Now().UnixNano(), userID)
	tpl := &models.UserCaptionTemplate{
		ID:        id,
		UserID:    userID,
		Code:      code,
		Caption:   caption,
		Reactions: reactions,
	}
	if err := s.repo.Create(ctx, tpl); err != nil {
		return nil, errors.Internal(err)
	}
	return tpl, nil
}

func (s *UserCaptionTemplateService) List(ctx context.Context, userID int64) ([]models.UserCaptionTemplate, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *UserCaptionTemplateService) GetByID(ctx context.Context, id string) (*models.UserCaptionTemplate, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	return tpl, nil
}

func (s *UserCaptionTemplateService) GetByUserAndCode(ctx context.Context, userID int64, code string) (*models.UserCaptionTemplate, error) {
	tpl, err := s.repo.GetByUserAndCode(ctx, userID, code)
	if err != nil {
		return nil, nil // not found is not an error here — caller falls back
	}
	return tpl, nil
}

func (s *UserCaptionTemplateService) Update(ctx context.Context, userID int64, id, code, caption, reactions string) error {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrNotFound
	}
	if tpl.UserID != userID {
		return errors.ErrForbidden
	}
	return s.repo.Update(ctx, id, map[string]interface{}{
		"code":       code,
		"caption":    caption,
		"reactions":  reactions,
		"updated_at": time.Now(),
	})
}

func (s *UserCaptionTemplateService) UpdateLayout(ctx context.Context, userID int64, id string, layout [][]types.ButtonLayoutItem) error {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrNotFound
	}
	if tpl.UserID != userID {
		return errors.ErrForbidden
	}

	return s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
		for y, row := range layout {
			for x, btn := range row {
				if err := tx.Model(&models.UserCaptionTemplateButton{}).
					Where("button_id = ? AND owner_template_id = ?", btn.ID, id).
					Updates(map[string]interface{}{
						"position_x": x,
						"position_y": y,
					}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *UserCaptionTemplateService) Delete(ctx context.Context, id string, userID int64) error {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return errors.ErrNotFound
	}
	if tpl.UserID != userID {
		return errors.ErrForbidden
	}
	return s.repo.Delete(ctx, id, userID)
}

func (s *UserCaptionTemplateService) CreateButton(ctx context.Context, button *models.UserCaptionTemplateButton) error {
	return s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
		return tx.Create(button).Error
	})
}

func (s *UserCaptionTemplateService) UpdateButton(ctx context.Context, templateID, buttonID, name, url, style string) error {
	return s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"name_button": name,
			"button_url":  url,
		}
		if style != "" {
			updates["style"] = style
		}
		return tx.Model(&models.UserCaptionTemplateButton{}).
			Where("button_id = ? AND owner_template_id = ?", buttonID, templateID).
			Updates(updates).Error
	})
}

func (s *UserCaptionTemplateService) DeleteButton(ctx context.Context, templateID, buttonID string) error {
	return s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
		return tx.Where("button_id = ? AND owner_template_id = ?", buttonID, templateID).
			Delete(&models.UserCaptionTemplateButton{}).Error
	})
}
