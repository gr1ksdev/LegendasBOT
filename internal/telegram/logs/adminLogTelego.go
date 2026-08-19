package logs

import (
	"context"
	"fmt"

	usermodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

func LogAdminTelego(b *telego.Bot, channel *usermodels.Channel) {
	user, err := b.GetChat(context.Background(), &telego.GetChatParams{
		ChatID: telego.ChatID{ID: channel.OwnerID},
	})
	if err != nil {
		logger.Error("ADMIN", "Erro ao buscar informacoes do usuario para o log: %v", err)
	}

	ownerName := user.FirstName
	if ownerName == "" {
		ownerName = "Desconhecido"
	}

	text := fmt.Sprintf(
		"📢 <b>Novo canal adicionado!</b>\n\n"+
			"🛰 <b>Canal:</b> %s\n"+
			"🆔 <b>ID:</b> <code>%d</code>\n"+
			"🔗 <b>Link:</b> %s\n"+
			"👤 <b>Owner:</b> %s (<code>%d</code>)\n"+
			"⏰ <b>Criado em:</b> <code>%s</code>",
		channel.Title,
		channel.ID,
		channel.InviteURL,
		ownerName,
		channel.OwnerID,
		channel.CreatedAt.In(utils.BrazilTZ()).Format("02/01/2006 15:04:05"),
	)

	_, _ = b.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: config.OwnerID},
		Text:      text,
		ParseMode: telego.ModeHTML,
	})
}
