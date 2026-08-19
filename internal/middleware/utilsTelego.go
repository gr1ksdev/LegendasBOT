package middleware

import (
	"github.com/mymmrac/telego"
)

func GetUpdateUserIDTelego(upt telego.Update) int64 {
	switch {
	case upt.Message != nil && upt.Message.From != nil:
		return upt.Message.From.ID
	case upt.CallbackQuery != nil:
		return upt.CallbackQuery.From.ID
	case upt.InlineQuery != nil:
		return upt.InlineQuery.From.ID
	case upt.ChosenInlineResult != nil:
		return upt.ChosenInlineResult.From.ID
	case upt.MyChatMember != nil:
		return upt.MyChatMember.From.ID
	case upt.ChatMember != nil:
		return upt.ChatMember.From.ID
	case upt.PreCheckoutQuery != nil:
		return upt.PreCheckoutQuery.From.ID
	case upt.ShippingQuery != nil:
		return upt.ShippingQuery.From.ID
	default:
		return 0
	}
}
