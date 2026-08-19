package mydrafts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	postbuilder "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/events/postBuilder"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// HandlerTelego exibe a lista principal de rascunhos salvos do usuário.
func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		return renderDraftsList(c, ctx.Bot(), update)
	}
}

func renderDraftsList(c *container.AppContainer, bot *telego.Bot, update telego.Update) error {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return nil
	}

	userID := update.CallbackQuery.From.ID

	templates, err := c.PostTemplateService.ListTemplates(context.Background(), userID)
	if err != nil {
		logger.Error("BOT", "Erro ao buscar rascunhos do usuário: %v", err)
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Erro ao buscar rascunhos.",
		})
		return nil
	}

	if len(templates) == 0 {
		var buttons [][]telego.InlineKeyboardButton
		buttons = append(buttons, []telego.InlineKeyboardButton{
			{Text: "🔙 Voltar ao Menu", CallbackData: "start"},
		})

		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID:   update.CallbackQuery.Message.GetMessageID(),
			Text:        "📂 <b>Meus Rascunhos</b>\n\n<i>Você não possui nenhum rascunho salvo no momento.</i>\n\n💡 Você pode salvar rascunhos durante a criação de posts no Post Builder!",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: buttons},
		})

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
		return nil
	}

	var text strings.Builder
	text.WriteString("📂 <b>Meus Rascunhos Salvos</b>\n\n")
	text.WriteString("Selecione um rascunho abaixo para visualizar, carregar ou enviar:\n\n")

	var buttons [][]telego.InlineKeyboardButton
	for _, tpl := range templates {
		createdStr := tpl.CreatedAt.In(utils.BrazilTZ()).Format("02/01/2006 às 15:04")
		text.WriteString(fmt.Sprintf("📝 <b>%s</b>\n🗓️ <i>Criado em %s</i>\n\n", tpl.Name, createdStr))

		buttons = append(buttons, []telego.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("📝 %s", tpl.Name),
				CallbackData: fmt.Sprintf("draft-detail:%s", tpl.ID),
			},
		})
	}

	buttons = append(buttons, []telego.InlineKeyboardButton{
		{Text: "🔙 Voltar ao Menu", CallbackData: "start"},
	})

	kb := &telego.InlineKeyboardMarkup{InlineKeyboard: buttons}

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
		MessageID:   update.CallbackQuery.Message.GetMessageID(),
		Text:        text.String(),
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})

	_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	return nil
}

// DetailHandlerTelego exibe detalhes e opções de um rascunho específico.
func DetailHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		draftID := strings.TrimPrefix(update.CallbackQuery.Data, "draft-detail:")
		userID := update.CallbackQuery.From.ID

		tpl, err := c.PostTemplateService.GetTemplateByID(context.Background(), draftID)
		if err != nil || tpl == nil || tpl.OwnerID != userID {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Rascunho não encontrado.",
				ShowAlert:       true,
			})
			return nil
		}

		var state cache.PostBuilderState
		_ = json.Unmarshal([]byte(tpl.TemplateData), &state)

		createdStr := tpl.CreatedAt.In(utils.BrazilTZ()).Format("02/01/2006 às 15:04 (BRT)")

		mediaStr := "Sem Mídia"
		if state.MediaType != "" {
			mediaStr = strings.ToUpper(state.MediaType)
		}

		text := fmt.Sprintf(
			"📝 <b>Detalhes do Rascunho</b>\n\n"+
				"📌 <b>Nome:</b> %s\n"+
				"🖼️ <b>Mídia:</b> %s\n"+
				"🔘 <b>Botões:</b> %d botão(ões)\n"+
				"🎭 <b>Reações:</b> %s\n"+
				"🗓️ <b>Salvo em:</b> %s\n\n"+
				"<i>O que deseja fazer com este rascunho?</i>",
			tpl.Name,
			mediaStr,
			len(state.Buttons),
			state.Reactions,
			createdStr,
		)

		buttons := [][]telego.InlineKeyboardButton{
			{
				{Text: "✏️ Carregar no PostBuilder", CallbackData: "draft-load:" + tpl.ID},
			},
			{
				{Text: "🏷️ Renomear Rascunho", CallbackData: "draft-rename:" + tpl.ID},
				{Text: "🗑️ Excluir Rascunho", CallbackData: "draft-delete:" + tpl.ID},
			},
			{
				{Text: "🔙 Voltar aos Rascunhos", CallbackData: "my-drafts"},
			},
		}

		kb := &telego.InlineKeyboardMarkup{InlineKeyboard: buttons}

		_, _ = ctx.Bot().EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID:   update.CallbackQuery.Message.GetMessageID(),
			Text:        text,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
		})

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}

// LoadHandlerTelego carrega o rascunho na sessão ativa do PostBuilder.
func LoadHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		draftID := strings.TrimPrefix(update.CallbackQuery.Data, "draft-load:")
		userID := update.CallbackQuery.From.ID

		tpl, err := c.PostTemplateService.GetTemplateByID(context.Background(), draftID)
		if err != nil || tpl == nil || tpl.OwnerID != userID {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Rascunho não encontrado.",
				ShowAlert:       true,
			})
			return nil
		}

		var state cache.PostBuilderState
		if err := json.Unmarshal([]byte(tpl.TemplateData), &state); err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao desserializar rascunho.",
				ShowAlert:       true,
			})
			return nil
		}

		state.MenuMessageID = 0
		state.PromptMessageID = 0
		state.Step = ""

		if err := c.CacheService.SetPostBuilderState(context.Background(), userID, state); err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao carregar rascunho no PostBuilder.",
				ShowAlert:       true,
			})
			return nil
		}

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Rascunho carregado com sucesso!",
			ShowAlert:       true,
		})

		chatID := update.CallbackQuery.Message.GetChat().ChatID()

		// 1. Notifica o usuário no chat privado
		_, _ = ctx.Bot().SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    chatID,
			Text:      fmt.Sprintf("✅ Rascunho <b>%s</b> carregado! Veja a prévia e o menu abaixo:", tpl.Name),
			ParseMode: telego.ModeHTML,
		})

		// 2. Envia a prévia da postagem no chat privado
		_ = postbuilder.SendPreviewTelego(ctx, chatID.ID, userID, c, &state)

		// 3. Abre o menu do PostBuilder logo abaixo para edição ou envio
		postbuilder.ShowMenuTelego(ctx, chatID.ID, userID, c, &state)

		return nil
	}
}

// DeleteHandlerTelego exclui um rascunho salvo.
func DeleteHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		draftID := strings.TrimPrefix(update.CallbackQuery.Data, "draft-delete:")
		userID := update.CallbackQuery.From.ID

		err := c.PostTemplateService.DeleteTemplate(context.Background(), draftID, userID)
		if err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao excluir rascunho.",
				ShowAlert:       true,
			})
			return nil
		}

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "🗑️ Rascunho excluído com sucesso!",
			ShowAlert:       true,
		})

		return renderDraftsList(c, ctx.Bot(), update)
	}
}

// RenameHandlerTelego solicita um novo nome para o rascunho.
func RenameHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		draftID := strings.TrimPrefix(update.CallbackQuery.Data, "draft-rename:")
		userID := update.CallbackQuery.From.ID

		tpl, err := c.PostTemplateService.GetTemplateByID(context.Background(), draftID)
		if err != nil || tpl == nil || tpl.OwnerID != userID {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Rascunho não encontrado.",
				ShowAlert:       true,
			})
			return nil
		}

		state, _ := c.CacheService.GetPostBuilderState(context.Background(), userID)
		if state == nil {
			state = &cache.PostBuilderState{}
		}
		state.Step = "awaiting_rename_draft:" + draftID
		_ = c.CacheService.SetPostBuilderState(context.Background(), userID, *state)

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		_, _ = ctx.Bot().SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      fmt.Sprintf("🏷️ <b>Renomear Rascunho</b>\n\nNome atual: <b>%s</b>\n\nEnvie o novo nome para o rascunho no chat:", tpl.Name),
			ParseMode: telego.ModeHTML,
		})

		return nil
	}
}
