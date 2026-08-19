package services

import (
	"context"
	"fmt"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/crypto"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type UserPostTemplateService struct {
	repo      *repositories.UserPostTemplateRepository
	secretKey string
}

func NewUserPostTemplateService(repo *repositories.UserPostTemplateRepository) *UserPostTemplateService {
	return &UserPostTemplateService{
		repo:      repo,
		secretKey: "freddybot-secure-draft-key-aes256",
	}
}

func (s *UserPostTemplateService) SetSecretKey(key string) {
	if key != "" {
		s.secretKey = key
	}
}

func (s *UserPostTemplateService) SaveTemplate(ctx context.Context, ownerID int64, name, templateData string) (*models.UserPostTemplate, error) {
	encryptedData, err := crypto.CompressAndEncrypt([]byte(templateData), s.secretKey)
	if err != nil {
		return nil, errors.Internal(err)
	}

	template := &models.UserPostTemplate{
		ID:           fmt.Sprintf("tpl_%d_%d", time.Now().UnixNano(), ownerID),
		OwnerID:      ownerID,
		Name:         name,
		TemplateData: encryptedData,
	}

	if err := s.repo.Create(ctx, template); err != nil {
		return nil, errors.Internal(err)
	}

	// Retorna com os dados descomprimidos/decifrados para o chamador
	template.TemplateData = templateData
	return template, nil
}

func (s *UserPostTemplateService) ListTemplates(ctx context.Context, ownerID int64) ([]models.UserPostTemplate, error) {
	templates, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	for i := range templates {
		decryptedBytes, err := crypto.DecryptAndDecompress(templates[i].TemplateData, s.secretKey)
		if err == nil {
			templates[i].TemplateData = string(decryptedBytes)
		}
	}

	return templates, nil
}

func (s *UserPostTemplateService) DeleteTemplate(ctx context.Context, id string, ownerID int64) error {
	return s.repo.Delete(ctx, id, ownerID)
}

func (s *UserPostTemplateService) GetTemplateByID(ctx context.Context, id string) (*models.UserPostTemplate, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	decryptedBytes, err := crypto.DecryptAndDecompress(tpl.TemplateData, s.secretKey)
	if err == nil {
		tpl.TemplateData = string(decryptedBytes)
	}

	return tpl, nil
}

func (s *UserPostTemplateService) UpdateTemplateName(ctx context.Context, id string, ownerID int64, newName string) error {
	return s.repo.UpdateName(ctx, id, ownerID, newName)
}
