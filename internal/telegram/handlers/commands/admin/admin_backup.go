package admin

import (
	"context"
	"errors"
	"fmt"
	"html"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// GetBackUpHandlerTelego processa o comando administrativo /backup.
func GetBackUpHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		chatID := update.Message.Chat.ChatID()
		userID := update.Message.From.ID

		// Validação estrita de autorização administrativa
		if userID != config.OwnerID {
			user, err := app.UserService.GetUserByID(context.Background(), userID)
			if err != nil || user == nil || !user.IsAdmin {
				logger.Warn("ADMIN", "Tentativa não autorizada de executar /backup por UserID=%d", userID)
				return nil
			}
		}

		if app.BackupService == nil || !app.BackupService.IsConfigured() {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    chatID,
				Text:      "❌ <b>Serviço de backup não configurado.</b>\nVerifique as variáveis de ambiente do PostgreSQL e Cloudflare R2.",
				ParseMode: telego.ModeHTML,
			})
			return nil
		}

		// Enviar mensagem inicial informativa
		initMsg, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    chatID,
			Text:      "⏳ <b>Criando backup do banco de dados...</b>\n\n<i>Exportando tabelas, validando integridade e enviando para o Cloudflare R2. Aguarde...</i>",
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		if err != nil {
			logger.Error("ADMIN", "Erro ao enviar mensagem inicial de backup: %v", err)
		}

		// Executar assincronamente para não prender o dispatcher do bot
		go func() {
			bgCtx := context.Background()
			res, err := app.BackupService.PerformBackup(bgCtx)
			if err != nil {
				logger.Error("ADMIN", "Erro ao executar backup para UserID=%d: %v", userID, err)

				var errMsg string
				if errors.Is(err, services.ErrBackupInProgress) {
					errMsg = "⏳ <b>Já existe um backup em andamento.</b>\nAguarde a conclusão do processo atual antes de iniciar outro."
				} else {
					errMsg = "❌ <b>Não foi possível concluir o backup.</b>\nVerifique os logs administrativos para mais detalhes."
				}

				if initMsg != nil {
					_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
						ChatID:    chatID,
						MessageID: initMsg.MessageID,
						Text:      errMsg,
						ParseMode: telego.ModeHTML,
					})
				} else {
					_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
						ChatID:    chatID,
						Text:      errMsg,
						ParseMode: telego.ModeHTML,
					})
				}
				return
			}

			// Sucesso
			sizeFormatted := formatBytes(res.Size)
			msgText := fmt.Sprintf("✅ <b>Backup concluído com sucesso!</b>\n\n"+
				"📦 <b>Arquivo:</b> <code>%s</code>\n"+
				"💾 <b>Tamanho:</b> %s\n"+
				"🗑️ <b>Logs de auditoria:</b> não incluídos (schema preservado)\n"+
				"🔐 <b>SHA-256:</b> <code>%s</code>\n"+
				"☁️ <b>Armazenamento:</b> Cloudflare R2\n"+
				"⏱️ <b>Tempo de execução:</b> %s\n\n"+
				"🔗 <b>Download temporário:</b>\n"+
				"<a href=\"%s\">Clique aqui para baixar o arquivo .dmp</a>\n\n"+
				"⏱️ <i>O link expira em %s (às %s UTC).</i>",
				html.EscapeString(res.Filename),
				sizeFormatted,
				res.SHA256,
				res.Duration.Round(time.Second).String(),
				html.EscapeString(res.DownloadURL),
				app.BackupService.GetURLExpirationString(),
				res.ExpiresAt.Format("15:04:05"),
			)

			kb := &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{
							Text: "📥 Baixar Backup (.dmp)",
							URL:  res.DownloadURL,
						},
					},
				},
			}

			if initMsg != nil {
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:      chatID,
					MessageID:   initMsg.MessageID,
					Text:        msgText,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: kb,
				})
			} else {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:      chatID,
					Text:        msgText,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: kb,
				})
			}
		}()

		return nil
	}
}
