package channelpost

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

// Estado por canal: quando recebeu !newpack e está aguardando sticker
type newPackState struct {
	WaitingForSticker bool
	MessageID         int
}

var newPackStates sync.Map // map[int64]newPackState
var cmdNewPackRegex = regexp.MustCompile(`^[!/]newpack(?:\s|$)`)

func TryHandleNewPackTelego(ctx context.Context, b *telego.Bot, channel dbmodels.Channel, post telego.Message) (handled bool, err error) {
	channelID := post.Chat.ID
	mgm := GetMediaGroupManagerTelego()

	if post.Text != "" && cmdNewPackRegex.MatchString(strings.TrimSpace(post.Text)) {
		mgm.SetNewPackActive(channelID, true)
		newPackStates.Store(channelID, newPackState{
			WaitingForSticker: true,
			MessageID:         post.MessageID,
		})

		_, err := b.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: channelID},
			MessageID: post.MessageID,
			Text:      "Envie-me um sticker do seu pack...",
		})
		if err != nil {
			logger.Error("BOT", "falha ao solicitar sticker newpack: %v", err)
			newPackStates.Delete(channelID)
			mgm.SetNewPackActive(channelID, false)
			return true, err
		}
		return true, nil
	}

	if post.Sticker != nil {
		v, ok := newPackStates.Load(channelID)
		if !ok {
			return false, nil
		}
		state := v.(newPackState)
		if !state.WaitingForSticker {
			return false, nil
		}

		setName := post.Sticker.SetName
		if strings.TrimSpace(setName) == "" {
			_, _ = b.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: channelID},
				Text:   "Sticker não faz parte de um pack público.",
			})
			newPackStates.Delete(channelID)
			mgm.SetNewPackActive(channelID, false)
			return true, nil
		}

		pack, err := b.GetStickerSet(context.Background(), &telego.GetStickerSetParams{
			Name: setName,
		})
		if err != nil {
			newPackStates.Delete(channelID)
			mgm.SetNewPackActive(channelID, false)
			return true, err
		}

		packURL := fmt.Sprintf("https://t.me/addstickers/%s", setName)

		title := strings.TrimSpace(pack.Title)
		if title == "" {
			title = "Meu Pack"
		}
		link := fmt.Sprintf("https://t.me/addstickers/%s", setName)

		tpl := channel.NewPackCaption
		if strings.TrimSpace(tpl) == "" {
			tpl = "$titulo\n$link"
		}

		stickerCount := len(pack.Stickers)
		caption := renderNewPackTemplate(tpl, title, link, stickerCount)
		captionHTML := renderNewPackHTML(caption)
		messageButtons := newPackButtonEnabled(channel.NewPackMessageButtons)
		stickerButtons := newPackButtonEnabled(channel.NewPackStickerButtons)
		messagePosition := newPackMessagePosition(channel.NewPackMessagePosition)
		replyToSticker := newPackReplyToSticker(channel.NewPackReplyToSticker) && messagePosition == "below"

		logger.Bot("🧩 NewPack canal=%d set=%s title=%q stickers=%d msgButton=%v stickerButton=%v position=%s replySticker=%v", channelID, setName, title, stickerCount, messageButtons, stickerButtons, messagePosition, replyToSticker)
		logger.Bot("🧩 NewPack template bruto: %q", tpl)
		logger.Bot("🧩 NewPack caption renderizada: %q", caption)
		logger.Bot("🧩 NewPack HTML final: %q", captionHTML)

		pm := GetPermissionManager()
		perms := pm.CheckPermissions(&channel, MessageTypeText)
		disableLP := !perms.CanUseLinkPreview
		logger.Bot("🧩 NewPack link preview habilitado=%v", !disableLP)

		kb := telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: title, URL: packURL},
				},
			},
		}

		if messagePosition == "below" {
			sendParams := telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: channelID},
				Text:      captionHTML,
				ParseMode: telego.ModeHTML,
			}
			if disableLP {
				sendParams.LinkPreviewOptions = &telego.LinkPreviewOptions{IsDisabled: true}
			}
			if messageButtons {
				sendParams.ReplyMarkup = &kb
			}
			if replyToSticker {
				sendParams.ReplyParameters = &telego.ReplyParameters{MessageID: post.MessageID, AllowSendingWithoutReply: true}
			}

			if _, err = b.SendMessage(context.Background(), &sendParams); err != nil {
				logger.Error("BOT", "falha ao enviar mensagem newpack abaixo: %v | html=%q", err, captionHTML)
				newPackStates.Delete(channelID)
				mgm.SetNewPackActive(channelID, false)
				return true, err
			}

			if err := b.DeleteMessage(context.Background(), &telego.DeleteMessageParams{ChatID: telego.ChatID{ID: channelID}, MessageID: state.MessageID}); err != nil {
				logger.Error("BOT", "falha ao apagar mensagem de espera newpack: %v", err)
			}
		} else {
			editParams := telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: channelID},
				MessageID: state.MessageID,
				Text:      captionHTML,
				ParseMode: telego.ModeHTML,
			}
			if disableLP {
				editParams.LinkPreviewOptions = &telego.LinkPreviewOptions{IsDisabled: true}
			}
			if messageButtons {
				editParams.ReplyMarkup = &kb
			}

			if _, err = b.EditMessageText(context.Background(), &editParams); err != nil {
				logger.Error("BOT", "falha ao editar mensagem newpack: %v | html=%q", err, captionHTML)
				newPackStates.Delete(channelID)
				mgm.SetNewPackActive(channelID, false)
				return true, err
			}
		}

		if stickerButtons {
			_, err = b.EditMessageReplyMarkup(context.Background(), &telego.EditMessageReplyMarkupParams{
				ChatID:      telego.ChatID{ID: channelID},
				MessageID:   post.MessageID,
				ReplyMarkup: &kb,
			})

			if err != nil {
				logger.Error("BOT", "falha ao editar reply markup newpack: %v", err)
			}
		}

		if channel.Separator != nil && channel.Separator.SeparatorID != "" {
			_, _ = b.SendSticker(context.Background(), &telego.SendStickerParams{
				ChatID:  telego.ChatID{ID: channelID},
				Sticker: telego.InputFile{FileID: channel.Separator.SeparatorID},
			})
		}

		newPackStates.Delete(channelID)
		mgm.SetNewPackActive(channelID, false)
		return true, nil
	}

	return false, nil
}

func newPackButtonEnabled(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func newPackMessagePosition(value *string) string {
	if value == nil || *value != "below" {
		return "above"
	}
	return "below"
}

func newPackReplyToSticker(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func renderNewPackTemplate(tpl, title, link string, stickerCount int) string {
	count := fmt.Sprintf("%d", stickerCount)
	res := strings.ReplaceAll(tpl, "$titulo", title)
	res = strings.ReplaceAll(res, "$title", title)
	res = strings.ReplaceAll(res, "$name", title)
	res = strings.ReplaceAll(res, "$link", link)
	res = strings.ReplaceAll(res, "$count", count)
	res = strings.ReplaceAll(res, "$total", count)
	res = strings.ReplaceAll(res, "$stickers", count)
	return res
}

func renderNewPackHTML(text string) string {
	if text == "" {
		return ""
	}

	res := html.EscapeString(text)
	linkRegex := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	links := make([]string, 0)
	res = linkRegex.ReplaceAllStringFunc(res, func(m string) string {
		matches := linkRegex.FindStringSubmatch(m)
		if len(matches) != 3 {
			return m
		}

		label := matches[1]
		url := strings.TrimSpace(matches[2])
		if url == "" {
			return m
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "tg://") {
			url = "https://" + url
		}

		token := fmt.Sprintf("NEWPACKLINKTOKEN%d", len(links))
		links = append(links, fmt.Sprintf(`<a href="%s">%s</a>`, url, label))
		return token
	})

	boldRegex := regexp.MustCompile(`\*([^\*\n]+)\*`)
	res = boldRegex.ReplaceAllString(res, "<b>$1</b>")

	italicRegex := regexp.MustCompile(`_([^_\n]+)_`)
	res = italicRegex.ReplaceAllString(res, "<i>$1</i>")

	codeRegex := regexp.MustCompile("`([^`\\n]+)`")
	res = codeRegex.ReplaceAllString(res, "<code>$1</code>")

	for i, link := range links {
		res = strings.ReplaceAll(res, fmt.Sprintf("NEWPACKLINKTOKEN%d", i), link)
	}

	return res
}
