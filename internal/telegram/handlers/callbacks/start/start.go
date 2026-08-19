package start

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		text, kb := parser.GetMessageTelego("start", map[string]string{
			"firstName": utils.RemoveHTMLTags(update.CallbackQuery.From.FirstName),
		})

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID: update.CallbackQuery.Message.GetMessageID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.EditMessageText(context.Background(), params)

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}

func CheckSubscriptionHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		const channelID = -1003767126116

		member, err := bot.GetChatMember(context.Background(), &telego.GetChatMemberParams{
			ChatID: telego.ChatID{ID: channelID},
			UserID: update.CallbackQuery.From.ID,
		})

		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao verificar sua inscrição. O bot precisa ser administrador no canal oficial!",
				ShowAlert:       true,
			})
			return nil
		}

		isMember := false
		if member != nil {
			status := member.MemberStatus()
			if status == telego.MemberStatusCreator || status == telego.MemberStatusAdministrator || status == telego.MemberStatusMember || status == telego.MemberStatusRestricted {
				isMember = true
			}
		}

		if !isMember {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Você ainda não entrou no canal!",
				ShowAlert:       true,
			})
			return nil
		}

		// Se entrou, mostra o start normal
		text, kb := parser.GetMessageTelego("start", map[string]string{
			"firstName": utils.RemoveHTMLTags(update.CallbackQuery.From.FirstName),
		})

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID: update.CallbackQuery.Message.GetMessageID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.EditMessageText(context.Background(), params)

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Obrigado por entrar!",
		})

		return nil
	}
}
