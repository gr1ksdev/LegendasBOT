package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
)

const MaxAPIRequestBodyBytes int64 = 1 << 20

// BodyLimit limita leituras de JSON e evita que endpoints ShouldBindJSON
// aceitem corpos ilimitados em memoria.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, types.NewErrorResponse("Corpo da requisição muito grande"))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
