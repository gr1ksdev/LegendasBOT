package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

// ── Constantes ──

const (
	// SubscriptionPeriodDays e a duracao de cada periodo de assinatura.
	SubscriptionPeriodDays = 30

	// InvoicePayloadPrefix e o prefixo do payload da invoice para identificar.
	InvoicePayloadPrefix = "premium_sub:"

	// InvoiceExtraPayloadPrefix e o prefixo para invoices de canais extras.
	InvoiceExtraPayloadPrefix = "premium_extra:"

	maxPremiumChannels = 100
)

// ── SubscriptionService ──

// SubscriptionService gerencia o ciclo de vida das assinaturas premium.
type SubscriptionService struct {
	subRepo           *repositories.SubscriptionRepository
	paymentIntentRepo *repositories.PaymentIntentRepository
	userRepo          *repositories.UserRepository
	bot               *telego.Bot
	featureSvc        *PremiumFeatureService
}

// NewSubscriptionService cria um novo servico de assinaturas.
func NewSubscriptionService(
	subRepo *repositories.SubscriptionRepository,
	paymentIntentRepo *repositories.PaymentIntentRepository,
	userRepo *repositories.UserRepository,
	bot *telego.Bot,
	featureSvc *PremiumFeatureService,
) *SubscriptionService {
	return &SubscriptionService{
		subRepo:           subRepo,
		paymentIntentRepo: paymentIntentRepo,
		userRepo:          userRepo,
		bot:               bot,
		featureSvc:        featureSvc,
	}
}

// ── DTOs ──

// InvoiceResult representa o resultado da criacao de uma invoice.
type InvoiceResult struct {
	InvoiceURL string `json:"invoiceUrl,omitempty"`
	Payload    string `json:"payload"`
	TotalStars int    `json:"totalStars"`
}

// SubscriptionStatusDTO e o DTO retornado para o frontend.
type SubscriptionStatusDTO struct {
	HasSubscription         bool                 `json:"hasSubscription"`
	Subscription            *models.Subscription `json:"subscription,omitempty"`
	Features                *models.UserFeatures `json:"features,omitempty"`
	BasePrice               int                  `json:"basePrice"`
	ExtraChannelPrice       int                  `json:"extraChannelPrice"`
	StarsTestMode           bool                 `json:"starsTestMode"`
	PremiumEnabled          bool                 `json:"premiumEnabled"`
	ConnectedAccountEnabled bool                 `json:"connectedAccountEnabled"`
}

// ── Metodos Publicos ──

// GetStatus retorna o status da assinatura de um usuario com precos dinamicos.
func (s *SubscriptionService) GetStatus(ctx context.Context, userID int64) (*SubscriptionStatusDTO, error) {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	features, err := s.GetUserFeatures(ctx, userID)
	if err != nil {
		return nil, errors.Internal(err)
	}

	basePrice, _ := s.featureSvc.CalculateBasePrice(ctx)
	extraPrice, _ := s.featureSvc.GetExtraChannelPrice(ctx)

	dto := &SubscriptionStatusDTO{
		HasSubscription:         isSubscriptionActive(sub, time.Now()),
		Subscription:            sub,
		Features:                features,
		BasePrice:               basePrice,
		ExtraChannelPrice:       extraPrice,
		StarsTestMode:           config.StarsTestMode,
		PremiumEnabled:          s.featureSvc.IsPremiumEnabled(ctx),
		ConnectedAccountEnabled: s.featureSvc.IsFeatureEnabled(ctx, "connected_account"),
	}

	logger.Bot("📋 GetStatus userID=%d: starsTestMode=%v hasSubscription=%v", userID, dto.StarsTestMode, dto.HasSubscription)

	return dto, nil
}

// CreateInvoice cria uma invoice do Telegram Stars para assinatura premium.
// Se o usuario ja tiver uma assinatura ativa, retorna erro. Se tiver uma
// expirada ou cancelada, renova.
// Se testMode=true e AppEnv for "dev", ativa a assinatura direto sem cobrar Stars.
// channelCount e o numero de canais que o usuario quer incluir no premium.
func (s *SubscriptionService) CreateInvoice(ctx context.Context, userID int64, testMode bool, channelCount int) (*InvoiceResult, error) {
	if channelCount < 1 || channelCount > maxPremiumChannels {
		return nil, errors.BadRequest("quantidade de canais premium inválida")
	}
	// Verificar se ja existe assinatura ativa
	existing, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	if isSubscriptionActive(existing, time.Now()) {
		return nil, errors.ErrConflict
	}

	// Calcular total com precos dinamicos
	// channelCount = total de canais que o usuario quer incluir
	// extraChannels = canais alem do primeiro
	basePrice, _ := s.featureSvc.CalculateBasePrice(ctx)
	extraPrice, _ := s.featureSvc.GetExtraChannelPrice(ctx)

	extraChannels := 0
	if channelCount > 1 {
		extraChannels = channelCount - 1
	}
	// Se ja existia subscription (expirada/cancelada), usa o maior valor
	if existing != nil && existing.ExtraChannels > extraChannels {
		extraChannels = existing.ExtraChannels
	}

	totalStars := basePrice + extraChannels*extraPrice

	logger.Bot("💰 CreateInvoice: userID=%d testMode=%v StarsTestMode=%v channels=%d extraChannels=%d total=%d",
		userID, testMode, config.StarsTestMode, channelCount, extraChannels, totalStars)

	// ── Modo teste: precos sao 1 star para testar fluxo real de pagamento ──
	if testMode && config.StarsTestMode {
		originalTotal := totalStars
		totalStars = 1
		logger.Bot("🧪 Modo teste: preco ajustado para 1 star (original era %d)", originalTotal)
	}
	payload, err := s.createPaymentIntent(ctx, userID, models.PaymentIntentSubscription, extraChannels, totalStars)
	if err != nil {
		return nil, errors.Internal(err)
	}

	// ── Criar invoice link ──
	params := &telego.CreateInvoiceLinkParams{
		Title:         "LegendasBr Premium",
		Description:   "Assinatura mensal do LegendasBr BOT com recursos premium.",
		Payload:       payload,
		ProviderToken: "",    // String vazia = Telegram Stars
		Currency:      "XTR", // Telegram Stars
		Prices: []telego.LabeledPrice{
			{Label: "LegendasBr Premium", Amount: totalStars},
		},
	}

	invoiceLink, err := s.bot.CreateInvoiceLink(ctx, params)
	if err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao criar invoice link para %d: %v", userID, err)
		return nil, errors.Internal(fmt.Errorf("create invoice link: %w", err))
	}

	logger.Bot("🧾 Invoice link criado para %d: %d stars (payload=%s)", userID, totalStars, utils.TruncateString(payload, 20))

	return &InvoiceResult{
		InvoiceURL: *invoiceLink,
		Payload:    payload,
		TotalStars: totalStars,
	}, nil
}

func (s *SubscriptionService) createPaymentIntent(ctx context.Context, userID int64, intentType string, extraChannels, amountStars int) (string, error) {
	if amountStars <= 0 || extraChannels < 0 {
		return "", fmt.Errorf("invalid payment intent amount or channels")
	}
	id := uuid.New().String()
	payload := InvoicePayloadPrefix + id
	if intentType == models.PaymentIntentExtraChannel {
		payload = InvoiceExtraPayloadPrefix + id
	}
	intent := &models.PaymentIntent{
		ID:            id,
		Payload:       payload,
		UserID:        userID,
		Type:          intentType,
		ExtraChannels: extraChannels,
		AmountStars:   amountStars,
		Status:        models.PaymentIntentPending,
		ExpiresAt:     time.Now().Add(30 * time.Minute),
	}
	if err := s.paymentIntentRepo.Create(ctx, intent); err != nil {
		return "", err
	}
	return payload, nil
}

func isSubscriptionActive(sub *models.Subscription, now time.Time) bool {
	return sub != nil && sub.Status == models.SubscriptionActive && now.Before(sub.CurrentPeriodEnd)
}

// activateSubscription ativa ou renova uma assinatura (usado tanto no pagamento real quanto no teste).
func (s *SubscriptionService) activateSubscription(ctx context.Context, userID int64, existing *models.Subscription, chargeID string, totalStars int, extraChannels int) error {
	return s.activateSubscriptionWithRepo(ctx, s.subRepo, userID, existing, chargeID, totalStars, extraChannels)
}

func (s *SubscriptionService) activateSubscriptionWithRepo(ctx context.Context, repo *repositories.SubscriptionRepository, userID int64, existing *models.Subscription, chargeID string, totalStars int, extraChannels int) error {
	now := time.Now()
	periodEnd := now.AddDate(0, 0, SubscriptionPeriodDays)

	if existing != nil {
		existing.Status = models.SubscriptionActive
		existing.CurrentPeriodStart = now
		existing.CurrentPeriodEnd = periodEnd
		existing.TelegramPaymentID = chargeID
		existing.CancelAtPeriodEnd = false
		existing.UpdatedAt = now
		existing.ExtraChannels = extraChannels

		if err := repo.Update(ctx, existing); err != nil {
			return errors.Internal(err)
		}
		logger.Bot("🔄 Assinatura renovada: user=%d ate %s extraChannels=%d", userID, periodEnd.Format("2006-01-02"), extraChannels)
	} else {
		sub := &models.Subscription{
			ID:                 uuid.New().String(),
			UserID:             userID,
			Status:             models.SubscriptionActive,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   periodEnd,
			ExtraChannels:      extraChannels,
			CancelAtPeriodEnd:  false,
			TelegramPaymentID:  chargeID,
		}

		if err := repo.Create(ctx, sub); err != nil {
			return errors.Internal(err)
		}
		logger.Bot("🎉 Nova assinatura: user=%d ate %s extraChannels=%d", userID, periodEnd.Format("2006-01-02"), extraChannels)
	}

	return nil
}

// HandlePreCheckout aprova ou rejeita uma pre-checkout query.
// Sempre aprova para payloads validos.
func (s *SubscriptionService) HandlePreCheckout(ctx context.Context, query *telego.PreCheckoutQuery) error {
	userID := query.From.ID
	payload := query.InvoicePayload

	logger.Bot("💳 PreCheckoutQuery: user=%d payload=%s amount=%d", userID, utils.TruncateString(payload, 20), query.TotalAmount)

	validPayload, err := s.paymentIntentRepo.IsPendingForPayment(ctx, payload, userID, query.TotalAmount, time.Now())
	if err != nil {
		return errors.Internal(err)
	}
	if !validPayload {
		logger.Warn("SUBSCRIPTION", "Payload invalido: user=%d payload=%s", userID, payload)
		return s.bot.AnswerPreCheckoutQuery(ctx, &telego.AnswerPreCheckoutQueryParams{
			PreCheckoutQueryID: query.ID,
			Ok:                 false,
			ErrorMessage:       "Erro de validação. Tente novamente.",
		})
	}

	// Aprovar
	return s.bot.AnswerPreCheckoutQuery(ctx, &telego.AnswerPreCheckoutQueryParams{
		PreCheckoutQueryID: query.ID,
		Ok:                 true,
	})
}

// HandlePayment ativa a assinatura apos um pagamento bem-sucedido.
func (s *SubscriptionService) HandlePayment(ctx context.Context, userID int64, payment *telego.SuccessfulPayment) error {
	payload := payment.InvoicePayload
	chargeID := payment.TelegramPaymentChargeID

	logger.Bot("💰 SuccessfulPayment: user=%d charge=%s payload=%s amount=%d",
		userID, chargeID, payload, payment.TotalAmount)

	processed, err := s.paymentIntentRepo.Process(ctx, payload, userID, payment.TotalAmount, chargeID, func(tx *gorm.DB, intent *models.PaymentIntent) error {
		repo := s.subRepo.WithTx(tx)
		existing, err := repo.FindByUserID(ctx, userID)
		if err != nil {
			return err
		}
		switch intent.Type {
		case models.PaymentIntentSubscription:
			return s.activateSubscriptionWithRepo(ctx, repo, userID, existing, chargeID, payment.TotalAmount, intent.ExtraChannels)
		case models.PaymentIntentExtraChannel:
			return s.addExtraChannelWithChargeWithRepo(ctx, repo, userID, chargeID)
		default:
			return repositories.ErrInvalidPaymentIntent
		}
	})
	if err != nil {
		return errors.BadRequest("pagamento inválido ou expirado")
	}
	if processed {
		if err := s.SyncFeatures(ctx, userID); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao sincronizar features apos pagamento: %v", err)
		}
	}
	return nil
}

// AdminListSubscriptions retorna todas as assinaturas para o painel admin.
func (s *SubscriptionService) AdminListSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	return s.subRepo.FindAll(ctx)
}

// AdminCancelSubscription cancela/expiura a assinatura de um usuario.
// Se instant=true, a assinatura e expirada imediatamente (status=expired, period end=now).
// Se instant=false, apenas marca cancelAtPeriodEnd=true (comportamento normal).
func (s *SubscriptionService) AdminCancelSubscription(ctx context.Context, userID int64, instant bool) error {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) {
		return errors.ErrNotFound
	}

	if instant {
		sub.Status = models.SubscriptionExpired
		sub.CurrentPeriodEnd = time.Now()
		sub.UpdatedAt = time.Now()
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return errors.Internal(err)
		}
		// Limpar features imediatamente
		if err := s.SyncFeatures(ctx, userID); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao limpar features apos cancelamento admin: %v", err)
		}
		logger.Bot("⚡ Admin cancelou assinatura de %d (instantaneo)", userID)
	} else {
		sub.CancelAtPeriodEnd = true
		sub.UpdatedAt = time.Now()
		if err := s.subRepo.Update(ctx, sub); err != nil {
			return errors.Internal(err)
		}
		logger.Bot("🛑 Admin marcou cancelamento de %d (fim do periodo)", userID)
	}

	return nil
}

// AdminRefundPayment reembolsa o pagamento Stars de um usuario e expira a assinatura.
// Reembolsa tanto a assinatura principal quanto todos os canais extras pagos separadamente.
func (s *SubscriptionService) AdminRefundPayment(ctx context.Context, userID int64, chargeID string, adminID int64) error {
	logger.Bot("🔁 AdminRefundPayment: user=%d charge=%s admin=%d", userID, chargeID, adminID)

	// 1. Buscar subscription pelo chargeID
	sub, err := s.subRepo.FindByChargeID(ctx, chargeID)
	if err != nil {
		return errors.Internal(err)
	}
	if sub == nil {
		return errors.BadRequest("assinatura nao encontrada para este charge ID")
	}
	if sub.UserID != userID {
		return errors.BadRequest("charge ID nao pertence a este usuario")
	}

	// 2. Verificar se chargeID ja foi reembolsado
	existingRefund, err := s.subRepo.FindRefundByChargeID(ctx, chargeID)
	if err != nil {
		return errors.Internal(err)
	}
	if existingRefund != nil {
		return errors.BadRequest("este pagamento ja foi reembolsado")
	}

	// 3. Reembolsar assinatura principal
	if err := s.bot.RefundStarPayment(ctx, &telego.RefundStarPaymentParams{
		UserID:                  userID,
		TelegramPaymentChargeID: chargeID,
	}); err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao reembolsar Stars de %d (charge=%s): %v", userID, chargeID, err)
		return errors.Internal(fmt.Errorf("refund star payment: %w", err))
	}
	logger.Bot("✅ Stars reembolsados (assinatura): user=%d charge=%s", userID, chargeID)

	// Criar registro de refund da assinatura principal
	refund := &models.Refund{
		ID:                      uuid.New().String(),
		SubscriptionID:          sub.ID,
		UserID:                  userID,
		TelegramPaymentChargeID: chargeID,
		AmountStars:             0,
		Status:                  "processed",
		RefundedAt:              time.Now(),
		RefundedBy:              adminID,
	}
	if err := s.subRepo.CreateRefund(ctx, refund); err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao salvar registro de refund: %v", err)
	}

	// 4. Reembolsar canais extras pagos separadamente
	if sub.ExtraChannelPayments != "" {
		extraCharges := strings.Split(sub.ExtraChannelPayments, ",")
		for _, extraCharge := range extraCharges {
			extraCharge = strings.TrimSpace(extraCharge)
			if extraCharge == "" {
				continue
			}

			// Verificar se ja foi reembolsado
			existingExtraRefund, _ := s.subRepo.FindRefundByChargeID(ctx, extraCharge)
			if existingExtraRefund != nil {
				logger.Bot("⏭️ Canal extra ja reembolsado: charge=%s", extraCharge)
				continue
			}

			// Reembolsar
			if err := s.bot.RefundStarPayment(ctx, &telego.RefundStarPaymentParams{
				UserID:                  userID,
				TelegramPaymentChargeID: extraCharge,
			}); err != nil {
				logger.Error("SUBSCRIPTION", "Erro ao reembolsar canal extra %s: %v", extraCharge, err)
				continue
			}
			logger.Bot("✅ Stars reembolsados (canal extra): user=%d charge=%s", userID, extraCharge)

			// Criar registro de refund
			extraRefund := &models.Refund{
				ID:                      uuid.New().String(),
				SubscriptionID:          sub.ID,
				UserID:                  userID,
				TelegramPaymentChargeID: extraCharge,
				AmountStars:             0,
				Status:                  "processed",
				RefundedAt:              time.Now(),
				RefundedBy:              adminID,
			}
			if err := s.subRepo.CreateRefund(ctx, extraRefund); err != nil {
				logger.Error("SUBSCRIPTION", "Erro ao salvar refund de canal extra: %v", err)
			}
		}
	}

	// 5. Expirar subscription
	if sub.Status == models.SubscriptionActive {
		sub.Status = models.SubscriptionExpired
		sub.CurrentPeriodEnd = time.Now()
		sub.UpdatedAt = time.Now()
		sub.ExtraChannels = 0
		sub.ExtraChannelPayments = ""
		if err := s.subRepo.Update(ctx, sub); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao expirar subscription apos refund: %v", err)
			return errors.Internal(err)
		}
		// Limpar features
		if err := s.SyncFeatures(ctx, userID); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao limpar features apos refund: %v", err)
		}
		logger.Bot("⚡ Subscription expirada apos refund: user=%d", userID)
	}

	return nil
}

// Cancel marca a assinatura para cancelamento no fim do periodo.
func (s *SubscriptionService) Cancel(ctx context.Context, userID int64) error {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) {
		return errors.ErrNotFound
	}

	sub.CancelAtPeriodEnd = true
	sub.UpdatedAt = time.Now()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return errors.Internal(err)
	}

	logger.Bot("🚫 Assinatura marcada para cancelamento: user=%d (expira %s)",
		userID, sub.CurrentPeriodEnd.Format("2006-01-02"))

	return nil
}

// CreateExtraChannelInvoice cria uma invoice para adicionar um canal extra.
// Retorna a URL da invoice para o usuario pagar via Telegram Stars.
func (s *SubscriptionService) CreateExtraChannelInvoice(ctx context.Context, userID int64, testMode bool) (*InvoiceResult, error) {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) {
		return nil, errors.ErrNotFound
	}

	extraPrice, _ := s.featureSvc.GetExtraChannelPrice(ctx)
	totalStars := extraPrice

	logger.Bot("💰 CreateExtraChannelInvoice: userID=%d testMode=%v StarsTestMode=%v price=%d",
		userID, testMode, config.StarsTestMode, totalStars)

	// Modo teste: preco 1 star
	if testMode && config.StarsTestMode {
		totalStars = 1
		logger.Bot("🧪 Modo teste: preco ajustado para 1 star (original era %d)", extraPrice)
	}
	payload, err := s.createPaymentIntent(ctx, userID, models.PaymentIntentExtraChannel, 1, totalStars)
	if err != nil {
		return nil, errors.Internal(err)
	}

	params := &telego.CreateInvoiceLinkParams{
		Title:         "Canal Extra",
		Description:   "Adicionar um canal extra a sua assinatura premium.",
		Payload:       payload,
		ProviderToken: "",
		Currency:      "XTR",
		Prices: []telego.LabeledPrice{
			{Label: "Canal Extra", Amount: totalStars},
		},
	}

	invoiceLink, err := s.bot.CreateInvoiceLink(ctx, params)
	if err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao criar invoice link extra para %d: %v", userID, err)
		return nil, errors.Internal(fmt.Errorf("create extra invoice link: %w", err))
	}

	logger.Bot("🧾 Invoice link extra criado para %d: %d stars (payload=%s)", userID, totalStars, payload)

	return &InvoiceResult{
		InvoiceURL: *invoiceLink,
		Payload:    payload,
		TotalStars: totalStars,
	}, nil
}

// AddExtraChannel adiciona um canal extra a assinatura.
func (s *SubscriptionService) AddExtraChannel(ctx context.Context, userID int64) error {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) {
		return errors.ErrNotFound
	}

	sub.ExtraChannels++
	sub.UpdatedAt = time.Now()

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return errors.Internal(err)
	}

	if err := s.SyncFeatures(ctx, userID); err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao sincronizar features apos add canal: %v", err)
	}

	logger.Bot("➕ Canal extra adicionado: user=%d total=%d", userID, sub.ExtraChannels)
	return nil
}

// addExtraChannelWithCharge adiciona um canal extra e salva o charge ID para reembolso futuro.
func (s *SubscriptionService) addExtraChannelWithCharge(ctx context.Context, userID int64, chargeID string) error {
	return s.addExtraChannelWithChargeWithRepo(ctx, s.subRepo, userID, chargeID)
}

func (s *SubscriptionService) addExtraChannelWithChargeWithRepo(ctx context.Context, repo *repositories.SubscriptionRepository, userID int64, chargeID string) error {
	sub, err := repo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) {
		return errors.ErrNotFound
	}

	sub.ExtraChannels++
	sub.UpdatedAt = time.Now()

	// Salvar charge ID do extra (separado por virgula)
	if sub.ExtraChannelPayments == "" {
		sub.ExtraChannelPayments = chargeID
	} else {
		sub.ExtraChannelPayments = sub.ExtraChannelPayments + "," + chargeID
	}

	if err := repo.Update(ctx, sub); err != nil {
		return errors.Internal(err)
	}

	logger.Bot("➕ Canal extra adicionado com charge: user=%d total=%d charge=%s", userID, sub.ExtraChannels, chargeID)
	return nil
}

// RemoveExtraChannel remove um canal extra da assinatura.
func (s *SubscriptionService) RemoveExtraChannel(ctx context.Context, userID int64) error {
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}
	if !isSubscriptionActive(sub, time.Now()) || sub.ExtraChannels <= 0 {
		return errors.ErrNotFound
	}

	sub.ExtraChannels--
	sub.UpdatedAt = time.Now()

	// Remover ultimo charge ID (remove o mais recente)
	if sub.ExtraChannelPayments != "" {
		parts := strings.Split(sub.ExtraChannelPayments, ",")
		if len(parts) > 0 {
			parts = parts[:len(parts)-1]
			sub.ExtraChannelPayments = strings.Join(parts, ",")
		}
	}

	if err := s.subRepo.Update(ctx, sub); err != nil {
		return errors.Internal(err)
	}

	if err := s.SyncFeatures(ctx, userID); err != nil {
		logger.Error("SUBSCRIPTION", "Erro ao sincronizar features apos remover canal: %v", err)
	}

	logger.Bot("➖ Canal extra removido: user=%d total=%d", userID, sub.ExtraChannels)
	return nil
}

// StartMaintenance mantem a persistencia e as features alinhadas ao horario
// real da assinatura. Deve ser chamado apenas pelo container compartilhado.
func (s *SubscriptionService) StartMaintenance(ctx context.Context) {
	if err := s.ExpireSubscriptions(ctx); err != nil {
		logger.Error("SUBSCRIPTION", "Erro na expiração inicial: %v", err)
	}
	if err := s.SendRenewalInvoices(ctx); err != nil {
		logger.Error("SUBSCRIPTION", "Erro nos lembretes iniciais: %v", err)
	}

	expirationTicker := time.NewTicker(time.Hour)
	defer expirationTicker.Stop()
	renewalTicker := time.NewTicker(24 * time.Hour)
	defer renewalTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-expirationTicker.C:
			if err := s.ExpireSubscriptions(ctx); err != nil {
				logger.Error("SUBSCRIPTION", "Erro ao expirar assinaturas: %v", err)
			}
		case <-renewalTicker.C:
			if err := s.SendRenewalInvoices(ctx); err != nil {
				logger.Error("SUBSCRIPTION", "Erro ao enviar lembretes: %v", err)
			}
		}
	}
}

// ── Gerenciamento de Features ──

// GetUserFeatures retorna as features atuais do usuario baseadas no JSON armazenado.
func (s *SubscriptionService) GetUserFeatures(ctx context.Context, userID int64) (*models.UserFeatures, error) {
	user, err := s.userRepo.GetUserById(ctx, userID)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	features := &models.UserFeatures{}
	if user.Features != "" {
		if err := json.Unmarshal([]byte(user.Features), features); err != nil {
			// Se der erro no parse, retorna features vazias
			return &models.UserFeatures{}, nil
		}
	}
	return features, nil
}

// SyncFeatures sincroniza o estado da assinatura com o campo Features do User.
// Respeita as features globalmente habilitadas — se o admin desativou uma feature,
// ela nao e concedida mesmo que o usuario tenha assinatura ativa.
func (s *SubscriptionService) SyncFeatures(ctx context.Context, userID int64) error {
	user, err := s.userRepo.GetUserById(ctx, userID)
	if err != nil {
		return errors.ErrNotFound
	}

	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.Internal(err)
	}

	features := &models.UserFeatures{}

	if sub != nil && sub.Status == models.SubscriptionActive {
		if time.Now().Before(sub.CurrentPeriodEnd) {
			// So concede features que estao habilitadas globalmente
			if s.featureSvc.IsFeatureEnabled(ctx, "managed_premium_account") {
				features.ManagedPremiumAccount = true
			}
			if s.featureSvc.IsFeatureEnabled(ctx, "custom_emojis") {
				features.CustomEmojis = true
			}
			// extra_channels sempre sincroniza com o valor da subscription
			// (mesmo se desativado, mantemos o contador para preservar dados)
			features.ExtraChannels = sub.ExtraChannels
		}
	}

	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return errors.Internal(err)
	}

	user.Features = string(featuresJSON)
	user.UpdatedAt = time.Now()

	// Salvar via repo — precisamos atualizar o campo features diretamente
	// Usamos UpdateColumn para evitar salvamento em cascata
	if err := s.userRepo.UpdateFeatures(ctx, userID, user.Features); err != nil {
		return errors.Internal(err)
	}

	return nil
}

// UserHasFeature verifica se um usuario tem uma feature especifica.
// Primeiro verifica se a feature esta habilitada globalmente (admin),
// depois verifica se o usuario possui a feature na assinatura.
func (s *SubscriptionService) UserHasFeature(ctx context.Context, userID int64, feature string) bool {
	// 1. Verificar se a feature existe e esta habilitada globalmente
	if !s.featureSvc.IsFeatureEnabled(ctx, feature) {
		return false
	}

	// 2. A fonte de verdade para acesso e a assinatura e sua data de vencimento;
	// o JSON do usuario e apenas uma projeção que pode aguardar o proximo job.
	sub, err := s.subRepo.FindByUserID(ctx, userID)
	if err != nil || !isSubscriptionActive(sub, time.Now()) {
		return false
	}

	// 3. Verificar se o usuario tem a feature na projeção da assinatura.
	features, err := s.GetUserFeatures(ctx, userID)
	if err != nil {
		return false
	}

	switch feature {
	case "managed_premium_account":
		return features.ManagedPremiumAccount
	case "custom_emojis":
		return features.CustomEmojis
	case "extra_channels":
		return features.ExtraChannels > 0
	default:
		return false
	}
}

// ── Jobs de Manutencao ──

// ExpireSubscriptions expira assinaturas cujo periodo ja terminou.
// Deve ser chamado periodicamente (ex: a cada hora).
func (s *SubscriptionService) ExpireSubscriptions(ctx context.Context) error {
	expired, err := s.subRepo.FindExpired(ctx)
	if err != nil {
		return errors.Internal(err)
	}

	for _, sub := range expired {
		sub.Status = models.SubscriptionExpired
		sub.UpdatedAt = time.Now()

		if err := s.subRepo.Update(ctx, &sub); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao expirar assinatura %s: %v", sub.ID, err)
			continue
		}

		// Remover features
		if err := s.SyncFeatures(ctx, sub.UserID); err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao limpar features de %d: %v", sub.UserID, err)
		}

		logger.Bot("⏰ Assinatura expirada: user=%d", sub.UserID)
	}

	return nil
}

// SendRenewalInvoices envia invoices de renovacao para assinaturas proximas do fim.
// Deve ser chamado periodicamente (ex: uma vez por dia).
func (s *SubscriptionService) SendRenewalInvoices(ctx context.Context) error {
	dueSubs, err := s.subRepo.FindDueForRenewal(ctx)
	if err != nil {
		return errors.Internal(err)
	}

	for _, sub := range dueSubs {
		basePrice, _ := s.featureSvc.CalculateBasePrice(ctx)
		extraPrice, _ := s.featureSvc.GetExtraChannelPrice(ctx)
		totalStars := basePrice + sub.ExtraChannels*extraPrice
		payload, err := s.createPaymentIntent(ctx, sub.UserID, models.PaymentIntentSubscription, sub.ExtraChannels, totalStars)
		if err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao criar intent de renovacao para %d: %v", sub.UserID, err)
			continue
		}

		params := &telego.SendInvoiceParams{
			ChatID:        telego.ChatID{ID: sub.UserID},
			Title:         "LegendasBr Premium - Renovação",
			Description:   "Sua assinatura LegendasBr Premium está próxima do vencimento. Renove agora!",
			Payload:       payload,
			ProviderToken: "",
			Currency:      "XTR",
			Prices: []telego.LabeledPrice{
				{Label: "LegendasBr Premium (Renovação)", Amount: totalStars},
			},
		}

		_, err = s.bot.SendInvoice(ctx, params)
		if err != nil {
			logger.Error("SUBSCRIPTION", "Erro ao enviar invoice de renovacao para %d: %v", sub.UserID, err)
			continue
		}

		logger.Bot("📬 Invoice de renovacao enviada para %d (%d stars)", sub.UserID, totalStars)
	}

	return nil
}
