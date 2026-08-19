package telegram

import (
	"context"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/middleware"
	"github.com/leirbagxis/FreddyBot/internal/telegram/events/channelPost"
	callbackAbout "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/about"
	callbackClaim "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/claimChannel"
	callbackMyChannel "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/my_channel"
	callbackMyDrafts "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/my_drafts"
	callbackMySchedules "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/my_schedules"
	callbackProfile "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/profile_info"
	callbackStart "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/start"
	callbackVote "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/callbacks/vote"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/commands/admin"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/commands/help"
	commandStart "github.com/leirbagxis/FreddyBot/internal/telegram/handlers/commands/start"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/commands/suporte"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/commands/tutorial"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/events/addChannel"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/events/postBuilder"
	"github.com/leirbagxis/FreddyBot/internal/telegram/handlers/payments"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func LoadHandlersTelegoWithBH(bh *telegohandler.BotHandler, c *container.AppContainer) {
	// Middlewares
	bh.Use(middleware.SaveUserMiddlewareTelego(c))
	bh.Use(middleware.CheckBlacklistMiddlewareTelego(c))
	bh.Use(middleware.CheckMaintenanceMiddlewareTelego(c))

	// Channel Post Handler
	bh.Handle(channelpost.HandlerTelego(c), telegohandler.AnyChannelPost())

	// Add Channel Handlers
	addChannelGroup := bh.Group(telegohandler.AnyMyChatMember())
	addChannelGroup.Use(middleware.CheckAddBotMiddlewareTelego(c))
	addChannelGroup.Handle(addchannel.AskAddChannelHandlerTelego(c))

	bh.Handle(addchannel.UpdateChannelInfoHandlerTelego(c), telegohandler.AnyMyChatMember())

	// Commands
	bh.Handle(commandStart.HandlerTelego(c), telegohandler.CommandEqual("start"))
	bh.Handle(help.HandlerTelego(c), telegohandler.CommandEqual("help"))
	bh.Handle(suporte.HandlerTelego(c), telegohandler.CommandEqual("ouvidoria"))
	bh.Handle(tutorial.HandlerTelego(c), telegohandler.CommandEqual("tutorial"))

	// Admin Commands (Owner and Admins)
	adminGroup := bh.Group(matchAdminOrOwnerTelego(c))
	adminGroup.Handle(admin.AdminHelpHandlerTelego(c), telegohandler.CommandEqual("admin"))
	adminGroup.Handle(admin.GetAllUsersHandlerTelego(c), telegohandler.CommandEqual("users"))
	adminGroup.Handle(admin.GetAllChannelsHandlerTelego(c), telegohandler.CommandEqual("channels"))
	adminGroup.Handle(admin.GetInfoUserHandlerTelego(c), telegohandler.CommandEqual("user"))
	adminGroup.Handle(admin.NoticeCommandHandlerTelego(c), telegohandler.CommandEqual("notice"))
	adminGroup.Handle(admin.NoticeChannelsHandlerTelego(c), telegohandler.CommandEqual("publi"))
	adminGroup.Handle(admin.SendMessageToIdHandlerTelego(c), telegohandler.CommandEqual("send"))
	adminGroup.Handle(admin.NoticeUsersReplyHandlerTelego(c), telegohandler.CommandEqual("allusers"))
	adminGroup.Handle(admin.NoticeChannelsReplyHandlerTelego(c), telegohandler.CommandEqual("allchannels"))
	adminGroup.Handle(admin.AddChannelCommandHandlerTelego(c), telegohandler.CommandEqual("add"))
	adminGroup.Handle(admin.RemoveChannelHandlerTelego(c), telegohandler.CommandEqual("remove"))
	adminGroup.Handle(admin.RegisterTransferHandlerTelego(c), telegohandler.CommandEqual("transfer"))
	adminGroup.Handle(admin.SetAdminHandlerTelego(c), telegohandler.CommandEqual("setadmin"))
	adminGroup.Handle(admin.ToggleMaintenceHandlerTelego(c), telegohandler.CommandEqual("maintence"))
	adminGroup.Handle(admin.GetBackUpHandlerTelego(c), telegohandler.CommandEqual("backup"))
	adminGroup.Handle(admin.CheckBotAdminHandlerTelego(c), telegohandler.CommandEqual("checkbot"))
	adminGroup.Handle(admin.GetMediaIDHandlerTelego(c), telegohandler.CommandEqual("getid"))

	adminOrOwnerGroup := bh.Group(matchAdminOrOwnerTelego(c))
	adminOrOwnerGroup.Handle(admin.GetInfoChannelHandlerTelego(c), telegohandler.CommandEqual("info"))

	// Message Handlers for active sessions (Text and Sticker inputs)
	bh.Handle(callbackMyChannel.SetSeparatorHandlerTelego(c), matchAwaitingSeparatorTelego(c))
	bh.Handle(callbackMyChannel.SetTransferAccessHandlerTelego(c), matchAwaitingTransferAccessTelego(c))
	bh.Handle(callbackMyChannel.SetCaptionHandlerTelego(c), matchAwaitingCaptionTelego(c))

	// Post Builder - Message Handler (Media and Text Input)
	bh.Handle(postbuilder.HandlerTelego(c), matchPostBuilderTelego(c))

	// Callbacks
	bh.Handle(callbackStart.HandlerTelego(c), telegohandler.CallbackDataEqual("start"))
	bh.Handle(callbackStart.CheckSubscriptionHandlerTelego(c), telegohandler.CallbackDataEqual("check_subscription"))
	bh.Handle(callbackProfile.HandlerTelego(c), telegohandler.CallbackDataEqual("profile-info"))
	bh.Handle(callbackMySchedules.HandlerTelego(c), telegohandler.CallbackDataEqual("my-schedules"))
	bh.Handle(callbackMySchedules.DetailHandlerTelego(c), telegohandler.CallbackDataPrefix("schedule-detail:"))
	bh.Handle(callbackMySchedules.PauseHandlerTelego(c), telegohandler.CallbackDataPrefix("schedule-pause:"))
	bh.Handle(callbackMySchedules.ResumeHandlerTelego(c), telegohandler.CallbackDataPrefix("schedule-resume:"))
	bh.Handle(callbackMySchedules.DeleteHandlerTelego(c), telegohandler.CallbackDataPrefix("schedule-delete:"))
	bh.Handle(callbackMyDrafts.HandlerTelego(c), telegohandler.CallbackDataEqual("my-drafts"))
	bh.Handle(callbackMyDrafts.DetailHandlerTelego(c), telegohandler.CallbackDataPrefix("draft-detail:"))
	bh.Handle(callbackMyDrafts.LoadHandlerTelego(c), telegohandler.CallbackDataPrefix("draft-load:"))
	bh.Handle(callbackMyDrafts.RenameHandlerTelego(c), telegohandler.CallbackDataPrefix("draft-rename:"))
	bh.Handle(callbackMyDrafts.DeleteHandlerTelego(c), telegohandler.CallbackDataPrefix("draft-delete:"))
	bh.Handle(callbackMyChannel.HandlerTelego(c), telegohandler.CallbackDataEqual("profile-user-channels"))
	bh.Handle(callbackMyChannel.ConfigHandlerTelego(c), telegohandler.CallbackDataPrefix("config:"))
	bh.Handle(callbackMyChannel.GroupChannelHandlerTelego(c), telegohandler.CallbackDataPrefix("gc-info:"))
	bh.Handle(callbackVote.HandlerTelego(c), telegohandler.CallbackDataPrefix("vote:"))
	bh.Handle(callbackMyChannel.AskDeleteChannelHandlerTelego(c), telegohandler.CallbackDataEqual("del"))
	bh.Handle(callbackMyChannel.ConfirmDeleteChannelHandlerTelego(c), telegohandler.CallbackDataPrefix("confirm-del:"))

	// Remaining Callbacks
	bh.Handle(callbackAbout.HandlerTelego(c), telegohandler.CallbackDataEqual("about"))
	bh.Handle(callbackClaim.AcceptClaimHandlerTelego(c), telegohandler.CallbackDataPrefix("accept-claim:"))

	// Caption Callbacks
	bh.Handle(callbackMyChannel.AskCaptionHandlerTelego(c), telegohandler.CallbackDataPrefix("setcaption:"))

	// Sticker Separator Callbacks
	bh.Handle(callbackMyChannel.AskStickerSeparatorHandlerTelego(c), telegohandler.CallbackDataEqual("sptc"))
	bh.Handle(callbackMyChannel.RequireStickerSeparatorHandlerTelego(c), telegohandler.CallbackDataEqual("sptc-config"))
	bh.Handle(callbackMyChannel.DeleteSeparatorHandlerTelego(c), telegohandler.CallbackDataEqual("spex"))

	// Premium Callbacks
	bh.Handle(callbackMyChannel.PremiumFeaturesHandlerTelego(c), telegohandler.CallbackDataPrefix("premium-features:"))

	// Transfer Access Callbacks
	bh.Handle(callbackMyChannel.AskTransferAccessHandlerTelego(c), telegohandler.CallbackDataEqual("paccess-info"))
	bh.Handle(callbackMyChannel.TransferAcessHandlerTelego(c), telegohandler.CallbackDataEqual("transfer"))

	// Help Callback
	bh.Handle(help.CallbackHandlerTelego(c), telegohandler.CallbackDataEqual("help"))

	// Add Channel Callbacks
	bh.Handle(addchannel.AddYesHandlerTelego(c), telegohandler.CallbackDataPrefix("add-yes:"))
	bh.Handle(addchannel.AddNotHandlerTelego(c), telegohandler.CallbackDataPrefix("add-not:"))

	// Post Builder Callbacks
	bh.Handle(postbuilder.CallbackHandlerTelego(c), telegohandler.CallbackDataPrefix("pb-"))

	// Payment Handlers (Telegram Stars)
	bh.Handle(payments.PreCheckoutHandler(c), telegohandler.AnyPreCheckoutQuery())
	bh.Handle(payments.SuccessfulPaymentHandler(c), anySuccessfulPayment())

	// Inline Handlers
	bh.HandleInlineQuery(postbuilder.InlineHandlerTelego(c), telegohandler.InlineQueryPrefix("pb "))
	bh.HandleInlineQuery(callbackClaim.HandlerTelego(c), telegohandler.InlineQueryPrefix("claim "))
	bh.HandleChosenInlineResult(postbuilder.ChosenInlineResultHandlerTelego(c), telegohandler.AnyChosenInlineResult())
}

func LoadHandlersTelego(bot *telego.Bot, c *container.AppContainer) *telegohandler.BotHandler {
	updates, _ := bot.UpdatesViaLongPolling(context.Background(), nil)
	bh, _ := telegohandler.NewBotHandler(bot, updates)
	LoadHandlersTelegoWithBH(bh, c)
	return bh
}

func matchAwaitingSeparatorTelego(c *container.AppContainer) telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.From == nil {
			return false
		}
		id, _ := c.CacheService.GetAwaitingStickerSeparator(context.Background(), update.Message.From.ID)
		if id == 0 {
			return false
		}
		// Aceita sticker OU texto com entidade custom_emoji
		if update.Message.Sticker != nil {
			return true
		}
		if update.Message.Text != "" && len(update.Message.Entities) > 0 {
			for _, entity := range update.Message.Entities {
				if entity.Type == "custom_emoji" {
					return true
				}
			}
		}
		return false
	}
}

func matchAwaitingTransferAccessTelego(c *container.AppContainer) telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.From == nil {
			return false
		}
		id, err := c.CacheService.GetTransferChannel(context.Background(), update.Message.From.ID)
		return err == nil && id != 0 && update.Message.Text != ""
	}
}

func matchAwaitingCaptionTelego(c *container.AppContainer) telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.From == nil {
			return false
		}
		id, _ := c.CacheService.GetAwaitingCaption(ctx, update.Message.From.ID)
		return id != 0
	}
}

func matchOwnerTelego() telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.From == nil {
			return false
		}
		return update.Message.From.ID == config.OwnerID
	}
}

func matchAdminOrOwnerTelego(c *container.AppContainer) telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.From == nil {
			return false
		}

		userID := update.Message.From.ID
		if userID == config.OwnerID {
			return true
		}

		user, err := c.UserService.GetUserByID(context.Background(), userID)
		return err == nil && user != nil && user.IsAdmin
	}
}

func anySuccessfulPayment() telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		return update.Message != nil && update.Message.SuccessfulPayment != nil
	}
}

func matchPostBuilderTelego(c *container.AppContainer) telegohandler.Predicate {
	return func(ctx context.Context, update telego.Update) bool {
		if update.Message == nil || update.Message.Chat.Type != telego.ChatTypePrivate {
			return false
		}
		// Se for comando, deixa os comandos tratarem
		if strings.HasPrefix(update.Message.Text, "/") {
			return false
		}

		// Prioridade para sessões ativas de outros fluxos
		userId := update.Message.From.ID
		if id, _ := c.CacheService.GetAwaitingStickerSeparator(context.Background(), userId); id != 0 {
			return false
		}
		if id, _ := c.CacheService.GetTransferChannel(context.Background(), userId); id != 0 {
			return false
		}
		if id, _ := c.CacheService.GetAwaitingCaption(context.Background(), userId); id != 0 {
			return false
		}

		// Match se o usuário estiver em uma etapa de entrada ativa do PostBuilder
		state, _ := c.CacheService.GetPostBuilderState(context.Background(), userId)
		if state != nil && state.Step != "" {
			logger.Bot("Predicate: matchPostBuilderTelego = true (etapa ativa %s) para UserID=%d", state.Step, userId)
			return true
		}

		// Match se o usuário estiver no fluxo de agendamento ativo
		scheduleState, _ := c.CacheService.GetScheduleState(context.Background(), userId)
		if scheduleState != nil && scheduleState.SessionID != "" {
			logger.Bot("Predicate: matchPostBuilderTelego = true (agendamento ativo) para UserID=%d", userId)
			return true
		}

		// Match se contiver mídia ou for encaminhada
		hasMedia := update.Message.Photo != nil ||
			update.Message.Video != nil ||
			update.Message.Animation != nil ||
			update.Message.Audio != nil ||
			update.Message.Document != nil ||
			update.Message.Sticker != nil

		isForwarded := update.Message.ForwardOrigin != nil

		if hasMedia || isForwarded {
			logger.Bot("Predicate: matchPostBuilderTelego = true (midia/encaminhamento) para UserID=%d", userId)
			return true
		}

		return false
	}
}
