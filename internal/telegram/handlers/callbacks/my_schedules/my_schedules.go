package myschedules

import (
	"context"
	"fmt"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// HandlerTelego exibe a lista principal de agendamentos do usuário.
func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		return renderScheduleList(c, ctx.Bot(), update)
	}
}

func renderScheduleList(c *container.AppContainer, bot *telego.Bot, update telego.Update) error {
	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return nil
	}

	userID := update.CallbackQuery.From.ID

	schedules, err := c.SchedulerService.GetUserSchedules(context.Background(), userID)
	if err != nil {
		logger.Error("BOT", "Erro ao buscar agendamentos: %v", err)
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Erro ao buscar agendamentos.",
		})
		return nil
	}

	if len(schedules) == 0 {
		var buttons [][]telego.InlineKeyboardButton
		buttons = append(buttons, []telego.InlineKeyboardButton{
			{Text: "🔙 Voltar", CallbackData: "start"},
		})

		_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
			ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID:   update.CallbackQuery.Message.GetMessageID(),
			Text:        "📅 <b>Meus Agendamentos</b>\n\n<i>Você não possui nenhum agendamento ativo no momento.</i>",
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: &telego.InlineKeyboardMarkup{InlineKeyboard: buttons},
		})

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
		return nil
	}

	var text strings.Builder
	text.WriteString("📅 <b>Meus Agendamentos</b>\n\n")
	text.WriteString("Selecione um agendamento abaixo para visualizar detalhes ou gerenciá-lo:\n\n")

	var buttons [][]telego.InlineKeyboardButton
	for _, s := range schedules {
		statusShort, _ := formatStatus(s.Status)
		typeShort, timeInfo := formatScheduleTypeShort(&s)

		text.WriteString(fmt.Sprintf("%s <b>%s</b>\n", statusShort, s.ChannelTitle))
		text.WriteString(fmt.Sprintf("⏱️ <i>Tipo: %s</i>\n", typeShort))
		text.WriteString(fmt.Sprintf("⏰ <i>Horário: %s</i>\n\n", timeInfo))

		buttons = append(buttons, []telego.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("📌 %s (%s)", s.ChannelTitle, typeShort),
				CallbackData: fmt.Sprintf("schedule-detail:%s", s.ID),
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

// DetailHandlerTelego exibe os detalhes completos de um agendamento específico.
func DetailHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		scheduleID := strings.TrimPrefix(update.CallbackQuery.Data, "schedule-detail:")
		return renderScheduleDetail(c, ctx.Bot(), update, scheduleID)
	}
}

func renderScheduleDetail(c *container.AppContainer, bot *telego.Bot, update telego.Update, scheduleID string) error {
	userID := update.CallbackQuery.From.ID

	post, err := c.SchedulerService.GetScheduleByID(context.Background(), scheduleID)
	if err != nil || post == nil || post.OwnerID != userID {
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Agendamento não encontrado.",
			ShowAlert:       true,
		})
		return nil
	}

	_, statusLong := formatStatus(post.Status)
	typeFull, timeFull := formatScheduleTypeFull(post)

	nextRunStr := "Nenhum disparo pendente"
	if !post.NextRunAt.IsZero() {
		nextRunStr = post.NextRunAt.In(utils.BrazilTZ()).Format("02/01/2006 às 15:04 (BRT)")
	}

	pinStr := "Não"
	if post.PinMessage {
		pinStr = "Sim 📌"
	}

	autoDelStr := "Desativada"
	if post.AutoDeleteMin > 0 {
		if post.AutoDeleteMin%60 == 0 {
			autoDelStr = fmt.Sprintf("%dh após envio ⏱️", post.AutoDeleteMin/60)
		} else {
			autoDelStr = fmt.Sprintf("%dmin após envio ⏱️", post.AutoDeleteMin)
		}
	}

	text := fmt.Sprintf(
		"📋 <b>Detalhes do Agendamento</b>\n\n"+
			"📢 <b>Canal:</b> %s\n"+
			"🆔 <b>ID do Canal:</b> <code>%d</code>\n\n"+
			"⚙️ <b>Tipo de Agendamento:</b> %s\n"+
			"⏰ <b>Frequência / Horário:</b> %s\n"+
			"🗓️ <b>Próximo Envio:</b> %s\n"+
			"📌 <b>Fixar Mensagem:</b> %s\n"+
			"⏱️ <b>Auto-Destruição:</b> %s\n"+
			"📊 <b>Status:</b> %s\n"+
			"🔢 <b>Envios Realizados:</b> %d\n",
		post.ChannelTitle,
		post.ChannelID,
		typeFull,
		timeFull,
		nextRunStr,
		pinStr,
		autoDelStr,
		statusLong,
		post.SentCount,
	)

	if post.LastError != "" {
		text += fmt.Sprintf("\n⚠️ <b>Último Erro Registrado:</b>\n<code>%s</code>\n", post.LastError)
	}

	var buttons [][]telego.InlineKeyboardButton

	if post.Status == "pending" {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			{Text: "🟡 Pausar Agendamento", CallbackData: "schedule-pause:" + post.ID},
		})
	} else if post.Status == "paused" {
		buttons = append(buttons, []telego.InlineKeyboardButton{
			{Text: "🟢 Retomar Agendamento", CallbackData: "schedule-resume:" + post.ID},
		})
	}

	buttons = append(buttons, []telego.InlineKeyboardButton{
		{Text: "🗑️ Excluir Agendamento", CallbackData: "schedule-delete:" + post.ID},
	})
	buttons = append(buttons, []telego.InlineKeyboardButton{
		{Text: "🔙 Voltar para Agendamentos", CallbackData: "my-schedules"},
	})

	kb := &telego.InlineKeyboardMarkup{InlineKeyboard: buttons}

	_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
		ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
		MessageID:   update.CallbackQuery.Message.GetMessageID(),
		Text:        text,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: kb,
	})

	_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	return nil
}

// PauseHandlerTelego pausa o agendamento e atualiza a interface.
func PauseHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		scheduleID := strings.TrimPrefix(update.CallbackQuery.Data, "schedule-pause:")
		userID := update.CallbackQuery.From.ID

		err := c.SchedulerService.PauseScheduledPost(context.Background(), scheduleID, userID)
		if err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao pausar agendamento.",
				ShowAlert:       true,
			})
			return nil
		}

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "🟡 Agendamento pausado com sucesso!",
		})

		return renderScheduleDetail(c, ctx.Bot(), update, scheduleID)
	}
}

// ResumeHandlerTelego retoma o agendamento pausado e atualiza a interface.
func ResumeHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		scheduleID := strings.TrimPrefix(update.CallbackQuery.Data, "schedule-resume:")
		userID := update.CallbackQuery.From.ID

		err := c.SchedulerService.ResumeScheduledPost(context.Background(), scheduleID, userID)
		if err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao retomar agendamento.",
				ShowAlert:       true,
			})
			return nil
		}

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "🟢 Agendamento ativado com sucesso!",
		})

		return renderScheduleDetail(c, ctx.Bot(), update, scheduleID)
	}
}

// DeleteHandlerTelego exclui o agendamento e retorna para a lista de agendamentos.
func DeleteHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		scheduleID := strings.TrimPrefix(update.CallbackQuery.Data, "schedule-delete:")
		userID := update.CallbackQuery.From.ID

		err := c.SchedulerService.DeleteScheduledPost(context.Background(), scheduleID, userID)
		if err != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao excluir agendamento.",
				ShowAlert:       true,
			})
			return nil
		}

		_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "🗑️ Agendamento excluído com sucesso!",
			ShowAlert:       true,
		})

		return renderScheduleList(c, ctx.Bot(), update)
	}
}

// Helpers de Formatação em Português

func formatStatus(status string) (shortLabel string, longLabel string) {
	switch status {
	case "pending":
		return "🟢", "🟢 Ativo (Aguardando horário)"
	case "paused":
		return "🟡", "🟡 Pausado pelo Usuário"
	case "processing":
		return "🔄", "🔄 Em Envio / Processando"
	case "completed", "sent":
		return "✅", "✅ Concluído"
	case "cancelled", "failed":
		return "🔴", "🔴 Cancelado / Erro"
	default:
		return "⚪", status
	}
}

func formatScheduleTypeShort(s *models.ScheduledPost) (typeShort string, timeInfo string) {
	switch s.ScheduleType {
	case "once":
		typeShort = "Envio Único"
		if s.ScheduledAt != nil {
			timeInfo = s.ScheduledAt.In(utils.BrazilTZ()).Format("02/01 às 15:04")
		} else {
			timeInfo = "Pontual"
		}

	case "daily":
		typeShort = "Diário"
		if s.ScheduleTime != "" {
			timeInfo = fmt.Sprintf("Todo dia às %s", s.ScheduleTime)
		} else {
			timeInfo = "Diário"
		}

	case "weekly":
		typeShort = "Semanal"
		timeInfo = fmt.Sprintf("Semanalmente às %s", s.ScheduleTime)

	case "interval":
		typeShort = fmt.Sprintf("Intervalo (%dmin)", s.IntervalMin)
		if s.WindowStart != "" && s.WindowEnd != "" {
			timeInfo = fmt.Sprintf("%s às %s", s.WindowStart, s.WindowEnd)
		} else {
			timeInfo = "24 Horas"
		}

	case "queue":
		typeShort = "Fila"
		timeInfo = "Ordem de Fila"

	default:
		typeShort = s.ScheduleType
		timeInfo = s.ScheduleTime
	}
	return typeShort, timeInfo
}

func formatScheduleTypeFull(s *models.ScheduledPost) (typeFull string, timeFull string) {
	switch s.ScheduleType {
	case "once":
		typeFull = "Envio Único (Pontual)"
		if s.ScheduledAt != nil {
			timeFull = s.ScheduledAt.In(utils.BrazilTZ()).Format("02/01/2006 às 15:04 (BRT)")
		} else {
			timeFull = "Pontual"
		}

	case "daily":
		typeFull = "Recorrente - Diário"
		timeFull = fmt.Sprintf("Todos os dias às %s (BRT)", s.ScheduleTime)

	case "weekly":
		typeFull = "Recorrente - Semanal"
		timeFull = fmt.Sprintf("Toda semana às %s (BRT)", s.ScheduleTime)

	case "interval":
		typeFull = fmt.Sprintf("Recorrente - Intervalo (A cada %d minutos)", s.IntervalMin)
		if s.WindowStart != "" && s.WindowEnd != "" {
			timeFull = fmt.Sprintf("Janela das %s às %s (BRT)", s.WindowStart, s.WindowEnd)
		} else {
			timeFull = "24 Horas (Sem restrição de horário)"
		}

	case "queue":
		typeFull = "Fila de Postagem"
		timeFull = fmt.Sprintf("Posição %d na Fila", s.QueuePosition)

	default:
		typeFull = s.ScheduleType
		timeFull = s.ScheduleTime
	}
	return typeFull, timeFull
}
