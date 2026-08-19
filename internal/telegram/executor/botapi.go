package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

// BotAPIExecutor implementa TelegramExecutor usando a Telegram Bot API
// por meio da biblioteca telego.
type BotAPIExecutor struct {
	bot *telego.Bot
}

// NewBotAPIExecutor cria um novo executor baseado na Bot API.
func NewBotAPIExecutor(bot *telego.Bot) *BotAPIExecutor {
	return &BotAPIExecutor{bot: bot}
}

// toTelegoKeyboard converte o InlineKeyboardMarkup generico para o formato do telego.
func toTelegoKeyboard(keyboard *InlineKeyboardMarkup) *telego.InlineKeyboardMarkup {
	if keyboard == nil || len(keyboard.InlineKeyboard) == 0 {
		return nil
	}

	rows := make([][]telego.InlineKeyboardButton, len(keyboard.InlineKeyboard))
	for i, row := range keyboard.InlineKeyboard {
		buttons := make([]telego.InlineKeyboardButton, len(row))
		for j, btn := range row {
			tb := telego.InlineKeyboardButton{
				Text:  btn.Text,
				Style: btn.Style,
			}
			if btn.URL != "" {
				tb.URL = btn.URL
			}
			if btn.CallbackData != "" {
				tb.CallbackData = btn.CallbackData
			}
			buttons[j] = tb
		}
		rows[i] = buttons
	}

	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func (e *BotAPIExecutor) EditMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
	}

	if opts != nil && opts.DisableLinkPreview {
		params.LinkPreviewOptions = &telego.LinkPreviewOptions{IsDisabled: true}
	}

	if kb := toTelegoKeyboard(keyboard); kb != nil {
		params.ReplyMarkup = kb
	}

	return e.retry(ctx, func() error {
		_, err := e.bot.EditMessageText(ctx, params)
		return err
	})
}

func (e *BotAPIExecutor) EditCaption(
	ctx context.Context,
	chatID int64,
	messageID int,
	caption string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	params := &telego.EditMessageCaptionParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Caption:   caption,
		ParseMode: telego.ModeHTML,
	}

	if kb := toTelegoKeyboard(keyboard); kb != nil {
		params.ReplyMarkup = kb
	}

	return e.retry(ctx, func() error {
		_, err := e.bot.EditMessageCaption(ctx, params)
		return err
	})
}

func (e *BotAPIExecutor) EditReplyMarkup(
	ctx context.Context,
	chatID int64,
	messageID int,
	keyboard *InlineKeyboardMarkup,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if keyboard == nil || len(keyboard.InlineKeyboard) == 0 {
		return nil
	}

	params := &telego.EditMessageReplyMarkupParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
	}

	if kb := toTelegoKeyboard(keyboard); kb != nil {
		params.ReplyMarkup = kb
	}

	return e.retry(ctx, func() error {
		_, err := e.bot.EditMessageReplyMarkup(ctx, params)
		return err
	})
}

func (e *BotAPIExecutor) SendSticker(
	ctx context.Context,
	chatID int64,
	stickerID string,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	params := &telego.SendStickerParams{
		ChatID:  telego.ChatID{ID: chatID},
		Sticker: telego.InputFile{FileID: stickerID},
	}

	return e.retry(ctx, func() error {
		_, err := e.bot.SendSticker(ctx, params)
		return err
	})
}

func (e *BotAPIExecutor) DeleteMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	params := &telego.DeleteMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
	}

	return e.retry(ctx, func() error {
		return e.bot.DeleteMessage(ctx, params)
	})
}

func (e *BotAPIExecutor) SendMessage(
	ctx context.Context,
	chatID int64,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
	}

	// Se tiver entities explicitas, usar ao inves de parse_mode
	if opts != nil && opts.Entities != "" {
		params.ParseMode = ""
		entities, err := entitiesJSONToTelego(opts.Entities)
		if err != nil {
			logger.Warn("BOTAPI", "SendMessage: erro ao parsear entities: %v", err)
		} else if len(entities) > 0 {
			params.Entities = entities
		}
	}

	if kb := toTelegoKeyboard(keyboard); kb != nil {
		params.ReplyMarkup = kb
	}

	return e.retry(ctx, func() error {
		_, err := e.bot.SendMessage(ctx, params)
		return err
	})
}

// retry executa fn com ate 3 tentativas em caso de rate limit.
func (e *BotAPIExecutor) retry(ctx context.Context, fn func() error) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if isRateLimit(err) {
			retryAfter := extractRetryAfter(err.Error())
			if retryAfter == 0 {
				retryAfter = (attempt + 1) * 2
			}
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}

		return err
	}
	return fmt.Errorf("failed after %d attempts", maxRetries)
}

func isRateLimit(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "too many requests") || strings.Contains(msg, "429")
}

func extractRetryAfter(errMsg string) int {
	fields := strings.Fields(errMsg)
	for i, field := range fields {
		if strings.EqualFold(field, "after") && i+1 < len(fields) {
			var n int
			if _, err := fmt.Sscanf(fields[i+1], "%d", &n); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
