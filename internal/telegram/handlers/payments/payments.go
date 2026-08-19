package payments

import (
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// PreCheckoutHandler retorna um handler para pre-checkout queries do Telegram Stars.
func PreCheckoutHandler(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		query := update.PreCheckoutQuery
		if query == nil {
			return nil
		}

		logger.Bot("💳 PreCheckoutQuery recebida: user=%d payload=%s amount=%d",
			query.From.ID, query.InvoicePayload, query.TotalAmount)

		if err := c.SubscriptionService.HandlePreCheckout(ctx, query); err != nil {
			logger.Error("PAYMENTS", "Erro ao processar PreCheckoutQuery: %v", err)
		}

		return nil
	}
}

// SuccessfulPaymentHandler retorna um handler para pagamentos bem-sucedidos.
func SuccessfulPaymentHandler(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		msg := update.Message
		if msg == nil || msg.SuccessfulPayment == nil {
			return nil
		}

		payment := msg.SuccessfulPayment
		userID := msg.From.ID

		logger.Bot("💰 SuccessfulPayment recebido: user=%d charge=%s amount=%d stars",
			userID, payment.TelegramPaymentChargeID, payment.TotalAmount)

		if err := c.SubscriptionService.HandlePayment(ctx, userID, payment); err != nil {
			logger.Error("PAYMENTS", "Erro ao processar SuccessfulPayment: %v", err)
		}

		return nil
	}
}
