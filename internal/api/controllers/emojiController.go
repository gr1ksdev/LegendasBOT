package controllers

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

// validEmojiID valida que o emoji ID contém apenas dígitos
var validEmojiID = regexp.MustCompile(`^\d+$`)

type EmojiController struct {
	container *container.AppContainer
}

func NewEmojiController(container *container.AppContainer) *EmojiController {
	return &EmojiController{container: container}
}

// ServeEmoji serve o arquivo de emoji customizado.
// Aceita /api/emoji/:id ou /api/emoji/:id.ext
func (ctrl *EmojiController) ServeEmoji(ctx *gin.Context) {
	rawID := ctx.Param("id")
	if rawID == "" {
		ctx.Error(errors.BadRequest("emoji ID é obrigatório"))
		return
	}

	// Separa extensão opcional do ID (ex: "123.tgs" → id="123", ext=".tgs")
	emojiID, ext := splitEmojiID(rawID)

	// Validar formato do emoji ID
	if !validEmojiID.MatchString(emojiID) {
		ctx.Error(errors.BadRequest("emoji ID inválido"))
		return
	}

	// Extrair userID do contexto JWT
	userID := getUserID(ctx)
	if userID == 0 {
		return
	}

	// Busca o emoji — concede acesso e baixa via Bot API se necessário
	emoji, err := ctrl.container.EmojiService.FetchForUser(ctx.Request.Context(), userID, emojiID)
	if err != nil {
		ctx.Error(errors.ErrNotFound)
		return
	}
	if emoji == nil {
		ctx.Error(errors.ErrNotFound)
		return
	}

	// Validar extensão se foi fornecida na URL
	if ext != "" && ext != emoji.FileType {
		ctx.Error(errors.ErrNotFound)
		return
	}

	// Determinar Content-Type
	contentType := fileTypeToContentType(emoji.FileType)

	// Cache: emoji não muda, pode cachear por 1 dia
	ctx.Header("Cache-Control", "public, max-age=86400")
	ctx.Header("X-Emoji-Format", emoji.FileType)
	ctx.Data(http.StatusOK, contentType, emoji.FileData)
}

// ListEmojiHistory retorna a lista de emoji IDs que o usuário já usou.
// GET /api/emoji/history
func (ctrl *EmojiController) ListEmojiHistory(ctx *gin.Context) {
	userID := getUserID(ctx)
	if userID == 0 {
		return
	}

	ctx.Header("Cache-Control", "private, max-age=300")

	ids, err := ctrl.container.EmojiService.ListAccessedEmojiIDs(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ids": []string{}})
		return
	}
	if ids == nil {
		ids = []string{}
	}

	ctx.JSON(http.StatusOK, gin.H{"ids": ids})
}

// getUserID extrai o userID do contexto JWT.
// Retorna 0 se não encontrado (e já escreve o erro na resposta).
func getUserID(ctx *gin.Context) int64 {
	userIDRaw, exists := ctx.Get("userID")
	if !exists {
		ctx.Error(errors.ErrUnauthorized)
		return 0
	}
	userID, ok := userIDRaw.(int64)
	if !ok {
		ctx.Error(errors.Internal(nil))
		return 0
	}
	return userID
}

// splitEmojiID separa "123.tgs" em ("123", ".tgs") ou "123" em ("123", "")
func splitEmojiID(raw string) (id, ext string) {
	idx := strings.LastIndex(raw, ".")
	if idx > 0 && len(raw)-idx <= 6 {
		return raw[:idx], raw[idx:]
	}
	return raw, ""
}

func fileTypeToContentType(ft string) string {
	switch ft {
	case ".webm":
		return "video/webm"
	case ".webp":
		return "image/webp"
	case ".tgs":
		return "application/x-tgsticker"
	default:
		return "application/octet-stream"
	}
}
