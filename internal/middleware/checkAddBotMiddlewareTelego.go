package middleware

import (
	"context"
	"sync"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

var (
	botUserCache     *telego.User
	botUserCacheOnce sync.Once
)

func getBotUser(b *telego.Bot) *telego.User {
	botUserCacheOnce.Do(func() {
		info, err := b.GetMe(context.Background())
		if err == nil {
			botUserCache = info
		}
	})
	return botUserCache
}

func CheckAddBotMiddlewareTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.MyChatMember != nil {
			if !handleMyChatMemberTelego(ctx.Bot(), update.MyChatMember, c) {
				return nil
			}
			return ctx.Next(update)
		}

		if update.Message != nil && update.Message.ForwardOrigin != nil {
			if origin, ok := update.Message.ForwardOrigin.(*telego.MessageOriginChannel); ok {
				if !handleForwardedMessageTelego(ctx.Bot(), update.Message, origin) {
					return nil
				}
				return ctx.Next(update)
			}
		}

		return ctx.Next(update)
	}
}

func handleMyChatMemberTelego(b *telego.Bot, chatMember *telego.ChatMemberUpdated, c *container.AppContainer) bool {
	if chatMember == nil {
		return false
	}

	chatId := chatMember.Chat.ID
	oldStatus := chatMember.OldChatMember.MemberStatus()
	newStatus := chatMember.NewChatMember.MemberStatus()

	logger.Bot("Status change Telego: %s -> %s in chat %d by user %d", oldStatus, newStatus, chatId, chatMember.From.ID)

	if newStatus == telego.MemberStatusLeft || newStatus == telego.MemberStatusBanned {
		logger.Bot("Bot was removed from channel %d. Cleaning up.", chatId)
		channel, err := c.ChannelService.GetChannelByID(context.Background(), chatId)
		if err == nil {
			_ = c.ChannelService.DisconnectChannel(context.Background(), channel.OwnerID, chatId)
		}
		return false
	}

	// Se já era admin ou criador, não dispara o convite novamente (evita spam ao mudar perms)
	if oldStatus == telego.MemberStatusAdministrator || oldStatus == telego.MemberStatusCreator {
		logger.Bot("Bot já era admin/creator no canal %d. Ignorando gatilho de adição.", chatId)
		return false
	}

	if newStatus != telego.MemberStatusAdministrator && newStatus != telego.MemberStatusCreator {
		logger.Bot("Novo status no canal %d não é admin/creator (%s).", chatId, newStatus)
		return false
	}

	if !hasRequiredPermissionsTelego(chatMember.NewChatMember) {
		logger.Bot("Bot no canal %d não tem todas as permissões necessárias.", chatId)
		sendPermissionErrorMessageTelego(b, chatMember.From.ID)
		return false
	}

	logger.Bot("Bot promovido a admin com sucesso no canal %d. Prosseguindo para convite.", chatId)
	return true
}

func handleForwardedMessageTelego(b *telego.Bot, message *telego.Message, origin *telego.MessageOriginChannel) bool {
	forwardedChatID := origin.Chat.ID

	botInfo := getBotUser(b)
	if botInfo == nil {
		return false
	}
	botMember, err := b.GetChatMember(context.Background(), &telego.GetChatMemberParams{
		ChatID: telego.ChatID{ID: forwardedChatID},
		UserID: botInfo.ID,
	})

	if err != nil {
		return false
	}

	if !hasRequiredPermissionsTelego(botMember) {
		sendPermissionErrorMessageTelego(b, message.From.ID)
		return false
	}

	return true
}

func hasRequiredPermissionsTelego(chatMember telego.ChatMember) bool {
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

func sendPermissionErrorMessageTelego(b *telego.Bot, userID int64) {
	text, button := parser.GetMessageTelego("toadd-notfound-permissions-message", nil)
	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: userID},
		Text:      text,
		ParseMode: telego.ModeHTML,
	}
	if button != nil {
		params.ReplyMarkup = button
	}
	_, _ = b.SendMessage(context.Background(), params)
}
