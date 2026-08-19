package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type ButtonService struct {
	buttonRepo        *repositories.ButtonRepository
	channelRepo       *repositories.ChannelRepository
	customCaptionRepo *repositories.CustomCaptionRepository
	cache             *cache.Service
}

func NewButtonService(
	buttonRepo *repositories.ButtonRepository,
	channelRepo *repositories.ChannelRepository,
	customCaptionRepo *repositories.CustomCaptionRepository,
	cache *cache.Service,
) *ButtonService {
	return &ButtonService{
		buttonRepo:        buttonRepo,
		channelRepo:       channelRepo,
		customCaptionRepo: customCaptionRepo,
		cache:             cache,
	}
}

// --- BOTÕES DE CANAL (DEFAULT) ---

func (s *ButtonService) CreateButton(ctx context.Context, channelID int64, buttonData types.ButtonCreateRequest) (*models.Button, error) {
	buttonData.ButtonURL = utils.NormalizeTelegramURL(buttonData.ButtonURL)
	if err := s.validateButtonData(buttonData); err != nil {
		return nil, err
	}

	position, err := s.calculateNextButtonPosition(ctx, channelID)
	if err != nil {
		return nil, err
	}

	newButton := &models.Button{
		ButtonID:       uuid.NewString(),
		OwnerChannelID: channelID,
		NameButton:     buttonData.NameButton,
		ButtonURL:      buttonData.ButtonURL,
		Style:          buttonData.Style,
		PositionX:      position.X,
		PositionY:      position.Y,
	}

	if err := s.buttonRepo.CreateButton(ctx, newButton); err != nil {
		return nil, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return newButton, nil
}

func (s *ButtonService) UpdateButton(ctx context.Context, channelID int64, buttonID string, buttonData types.ButtonCreateRequest) (int64, error) {
	buttonData.ButtonURL = utils.NormalizeTelegramURL(buttonData.ButtonURL)
	if err := s.validateButtonData(buttonData); err != nil {
		return 0, err
	}

	rowsAffected, err := s.buttonRepo.UpdateButton(ctx, channelID, buttonID, buttonData.NameButton, buttonData.ButtonURL, buttonData.Style)
	if err != nil {
		return 0, errors.Internal(err)
	}

	if rowsAffected > 0 {
		s.cache.InvalidateChannel(ctx, channelID)
	}

	return rowsAffected, nil
}

func (s *ButtonService) DeleteButton(ctx context.Context, channelID int64, buttonID string) (int64, error) {
	rowsAffected, err := s.buttonRepo.DeleteButton(ctx, channelID, buttonID)
	if err != nil {
		return 0, errors.Internal(err)
	}

	if rowsAffected > 0 {
		s.cache.InvalidateChannel(ctx, channelID)
	}

	return rowsAffected, nil
}

func (s *ButtonService) UpdateButtonsLayout(ctx context.Context, channelID int64, layoutData types.UpdateLayoutRequest) (int, error) {
	channel, err := s.channelRepo.GetChannelByIDLight(ctx, channelID)
	if err != nil {
		return 0, errors.ErrNotFound
	}

	var buttonsToUpdate []struct {
		ID string
		X  int
		Y  int
	}
	for y, row := range layoutData.Layout {
		for x, button := range row {
			buttonsToUpdate = append(buttonsToUpdate, struct {
				ID string
				X  int
				Y  int
			}{ID: button.ID, X: x, Y: y})
		}
	}

	err = s.buttonRepo.UpdateButtonsLayout(ctx, channelID, buttonsToUpdate, channel.ReactionPosition)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return len(buttonsToUpdate), nil
}

// --- BOTÕES DE LEGENDA CUSTOMIZADA ---

func (s *ButtonService) CreateCustomCaptionButton(ctx context.Context, channelID int64, captionID string, body types.ButtonCreateRequest) (*models.CustomCaptionButton, error) {
	body.ButtonURL = utils.NormalizeTelegramURL(body.ButtonURL)
	if err := s.validateButtonData(body); err != nil {
		return nil, err
	}

	position, err := s.calculateNextCustomButtonPosition(ctx, captionID)
	if err != nil {
		return nil, err
	}

	// Verificar se a legenda pertence ao canal
	_, err = s.customCaptionRepo.GetCustomCaptionByID(ctx, channelID, captionID)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	newButton := &models.CustomCaptionButton{
		ButtonID:       uuid.NewString(),
		NameButton:     body.NameButton,
		ButtonURL:      body.ButtonURL,
		Style:          body.Style,
		PositionX:      position.X,
		PositionY:      position.Y,
		OwnerCaptionID: captionID,
	}

	if err := s.customCaptionRepo.CreateCustomCaptionButton(ctx, newButton); err != nil {
		return nil, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return newButton, nil
}

func (s *ButtonService) UpdateCustomCaptionButton(ctx context.Context, channelID int64, captionID, buttonID string, body types.ButtonCreateRequest) (int64, error) {
	body.ButtonURL = utils.NormalizeTelegramURL(body.ButtonURL)
	if err := s.validateButtonData(body); err != nil {
		return 0, err
	}

	_, err := s.customCaptionRepo.GetCustomCaptionByID(ctx, channelID, captionID)
	if err != nil {
		return 0, errors.ErrNotFound
	}

	updates := map[string]interface{}{
		"name_button": body.NameButton,
		"button_url":  body.ButtonURL,
		"style":       body.Style,
	}

	rowsAffected, err := s.customCaptionRepo.UpdateCustomCaptionButton(ctx, captionID, buttonID, updates)
	if err != nil {
		return 0, errors.Internal(err)
	}

	if rowsAffected > 0 {
		s.cache.InvalidateChannel(ctx, channelID)
	}

	return rowsAffected, nil
}

func (s *ButtonService) DeleteCustomCaptionButton(ctx context.Context, channelID int64, captionID, buttonID string) (int64, error) {
	_, err := s.customCaptionRepo.GetCustomCaptionByID(ctx, channelID, captionID)
	if err != nil {
		return 0, errors.ErrNotFound
	}

	rowsAffected, err := s.customCaptionRepo.DeleteCustomCaptionButton(ctx, captionID, buttonID)
	if err != nil {
		return 0, errors.Internal(err)
	}

	if rowsAffected > 0 {
		s.cache.InvalidateChannel(ctx, channelID)
	}

	return rowsAffected, nil
}

func (s *ButtonService) UpdateCustomCaptionLayout(ctx context.Context, channelID int64, captionID string, layoutData types.UpdateCustomCaptionLayoutRequest) (int, error) {
	// Verificar se a legenda pertence ao canal
	_, err := s.customCaptionRepo.GetCustomCaptionByID(ctx, channelID, captionID)
	if err != nil {
		return 0, errors.ErrNotFound
	}

	var buttonsToUpdate []struct {
		ID string
		X  int
		Y  int
	}
	for y, row := range layoutData.Layout {
		for x, button := range row {
			buttonsToUpdate = append(buttonsToUpdate, struct {
				ID string
				X  int
				Y  int
			}{ID: button.ID, X: x, Y: y})
		}
	}

	err = s.customCaptionRepo.UpdateCustomCaptionLayout(ctx, captionID, buttonsToUpdate)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return len(buttonsToUpdate), nil
}

// --- AUXILIARES ---

func (s *ButtonService) calculateNextButtonPosition(ctx context.Context, channelId int64) (*types.ButtonPosition, error) {
	channel, err := s.channelRepo.GetChannelByIDLight(ctx, channelId)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	buttons, err := s.channelRepo.GetChannelButtons(ctx, channelId)
	if err != nil {
		return nil, errors.Internal(err)
	}

	return s.getGridPosition(buttons, channel.ReactionPosition), nil
}

func (s *ButtonService) calculateNextCustomButtonPosition(ctx context.Context, captionID string) (*types.ButtonPosition, error) {
	buttons, err := s.customCaptionRepo.GetCustomCaptionButtons(ctx, captionID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	// Legendass customizadas não têm linha de reações reservada no grid
	return s.getGridPositionGeneric(buttons, -1), nil
}

func (s *ButtonService) getGridPosition(buttons []models.Button, skipY int) *types.ButtonPosition {
	if len(buttons) == 0 {
		y := 0
		if skipY == 0 {
			y = 1
		}
		return &types.ButtonPosition{X: 0, Y: y}
	}

	maxY := 0
	for _, b := range buttons {
		if b.PositionY > maxY {
			maxY = b.PositionY
		}
	}

	countInLastRow := 0
	for _, b := range buttons {
		if b.PositionY == maxY {
			countInLastRow++
		}
	}

	if countInLastRow < 3 {
		return &types.ButtonPosition{X: countInLastRow, Y: maxY}
	}

	nextY := maxY + 1
	if nextY == skipY {
		nextY++
	}
	return &types.ButtonPosition{X: 0, Y: nextY}
}

func (s *ButtonService) getGridPositionGeneric(buttons []models.CustomCaptionButton, skipY int) *types.ButtonPosition {
	if len(buttons) == 0 {
		return &types.ButtonPosition{X: 0, Y: 0}
	}

	maxY := 0
	for _, b := range buttons {
		if b.PositionY > maxY {
			maxY = b.PositionY
		}
	}

	countInLastRow := 0
	for _, b := range buttons {
		if b.PositionY == maxY {
			countInLastRow++
		}
	}

	if countInLastRow < 3 {
		return &types.ButtonPosition{X: countInLastRow, Y: maxY}
	}

	return &types.ButtonPosition{X: 0, Y: maxY + 1}
}

func (s *ButtonService) validateButtonData(data types.ButtonCreateRequest) error {
	if strings.TrimSpace(data.NameButton) == "" {
		return errors.BadRequest("O nome do botão é obrigatório")
	}

	if len(data.NameButton) > 64 {
		return errors.BadRequest("Nome do botão muito longo")
	}

	if data.ButtonURL != "" && !utils.IsValidButtonURL(data.ButtonURL) {
		return errors.BadRequest("URL inválida")
	}

	return nil
}
