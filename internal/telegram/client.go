package telegram

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"gorm.io/gorm"
)

const maxWebhookBodyBytes int64 = 1 << 20

func StartBot(ctx context.Context, db *gorm.DB) (http.Handler, *telego.Bot, *container.AppContainer, func(), error) {
	cache.GetRedisClient()

	// Inicializar telego
	tb, err := telego.NewBot(config.TelegramBotToken)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create Telegram bot: %w", err)
	}

	app := container.NewAppContainer(db, tb)

	botInfo, err := tb.GetMe(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("get Telegram bot info: %w", err)
	}
	logger.Bot("🤖 Bot iniciado (Telego): %s", botInfo.Username)

	// Updates channel
	updates := make(chan telego.Update, 1000)
	bh, err := telegohandler.NewBotHandler(tb, updates)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create Telegram handler: %w", err)
	}

	// Custom HTTP Handler for Webhook
	webhookHandler := newWebhookHandler(config.TelegramWebhookSecret, updates)

	// Load Handlers
	LoadHandlersTelegoWithBH(bh, app)

	cleanupFunc := func() {
		logger.Bot("🧹 Encerrando bot handler do Telegram...")
		bh.Stop()
	}

	webhookUrl := config.WebhookURL
	if webhookUrl != "" {
		if config.TelegramWebhookSecret == "" {
			return nil, nil, nil, nil, fmt.Errorf("TELEGRAM_WEBHOOK_SECRET is required when WEBHOOK_URL is configured")
		}
		logger.Bot("🔗 Bot configurado para modo webhook: %s", webhookUrl)

		if err := tb.SetWebhook(ctx, &telego.SetWebhookParams{
			URL:            webhookUrl,
			SecretToken:    config.TelegramWebhookSecret,
			AllowedUpdates: []string{"message", "edited_message", "callback_query", "inline_query", "chosen_inline_result", "my_chat_member", "channel_post", "edited_channel_post", "pre_checkout_query", "successful_payment"},
		}); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("set webhook: %w", err)
		}

		logger.Bot("✅ Webhook configurado com sucesso")

		webhookInfo, err := tb.GetWebhookInfo(ctx)
		if err == nil {
			logger.Bot("📊 Webhook Info - URL: %s, Pending: %d",
				webhookInfo.URL, webhookInfo.PendingUpdateCount)
		}

		logger.Bot("🚀 Iniciando processamento de updates...")
		go bh.Start()

	} else {
		logger.Bot("🔄 Bot iniciado em modo polling")
		if err := tb.DeleteWebhook(ctx, &telego.DeleteWebhookParams{}); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("delete webhook for polling: %w", err)
		}

		pollingUpdates, err := tb.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
			Timeout: 8,
			AllowedUpdates: []string{"message", "edited_message", "callback_query", "inline_query",
				"chosen_inline_result", "my_chat_member", "channel_post", "edited_channel_post",
				"pre_checkout_query", "successful_payment"},
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("start long polling: %w", err)
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					logger.Bot("⚠️ LongPolling cancelado via contexto!")
					return
				case u, ok := <-pollingUpdates:
					if !ok {
						logger.Bot("⚠️ LongPolling channel fechou!")
						return
					}
					if u.ChannelPost != nil {
						logger.Bot("📥 LongPolling recebeu ChannelPost #%d do canal %d",
							u.ChannelPost.MessageID, u.ChannelPost.Chat.ID)
					}
					select {
					case updates <- u:
					case <-ctx.Done():
						return
					}
				}
			}
		}()

		go bh.Start()
	}

	return webhookHandler, tb, app, cleanupFunc, nil
}

func newWebhookHandler(secret string, updates chan<- telego.Update) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get(telego.WebhookSecretTokenHeader)), []byte(secret)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
		defer r.Body.Close()
		var update telego.Update
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&update); err != nil {
			logger.Error("BOT", "❌ Erro ao ler body: %v", err)
			status := http.StatusBadRequest
			if _, ok := err.(*http.MaxBytesError); ok {
				status = http.StatusRequestEntityTooLarge
			}
			http.Error(w, "Bad Request", status)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			logger.Error("BOT", "❌ Erro ao deserealizar update: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		select {
		case updates <- update:
			w.WriteHeader(http.StatusOK)
		default:
			logger.Warn("BOT", "Webhook recusado: fila de updates cheia")
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		logger.Bot("✅ Webhook processado com sucesso")
	})
}
