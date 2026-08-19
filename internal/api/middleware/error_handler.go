package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Se houver erros no contexto do Gin
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			if appErr, ok := err.(*errors.AppError); ok {
				c.JSON(appErr.Code, types.APIResponse[any]{
					Success: false,
					Message: appErr.Message,
				})
				return
			}

			// Erro genérico
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse("Erro interno inesperado"))
		}
	}
}
