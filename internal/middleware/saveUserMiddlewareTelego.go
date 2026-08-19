package middleware

import (
	"context"
	"fmt"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func SaveUserMiddlewareTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		var userId int64
		var firstName string
		var username string

		if update.Message != nil && update.Message.From != nil {
			userId = update.Message.From.ID
			firstName = update.Message.From.FirstName
			username = fmt.Sprintf("@%s", update.Message.From.Username)
		} else if update.CallbackQuery != nil {
			userId = update.CallbackQuery.From.ID
			firstName = update.CallbackQuery.From.FirstName
			username = fmt.Sprintf("@%s", update.CallbackQuery.From.Username)
		} else if update.InlineQuery != nil {
			userId = update.InlineQuery.From.ID
			firstName = update.InlineQuery.From.FirstName
			username = fmt.Sprintf("@%s", update.InlineQuery.From.Username)
		}

		if userId != 0 {
			err := c.UserService.UpsertUser(context.Background(), &models.User{
				UserId:    userId,
				FirstName: utils.RemoveHTMLTags(firstName),
				Username:  username,
			})
			if err != nil {
				logger.Error("DB", "Erro ao upsert do usuário: %v", err)
			}
		}

		return ctx.Next(update)
	}
}
