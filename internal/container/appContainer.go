package container

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/internal/database"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/telegram/executor"
	mtprotoAuth "github.com/leirbagxis/FreddyBot/internal/telegram/mtproto/auth"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

// accountSaverAdapter adapta *services.ConnectedAccountService para a interface
// mtprotoAuth.AccountSaver, ignorando o retorno do modelo no SaveSession.
type accountSaverAdapter struct {
	svc *services.ConnectedAccountService
}

func (a *accountSaverAdapter) SaveSession(ctx context.Context, userID int64, telegramUserID int64, username string, firstName string, sessionData []byte) error {
	_, err := a.svc.SaveSession(ctx, userID, telegramUserID, username, firstName, sessionData)
	return err
}

// providerAdapter adapta *services.ConnectedAccountService e *services.SubscriptionService
// para executor.Provider.
type providerAdapter struct {
	connSvc *services.ConnectedAccountService
	subSvc  *services.SubscriptionService
}

func (a *providerAdapter) HasConnectedAccount(ctx context.Context, userID int64) bool {
	return a.connSvc.HasActiveAccount(ctx, userID)
}

func (a *providerAdapter) HasPremiumManagedAccount(ctx context.Context, userID int64) bool {
	return a.subSvc != nil && a.subSvc.UserHasFeature(ctx, userID, "managed_premium_account")
}

// adminSessionAdapter adapta *services.AdminAccountService para executor.AdminSessionProvider.
type adminSessionAdapter struct {
	svc *services.AdminAccountService
}

func (a *adminSessionAdapter) GetAdminSession(ctx context.Context) ([]byte, string, error) {
	return a.svc.GetAdminSession(ctx)
}

type BroadcastButton struct {
	Text  string
	Type  string
	Value string
}

type BroadcastJob struct {
	ChatID   int64
	Text     string
	ImageUrl string
	Buttons  []BroadcastButton
}

type AppContainer struct {
	DB        *gorm.DB
	TelegoBot *telego.Bot

	BroadcastQueue chan BroadcastJob

	// ## SERVICES ## \\
	UserService          *services.UserService
	ChannelService       *services.ChannelService
	ButtonService        *services.ButtonService
	CaptionService       *services.CaptionService
	PermissionsService   *services.PermissionsService
	CustomCaptionService *services.CustomCaptionService
	SeparatorService     *services.SeparatorService
	VoteService          *services.VoteService
	ServerService        *services.ServerService
	ChannelEventService  *services.ChannelEventService
	EmojiService         *services.EmojiService

	// ## MTProto / CONNECTED ACCOUNTS ## \\
	ConnectedAccountService *services.ConnectedAccountService
	AdminAccountService     *services.AdminAccountService
	MTProtoAuthService      *mtprotoAuth.Service
	BotAPIExecutor          *executor.BotAPIExecutor
	ExecutorFactory         *executor.ExecutorFactory

	// ## SUBSCRIPTION / PREMIUM ## \\
	SubscriptionService   *services.SubscriptionService
	PremiumFeatureService *services.PremiumFeatureService

	// ## SCHEDULER & AUTODELETE ## \\
	SchedulerService           *services.SchedulerService
	AutoDeleteService          *services.AutoDeleteService
	PostTemplateService        *services.UserPostTemplateService
	UserCaptionTemplateService *services.UserCaptionTemplateService

	// ## CACHE ## \\
	CacheService   *cache.Service
	SessionManager *cache.SessionManager

	startOnce sync.Once
}

func NewAppContainer(db *gorm.DB, telegoClient *telego.Bot) *AppContainer {
	cacheService := cache.NewService()

	// Repositories (Removed cache from repositories)
	userRepo := repositories.NewUserRepository(db)
	channelRepo := repositories.NewChannelRepository(db)
	buttonRepo := repositories.NewButtonRepository(db)
	separatorRepo := repositories.NewSeparatorRepository(db)
	voteRepo := repositories.NewVoteRepository(db)
	customCaptionRepo := repositories.NewCustomCaptionRepository(db)
	permissionsRepo := repositories.NewPermissionsRepository(db)
	serverRepo := repositories.NewServerConfigRepository(db)
	channelEventRepo := repositories.NewChannelEventRepository(db)
	emojiRepo := repositories.NewEmojiRepository(db)

	// MTProto Repositories
	connectedAccountRepo := repositories.NewConnectedAccountRepository(db)

	// Services
	userService := services.NewUserService(userRepo)
	channelService := services.NewChannelService(channelRepo, userRepo, separatorRepo, cacheService, telegoClient)
	buttonService := services.NewButtonService(buttonRepo, channelRepo, customCaptionRepo, cacheService)
	captionService := services.NewCaptionService(channelRepo, buttonRepo, cacheService)
	permissionsService := services.NewPermissionsService(permissionsRepo, channelRepo, cacheService)
	customCaptionService := services.NewCustomCaptionService(customCaptionRepo, channelRepo, cacheService)
	separatorService := services.NewSeparatorService(separatorRepo)
	voteService := services.NewVoteService(voteRepo)
	serverService := services.NewServerService(serverRepo)
	channelEventService := services.NewChannelEventService(channelEventRepo, serverService.IsLogsEnabled)

	// Emoji Service
	emojiService := services.NewEmojiService(emojiRepo, telegoClient)

	// MTProto Services
	connectedAccountService := services.NewConnectedAccountService(connectedAccountRepo)

	// Redis client for auth state
	redisClient := cache.GetRedisClient()

	// MTProto Auth Service - uses AppID/AppHash from config (placeholder: 0/"" until configured)
	mtprotoAppID := config.GetMTProtoAppID()
	mtprotoAppHash := config.GetMTProtoAppHash()

	// Admin MTProto Service
	adminAccountRepo := repositories.NewAdminAccountRepository(db)
	adminAccountService := services.NewAdminAccountService(adminAccountRepo, redisClient, mtprotoAppID, mtprotoAppHash)

	// Premium Feature Service (modular — admin pode ativar/desativar)
	premiumFeatureRepo := repositories.NewPremiumFeatureRepository(db)
	premiumFeatureService := services.NewPremiumFeatureService(premiumFeatureRepo)

	// Subscription Service
	subscriptionRepo := repositories.NewSubscriptionRepository(db)
	paymentIntentRepo := repositories.NewPaymentIntentRepository(db)
	subscriptionService := services.NewSubscriptionService(subscriptionRepo, paymentIntentRepo, userRepo, telegoClient, premiumFeatureService)

	// AutoDelete Service
	autoDeleteRepo := repositories.NewAutoDeleteRepository(db)
	autoDeleteService := services.NewAutoDeleteService(autoDeleteRepo, telegoClient)

	// Scheduler Service
	schedulerQueue := cache.NewRedisSchedulerQueue()
	scheduledPostRepo := repositories.NewScheduledPostRepository(db)
	schedulerService := services.NewSchedulerService(scheduledPostRepo, channelRepo, cacheService, telegoClient, schedulerQueue)
	schedulerService.SetAutoDeleteService(autoDeleteService)

	postTemplateRepo := repositories.NewUserPostTemplateRepository(db)
	postTemplateService := services.NewUserPostTemplateService(postTemplateRepo)

	// User Caption Template Service
	userCaptionTemplateRepo := repositories.NewUserCaptionTemplateRepository(db)
	userCaptionTemplateService := services.NewUserCaptionTemplateService(userCaptionTemplateRepo)

	saverAdapter := &accountSaverAdapter{svc: connectedAccountService}
	mtprotoAuthService := mtprotoAuth.NewService(redisClient, mtprotoAppID, mtprotoAppHash, saverAdapter)

	// Executors
	botAPIExecutor := executor.NewBotAPIExecutor(telegoClient)

	// MTProto Executor (opcional — so cria se AppID estiver configurado)
	var mtprotoExecutor *executor.MTProtoExecutor
	if mtprotoAppID > 0 && mtprotoAppHash != "" {
		mtprotoExecutor = executor.NewMTProtoExecutor(mtprotoAppID, mtprotoAppHash, connectedAccountService, connectedAccountService)
		logger.Bot("🏗️ MTProtoExecutor criado (AppID: %d)", mtprotoAppID)
	} else {
		logger.Bot("⚠️ MTProtoExecutor nao criado: AppID ou AppHash ausentes")
	}

	// Factory
	provAdapter := &providerAdapter{connSvc: connectedAccountService, subSvc: subscriptionService}
	var adminSP executor.AdminSessionProvider
	if mtprotoExecutor != nil {
		adminSP = &adminSessionAdapter{svc: adminAccountService}
	}
	executorFactory := executor.NewExecutorFactory(botAPIExecutor, mtprotoExecutor, adminSP, provAdapter)

	container := &AppContainer{
		DB:        db,
		TelegoBot: telegoClient,

		BroadcastQueue: make(chan BroadcastJob, 10000),

		// Services
		UserService:          userService,
		ChannelService:       channelService,
		ButtonService:        buttonService,
		CaptionService:       captionService,
		PermissionsService:   permissionsService,
		CustomCaptionService: customCaptionService,
		SeparatorService:     separatorService,
		VoteService:          voteService,
		ServerService:        serverService,
		ChannelEventService:  channelEventService,

		// Emoji
		EmojiService: emojiService,

		// MTProto
		ConnectedAccountService: connectedAccountService,
		AdminAccountService:     adminAccountService,
		MTProtoAuthService:      mtprotoAuthService,
		BotAPIExecutor:          botAPIExecutor,
		ExecutorFactory:         executorFactory,

		// Subscription / Premium
		SubscriptionService:   subscriptionService,
		PremiumFeatureService: premiumFeatureService,

		// Scheduler & AutoDelete
		SchedulerService:           schedulerService,
		AutoDeleteService:          autoDeleteService,
		PostTemplateService:        postTemplateService,
		UserCaptionTemplateService: userCaptionTemplateService,

		CacheService:   cacheService,
		SessionManager: cache.NewSessionManager(cacheService),
	}

	return container
}

// StartBackground inicia os workers que devem existir uma unica vez por processo.
// O container e compartilhado pela API e pelo bot para que dois schedulers nao
// processem a mesma postagem.
func (c *AppContainer) StartBackground(ctx context.Context) {
	c.startOnce.Do(func() {
		c.syncFixedPostBuilderSession(ctx)
		go c.ChannelEventService.StartCleanupScheduler(ctx, c.ServerService.GetLogRetentionDays)
		c.startBroadcastWorkers(ctx, 5)

		// Reconstruir fila Redis no startup a partir dos registros pendentes no PostgreSQL
		if err := c.SchedulerService.RebuildQueue(ctx); err != nil {
			logger.Error("APP", "Erro ao reconstruir fila do scheduler: %v", err)
		}

		go c.SchedulerService.Start(ctx)
		go c.AutoDeleteService.Start(ctx)
		go c.SubscriptionService.StartMaintenance(ctx)
	})
}

// HasPremiumAccess verifica se o usuario tem acesso a recursos premium,
// seja por assinatura ativa ou conta Telegram conectada (se a feature estiver habilitada).
func (c *AppContainer) HasPremiumAccess(ctx context.Context, userID int64) bool {
	status, err := c.SubscriptionService.GetStatus(ctx, userID)
	if err == nil && status != nil && status.HasSubscription {
		return true
	}
	// Conta conectada só concede acesso se a feature "connected_account" estiver habilitada
	if c.PremiumFeatureService.IsFeatureEnabled(ctx, "connected_account") {
		return c.ConnectedAccountService.HasActiveAccount(ctx, userID)
	}
	return false
}

func (c *AppContainer) syncFixedPostBuilderSession(ctx context.Context) {
	config, err := c.ServerService.GetConfig(ctx)
	if err != nil {
		logger.Error("APP", "Erro ao carregar PostBuilder fixo: %v", err)
		return
	}

	if config.FixedPostBuilderKey == "" {
		return
	}

	if !config.FixedPostBuilderEnabled {
		_ = c.CacheService.DeletePostBuilderSession(ctx, config.FixedPostBuilderKey)
		return
	}

	var state cache.PostBuilderState
	if err := json.Unmarshal([]byte(config.FixedPostBuilderPayload), &state); err != nil {
		logger.Warn("APP", "Payload do PostBuilder fixo inválida, restaurando padrão: %v", err)
		config.FixedPostBuilderPayload = database.DefaultFixedPostBuilderPayload()
		config.FixedPostBuilderEnabled = true
		if err := c.ServerService.SaveConfig(ctx, config); err != nil {
			logger.Error("APP", "Erro ao salvar reparo do PostBuilder fixo: %v", err)
			return
		}
		if err := json.Unmarshal([]byte(config.FixedPostBuilderPayload), &state); err != nil {
			logger.Error("APP", "Payload padrão do PostBuilder fixo inválida: %v", err)
			return
		}
	}

	if err := c.CacheService.SetPostBuilderSession(ctx, config.FixedPostBuilderKey, state, 0); err != nil {
		logger.Error("APP", "Erro ao sincronizar PostBuilder fixo no Redis: %v", err)
	}
}

func (c *AppContainer) startBroadcastWorkers(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go c.broadcastWorker(ctx)
	}
}

func (c *AppContainer) broadcastWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-c.BroadcastQueue:
			if !ok {
				return
			}
			var keyboard [][]telego.InlineKeyboardButton
			var replyMarkup *telego.InlineKeyboardMarkup

			if len(job.Buttons) > 0 {
				for _, btn := range job.Buttons {
					button := telego.InlineKeyboardButton{
						Text: btn.Text,
					}

					if btn.Type == "url" {
						button.URL = btn.Value
					} else if btn.Type == "callback" {
						button.CallbackData = btn.Value
					}

					keyboard = append(keyboard, []telego.InlineKeyboardButton{button})
				}
				replyMarkup = &telego.InlineKeyboardMarkup{
					InlineKeyboard: keyboard,
				}
			}

			var err error
			if job.ImageUrl != "" {
				params := &telego.SendPhotoParams{
					ChatID:    telego.ChatID{ID: job.ChatID},
					Photo:     telego.InputFile{URL: job.ImageUrl},
					Caption:   job.Text,
					ParseMode: telego.ModeHTML,
				}
				if replyMarkup != nil {
					params.ReplyMarkup = replyMarkup
				}
				_, err = c.TelegoBot.SendPhoto(ctx, params)
			} else {
				params := &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: job.ChatID},
					Text:      job.Text,
					ParseMode: telego.ModeHTML,
				}
				if replyMarkup != nil {
					params.ReplyMarkup = replyMarkup
				}
				_, err = c.TelegoBot.SendMessage(ctx, params)
			}

			if err != nil {
				logger.Error("APP", "Erro ao enviar para %d: %v", job.ChatID, err)
				continue
			}

			// 🔥 Controle de rate limit global
			time.Sleep(35 * time.Millisecond)
		}
	}
}
