package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

// PremiumFeatureService gerencia as features premium configuraveis por admin.
// Cada feature tem um identificador unico (key), nome, descricao e um flag enabled
// que determina se esta disponivel globalmente para usuarios premium.
type PremiumFeatureService struct {
	repo *repositories.PremiumFeatureRepository
}

// NewPremiumFeatureService cria um novo servico de features premium.
func NewPremiumFeatureService(repo *repositories.PremiumFeatureRepository) *PremiumFeatureService {
	return &PremiumFeatureService{repo: repo}
}

// ListFeatures retorna todas as features premium cadastradas.
func (s *PremiumFeatureService) ListFeatures(ctx context.Context) ([]models.PremiumFeature, error) {
	return s.repo.List(ctx)
}

// ListEnabledFeatures retorna apenas as features globalmente habilitadas.
func (s *PremiumFeatureService) ListEnabledFeatures(ctx context.Context) ([]models.PremiumFeature, error) {
	return s.repo.ListEnabled(ctx)
}

// GetFeature retorna uma feature pela chave.
func (s *PremiumFeatureService) GetFeature(ctx context.Context, key string) (*models.PremiumFeature, error) {
	return s.repo.GetByKey(ctx, key)
}

// ToggleFeature ativa ou desativa uma feature globalmente.
func (s *PremiumFeatureService) ToggleFeature(ctx context.Context, key string, enabled bool) error {
	return s.repo.Toggle(ctx, key, enabled)
}

// UpdateFeature atualiza nome, descricao e preco de uma feature.
func (s *PremiumFeatureService) UpdateFeature(ctx context.Context, feature *models.PremiumFeature) error {
	existing, err := s.repo.GetByKey(ctx, feature.Key)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.ErrNotFound
	}

	existing.Name = feature.Name
	existing.Description = feature.Description
	existing.Price = feature.Price
	existing.Enabled = feature.Enabled

	return s.repo.Upsert(ctx, existing)
}

// IsFeatureEnabled verifica se uma feature especifica esta habilitada globalmente.
func (s *PremiumFeatureService) IsFeatureEnabled(ctx context.Context, key string) bool {
	ok, err := s.repo.IsEnabled(ctx, key)
	return err == nil && ok
}

// IsPremiumEnabled verifica se pelo menos uma feature premium esta habilitada.
// Se nenhuma feature estiver ativa, o sistema premium como todo deve ser desconsiderado.
func (s *PremiumFeatureService) IsPremiumEnabled(ctx context.Context) bool {
	features, err := s.repo.ListEnabled(ctx)
	return err == nil && len(features) > 0
}

// CalculateBasePrice calcula o preco base da assinatura somando os precos de
// todas as features habilitadas que possuem preco > 0.
func (s *PremiumFeatureService) CalculateBasePrice(ctx context.Context) (int, error) {
	features, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, f := range features {
		if f.Price > 0 {
			total += f.Price
		}
	}
	return total, nil
}

// GetExtraChannelPrice retorna o preco por canal extra configurado na feature.
// Se a feature "extra_channels" nao estiver habilitada ou nao existir, retorna 0.
func (s *PremiumFeatureService) GetExtraChannelPrice(ctx context.Context) (int, error) {
	feature, err := s.repo.GetByKey(ctx, "extra_channels")
	if err != nil {
		return 0, err
	}
	if feature == nil || !feature.Enabled {
		return 0, nil
	}
	return feature.Price, nil
}
