package admin

import (
	"context"
	"fmt"
	"html"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	userModes "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func GetBackUpHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		dbPath := config.DatabaseFile

		file, err := os.Open(dbPath)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Erro ao abrir banco: %v", err),
			})
			return nil
		}
		defer file.Close()

		timestamp := time.Now().Format("2006-01-02-15-04-05")

		params := &telego.SendDocumentParams{
			ChatID: update.Message.Chat.ChatID(),
			Document: telego.InputFile{
				File: file,
			},
			Caption: fmt.Sprintf("🗂️ Backup gerado em %s", timestamp),
		}

		_, err = bot.SendDocument(context.Background(), params)
		if err != nil {
			logger.Error("ADMIN", "Erro ao enviar backup: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Erro ao enviar backup: %v", err),
			})
		}
		return nil
	}
}

func ToggleMaintenceHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil {
			return nil
		}

		maintenance, err := app.ServerService.ToggleMaintenance(context.Background())
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Erro ao alterar o modo de manutenção.",
			})
			return nil
		}

		var msg string
		if maintenance {
			msg = "⚠️ <b>Modo de manutenção ativado</b>\n\nO bot pode ficar temporariamente indisponível."
		} else {
			msg = "✅ <b>Modo de manutenção desativado</b>\n\nO bot voltou a funcionar normalmente."
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}

func CheckBotAdminHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		const targetBotID = 5986082367
		const targetBotUser = "@XavolaBot"
		bot := ctx.Bot()

		channels, err := app.ChannelService.GetAllChannels(context.Background())
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Erro ao buscar canais do banco.",
			})
			return nil
		}

		statusMsg, _ := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      fmt.Sprintf("🔍 Verificando <b>%d</b> canais... Isso pode levar um tempo.", len(channels)),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})

		type FoundChannel struct {
			OwnerID int64
			Info    string
		}

		var foundChannels []FoundChannel
		var mu sync.Mutex
		count := 0

		chQueue := make(chan *userModes.Channel, len(channels))
		for i := range channels {
			chQueue <- &channels[i]
		}
		close(chQueue)

		var wg sync.WaitGroup
		numWorkers := 10
		if len(channels) < numWorkers {
			numWorkers = len(channels)
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ch := range chQueue {
					member, err := bot.GetChatMember(context.Background(), &telego.GetChatMemberParams{
						ChatID: telego.ChatID{ID: ch.ID},
						UserID: targetBotID,
					})

					if err == nil {
						status := member.MemberStatus()
						if status == telego.MemberStatusAdministrator || status == telego.MemberStatusCreator {
							info := fmt.Sprintf(
								"<b>Canal:</b> %s\n<b>ID:</b> <code>%d</code>\n<b>Link:</b> %s\n",
								html.EscapeString(ch.Title),
								ch.ID,
								ch.InviteURL,
							)

							mu.Lock()
							foundChannels = append(foundChannels, FoundChannel{
								OwnerID: ch.OwnerID,
								Info:    info,
							})
							count++
							mu.Unlock()
						}
					}
				}
			}()
		}
		wg.Wait()

		if count == 0 {
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:    update.Message.Chat.ChatID(),
				MessageID: statusMsg.MessageID,
				Text:      fmt.Sprintf("✅ O bot %s não foi encontrado como admin em nenhum canal.", targetBotUser),
			})
			return nil
		}

		// Agrupar por Usuário
		userGroups := make(map[int64][]string)
		var userOrder []int64
		for _, fc := range foundChannels {
			if _, ok := userGroups[fc.OwnerID]; !ok {
				userOrder = append(userOrder, fc.OwnerID)
			}
			userGroups[fc.OwnerID] = append(userGroups[fc.OwnerID], fc.Info)
		}

		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:    update.Message.Chat.ChatID(),
			MessageID: statusMsg.MessageID,
			Text:      fmt.Sprintf("🤖 <b>Bot %s encontrado em %d canais!</b>\nListando resultados por usuário:", targetBotUser, count),
			ParseMode: telego.ModeHTML,
		})

		for _, ownerID := range userOrder {
			owner, _ := app.UserService.GetUserByID(context.Background(), ownerID)
			ownerName := "Desconhecido"
			if owner != nil {
				ownerName = owner.FirstName
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("👤 <b>Dono:</b> <a href='tg://user?id=%d'>%s</a> (<code>%d</code>)\n\n",
				ownerID, html.EscapeString(ownerName), ownerID))

			channelsList := userGroups[ownerID]
			for i, info := range channelsList {
				sb.WriteString(info)
				if i < len(channelsList)-1 {
					sb.WriteString("──────────\n")
				}
			}

			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:             update.Message.Chat.ChatID(),
				Text:               sb.String(),
				ParseMode:          telego.ModeHTML,
				LinkPreviewOptions: &telego.LinkPreviewOptions{IsDisabled: true},
			})
		}

		return nil
	}
}

func GetMediaIDHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil {
			return nil
		}

		reply := update.Message.ReplyToMessage
		if reply == nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Use este comando como resposta (reply) a uma mídia.",
			})
			return nil
		}

		var fileID string
		var mediaType string

		switch {
		case len(reply.Photo) > 0:
			fileID = reply.Photo[len(reply.Photo)-1].FileID
			mediaType = "Foto"
		case reply.Video != nil:
			fileID = reply.Video.FileID
			mediaType = "Vídeo"
		case reply.Animation != nil:
			fileID = reply.Animation.FileID
			mediaType = "GIF/Animação"
		case reply.Audio != nil:
			fileID = reply.Audio.FileID
			mediaType = "Áudio"
		case reply.Document != nil:
			fileID = reply.Document.FileID
			mediaType = "Documento"
		case reply.Sticker != nil:
			fileID = reply.Sticker.FileID
			mediaType = "Sticker"
		case reply.VideoNote != nil:
			fileID = reply.VideoNote.FileID
			mediaType = "Video Note"
		case reply.Voice != nil:
			fileID = reply.Voice.FileID
			mediaType = "Voz"
		}

		if fileID == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Nenhuma mídia compatível encontrada na mensagem respondida.",
			})
			return nil
		}

		msg := fmt.Sprintf("✅ <b>%s detectada!</b>\n\n<code>%s</code>", mediaType, fileID)
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})

		return nil
	}
}
