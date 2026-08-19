package postbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	channelpost "github.com/leirbagxis/FreddyBot/internal/telegram/events/channelPost"
	addchannel "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/events/addChannel"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func isBotAdminTelego(chatMember telego.ChatMember) bool {
	if chatMember == nil {
		return false
	}
	status := chatMember.MemberStatus()
	if status == telego.MemberStatusCreator {
		return true
	}
	if status == telego.MemberStatusAdministrator {
		if admin, ok := chatMember.(*telego.ChatMemberAdministrator); ok {
			return admin.CanPostMessages && admin.CanEditMessages && admin.CanDeleteMessages && admin.CanInviteUsers
		}
	}
	return false
}

func isEmoji(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func utf16CodeUnitLen(text string) int {
	units := 0
	for _, r := range text {
		units += len(utf16.Encode([]rune{r}))
	}
	return units
}

func stripLeadingEmojiFallback(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}

	first, size := utf8.DecodeRuneInString(label)
	if first == utf8.RuneError && size == 0 {
		return ""
	}
	if unicode.IsLetter(first) || unicode.IsDigit(first) {
		return label
	}

	end := size
	for end < len(label) {
		r, rSize := utf8.DecodeRuneInString(label[end:])
		if r == '\ufe0e' || r == '\ufe0f' || unicode.Is(unicode.Mn, r) {
			end += rSize
			continue
		}
		if r == '\u200d' {
			end += rSize
			if end < len(label) {
				_, nextSize := utf8.DecodeRuneInString(label[end:])
				end += nextSize
			}
			continue
		}
		break
	}

	return strings.TrimSpace(label[end:])
}

func buildPostBuilderURLButton(btn cache.PostBuilderButton, useCustomEmoji bool) telego.InlineKeyboardButton {
	label := strings.TrimSpace(btn.Text)
	button := telego.InlineKeyboardButton{
		Text: label,
		URL:  btn.URL,
	}

	if btn.CustomEmojiID == "" || !useCustomEmoji {
		if button.Text == "" {
			button.Text = " "
		}
		return button
	}

	button.IconCustomEmojiID = btn.CustomEmojiID
	button.Text = stripLeadingEmojiFallback(label)
	if button.Text == "" {
		button.Text = " "
	}

	return button
}

func ProcessIncomingContentTelego(ctx *telegohandler.Context, update telego.Update, c *container.AppContainer) error {
	if update.Message == nil || update.Message.From == nil {
		return nil
	}

	bot := ctx.Bot()
	userID := update.Message.From.ID

	logger.Bot("PostBuilder: Processando mensagem recebida de UserID=%d | TextLen=%d | CaptionLen=%d | Photo=%v | Video=%v | ForwardOrigin=%v",
		userID, len(update.Message.Text), len(update.Message.Caption), update.Message.Photo != nil, update.Message.Video != nil, update.Message.ForwardOrigin != nil)

	// Check Blacklist
	user, err := c.UserService.GetUserByID(context.Background(), userID)
	if err == nil && user != nil && user.IsBlacklisted {
		logger.Bot("PostBuilder: Usuario %d esta na blacklist. Ignorando.", userID)
		return nil
	}

	// Detect media
	var mediaID string
	var mediaType string

	// Se o usuário estiver configurando um sticker separador, o PostBuilder não deve interceptar
	awaitingStickerChannel, _ := c.CacheService.GetAwaitingStickerSeparator(context.Background(), userID)
	if awaitingStickerChannel != 0 && update.Message.Sticker != nil {
		return nil
	}

	if update.Message.Photo != nil {
		mediaID = update.Message.Photo[len(update.Message.Photo)-1].FileID
		mediaType = "photo"
	} else if update.Message.Video != nil {
		mediaID = update.Message.Video.FileID
		mediaType = "video"
	} else if update.Message.Animation != nil {
		mediaID = update.Message.Animation.FileID
		mediaType = "animation"
	} else if update.Message.Audio != nil {
		mediaID = update.Message.Audio.FileID
		mediaType = "audio"
	} else if update.Message.Document != nil {
		mediaID = update.Message.Document.FileID
		mediaType = "document"
	} else if update.Message.Sticker != nil {
		mediaID = update.Message.Sticker.FileID
		mediaType = "sticker"
	}

	// Primeiro verifica se o usuário está em uma etapa de entrada ativa (ex: digitando título, agendamento)
	scheduleState, _ := c.CacheService.GetScheduleState(context.Background(), userID)
	if scheduleState != nil && scheduleState.SessionID != "" {
		logger.Bot("PostBuilder: Usuario %d em fluxo de agendamento ativo", userID)
		handleScheduleTextInput(ctx, update.Message.Chat.ID, userID, update.Message.Text, scheduleState, c)
		return nil
	}
	activeState, _ := c.CacheService.GetPostBuilderState(context.Background(), userID)
	if activeState != nil && activeState.Step != "" {
		logger.Bot("PostBuilder: Usuario %d em etapa de entrada ativa '%s'", userID, activeState.Step)
		return handleTextInputTelego(ctx, update, c, activeState)
	}

	// Se não tiver mídia nem for mensagem encaminhada, ignora (evita interceptar texto/links comuns)
	if mediaID == "" && update.Message.ForwardOrigin == nil {
		logger.Bot("PostBuilder: Mensagem comum sem midia nem encaminhamento para o usuario %d. Ignorando.", userID)
		return nil
	}

	if mediaType == "" && update.Message.Text != "" {
		mediaType = "text"
	}

	// Verificar se é mensagem encaminhada de um canal
	if update.Message.ForwardOrigin != nil {
		if origin, ok := update.Message.ForwardOrigin.(*telego.MessageOriginChannel); ok {
			channelID := origin.Chat.ID
			existingChannel, _ := c.ChannelService.GetChannelByID(context.Background(), channelID)
			if existingChannel == nil {
				// Canal NÃO está configurado no banco. Verificar se o bot é admin no canal.
				botInfo, _ := bot.GetMe(context.Background())
				if botInfo != nil {
					botMember, err := bot.GetChatMember(context.Background(), &telego.GetChatMemberParams{
						ChatID: telego.ChatID{ID: channelID},
						UserID: botInfo.ID,
					})
					if err == nil && isBotAdminTelego(botMember) {
						logger.Bot("PostBuilder: Mensagem encaminhada do canal %d. Bot e admin porem canal NAO esta configurado no banco. Disparando prompt de vinculacao.", channelID)
						return addchannel.SendAddChannelPromptTelego(bot, userID, channelID, origin.Chat.Title, update.Message.From.FirstName)
					}
				}
				logger.Bot("PostBuilder: Mensagem encaminhada do canal %d. Bot NAO e admin no canal. Prosseguindo para o PostBuilder.", channelID)
			} else {
				logger.Bot("PostBuilder: Mensagem encaminhada do canal %d que JA ESTA configurado no banco. Prosseguindo para o PostBuilder.", channelID)
			}
		}
	}

	// Conteúdo/Mídia detectado, oferecer Post Builder
	kb := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🛠️ Post Builder", CallbackData: "pb-start"},
			},
		},
	}

	// Capturar legenda ou texto existente na mídia/mensagem
	var captionText string
	if update.Message.Caption != "" {
		captionText = channelpost.ProcessTextWithFormattingTelego(update.Message.Caption, update.Message.CaptionEntities)
	} else if update.Message.Text != "" {
		captionText = channelpost.ProcessTextWithFormattingTelego(update.Message.Text, update.Message.Entities)
	}

	// Extrair botões inline pré-existentes na mídia recebida
	var initialButtons []cache.PostBuilderButton
	if update.Message.ReplyMarkup != nil {
		for _, row := range update.Message.ReplyMarkup.InlineKeyboard {
			for _, btn := range row {
				if btn.URL != "" {
					initialButtons = append(initialButtons, cache.PostBuilderButton{
						Text:          btn.Text,
						URL:           btn.URL,
						CustomEmojiID: btn.IconCustomEmojiID,
					})
				}
			}
		}
	}

	// Store initial state
	state := cache.PostBuilderState{
		MediaType:   mediaType,
		MediaFileID: mediaID,
		Body:        captionText,
		Buttons:     initialButtons,
		Step:        "",
	}
	c.CacheService.SetPostBuilderState(context.Background(), userID, state)
	recordPostBuilderEvent(c, "postbuilder_started", services.ChannelEventStatusInfo, userID, 0, "", map[string]any{"media_type": mediaType, "chat_id": update.Message.Chat.ID, "has_caption": captionText != ""}, nil)

	logger.Bot("PostBuilder: Rascunho inicial criado com sucesso para UserID=%d | MediaType=%s | MediaID=%s | BodyLen=%d | Buttons=%d",
		userID, mediaType, mediaID, len(captionText), len(initialButtons))

	msgText := "✨ Conteúdo detectado! Deseja usar o <b>Post Builder</b> para criar uma postagem personalizada?"
	if captionText != "" {
		msgText += "\n\n📝 <b>Texto/Legenda detectado!</b> Você poderá editá-lo no Post Builder."
	}

	_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      update.Message.Chat.ChatID(),
		Text:        msgText,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: update.Message.MessageID,
		},
	})

	return nil
}

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		return ProcessIncomingContentTelego(ctx, update, c)
	}
}

func handleTextInputTelego(ctx *telegohandler.Context, update telego.Update, c *container.AppContainer, state *cache.PostBuilderState) error {
	text := update.Message.Text
	bot := ctx.Bot()
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	// Check if user is in schedule input flow
	scheduleState, _ := c.CacheService.GetScheduleState(context.Background(), userID)
	if scheduleState != nil && scheduleState.SessionID != "" {
		handleScheduleTextInput(ctx, chatID, userID, text, scheduleState, c)
		return nil
	}

	// Processar texto com entidades
	formattedText := channelpost.ProcessTextWithFormattingTelego(text, update.Message.Entities)

	if state.PromptMessageID != 0 {
		state.PromptMessageID = 0
	}

	previousStep := state.Step

	if strings.HasPrefix(state.Step, "awaiting_rename_draft:") {
		draftID := strings.TrimPrefix(state.Step, "awaiting_rename_draft:")
		newName := strings.TrimSpace(text)
		if newName != "" {
			err := c.PostTemplateService.UpdateTemplateName(context.Background(), draftID, userID, newName)
			if err != nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Erro ao renomear rascunho.",
				})
			} else {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: chatID},
					Text:      fmt.Sprintf("✅ Rascunho renomeado com sucesso para <b>%s</b>!", newName),
					ParseMode: telego.ModeHTML,
				})
			}
		}
		state.Step = ""
		c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		return nil
	}

	switch state.Step {
	case "awaiting_title":
		state.Title = formattedText
		state.Step = ""
	case "awaiting_body":
		state.Body = formattedText
		state.Step = ""
	case "awaiting_footer":
		state.Footer = formattedText
		state.Step = ""
	case "awaiting_reactions":
		parts := strings.Split(text, ",")
		var finalReactions []string
		valid := true

		// Mapear entidades para facilitar a busca por posição
		entityMap := make(map[int]string)
		for _, e := range update.Message.Entities {
			if e.Type == "custom_emoji" {
				entityMap[e.Offset] = e.CustomEmojiID
			}
		}

		currentOffset := 0
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed == "" {
				currentOffset += len(p) + 1
				continue
			}

			// Verificar se nesta posição do texto original existe uma entidade de emoji customizado
			pos := strings.Index(text[currentOffset:], trimmed) + currentOffset
			if eid, ok := entityMap[pos]; ok {
				finalReactions = append(finalReactions, "eid:"+eid)
			} else if isEmoji(trimmed) {
				finalReactions = append(finalReactions, trimmed)
			} else {
				valid = false
				break
			}
			currentOffset += len(p) + 1
		}

		if !valid {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ Apenas emojis (padrão ou customizados) são permitidos como reações. Tente novamente:",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		state.Reactions = strings.Join(finalReactions, ",")
		state.Step = ""
	case "awaiting_template_name":
		name := strings.TrimSpace(text)
		if name == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ O nome do template não pode estar vazio. Envie um nome válido:",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}
		var sb strings.Builder
		if state.Title != "" {
			sb.WriteString(state.Title + "\n\n")
		}
		if state.Body != "" {
			sb.WriteString(state.Body + "\n\n")
		}
		if state.Footer != "" {
			sb.WriteString(state.Footer)
		}
		caption := sb.String()

		tpl, err := c.UserCaptionTemplateService.Create(context.Background(), userID, name, caption, state.Reactions)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      fmt.Sprintf("❌ Erro ao salvar template: %s", err.Error()),
				ParseMode: telego.ModeHTML,
			})
			return nil
		}
		for i, btn := range state.Buttons {
			err = c.UserCaptionTemplateService.CreateButton(context.Background(), &models.UserCaptionTemplateButton{
				ButtonID:        fmt.Sprintf("btn_%d_%d", time.Now().UnixNano(), i),
				NameButton:      btn.Text,
				ButtonURL:       btn.URL,
				Style:           btn.Style,
				PositionX:       0,
				PositionY:       i,
				OwnerTemplateID: tpl.ID,
			})
		}

		sessionID := state.TemplateSessionID
		state.Step = ""
		state.TemplateSessionID = ""
		c.CacheService.SetPostBuilderState(context.Background(), userID, *state)

		var completionKB *telego.InlineKeyboardMarkup
		if sessionID != "" {
			completionKB = &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
					},
				},
			}
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      update.Message.Chat.ChatID(),
			Text:        fmt.Sprintf("✅ Template <b>%s</b> salvo com sucesso!\n\nUse 📋 Templates no menu para carregá-lo depois.", tpl.Code),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})

		if sessionID == "" {
			showMenuTelego(ctx, chatID, userID, c, state)
		}
		return nil

	case "awaiting_button":
		lines := strings.Split(text, "\n")
		if len(lines) < 2 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ Formato inválido. Envie o <b>Nome</b> em uma linha e o <b>Link</b> na linha de baixo.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}
		rawName := lines[0]
		name := strings.TrimSpace(rawName)
		url := utils.NormalizeTelegramURL(lines[1])

		// Extrair CustomEmojiID do nome (primeira linha). Mantemos o emoji textual
		// como fallback para resultados inline, onde IconCustomEmojiID pode ser ignorado.
		var customEmojiID string
		firstLineLen := utf16CodeUnitLen(rawName)
		for _, entity := range update.Message.Entities {
			if entity.Type == "custom_emoji" && entity.Offset < firstLineLen {
				customEmojiID = entity.CustomEmojiID
				break
			}
		}

		if !utils.IsValidButtonURL(url) {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ URL inválida. Use https://, t.me/canal, @canal ou tg://. Tente novamente:",
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		state.Buttons = append(state.Buttons, cache.PostBuilderButton{Text: name, URL: url, CustomEmojiID: customEmojiID})
		state.Step = ""
	default:
		return nil
	}

	state.MenuMessageID = 0
	eventType := "postbuilder_field_updated"
	field := strings.TrimPrefix(previousStep, "awaiting_")
	metadata := map[string]any{"field": field}
	if update.Message != nil {
		metadata["message_id"] = update.Message.MessageID
	}
	if strings.TrimSpace(text) != "" {
		metadata["text_len"] = len(text)
	}
	if previousStep == "awaiting_button" {
		eventType = "postbuilder_button_added"
		metadata["buttons"] = len(state.Buttons)
	}
	recordPostBuilderEvent(c, eventType, services.ChannelEventStatusInfo, update.Message.From.ID, 0, "", metadata, nil)
	c.CacheService.SetPostBuilderState(context.Background(), update.Message.From.ID, *state)

	if state.Step == "awaiting_button" {
		showButtonManagerTelego(ctx, update.Message.Chat.ID, update.Message.From.ID, c, state)
	} else {
		showMenuTelego(ctx, update.Message.Chat.ID, update.Message.From.ID, c, state, update.Message.MessageID)
	}

	return nil
}

func ShowMenuTelego(ctx *telegohandler.Context, chatID, userID int64, c *container.AppContainer, state *cache.PostBuilderState, replyToMessageID ...int) {
	showMenuTelego(ctx, chatID, userID, c, state, replyToMessageID...)
}

func SendPreviewTelego(ctx *telegohandler.Context, chatID, userID int64, c *container.AppContainer, state *cache.PostBuilderState) error {
	return sendFinalPostTelego(ctx, chatID, userID, c, state, false)
}

func showMenuTelego(ctx *telegohandler.Context, chatID, userID int64, c *container.AppContainer, state *cache.PostBuilderState, replyToMessageID ...int) {
	var sb strings.Builder
	bot := ctx.Bot()

	isSticker := state.MediaType == "sticker"

	check := func(filled bool) string {
		if filled {
			return "✅"
		}
		return "❌"
	}

	sb.WriteString("🛠️ <b>Post Builder - Menu</b>\n\n")
	if isSticker {
		sb.WriteString("📝 <b>Texto:</b> não suportado para stickers\n")
	} else {
		sb.WriteString(fmt.Sprintf("📝 <b>Título:</b> %s\n", check(state.Title != "")))
		sb.WriteString(fmt.Sprintf("📄 <b>Corpo:</b> %s\n", check(state.Body != "")))
		sb.WriteString(fmt.Sprintf("👣 <b>Rodapé:</b> %s\n", check(state.Footer != "")))
	}
	sb.WriteString(fmt.Sprintf("🎭 <b>Reações:</b> %s\n", check(state.Reactions != "")))
	sb.WriteString(fmt.Sprintf("🔘 <b>Botões:</b> %s\n\n", check(len(state.Buttons) > 0)))
	sb.WriteString("Escolha o que deseja editar:")

	var kbRows [][]telego.InlineKeyboardButton
	if isSticker {
		kbRows = [][]telego.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("🎭 Reações %s", check(state.Reactions != "")), CallbackData: "pb-edit-reactions"},
			},
			{
				{Text: fmt.Sprintf("🔘 Botões %s", check(len(state.Buttons) > 0)), CallbackData: "pb-manage-buttons"},
				{Text: "📥 Importar Canal", CallbackData: "pb-import-channel"},
			},
			{
				{Text: "👁️ Preview", CallbackData: "pb-preview"},
			},
			{
				{Text: "✅ Salvar", CallbackData: "pb-save"},
				{Text: "❌ Cancelar", CallbackData: "pb-cancel", Style: "danger"},
			},
		}
	} else {
		kbRows = [][]telego.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("📝 Título %s", check(state.Title != "")), CallbackData: "pb-edit-title"},
				{Text: fmt.Sprintf("📄 Corpo %s", check(state.Body != "")), CallbackData: "pb-edit-body"},
			},
			{
				{Text: fmt.Sprintf("👣 Rodapé %s", check(state.Footer != "")), CallbackData: "pb-edit-footer"},
				{Text: fmt.Sprintf("🎭 Reações %s", check(state.Reactions != "")), CallbackData: "pb-edit-reactions"},
			},
			{
				{Text: fmt.Sprintf("🔘 Botões %s", check(len(state.Buttons) > 0)), CallbackData: "pb-manage-buttons"},
				{Text: "📥 Importar Canal", CallbackData: "pb-import-channel"},
			},
			{
				{Text: "📋 Templates", CallbackData: "pb-list-templates"},
				{Text: "💾 Salvar Rascunho", CallbackData: "pb-save-template:current"},
			},
			{
				{Text: "👁️ Preview", CallbackData: "pb-preview"},
				{Text: "✅ Salvar", CallbackData: "pb-save"},
			},
			{
				{Text: "❌ Cancelar", CallbackData: "pb-cancel", Style: "danger"},
			},
		}
	}

	kb := &telego.InlineKeyboardMarkup{InlineKeyboard: kbRows}

	if len(replyToMessageID) > 0 && replyToMessageID[0] != 0 {
		msg, _ := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        sb.String(),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: replyToMessageID[0],
			},
		})
		if msg != nil {
			state.MenuMessageID = msg.MessageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		}
		return
	}

	if state.MenuMessageID != 0 {
		_, err := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   state.MenuMessageID,
			Text:        sb.String(),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
		})
		if err == nil {
			return
		}
	}

	msg, _ := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: chatID},
		Text:        sb.String(),
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})

	if msg != nil {
		state.MenuMessageID = msg.MessageID
		c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
	}
}

func showButtonManagerTelego(ctx *telegohandler.Context, chatID, userID int64, c *container.AppContainer, state *cache.PostBuilderState) {
	var sb strings.Builder
	bot := ctx.Bot()

	sb.WriteString("🔘 <b>Gerenciamento de Botões</b>\n\n")
	if len(state.Buttons) == 0 {
		sb.WriteString("<i>Nenhum botão adicionado ainda.</i>")
	} else {
		sb.WriteString("Clique em um botão para <b>excluí-lo</b>:")
	}

	var rows [][]telego.InlineKeyboardButton

	for i, btn := range state.Buttons {
		rows = append(rows, []telego.InlineKeyboardButton{
			{Text: "❌ " + btn.Text, CallbackData: fmt.Sprintf("pb-del-button:%d", i)},
		})
	}

	rows = append(rows, []telego.InlineKeyboardButton{
		{Text: "➕ Adicionar Novo Botão", CallbackData: "pb-add-button"},
	})
	rows = append(rows, []telego.InlineKeyboardButton{
		{Text: "🔙 Voltar ao Menu", CallbackData: "pb-start"},
	})

	kb := &telego.InlineKeyboardMarkup{InlineKeyboard: rows}

	if state.MenuMessageID != 0 {
		_, err := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   state.MenuMessageID,
			Text:        sb.String(),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
		})
		if err == nil {
			return
		}
	}

	msg, _ := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: chatID},
		Text:        sb.String(),
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})

	if msg != nil {
		state.MenuMessageID = msg.MessageID
		c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
	}
}

func CallbackHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		userID := update.CallbackQuery.From.ID
		chatID := update.CallbackQuery.Message.GetChat().ID
		data := update.CallbackQuery.Data
		bot := ctx.Bot()

		user, err := c.UserService.GetUserByID(context.Background(), userID)
		if err == nil && user != nil && user.IsBlacklisted {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Você está na blacklist.",
				ShowAlert:       true,
			})
			return nil
		}

		state, _ := c.CacheService.GetPostBuilderState(context.Background(), userID)
		if state == nil &&
			data != "pb-cancel" &&
			!strings.HasPrefix(data, "pb-saved-menu:") &&
			!strings.HasPrefix(data, "pb-send-") &&
			!strings.HasPrefix(data, "pb-schedule:") &&
			!strings.HasPrefix(data, "pb-sch:") &&
			!strings.HasPrefix(data, "pb-sch-type:") &&
			!strings.HasPrefix(data, "pb-sch-confirm:") &&
			!strings.HasPrefix(data, "pb-sch-edit:") &&
			!strings.HasPrefix(data, "pb-save-template:") &&
			!strings.HasPrefix(data, "pb-load-template:") &&
			!strings.HasPrefix(data, "pb-del-template:") {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "Sessão expirada ou não encontrada.",
			})
			return nil
		}

		if strings.HasPrefix(data, "pb-saved-menu:") {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			})
			sessionID := strings.TrimPrefix(data, "pb-saved-menu:")
			_ = c.CacheService.DeleteScheduleState(context.Background(), userID)
			messageID := update.CallbackQuery.Message.GetMessageID()
			showSavedPostMenu(ctx, chatID, userID, messageID, sessionID, c)
			return nil
		}

		if strings.HasPrefix(data, "pb-import-apply:") {
			channelIDStr := strings.TrimPrefix(data, "pb-import-apply:")
			channelID, _ := strconv.ParseInt(channelIDStr, 10, 64)

			channel, err := c.ChannelService.GetChannelWithRelations(context.Background(), channelID)
			if err != nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Erro ao obter dados do canal.",
				})
				return nil
			}

			if channel.DefaultCaption != nil {
				state.Body = channelpost.DetectParseMode(channel.DefaultCaption.Caption)
			}
			state.Reactions = channel.Reactions

			state.Buttons = make([]cache.PostBuilderButton, 0)
			for _, btn := range channel.Buttons {
				state.Buttons = append(state.Buttons, cache.PostBuilderButton{
					Text: btn.NameButton,
					URL:  btn.ButtonURL,
				})
			}

			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      fmt.Sprintf("✅ Dados importados do canal <b>%s</b>!", channel.Title),
				ParseMode: telego.ModeHTML,
			})
			showMenuTelego(ctx, chatID, userID, c, state)
			return nil
		}

		if strings.HasPrefix(data, "pb-del-button:") {
			indexStr := strings.TrimPrefix(data, "pb-del-button:")
			index, _ := strconv.Atoi(indexStr)

			if index >= 0 && index < len(state.Buttons) {
				removed := state.Buttons[index]
				state.Buttons = append(state.Buttons[:index], state.Buttons[index+1:]...)
				recordPostBuilderEvent(c, "postbuilder_button_deleted", services.ChannelEventStatusInfo, userID, 0, "", map[string]any{"button_text": removed.Text, "remaining_buttons": len(state.Buttons)}, nil)
				c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
			}
			showButtonManagerTelego(ctx, chatID, userID, c, state)
			return nil
		}

		if strings.HasPrefix(data, "pb-save-template:") {
			sessionID := strings.TrimPrefix(data, "pb-save-template:")
			var sessionState *cache.PostBuilderState

			if sessionID != "" && sessionID != "current" {
				sessionState, _ = c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
			}
			if sessionState == nil {
				sessionState, _ = c.CacheService.GetPostBuilderState(context.Background(), userID)
			}
			if sessionState == nil {
				sessionState = state
			}
			if sessionState == nil {
				_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "❌ Nenhuma postagem ativa para salvar como rascunho.",
					ShowAlert:       true,
				})
				return nil
			}

			draftName := sessionState.Title
			if draftName == "" {
				draftName = fmt.Sprintf("Rascunho - %s", time.Now().In(utils.BrazilTZ()).Format("02/01 às 15:04"))
			}

			sessionBytes, _ := json.Marshal(sessionState)
			_, err := c.PostTemplateService.SaveTemplate(context.Background(), userID, draftName, string(sessionBytes))
			if err != nil {
				_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "❌ Erro ao salvar rascunho no banco de dados.",
					ShowAlert:       true,
				})
				return nil
			}

			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            fmt.Sprintf("💾 Rascunho '%s' salvo com sucesso na sua biblioteca!", draftName),
				ShowAlert:       true,
			})
			return nil
		}

		if strings.HasPrefix(data, "pb-load-template:") {
			templateID := strings.TrimPrefix(data, "pb-load-template:")
			tpl, err := c.UserCaptionTemplateService.GetByID(context.Background(), templateID)
			if err != nil || tpl == nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Template não encontrado.",
				})
				return nil
			}
			if tpl.UserID != userID {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Este template não pertence a você.",
				})
				return nil
			}

			state.Title = ""
			state.Body = tpl.Caption
			state.Footer = ""
			state.Reactions = tpl.Reactions

			state.Buttons = []cache.PostBuilderButton{}
			for _, btn := range tpl.Buttons {
				state.Buttons = append(state.Buttons, cache.PostBuilderButton{
					Text:  btn.NameButton,
					URL:   btn.ButtonURL,
					Style: btn.Style,
				})
			}

			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)

			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      fmt.Sprintf("✅ Template <b>%s</b> carregado! Envie uma nova mídia para começar.", tpl.Code),
				ParseMode: telego.ModeHTML,
			})
			showMenuTelego(ctx, chatID, userID, c, state)
			return nil
		}

		if strings.HasPrefix(data, "pb-del-template:") {
			templateID := strings.TrimPrefix(data, "pb-del-template:")
			if err := c.UserCaptionTemplateService.Delete(context.Background(), templateID, userID); err != nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   fmt.Sprintf("❌ Erro ao excluir template: %s", err.Error()),
				})
				return nil
			}
			recordPostBuilderEvent(c, "template_deleted", services.ChannelEventStatusInfo, userID, 0, templateID, map[string]any{"action": "delete_template"}, nil)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "✅ Template excluído com sucesso!",
				ParseMode: telego.ModeHTML,
			})
			showMenuTelego(ctx, chatID, userID, c, state)
			return nil
		}

		if strings.HasPrefix(data, "pb-send-to-channels:") {
			sessionID := strings.TrimPrefix(data, "pb-send-to-channels:")
			messageID := update.CallbackQuery.Message.GetMessageID()
			handleSendToChannelsTelego(ctx, chatID, userID, messageID, sessionID, update.CallbackQuery.ID, c)
			return nil
		}

		if strings.HasPrefix(data, "pb-send-apply:") {
			parts := strings.Split(strings.TrimPrefix(data, "pb-send-apply:"), ":")
			if len(parts) == 2 {
				channelID, _ := strconv.ParseInt(parts[0], 10, 64)
				sessionID := parts[1]
				handleSendApplyTelego(ctx, chatID, userID, channelID, sessionID, c)
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-sch:") {
			// pb-sch:<sessionID>:<channelID>  OR  pb-sch:<sessionID>
			payload := strings.TrimPrefix(data, "pb-sch:")
			parts := strings.Split(payload, ":")
			sessionID := parts[0]
			messageID := update.CallbackQuery.Message.GetMessageID()

			if len(parts) == 2 {
				// Channel selected → show schedule type options
				channelID := parts[1]
				handleScheduleTypeSelection(ctx, chatID, userID, messageID, sessionID, channelID, c)
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-sch-type:") {
			// pb-sch-type:<sessionID>:<channelID>:<type>
			payload := strings.TrimPrefix(data, "pb-sch-type:")
			parts := strings.Split(payload, ":")
			if len(parts) == 3 {
				sessionID := parts[0]
				channelID := parts[1]
				scheduleType := parts[2]
				messageID := update.CallbackQuery.Message.GetMessageID()
				handleScheduleTypeAction(ctx, chatID, userID, messageID, sessionID, channelID, scheduleType, c)
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-sch-confirm:") {
			// pb-sch-confirm:<scheduleID>:<channelID>:<sessionID>
			payload := strings.TrimPrefix(data, "pb-sch-confirm:")
			parts := strings.Split(payload, ":")
			if len(parts) == 3 {
				scheduleID := parts[0]
				channelID, _ := strconv.ParseInt(parts[1], 10, 64)
				sessionID := parts[2]
				messageID := update.CallbackQuery.Message.GetMessageID()
				handleScheduleConfirm(ctx, chatID, userID, messageID, scheduleID, channelID, sessionID, c)
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-sch-edit:") {
			// pb-sch-edit:<scheduleID>:<channelID>:<sessionID>
			payload := strings.TrimPrefix(data, "pb-sch-edit:")
			parts := strings.Split(payload, ":")
			if len(parts) == 3 {
				scheduleID := parts[0]
				channelID, _ := strconv.ParseInt(parts[1], 10, 64)
				sessionID := parts[2]
				messageID := update.CallbackQuery.Message.GetMessageID()
				handleScheduleTypeSelection(ctx, chatID, userID, messageID, sessionID, fmt.Sprintf("%d", channelID), c)
				_ = scheduleID
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-sch-autodel:") {
			payload := strings.TrimPrefix(data, "pb-sch-autodel:")
			parts := strings.Split(payload, ":")
			if len(parts) >= 2 {
				scheduleID := parts[0]
				sessionID := parts[1]
				schedule, err := c.SchedulerService.GetScheduleByID(context.Background(), scheduleID)
				if err == nil && schedule != nil && schedule.OwnerID == userID {
					newMin := 0
					switch schedule.AutoDeleteMin {
					case 0:
						newMin = 60
					case 60:
						newMin = 360
					case 360:
						newMin = 720
					case 720:
						newMin = 1440
					case 1440:
						newMin = 2880
					default:
						newMin = 0
					}
					_ = c.SchedulerService.UpdateScheduleAutoDelete(context.Background(), scheduleID, userID, newMin)
					_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
						CallbackQueryID: update.CallbackQuery.ID,
						Text:            "⏱️ Auto-destruição atualizada com sucesso!",
					})
					messageID := update.CallbackQuery.Message.GetMessageID()
					handleScheduleConfirm(ctx, chatID, userID, messageID, scheduleID, schedule.ChannelID, sessionID, c)
					return nil
				}
			}
			return nil
		}

		if strings.HasPrefix(data, "pb-schedule:") {
			sessionID := strings.TrimPrefix(data, "pb-schedule:")
			messageID := update.CallbackQuery.Message.GetMessageID()
			handleSchedulePost(ctx, chatID, userID, messageID, sessionID, update.CallbackQuery.ID, c)
			return nil
		}

		switch data {
		case "pb-start":
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			})
			if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
				state.MenuMessageID = update.CallbackQuery.Message.GetMessageID()
				c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
			}
			showMenuTelego(ctx, chatID, userID, c, state)
		case "pb-manage-buttons":
			showButtonManagerTelego(ctx, chatID, userID, c, state)
		case "pb-import-channel":
			channels, err := c.ChannelService.GetUserChannels(context.Background(), userID)
			if err != nil || len(channels) == 0 {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: chatID},
					Text:      "❌ Você não possui canais cadastrados ou ocorreu um erro.",
					ParseMode: telego.ModeHTML,
				})
				return nil
			}

			var rows [][]telego.InlineKeyboardButton
			for _, ch := range channels {
				rows = append(rows, []telego.InlineKeyboardButton{
					{Text: "📣 " + ch.Title, CallbackData: fmt.Sprintf("pb-import-apply:%d", ch.ID)},
				})
			}
			rows = append(rows, []telego.InlineKeyboardButton{
				{Text: "🔙 Voltar", CallbackData: "pb-start"},
			})

			kb := &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
			text := "📥 <b>Importar de Canal</b>\n\nEscolha o canal de onde deseja copiar a legenda padrão, reações e botões:"

			if state.MenuMessageID != 0 {
				_, err := bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:      telego.ChatID{ID: chatID},
					MessageID:   state.MenuMessageID,
					Text:        text,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: kb,
				})
				if err == nil {
					return nil
				}
			}

			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:      telego.ChatID{ID: chatID},
				Text:        text,
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: kb,
			})
		case "pb-edit-title":
			state.Step = "awaiting_title"
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        "📝 Envie o <b>Título</b> da postagem (suporta formatação):",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: promptBackKB(),
			})
			state.PromptMessageID = messageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		case "pb-edit-body":
			state.Step = "awaiting_body"
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        "📄 Envie o <b>Corpo</b> da postagem (suporta formatação):",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: promptBackKB(),
			})
			state.PromptMessageID = messageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		case "pb-edit-footer":
			state.Step = "awaiting_footer"
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        "👣 Envie o <b>Rodapé</b> da postagem (suporta formatação):",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: promptBackKB(),
			})
			state.PromptMessageID = messageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		case "pb-edit-reactions":
			state.Step = "awaiting_reactions"
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        "🎭 Envie as <b>Reações</b> separadas por vírgula (ex: 👍,👎,❤️):",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: promptBackKB(),
			})
			state.PromptMessageID = messageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		case "pb-add-button":
			state.Step = "awaiting_button"
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        "🔘 Envie os dados do botão no formato:\n\n<code>Nome do Botão\nhttps://link.com</code>",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: promptBackKB(),
			})
			state.PromptMessageID = messageID
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
		case "pb-autodelete":
			switch state.AutoDeleteMin {
			case 0:
				state.AutoDeleteMin = 60
			case 60:
				state.AutoDeleteMin = 360
			case 360:
				state.AutoDeleteMin = 720
			case 720:
				state.AutoDeleteMin = 1440
			case 1440:
				state.AutoDeleteMin = 2880
			default:
				state.AutoDeleteMin = 0
			}
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
			showMenuTelego(ctx, chatID, userID, c, state)
		case "pb-preview":
			err := sendFinalPostTelego(ctx, chatID, userID, c, state, false)
			status := services.ChannelEventStatusSuccess
			if err != nil {
				status = services.ChannelEventStatusError
			}
			recordPostBuilderEvent(c, "postbuilder_preview_sent", status, userID, 0, "", map[string]any{"media_type": state.MediaType, "buttons": len(state.Buttons)}, err)
			// Reset MenuMessageID so showMenuTelego sends a NEW message below the preview
			state.MenuMessageID = 0
			c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
			showMenuTelego(ctx, chatID, userID, c, state)
		case "pb-save":
			id, err := c.CacheService.SavePostBuilderSession(context.Background(), *state)
			if err != nil {
				logger.Error("BOT", "PostBuilder: Error saving session: %v", err)
				recordPostBuilderEvent(c, "postbuilder_failed", services.ChannelEventStatusError, userID, 0, "", map[string]any{"action": "save"}, err)
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Erro ao salvar postagem.",
				})
				return nil
			}
			botInfo, _ := bot.GetMe(context.Background())

			query := "pb " + id
			kb := &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{Text: "🚀 Compartilhar", SwitchInlineQuery: &query},
					},
					{
						{Text: "📢 Enviar para Canais", CallbackData: "pb-send-to-channels:" + id},
					},
					{
						{Text: "📅 Agendar Envio", CallbackData: "pb-schedule:" + id},
					},
					{
						{Text: "💾 Salvar como Template", CallbackData: "pb-save-template:" + id},
					},
					{
						{Text: "❌ Cancelar", CallbackData: "pb-cancel", Style: "danger"},
					},
				},
			}

			recordPostBuilderEvent(c, "postbuilder_saved", services.ChannelEventStatusSuccess, userID, 0, id, map[string]any{"media_type": state.MediaType, "buttons": len(state.Buttons), "has_reactions": state.Reactions != ""}, nil)
			messageID := update.CallbackQuery.Message.GetMessageID()
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      telego.ChatID{ID: chatID},
				MessageID:   messageID,
				Text:        fmt.Sprintf("✅ <b>Postagem salva com sucesso!</b>\n\nUtilize o modo inline para enviar:\n<code>@%s pb %s</code>", botInfo.Username, id),
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: kb,
			})
			c.CacheService.DeletePostBuilderState(context.Background(), userID)
		case "pb-list-templates":
			templates, err := c.UserCaptionTemplateService.List(context.Background(), userID)
			if err != nil || len(templates) == 0 {
				text := "📋 <b>Meus Templates</b>\n\nVocê não possui templates salvos ainda.\n\n💡 Salve um post como template no menu do Post Builder."
				if state.MenuMessageID != 0 {
					_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
						ChatID:      telego.ChatID{ID: chatID},
						MessageID:   state.MenuMessageID,
						Text:        text,
						ParseMode:   telego.ModeHTML,
						ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{{{Text: "🔙 Voltar ao Menu", CallbackData: "pb-start"}}}},
					})
				} else {
					_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
						ChatID:    telego.ChatID{ID: chatID},
						Text:      text,
						ParseMode: telego.ModeHTML,
					})
				}
				return nil
			}

			var rows [][]telego.InlineKeyboardButton
			for _, t := range templates {
				label := t.Code
				if len(label) > 30 {
					label = label[:27] + "..."
				}
				rows = append(rows, []telego.InlineKeyboardButton{
					{Text: "📄 " + label, CallbackData: "pb-load-template:" + t.ID},
				})
			}
			rows = append(rows, []telego.InlineKeyboardButton{
				{Text: "🗑 Gerenciar Templates", CallbackData: "pb-manage-templates"},
			})
			rows = append(rows, []telego.InlineKeyboardButton{
				{Text: "🔙 Voltar ao Menu", CallbackData: "pb-start"},
			})

			text := fmt.Sprintf("📋 <b>Meus Templates</b>\n\n%d template(s) salvo(s). Selecione um para carregar:", len(templates))
			if state.MenuMessageID != 0 {
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:      telego.ChatID{ID: chatID},
					MessageID:   state.MenuMessageID,
					Text:        text,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: rows},
				})
			} else {
				msg, _ := bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:      telego.ChatID{ID: chatID},
					Text:        text,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: rows},
				})
				if msg != nil {
					state.MenuMessageID = msg.MessageID
					c.CacheService.SetPostBuilderState(context.Background(), userID, *state)
				}
			}

		case "pb-manage-templates":
			templates, err := c.UserCaptionTemplateService.List(context.Background(), userID)
			if err != nil || len(templates) == 0 {
				_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "Nenhum template para gerenciar.",
				})
				return nil
			}

			var rows [][]telego.InlineKeyboardButton
			for _, t := range templates {
				label := t.Code
				if len(label) > 25 {
					label = label[:22] + "..."
				}
				rows = append(rows, []telego.InlineKeyboardButton{
					{Text: "❌ " + label, CallbackData: "pb-del-template:" + t.ID},
				})
			}
			rows = append(rows, []telego.InlineKeyboardButton{
				{Text: "🔙 Voltar", CallbackData: "pb-list-templates"},
			})

			text := "🗑 <b>Gerenciar Templates</b>\n\nClique em um template para excluí-lo:"
			if state.MenuMessageID != 0 {
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:      telego.ChatID{ID: chatID},
					MessageID:   state.MenuMessageID,
					Text:        text,
					ParseMode:   telego.ModeHTML,
					ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: rows},
				})
			}

		case "pb-cancel":
			c.CacheService.DeletePostBuilderState(context.Background(), userID)
			if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
				messageID := update.CallbackQuery.Message.GetMessageID()
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:    telego.ChatID{ID: chatID},
					MessageID: messageID,
					Text:      "❌ Post Builder cancelado.",
				})
			} else {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: telego.ChatID{ID: chatID},
					Text:   "❌ Post Builder cancelado.",
				})
			}
		}

		return nil
	}
}

func handleSendToChannelsTelego(ctx *telegohandler.Context, chatID, userID int64, messageID int, sessionID, callbackQueryID string, c *container.AppContainer) {
	bot := ctx.Bot()

	state, err := c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
	if err != nil || state == nil {
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQueryID,
			Text:            "⚠️ Sessão da postagem expirada ou não encontrada.",
			ShowAlert:       true,
		})
		backKB := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
				},
			},
		}
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   messageID,
			Text:        "❌ <b>Sessão Expirada</b>\n\nEsta postagem não foi encontrada no servidor.",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: backKB,
		})
		return
	}

	channels, err := c.ChannelService.GetUserChannels(context.Background(), userID)
	if err != nil || len(channels) == 0 {
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQueryID,
			Text:            "⚠️ Você não possui nenhum canal cadastrado para enviar postagens!",
			ShowAlert:       true,
		})
		backKB := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
				},
			},
		}
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   messageID,
			Text:        "⚠️ <b>Nenhum Canal Cadastrado</b>\n\nVocê precisa cadastrar pelo menos um canal no bot para enviar postagens.",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: backKB,
		})
		return
	}

	_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQueryID,
	})

	var rows [][]telego.InlineKeyboardButton
	for _, ch := range channels {
		rows = append(rows, []telego.InlineKeyboardButton{
			{Text: "📣 " + ch.Title, CallbackData: fmt.Sprintf("pb-send-apply:%d:%s", ch.ID, sessionID)},
		})
	}
	rows = append(rows, []telego.InlineKeyboardButton{
		{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
	})

	kb := &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: chatID},
		MessageID:   messageID,
		Text:        "📢 <b>Enviar para Canal</b>\n\nSelecione o canal para o qual deseja enviar esta postagem:",
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})
}

func handleSendApplyTelego(ctx *telegohandler.Context, chatID, userID int64, channelID int64, sessionID string, c *container.AppContainer) {
	bot := ctx.Bot()
	state, err := c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
	if err != nil || state == nil {
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   "❌ Sessão de postagem não encontrada ou expirada.",
		})
		return
	}

	err = sendFinalPostTelego(ctx, channelID, userID, c, state, false)
	if err != nil {
		recordPostBuilderEvent(c, "postbuilder_failed", services.ChannelEventStatusError, userID, channelID, sessionID, map[string]any{"action": "send_to_channel"}, err)
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   "❌ Erro ao enviar postagem para o canal.",
		})
		return
	}
	recordPostBuilderEvent(c, "postbuilder_sent_to_channel", services.ChannelEventStatusSuccess, userID, channelID, sessionID, map[string]any{"media_type": state.MediaType, "buttons": len(state.Buttons)}, nil)

	completionKB := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
			},
		},
	}

	_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: chatID},
		Text:        "✅ Postagem enviada com sucesso para o canal!",
		ReplyMarkup: completionKB,
	})
}

func sendFinalPostTelego(ctx *telegohandler.Context, chatID, userID int64, c *container.AppContainer, state *cache.PostBuilderState, deleteState bool) error {
	var sb strings.Builder
	bot := ctx.Bot()

	if state.Title != "" {
		sb.WriteString(state.Title + "\n\n")
	}
	if state.Body != "" {
		sb.WriteString(state.Body + "\n\n")
	}
	if state.Footer != "" {
		sb.WriteString(state.Footer)
	}
	caption := sb.String()

	// Safeguard: converte links Markdown crus preservando HTML ja existente, como <tg-emoji>.
	if utils.HasMarkdownLink(caption) {
		if strings.Contains(caption, "<") {
			caption = utils.NormalizeMarkdownLinks(caption, "postbuilder.final")
		} else {
			caption = channelpost.DetectParseMode(caption)
		}
	} else if channelpost.IsMarkdown(caption) && !strings.Contains(caption, "<a href=") && !strings.Contains(caption, "<b>") && !strings.Contains(caption, "<tg-emoji") {
		caption = channelpost.DetectParseMode(caption)
	}

	var kb telego.ReplyMarkup
	if len(state.Buttons) > 0 || state.Reactions != "" {
		ikb := &telego.InlineKeyboardMarkup{}
		for _, btn := range state.Buttons {
			ikb.InlineKeyboard = append(ikb.InlineKeyboard, []telego.InlineKeyboardButton{
				buildPostBuilderURLButton(btn, true),
			})
		}

		if state.Reactions != "" {
			reactions := strings.Split(state.Reactions, ",")
			var reactionRow []telego.InlineKeyboardButton
			for _, r := range reactions {
				val := strings.TrimSpace(r)
				if val != "" {
					btn := telego.InlineKeyboardButton{
						CallbackData: "vote:" + val,
					}
					if strings.HasPrefix(val, "eid:") {
						btn.IconCustomEmojiID = strings.TrimPrefix(val, "eid:")
						btn.Text = " " // Texto mínimo para botões com ícone
					} else {
						btn.Text = val
					}
					reactionRow = append(reactionRow, btn)
				}
			}
			if len(reactionRow) > 0 {
				ikb.InlineKeyboard = append(ikb.InlineKeyboard, reactionRow)
			}
		}
		kb = ikb
	}

	paramsPhoto := &telego.SendPhotoParams{
		ChatID:    telego.ChatID{ID: chatID},
		Photo:     telego.InputFile{FileID: state.MediaFileID},
		Caption:   caption,
		ParseMode: telego.ModeHTML,
	}
	if kb != nil {
		paramsPhoto.ReplyMarkup = kb
	}

	var sentMsgID int
	var err error
	switch state.MediaType {
	case "photo":
		msg, e := bot.SendPhoto(context.Background(), paramsPhoto)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	case "video":
		params := &telego.SendVideoParams{
			ChatID:    telego.ChatID{ID: chatID},
			Video:     telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendVideo(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	case "animation":
		params := &telego.SendAnimationParams{
			ChatID:    telego.ChatID{ID: chatID},
			Animation: telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendAnimation(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	case "audio":
		params := &telego.SendAudioParams{
			ChatID:    telego.ChatID{ID: chatID},
			Audio:     telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendAudio(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	case "document":
		params := &telego.SendDocumentParams{
			ChatID:    telego.ChatID{ID: chatID},
			Document:  telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendDocument(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	case "sticker":
		params := &telego.SendStickerParams{
			ChatID:  telego.ChatID{ID: chatID},
			Sticker: telego.InputFile{FileID: state.MediaFileID},
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendSticker(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	default:
		params := &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			Text:      caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, e := bot.SendMessage(context.Background(), params)
		err = e
		if msg != nil {
			sentMsgID = msg.MessageID
		}
	}

	if err != nil {
		logger.Error("BOT", "PostBuilder: Error sending final post: %v", err)
	} else if deleteState && sentMsgID > 0 && state.AutoDeleteMin > 0 && c.AutoDeleteService != nil {
		_ = c.AutoDeleteService.ScheduleAutoDelete(context.Background(), chatID, sentMsgID, state.AutoDeleteMin)
	}

	if deleteState {
		c.CacheService.DeletePostBuilderState(context.Background(), userID)
	}
	return err
}

func ChosenInlineResultHandlerTelego(c *container.AppContainer) telegohandler.ChosenInlineResultHandler {
	return func(ctx *telegohandler.Context, result telego.ChosenInlineResult) error {
		sessionID := result.ResultID
		inlineMessageID := result.InlineMessageID

		if sessionID != "" && inlineMessageID != "" {
			key := fmt.Sprintf("pb_inline_map:%s", inlineMessageID)
			err := c.CacheService.Set(context.Background(), key, sessionID, 24*time.Hour)
			if err != nil {
				logger.Error("BOT", "❌ Erro ao salvar mapeamento inline no Redis: %v", err)
			}
		}

		return nil
	}
}

func InlineHandlerTelego(c *container.AppContainer) telegohandler.InlineQueryHandler {
	return func(ctx *telegohandler.Context, inlineQuery telego.InlineQuery) error {
		bot := ctx.Bot()
		query := inlineQuery.Query
		if !strings.HasPrefix(query, "pb ") {
			return nil
		}

		id := strings.TrimSpace(strings.TrimPrefix(query, "pb "))
		if id == "" {
			return nil
		}

		state, err := c.CacheService.GetPostBuilderSession(context.Background(), id)
		if err != nil || state == nil {
			logger.Warn("BOT", "InlineHandler: Sessão %s não encontrada ou expirada", id)
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results: []telego.InlineQueryResult{
					&telego.InlineQueryResultArticle{
						Type:  "article",
						ID:    "not_found",
						Title: "❌ Postagem não encontrada",
						InputMessageContent: &telego.InputTextMessageContent{
							MessageText: "Esta postagem não existe ou já expirou.",
						},
					},
				},
				CacheTime: 0,
			})
			return nil
		}

		var sb strings.Builder
		if state.Title != "" {
			sb.WriteString(state.Title + "\n\n")
		}
		if state.Body != "" {
			sb.WriteString(state.Body + "\n\n")
		}
		if state.Footer != "" {
			sb.WriteString(state.Footer)
		}
		caption := sb.String()

		// Safeguard: converte links Markdown crus preservando HTML ja existente, como <tg-emoji>.
		if utils.HasMarkdownLink(caption) {
			if strings.Contains(caption, "<") {
				caption = utils.NormalizeMarkdownLinks(caption, "postbuilder.inline")
			} else {
				caption = channelpost.DetectParseMode(caption)
			}
		} else if channelpost.IsMarkdown(caption) && !strings.Contains(caption, "<a href=") && !strings.Contains(caption, "<b>") && !strings.Contains(caption, "<tg-emoji") {
			caption = channelpost.DetectParseMode(caption)
		}

		displayCaption := caption
		if displayCaption == "" {
			displayCaption = "Postagem sem texto."
		}

		var kb *telego.InlineKeyboardMarkup
		if len(state.Buttons) > 0 || state.Reactions != "" {
			kb = &telego.InlineKeyboardMarkup{}
			for _, btn := range state.Buttons {
				kb.InlineKeyboard = append(kb.InlineKeyboard, []telego.InlineKeyboardButton{
					buildPostBuilderURLButton(btn, false),
				})
			}

			if state.Reactions != "" {
				reactions := strings.Split(state.Reactions, ",")
				var reactionRow []telego.InlineKeyboardButton
				for _, r := range reactions {
					val := strings.TrimSpace(r)
					if val != "" {
						btn := telego.InlineKeyboardButton{
							CallbackData: "vote:" + val,
						}
						if strings.HasPrefix(val, "eid:") {
							btn.IconCustomEmojiID = strings.TrimPrefix(val, "eid:")
							btn.Text = " "
						} else {
							btn.Text = val
						}
						reactionRow = append(reactionRow, btn)
					}
				}
				if len(reactionRow) > 0 {
					kb.InlineKeyboard = append(kb.InlineKeyboard, reactionRow)
				}
			}
		}

		var result telego.InlineQueryResult
		switch state.MediaType {
		case "photo":
			res := &telego.InlineQueryResultCachedPhoto{
				Type:        "photo",
				ID:          id,
				PhotoFileID: state.MediaFileID,
				Caption:     caption,
				ParseMode:   telego.ModeHTML,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		case "video":
			res := &telego.InlineQueryResultCachedVideo{
				Type:        "video",
				ID:          id,
				VideoFileID: state.MediaFileID,
				Title:       "Video Post",
				Caption:     caption,
				ParseMode:   telego.ModeHTML,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		case "animation":
			res := &telego.InlineQueryResultCachedMpeg4Gif{
				Type:        "mpeg4_gif",
				ID:          id,
				Mpeg4FileID: state.MediaFileID,
				Caption:     caption,
				ParseMode:   telego.ModeHTML,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		case "audio":
			res := &telego.InlineQueryResultCachedAudio{
				Type:        "audio",
				ID:          id,
				AudioFileID: state.MediaFileID,
				Caption:     caption,
				ParseMode:   telego.ModeHTML,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		case "document":
			res := &telego.InlineQueryResultCachedDocument{
				Type:           "document",
				ID:             id,
				DocumentFileID: state.MediaFileID,
				Title:          "Document Post",
				Caption:        caption,
				ParseMode:      telego.ModeHTML,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		case "sticker":
			res := &telego.InlineQueryResultCachedSticker{
				Type:          "sticker",
				ID:            id,
				StickerFileID: state.MediaFileID,
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		default:
			res := &telego.InlineQueryResultArticle{
				Type:  "article",
				ID:    id,
				Title: "Post Builder",
				InputMessageContent: &telego.InputTextMessageContent{
					MessageText: displayCaption,
					ParseMode:   telego.ModeHTML,
				},
			}
			if kb != nil {
				res.ReplyMarkup = kb
			}
			result = res
		}

		if err := bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
			InlineQueryID: inlineQuery.ID,
			Results:       []telego.InlineQueryResult{result},
			CacheTime:     0,
			IsPersonal:    true,
		}); err != nil {
			logger.Error("BOT", "Erro ao responder Inline Query: %v", err)
		}

		return nil
	}
}

func handleSchedulePost(ctx *telegohandler.Context, chatID, userID int64, messageID int, sessionID, callbackQueryID string, c *container.AppContainer) {
	bot := ctx.Bot()

	state, err := c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
	if err != nil || state == nil {
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQueryID,
			Text:            "⚠️ Sessão da postagem expirada ou não encontrada.",
			ShowAlert:       true,
		})
		backKB := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
				},
			},
		}
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   messageID,
			Text:        "❌ <b>Sessão Expirada</b>\n\nEsta postagem não foi encontrada no servidor.",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: backKB,
		})
		return
	}

	channels, err := c.ChannelService.GetUserChannels(context.Background(), userID)
	if err != nil || len(channels) == 0 {
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQueryID,
			Text:            "⚠️ Você não possui nenhum canal cadastrado para usar esta funcionalidade!",
			ShowAlert:       true,
		})
		backKB := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
				},
			},
		}
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   messageID,
			Text:        "⚠️ <b>Nenhum Canal Cadastrado</b>\n\nVocê precisa cadastrar pelo menos um canal no bot para agendar postagens.",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: backKB,
		})
		return
	}

	_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: callbackQueryID,
	})

	var buttons [][]telego.InlineKeyboardButton
	for _, ch := range channels {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			{Text: ch.Title, CallbackData: "pb-sch:" + sessionID + ":" + fmt.Sprintf("%d", ch.ID)},
		})
	}
	buttons = append(buttons, []telego.InlineKeyboardButton{
		{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
	})

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: chatID},
		MessageID:   messageID,
		Text:        "📅 <b>Agendar Envio</b>\n\nSelecione o canal de destino:",
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
}

func handleScheduleTypeSelection(ctx *telegohandler.Context, chatID, userID int64, messageID int, sessionID, channelID string, c *container.AppContainer) {
	bot := ctx.Bot()

	buttons := [][]telego.InlineKeyboardButton{
		{
			{Text: "📌 Agendar uma vez", CallbackData: "pb-sch-type:" + sessionID + ":" + channelID + ":once"},
		},
		{
			{Text: "🔄 Recorrente (diário)", CallbackData: "pb-sch-type:" + sessionID + ":" + channelID + ":daily"},
		},
		{
			{Text: "📅 Recorrente (semanal)", CallbackData: "pb-sch-type:" + sessionID + ":" + channelID + ":weekly"},
		},
		{
			{Text: "⏱️ Intervalo fixo", CallbackData: "pb-sch-type:" + sessionID + ":" + channelID + ":interval"},
		},
		{
			{Text: "📋 Fila de envio", CallbackData: "pb-sch-type:" + sessionID + ":" + channelID + ":queue"},
		},
		{
			{Text: "🔙 Voltar ao Canal", CallbackData: "pb-schedule:" + sessionID},
			{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
		},
	}

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: chatID},
		MessageID:   messageID,
		Text:        "📅 <b>Tipo de Agendamento</b>\n\nEscolha como deseja agendar:",
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
}

func handleScheduleTypeAction(ctx *telegohandler.Context, chatID, userID int64, messageID int, sessionID, channelID, scheduleType string, c *container.AppContainer) {
	bot := ctx.Bot()

	scheduleState := cache.ScheduleState{
		SessionID:    sessionID,
		ChannelID:    channelID,
		ScheduleType: scheduleType,
	}
	_ = c.CacheService.SetScheduleState(context.Background(), userID, scheduleState)

	var prompt string
	switch scheduleType {
	case "once":
		prompt = "📅 <b>Agendar uma vez</b>\n\nEnvie a data e hora no formato:\n<code>DD/MM/AAAA HH:MM</code>\n\nExemplo: <code>25/07/2026 14:30</code>"
	case "daily":
		prompt = "🔄 <b>Recorrência Diária</b>\n\nEnvie o horário diário no formato:\n<code>HH:MM</code>\n\nExemplo: <code>14:30</code>"
	case "weekly":
		prompt = "📅 <b>Recorrência Semanal</b>\n\nEnvie o horário e dias da semana:\n<code>HH:MM</code>\n<code>1,3,5</code> (seg,qua,sex)"
	case "interval":
		prompt = "⏱️ <b>Intervalo Fixo</b>\n\nEnvie o intervalo em minutos e, opcionalmente, a janela de horário:\n\n<b>Linha 1:</b> intervalo em minutos\n<b>Linha 2:</b> horário início-fim (opcional)\n\nExemplos:\n<code>30</code>\n→ A cada 30 minutos, 24h/dia\n\n<code>60\n08:00-22:00</code>\n→ A cada 1 hora, das 08:00 às 22:00"
	case "queue":
		prompt = "📋 <b>Fila de Envio</b>\n\nEnvie a posição na fila (número inteiro):\n<code>1</code> = próximo\n<code>2</code> = depois do próximo"
	default:
		return
	}

	buttons := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔙 Voltar", CallbackData: "pb-sch:" + sessionID + ":" + channelID},
			{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
		},
	}

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: chatID},
		MessageID:   messageID,
		Text:        prompt,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
}

func handleScheduleConfirm(ctx *telegohandler.Context, chatID, userID int64, messageID int, scheduleID string, channelID int64, sessionID string, c *container.AppContainer) {
	bot := ctx.Bot()

	schedule, err := c.SchedulerService.GetScheduleByID(context.Background(), scheduleID)
	if err != nil || schedule == nil {
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      "❌ Agendamento não encontrado.",
		})
		return
	}

	_ = channelID

	autoDelStr := "Desativada"
	if schedule.AutoDeleteMin > 0 {
		if schedule.AutoDeleteMin%60 == 0 {
			autoDelStr = fmt.Sprintf("%dh pós-envio ⏱️", schedule.AutoDeleteMin/60)
		} else {
			autoDelStr = fmt.Sprintf("%dmin pós-envio ⏱️", schedule.AutoDeleteMin)
		}
	}

	text := fmt.Sprintf(
		"✅ <b>Agendamento Criado com Sucesso!</b>\n\n"+
			"📢 <b>Canal:</b> %s\n"+
			"🗓️ <b>Próximo Envio:</b> %s (BRT)\n"+
			"⏱️ <b>Auto-Destruição:</b> %s\n\n"+
			"<i>Você pode personalizar a auto-destruição ou salvar este post nos rascunhos abaixo:</i>",
		schedule.ChannelTitle,
		schedule.NextRunAt.In(utils.BrazilTZ()).Format("02/01/2006 às 15:04"),
		autoDelStr,
	)

	kb := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("⏱️ Auto-Destruição: %s", autoDelStr), CallbackData: fmt.Sprintf("pb-sch-autodel:%s:%s", scheduleID, sessionID)},
			},
			{
				{Text: "💾 Salvar como Rascunho", CallbackData: "pb-save-template:" + sessionID},
			},
			{
				{Text: "📋 Meus Agendamentos", CallbackData: "my-schedules"},
				{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + sessionID},
			},
		},
	}

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      telego.ChatID{ID: chatID},
		MessageID:   messageID,
		Text:        text,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})
}

func handleScheduleTextInput(ctx *telegohandler.Context, chatID, userID int64, text string, scheduleState *cache.ScheduleState, c *container.AppContainer) {
	bot := ctx.Bot()

	// Get PostBuilder state from session
	pbState, err := c.CacheService.GetPostBuilderSession(context.Background(), scheduleState.SessionID)
	if err != nil || pbState == nil {
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   "❌ Sessão do PostBuilder expirada.",
		})
		c.CacheService.DeleteScheduleState(context.Background(), userID)
		return
	}

	channelID, _ := strconv.ParseInt(scheduleState.ChannelID, 10, 64)
	brazilTZ := utils.BrazilTZ()
	postData := mustMarshal(pbState)

	completionKB := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🔙 Voltar ao Post", CallbackData: "pb-saved-menu:" + scheduleState.SessionID},
				{Text: "📋 Meus Agendamentos", CallbackData: "my-schedules"},
			},
		},
	}

	switch scheduleState.ScheduleType {
	case "once":
		// Parse "DD/MM/AAAA HH:MM"
		parsedTime, err := time.ParseInLocation("02/01/2006 15:04", text, brazilTZ)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Formato inválido. Envie no formato:\n<code>DD/MM/AAAA HH:MM</code>",
				ParseMode: telego.ModeHTML,
			})
			return
		}

		schedule, err := c.SchedulerService.CreateScheduledPost(context.Background(), userID, channelID, postData, services.ScheduleOptions{
			ScheduleType: "once",
			ScheduledAt:  &parsedTime,
		})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Erro ao criar agendamento: %s", err.Error()),
			})
			return
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        fmt.Sprintf("✅ <b>Agendamento criado!</b>\n\nEnvio único em: %s", parsedTime.Format("02/01/2006 15:04")),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})
		_ = schedule

	case "daily":
		// Parse "HH:MM"
		parsedTime, err := time.ParseInLocation("15:04", text, brazilTZ)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Formato inválido. Envie no formato:\n<code>HH:MM</code>",
				ParseMode: telego.ModeHTML,
			})
			return
		}

		scheduleTime := parsedTime.Format("15:04")
		schedule, err := c.SchedulerService.CreateScheduledPost(context.Background(), userID, channelID, postData, services.ScheduleOptions{
			ScheduleType: "daily",
			ScheduleTime: scheduleTime,
		})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Erro ao criar agendamento: %s", err.Error()),
			})
			return
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        fmt.Sprintf("✅ <b>Agendamento diário criado!</b>\n\nHorário: %s", scheduleTime),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})
		_ = schedule

	case "weekly":
		// Parse "HH:MM\n1,3,5"
		lines := strings.Split(text, "\n")
		if len(lines) < 2 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Formato inválido. Envie:\n<code>HH:MM</code>\n<code>1,3,5</code>",
				ParseMode: telego.ModeHTML,
			})
			return
		}
		parsedTime, err := time.ParseInLocation("15:04", strings.TrimSpace(lines[0]), brazilTZ)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Horário inválido.",
				ParseMode: telego.ModeHTML,
			})
			return
		}
		scheduleDays := strings.TrimSpace(lines[1])
		scheduleTime := parsedTime.Format("15:04")

		// Parse days string to []int
		var days []int
		for _, d := range strings.Split(scheduleDays, ",") {
			d = strings.TrimSpace(d)
			if num, err := strconv.Atoi(d); err == nil {
				days = append(days, num)
			}
		}

		schedule, err := c.SchedulerService.CreateScheduledPost(context.Background(), userID, channelID, postData, services.ScheduleOptions{
			ScheduleType: "weekly",
			ScheduleTime: scheduleTime,
			ScheduleDays: days,
		})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Erro ao criar agendamento: %s", err.Error()),
			})
			return
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        fmt.Sprintf("✅ <b>Agendamento semanal criado!</b>\n\nHorário: %s\nDias: %s", scheduleTime, scheduleDays),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})
		_ = schedule

	case "interval":
		lines := strings.Split(text, "\n")
		intervalMin, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil || intervalMin < 5 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Intervalo inválido. Envie um número em minutos (mínimo 5).",
				ParseMode: telego.ModeHTML,
			})
			return
		}

		var windowStart, windowEnd string
		if len(lines) >= 2 && strings.TrimSpace(lines[1]) != "" {
			windowParts := strings.Split(strings.TrimSpace(lines[1]), "-")
			if len(windowParts) == 2 {
				windowStart = strings.TrimSpace(windowParts[0])
				windowEnd = strings.TrimSpace(windowParts[1])
			} else {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:    telego.ChatID{ID: chatID},
					Text:      "❌ Janela de horário inválida. Use o formato:\n<code>HH:MM-HH:MM</code>",
					ParseMode: telego.ModeHTML,
				})
				return
			}
		}

		opts := services.ScheduleOptions{
			ScheduleType: "interval",
			IntervalMin:  intervalMin,
			WindowStart:  windowStart,
			WindowEnd:    windowEnd,
		}
		schedule, err := c.SchedulerService.CreateScheduledPost(context.Background(), userID, channelID, postData, opts)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Erro ao criar agendamento: %s", err.Error()),
			})
			return
		}

		confirmText := fmt.Sprintf("✅ <b>Agendamento por intervalo criado!</b>\n\nA cada %d minutos", intervalMin)
		if windowStart != "" {
			confirmText += fmt.Sprintf("\nJanela: %s às %s", windowStart, windowEnd)
		}
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        confirmText,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})
		_ = schedule

	case "queue":
		// Parse position
		pos, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || pos < 1 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      "❌ Posição inválida. Envie um número inteiro (1, 2, 3...).",
				ParseMode: telego.ModeHTML,
			})
			return
		}

		groupID := fmt.Sprintf("queue_%d_%d", userID, time.Now().Unix())
		schedule, err := c.SchedulerService.CreateScheduledPost(context.Background(), userID, channelID, postData, services.ScheduleOptions{
			ScheduleType:  "queue",
			QueueGroupID:  groupID,
			QueuePosition: pos,
		})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   fmt.Sprintf("❌ Erro ao criar fila: %s", err.Error()),
			})
			return
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        fmt.Sprintf("✅ <b>Fila criada!</b>\n\nPosição na fila: %d", pos),
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: completionKB,
		})
		_ = schedule
	}

	c.CacheService.DeleteScheduleState(context.Background(), userID)
}

func showSavedPostMenu(ctx *telegohandler.Context, chatID, userID int64, messageID int, sessionID string, c *container.AppContainer) {
	bot := ctx.Bot()
	state, err := c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
	if err != nil || state == nil {
		if messageID != 0 {
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: chatID},
				MessageID: messageID,
				Text:      "❌ Sessão da postagem expirada ou não encontrada.",
			})
		} else {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   "❌ Sessão da postagem expirada ou não encontrada.",
			})
		}
		return
	}

	botInfo, _ := bot.GetMe(context.Background())
	query := "pb " + sessionID
	kb := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🚀 Compartilhar", SwitchInlineQuery: &query},
			},
			{
				{Text: "📢 Enviar para Canais", CallbackData: "pb-send-to-channels:" + sessionID},
			},
			{
				{Text: "📅 Agendar Envio", CallbackData: "pb-schedule:" + sessionID},
			},
			{
				{Text: "💾 Salvar como Template", CallbackData: "pb-save-template:" + sessionID},
			},
			{
				{Text: "❌ Cancelar", CallbackData: "pb-cancel", Style: "danger"},
			},
		},
	}

	text := fmt.Sprintf("✅ <b>Postagem salva com sucesso!</b>\n\nUtilize o modo inline para enviar:\n<code>@%s pb %s</code>", botInfo.Username, sessionID)

	if messageID != 0 {
		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      telego.ChatID{ID: chatID},
			MessageID:   messageID,
			Text:        text,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
		})
	} else {
		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:      telego.ChatID{ID: chatID},
			Text:        text,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: kb,
		})
	}
}

func promptBackKB() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "🔙 Voltar ao Menu", CallbackData: "pb-start"},
			},
		},
	}
}

func mustMarshal(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
