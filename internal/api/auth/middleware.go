package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
)

func AuthMiddlewareJWT(v *container.AppContainer) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("token")
		if err != nil {
			// Fallback para header Authorization se o cookie falhar
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Acesso não autorizado"})
				return
			}
		}

		claims, err := ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Sessão expirada ou inválida"})
			return
		}

		// Verificar Status Real no banco para garantir efeito imediato de Admin/Blacklist
		user, err := v.UserService.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			// Se o usuário não existir no banco, invalidamos o acesso
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Usuário não encontrado ou inativo"})
			return
		}

		if user.IsBlacklisted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Você está na blacklist e seu acesso foi bloqueado."})
			return
		}

		// Determinar Role atual baseada no Banco de Dados (sobrescreve o que está no Token)
		role := RoleUser
		if user.UserId == config.OwnerID {
			role = RoleOwner
		} else if user.IsAdmin {
			role = RoleAdmin
		}

		// Injetar dados ATUALIZADOS no contexto para uso nos controllers
		c.Set("userID", claims.UserID)
		c.Set("role", role)
		c.Next()
	}
}

// RequireRole garante que o usuário tenha um dos cargos permitidos
func RequireRole(roles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Cargo não identificado"})
			return
		}

		role, ok := userRole.(Role)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Cargo com formato inválido"})
			return
		}

		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		// Owner sempre tem acesso a tudo
		if role == RoleOwner {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Você não tem permissão para esta ação"})
	}
}

// AuthorizeChannel garante que o usuário tenha permissão sobre o canal especificado na URL
func AuthorizeChannel(v *container.AppContainer) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelIdStr := c.Param("channelId")
		if channelIdStr == "" {
			c.Next()
			return
		}

		channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID do canal inválido"})
			return
		}

		ctxUserID, userExists := c.Get("userID")
		ctxRole, roleExists := c.Get("role")
		if !userExists || !roleExists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Usuário não autenticado"})
			return
		}

		userID, okUser := ctxUserID.(int64)
		role, okRole := ctxRole.(Role)
		if !okUser || !okRole {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Dados de autenticação inválidos"})
			return
		}

		c.Set("channelID", channelId)

		// Owner e Admin têm passe livre
		if role == RoleOwner || role == RoleAdmin {
			c.Next()
			return
		}

		// Usuário comum: verificar se ele é o dono no banco
		channel, err := v.ChannelService.GetChannelByID(c.Request.Context(), channelId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"success": false, "message": "Canal não encontrado"})
			return
		}

		if channel.OwnerID != userID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Você não tem permissão para gerenciar este canal"})
			return
		}

		c.Next()
	}
}

func ValidateTelegramInitData(initData string, secondsToExpire int64) types.ValidateResult {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return types.ValidateResult{IsValid: false}
	}

	data := make(map[string]string)
	var hash string

	for key, val := range params {
		if key == "hash" {
			hash = val[0]
			continue
		}
		data[key] = val[0]
	}

	// ordenar keys
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+data[k])
	}
	dataCheckString := strings.Join(parts, "\n")

	// gerar secret
	step1 := hmac.New(sha256.New, []byte("WebAppData"))
	step1.Write([]byte(config.TelegramBotToken))
	secretKey := step1.Sum(nil)

	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(hash)) {
		return types.ValidateResult{IsValid: false, Data: data}
	}

	authUnix, err := strconv.ParseInt(data["auth_date"], 10, 64)
	if err != nil {
		return types.ValidateResult{IsValid: false, Data: data}
	}

	if secondsToExpire != 0 {
		if time.Now().Unix()-authUnix > secondsToExpire {
			return types.ValidateResult{IsValid: false, Data: data}
		}
	}

	return types.ValidateResult{IsValid: true, Data: data}
}
