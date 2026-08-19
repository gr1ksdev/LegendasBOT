package admin

import (
	"context"
	"fmt"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func AdminHelpHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		msg := `👨‍💻 <b>Painel de Administração</b>

<b>Comandos de Listagem:</b>
/users - Lista todos os usuários
/channels - Lista todos os canais
/user [id] - Informações detalhadas do usuário
/info [id] - Informações detalhadas do canal

<b>Comandos de Mensagem:</b>
/notice [msg] - Envia aviso para todos os usuários (primeira linha é comando)
/publi - Envia mensagem de publicidade padrão para todos os canais
/send [id]\n[msg] - Envia mensagem privada para um ID específico
/allusers (reply) - Envia a mensagem respondida para todos os usuários
/allchannels (reply) - Envia a mensagem respondida para todos os canais

<b>Comandos de Gerenciamento:</b>
/add [canalID] [donoID] - Adiciona canal e dono manualmente
/remove [id] - Remove um canal do sistema
/transfer [canalID] [novoDonoID] - Transfere posse de um canal
/setadmin [id] - Alterna status de administrador de um usuário
/maintence - Ativa/Desativa modo de manutenção
/backup - Gera backup do banco de dados

<b>Utilidades:</b>
/checkbot - Verifica se o XavolaBot é admin nos canais
/getid (reply) - Captura ID da mídia para Broadcast
/emoji - Log de update (debug)`

		kb := &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text: "📊 Abrir Dashboard Admin",
						WebApp: &telego.WebAppInfo{
							URL: fmt.Sprintf("%s/admin/dash", config.WebAppURL),
						},
					},
				},
			},
		}

		params := &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.SendMessage(context.Background(), params)
		return nil
	}
}
