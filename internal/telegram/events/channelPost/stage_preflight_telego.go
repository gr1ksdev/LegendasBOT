package channelpost

import (
	"context"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

func StagePreflightTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		post := pCtx.Update.ChannelPost
		if post == nil {
			pCtx.StopPipeline = true
			return nil
		}

		// 1. Basic Filters
		botInfo, err := pCtx.Bot.GetMe(pCtx.Ctx)
		if err != nil || botInfo == nil {
			logger.Error("PIPELINE", "❌ Falha ao obter dados do Bot em GetMe: %v", err)
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusError, map[string]any{"reason": "get_me_failed"}, err)
			pCtx.StopPipeline = true
			return nil
		}

		if post.ViaBot != nil && post.ViaBot.ID == botInfo.ID {
			logger.Bot("⏭️ Ignoring inline message.")
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "via_bot"}, nil)
			pCtx.StopPipeline = true
			return nil
		}

		// Evita loop infinito: ignora mensagens enviadas pelo proprio bot
		// (ex: separador sticker/emoji enviado pelo bot ao canal)
		if post.From != nil && post.From.ID == botInfo.ID {
			logger.Bot("⏭️ Ignorando mensagem do proprio bot (ID: %d)", post.MessageID)
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "bot_own_message"}, nil)
			pCtx.StopPipeline = true
			return nil
		}
		// Verificacao adicional: se o bot acabou de enviar um separador para este canal
		// (ChannelPost de canais tem From=nil, entao o check acima nao pega)
		if lastTime, ok := BotLastSeparatorSent.Load(post.Chat.ID); ok {
			elapsed := time.Since(lastTime.(time.Time))
			if elapsed < 3*time.Second {
				logger.Bot("⏭️ Ignorando ChannelPost do proprio separador (canal %d, ha %.1fs)", post.Chat.ID, elapsed.Seconds())
				recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "bot_separator_message"}, nil)
				pCtx.StopPipeline = true
				return nil
			}
		}

		maintenance, _ := c.ServerService.GetMaintenance(pCtx.Ctx)
		if maintenance {
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "maintenance"}, nil)
			pCtx.StopPipeline = true
			return nil
		}

		// 2. Load Channel
		channel, err := c.ChannelService.GetChannelWithRelations(pCtx.Ctx, post.Chat.ID)
		if err != nil {
			logger.Error("PIPELINE", "❌ Canal %d não encontrado no banco: %v", post.Chat.ID, err)
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "channel_not_found"}, err)
			pCtx.StopPipeline = true
			return nil
		}
		pCtx.Channel = channel
		logger.Bot("📁 Canal carregado: %s (%d)", channel.Title, channel.ID)

		// 3. Blacklist Check
		if channel.Owner != nil && channel.Owner.IsBlacklisted {
			logger.Bot("🚫 Canal %d ignorado: Proprietário está na Blacklist", channel.ID)
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "owner_blacklisted", "owner_id": channel.OwnerID}, nil)
			pCtx.StopPipeline = true
			return nil
		}

		// 4. Metadata Sync
		syncMetadataTelego(pCtx, c)

		// 5. Detect Message Type
		pCtx.MessageType = GetMessageTypeTelego(post)
		if pCtx.MessageType == "" {
			logger.Bot("⏭️ Tipo de mensagem não suportado ignorado")
			recordChannelPostEvent(c, pCtx, "post_skipped", services.ChannelEventStatusSkipped, map[string]any{"reason": "unsupported_message_type"}, nil)
			pCtx.StopPipeline = true
			return nil
		}
		logger.Bot("📝 Tipo detectado: %s", pCtx.MessageType)

		// 6. Check Permissions
		pm := GetPermissionManager()
		pCtx.Permissions = pm.CheckPermissions(channel, pCtx.MessageType)
		if !pCtx.Permissions.CanEdit && !pCtx.Permissions.CanAddButtons {
			logger.Error("PIPELINE", "❌ Sem permissões de Edição ou Botões para o canal %d (Tipo: %s)", channel.ID, pCtx.MessageType)
			recordChannelPostEvent(c, pCtx, "permission_missing", services.ChannelEventStatusSkipped, map[string]any{"can_edit": false, "can_add_buttons": false}, nil)
			pCtx.StopPipeline = true
			return nil
		}
		logger.Bot("⚖️ Permissões: Edit=%v, Buttons=%v", pCtx.Permissions.CanEdit, pCtx.Permissions.CanAddButtons)

		return nil
	}
}

func syncMetadataTelego(pCtx *ProcessingContextTelego, c *container.AppContainer) {
	post := pCtx.Update.ChannelPost
	channel := pCtx.Channel

	titleChanged := post.Chat.Title != "" && utils.RemoveHTMLTags(post.Chat.Title) != channel.Title
	usernameURL := utils.NormalizeTelegramURL("@" + post.Chat.Username)
	channelURL := utils.NormalizeTelegramURL(channel.InviteURL)
	usernameChanged := post.Chat.Username != "" && usernameURL != channelURL && !strings.HasPrefix(channelURL, "https://t.me/+")

	shouldUpdateNow := titleChanged || usernameChanged

	if !shouldUpdateNow && !c.CacheService.ShouldUpdateChannel(context.Background(), post.Chat.ID) {
		return
	}

	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var chatObj interface{}
		if shouldUpdateNow {
			chatObj = &post.Chat
		}

		updatedChannel, hasChanges := UpdateChannelBasicInfoTelego(updateCtx, pCtx.Bot, post.Chat.ID, channel, chatObj)
		_ = c.CacheService.SetLastChannelUpdate(updateCtx, post.Chat.ID)

		if hasChanges {
			if err := c.ChannelService.UpdateChannelBasicInfoAndFirstButton(updateCtx, updatedChannel); err != nil {
				logger.Error("PIPELINE", "❌ Error saving metadata for %d: %v", post.Chat.ID, err)
				recordChannelPostEvent(c, pCtx, "metadata_updated", services.ChannelEventStatusError, map[string]any{"title_changed": titleChanged, "username_changed": usernameChanged}, err)
			} else {
				recordChannelPostEvent(c, pCtx, "metadata_updated", services.ChannelEventStatusInfo, map[string]any{"title_changed": titleChanged, "username_changed": usernameChanged}, nil)
			}
		}
	}()
}
