package start

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		userID := update.Message.From.ID
		bot := ctx.Bot()

		// 0. Verificar Blacklist
		user, err := c.UserService.GetUserByID(context.Background(), userID)
		if err == nil && user != nil && user.IsBlacklisted {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "🚫 <b>Acesso Bloqueado</b>\n\nVocê está na nossa blacklist e seus comandos estão bloqueados.\n\nCaso tenha dúvidas ou queira solicitar a remoção, acione o comando /ouvidoria.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		// 1. Verificar se Force Join está ativado
		configData, err := c.ServerService.GetConfig(context.Background())
		if err == nil && configData.ForceJoin {

			// Bypass para Owner e Admins
			isAdmin := false
			if userID == config.OwnerID {
				isAdmin = true
			} else {
				user, err := c.UserService.GetUserByID(context.Background(), userID)
				if err == nil && user != nil && user.IsAdmin {
					isAdmin = true
				}
			}

			if !isAdmin {
				const channelID = -1003767126116
				const inviteLink = "https://t.me/LegendasBOTTopic"

				member, err := bot.GetChatMember(context.Background(), &telego.GetChatMemberParams{
					ChatID: telego.ChatID{ID: channelID},
					UserID: userID,
				})

				isMember := false
				if err == nil && member != nil {
					status := member.MemberStatus()
					if status == telego.MemberStatusCreator || status == telego.MemberStatusAdministrator || status == telego.MemberStatusMember || status == telego.MemberStatusRestricted {
						isMember = true
					}
				}

				if !isMember {
					kb := &telego.InlineKeyboardMarkup{
						InlineKeyboard: [][]telego.InlineKeyboardButton{
							{
								{Text: "📢 Entrar no Canal", URL: inviteLink},
							},
							{
								{Text: "✅ Já entrei", CallbackData: "check_subscription"},
							},
						},
					}

					params := &telego.SendMessageParams{
						ChatID:    update.Message.Chat.ChatID(),
						Text:      "⚠️ <b>Acesso Restrito</b>\n\nPara utilizar este bot, você precisa estar inscrito em nosso canal oficial.\n\nClique no botão abaixo para entrar e depois tente novamente ou clique em verificar.",
						ParseMode: telego.ModeHTML,
						ReplyParameters: &telego.ReplyParameters{
							MessageID: update.Message.MessageID,
						},
					}
					if kb != nil {
						params.ReplyMarkup = kb
					}

					_, _ = bot.SendMessage(context.Background(), params)
					return nil
				}
			}
		}

		firstName := utils.RemoveHTMLTags(update.Message.From.FirstName)

		text, kb := parser.GetMessageTelego("start", map[string]string{
			"firstName": firstName,
		})

		params := &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		}

		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.SendMessage(context.Background(), params)

		return nil
	}
}
