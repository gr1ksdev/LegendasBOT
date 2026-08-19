package channelpost

import (
	"context"
	"strings"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

func UpdateChannelBasicInfoTelego(ctx context.Context, b *telego.Bot, chatID int64, channel *dbmodels.Channel, chatObj interface{}) (*dbmodels.Channel, bool) {
	var title, username, inviteLink string

	if chatObj != nil {
		switch c := chatObj.(type) {
		case *telego.Chat:
			title = c.Title
			username = c.Username
		case telego.Chat:
			title = c.Title
			username = c.Username
		case *telego.ChatFullInfo:
			title = c.Title
			username = c.Username
			inviteLink = c.InviteLink
		case telego.ChatFullInfo:
			title = c.Title
			username = c.Username
			inviteLink = c.InviteLink
		}
	} else {
		chat, err := b.GetChat(context.Background(), &telego.GetChatParams{
			ChatID: telego.ChatID{ID: chatID},
		})
		if err != nil {
			return channel, false
		}
		title = chat.Title
		username = chat.Username
		inviteLink = chat.InviteLink
	}

	updated := false
	if title != "" {
		cleanTitle := utils.RemoveHTMLTags(title)
		if cleanTitle != channel.Title {
			channel.Title = cleanTitle
			updated = true
		}
	}

	// Lógica de URL: Username Público (@) sempre tem prioridade sobre link privado (t.me/+)
	var newInviteURL string
	if username != "" {
		newInviteURL = utils.NormalizeTelegramURL("@" + username)
	} else if inviteLink != "" {
		newInviteURL = utils.NormalizeTelegramURL(inviteLink)
	}

	oldInviteURL := channel.InviteURL
	if newInviteURL != "" && newInviteURL != oldInviteURL {
		channel.InviteURL = newInviteURL
		updated = true
	}

	// Atualizar botões que apontam para o link antigo do canal
	if updated && len(channel.Buttons) > 0 {
		buttonsUpdated := syncChannelButtons(ctx, channel, title, newInviteURL, oldInviteURL)
		if buttonsUpdated {
			updated = true
		}
	}

	return channel, updated
}

// syncChannelButtons verifica se o canal possui botões que apontam para o link antigo do canal
// e o atualiza para o novo nome/url, respeitando a escolha do usuário de ter ou não esse botão.
func syncChannelButtons(ctx context.Context, channel *dbmodels.Channel, title, newURL, oldURL string) bool {
	newURL = utils.NormalizeTelegramURL(newURL)
	oldURL = utils.NormalizeTelegramURL(oldURL)
	if newURL == "" {
		return false
	}

	hasChanges := false
	for i := range channel.Buttons {
		btn := &channel.Buttons[i]

		// Caso 1: O botão é exatamente o link antigo
		btnURL := utils.NormalizeTelegramURL(btn.ButtonURL)

		if btnURL == oldURL {
			logger.Bot("🔄 Sincronizando botão '%s': '%s' -> '%s'", btn.NameButton, btn.ButtonURL, newURL)
			btn.NameButton = utils.RemoveHTMLTags(title)
			btn.ButtonURL = newURL
			hasChanges = true
		} else {
			// Caso 2: O botão contém o link antigo como substring (ex: joinchat/...)
			// mas o novo link é um @username
			cleanOldURL := strings.TrimPrefix(oldURL, "https://t.me/+")
			cleanNewURL := newURL

			if strings.Contains(btnURL, cleanOldURL) && !strings.Contains(btnURL, cleanNewURL) {
				logger.Bot("🔄 Sincronizando botão '%s': '%s' -> '%s'", btn.NameButton, btn.ButtonURL, cleanNewURL)
				btn.NameButton = utils.RemoveHTMLTags(title)
				btn.ButtonURL = cleanNewURL
				hasChanges = true
			}
		}
	}

	return hasChanges
}
