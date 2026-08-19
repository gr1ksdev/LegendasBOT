package admin

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/container"
	userModes "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func GetAllChannelsHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		const chunkSize = 50
		offset := 0
		bot := ctx.Bot()

		for {
			channels, total, err := app.ChannelService.GetAllChannelsPaginated(context.Background(), chunkSize, offset)
			if err != nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: update.Message.Chat.ChatID(),
					Text:   "Erro ao buscar canais.",
				})
				return nil
			}

			if len(channels) == 0 {
				if offset == 0 {
					_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
						ChatID: update.Message.Chat.ChatID(),
						Text:   "Nenhum canal encontrado.",
					})
				}
				break
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("📦 Total de Canais: <b>%d</b>\n<blockquote>Página %d</blockquote>\n",
				total, (offset/chunkSize)+1))

			for _, c := range channels {
				sb.WriteString(fmt.Sprintf(`<a href='%s'>%s</a> - <code>%d</code>`+"\n", c.InviteURL, c.Title, c.ID))
			}

			_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:             update.Message.Chat.ChatID(),
				Text:               sb.String(),
				ParseMode:          telego.ModeHTML,
				LinkPreviewOptions: &telego.LinkPreviewOptions{IsDisabled: true},
			})
			if err != nil {
				break
			}

			offset += chunkSize
			if int64(offset) >= total {
				break
			}
		}
		return nil
	}
}

func GetInfoChannelHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		channelIDStr := strings.TrimSpace(update.Message.Text[len("/info"):])
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ ID inválido: %v", err),
			})
			return nil
		}

		channel, err := app.ChannelService.GetChannelByID(context.Background(), channelID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Canal não encontrado!: %v", err),
			})
			return nil
		}

		owner, err := app.UserService.GetUserByID(context.Background(), channel.OwnerID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Dono não encontrado!",
			})
			return nil
		}

		miniAppURL := auth.GenerateMiniAppUrl(fmt.Sprintf("%d", update.Message.From.ID), channelIDStr)
		logsURL := fmt.Sprintf("%s/admin/dash?tab=logs&channelId=%d", strings.TrimRight(config.WebAppURL, "/"), channel.ID)
		msg := fmt.Sprintf(
			"ID: <code>%d</code>\nCanal: %s\nLink: %s\nDono: <a href='tg://user?id=%d'>%s</a> (<code>%d</code>)",
			channel.ID,
			html.EscapeString(channel.Title),
			html.EscapeString(channel.InviteURL),
			owner.UserId,
			html.EscapeString(owner.FirstName),
			owner.UserId,
		)

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:             update.Message.Chat.ChatID(),
			Text:               msg,
			ParseMode:          telego.ModeHTML,
			LinkPreviewOptions: &telego.LinkPreviewOptions{IsDisabled: true},
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{
							Text: "Abrir canal na dashboard",
							WebApp: &telego.WebAppInfo{
								URL: miniAppURL,
							},
						},
					},
					{
						{
							Text: "Abrir logs do canal",
							WebApp: &telego.WebAppInfo{
								URL: logsURL,
							},
						},
					},
				},
			},
		})
		return nil
	}
}

func AddChannelCommandHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		botInfo, _ := bot.GetMe(context.Background())

		msgText := strings.TrimSpace(update.Message.Text)
		args := strings.SplitN(msgText, " ", 3)
		if len(args) < 3 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Uso correto: /add <channel_id> <owner_id>",
			})
			return nil
		}

		channelIDStr := args[1]
		ownerIDStr := args[2]
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		ownerID, err2 := strconv.ParseInt(ownerIDStr, 10, 64)
		if err != nil || err2 != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ IDs inválidos. Certifique-se de que ambos são numéricos.",
			})
			return nil
		}

		existingChannel, _ := c.ChannelService.GetChannelByID(context.Background(), channelID)
		if existingChannel != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Canal já existe no banco de dados.",
			})
			return nil
		}

		channelInfo, err := bot.GetChat(context.Background(), &telego.GetChatParams{ChatID: telego.ChatID{ID: channelID}})
		if err != nil {
			logger.Error("ADMIN", "Erro ao buscar canal: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{ChatID: update.Message.Chat.ChatID(), Text: "❌ Erro ao buscar informações do canal."})
			return nil
		}

		ownerInfo, err := bot.GetChat(context.Background(), &telego.GetChatParams{ChatID: telego.ChatID{ID: ownerID}})
		if err != nil {
			logger.Error("ADMIN", "Erro ao buscar usuário: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{ChatID: update.Message.Chat.ChatID(), Text: "❌ Erro ao buscar informações do usuário."})
			return nil
		}

		_ = c.UserService.UpsertUser(context.Background(), &userModes.User{
			UserId:    ownerID,
			FirstName: utils.RemoveHTMLTags(ownerInfo.FirstName),
		})

		newPackCaption := fmt.Sprintf(`╔═━──━═༻✧༺═━──━═╗

        𖦹⁠⁠⁠ ࣪ ⭑ ᥫ᭡
        (｡•́︿•̀｡)っ✧.*ೃ༄
        ˗ˏˋ [$name]($link) ⁠⋆｡˚ ☁︎
             彡♡ ₊˚

⋆｡˚ ❀ @%s ☽⁺₊

╚═━──━═༻✧༺═━──━═╝`, botInfo.Username)

		defaultCaption := fmt.Sprintf("➽ 𝐛𝐲 @%s", botInfo.Username)
		inviteURL := channelInfo.InviteLink
		if channelInfo.Username != "" {
			inviteURL = fmt.Sprintf("t.me/%s", channelInfo.Username)
		}

		channel, err := c.ChannelService.CreateChannelWithDefaults(context.Background(), channelID, channelInfo.Title, inviteURL, newPackCaption, defaultCaption, ownerID)
		if err != nil {
			logger.Error("ADMIN", "Erro ao criar canal: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{ChatID: update.Message.Chat.ChatID(), Text: "❌ Erro ao salvar canal."})
			return nil
		}

		miniApp := auth.GenerateMiniAppUrl(fmt.Sprintf("%d", ownerID), fmt.Sprintf("%d", channelID))
		msg := fmt.Sprintf("✅ Canal salvo com sucesso - (%s - %d)\n\n%s", channel.Title, channel.ID, miniApp)

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
		})
		return nil
	}
}

func RemoveChannelHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		channelIDStr := strings.TrimSpace(update.Message.Text[len("/remove"):])
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ ID inválido: %v", err),
			})
			return nil
		}

		channel, err := app.ChannelService.GetChannelByID(context.Background(), channelID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Canal não encontrado!: %v", err),
			})
			return nil
		}

		if err = app.ChannelService.DisconnectChannel(context.Background(), channel.OwnerID, channelID); err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Não foi possivel deletar o canal: %v", err),
			})
			return nil
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      "✅ Canal excluído com sucesso!",
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}

func RegisterTransferHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		input := strings.TrimSpace(update.Message.Text[len("/transfer"):])
		parts := strings.Fields(input)
		if len(parts) < 2 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Uso: /transfer <channelId> <newOwnerId>",
			})
			return nil
		}

		channelID, _ := strconv.ParseInt(parts[0], 10, 64)
		newOwnerID, _ := strconv.ParseInt(parts[1], 10, 64)

		channel, err := app.ChannelService.GetChannelByID(context.Background(), channelID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Canal não encontrado!: %v", err),
			})
			return nil
		}

		tgUser, err := bot.GetChat(context.Background(), &telego.GetChatParams{ChatID: telego.ChatID{ID: newOwnerID}})
		if err != nil || tgUser.FirstName == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ ID de usuário inválido: %d", newOwnerID),
			})
			return nil
		}

		err = app.ChannelService.UpdateOwnerChannel(context.Background(), channelID, channel.OwnerID, newOwnerID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Erro ao transferir canal",
			})
			return nil
		}

		msg := fmt.Sprintf(
			"✅ <b>Transferência realizada com sucesso!</b>\n<b>Canal:</b> %s\n<b>ID:</b> %d\n<b>Novo Dono:</b> %s (%d)\n\n🔗 <a href=\"%s\">Abrir painel do canal</a>",
			html.EscapeString(channel.Title),
			channelID,
			html.EscapeString(tgUser.FirstName),
			newOwnerID,
			auth.GenerateMiniAppUrl(parts[1], parts[0]),
		)

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
		})
		return nil
	}
}
