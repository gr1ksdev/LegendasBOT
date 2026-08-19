package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type AuthController struct {
	container *container.AppContainer
}

func NewAuthController(c *container.AppContainer) *AuthController {
	return &AuthController{container: c}
}

type LoginRequest struct {
	UserID int64 `json:"userID"`
}

type telegramLoginUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

func parseTelegramLoginUser(raw string) (telegramLoginUser, error) {
	var user telegramLoginUser
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		return telegramLoginUser{}, err
	}
	return user, nil
}

func telegramUserMatches(raw string, userID int64) bool {
	user, err := parseTelegramLoginUser(raw)
	return err == nil && user.ID == userID
}

func (ac *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos"))
		return
	}

	authData := ctx.GetHeader("x-telegram-init-data")
	if authData == "" {
		ctx.Error(errors.New(http.StatusUnauthorized, "InitData ausente"))
		return
	}

	// 1. Validação RIGOROSA do Telegram
	validateResult := auth.ValidateTelegramInitData(authData, 3600)
	if !validateResult.IsValid {
		ctx.Error(errors.New(http.StatusUnauthorized, "Autenticação do Telegram falhou"))
		return
	}

	// 1.1 Verificar se o userID do InitData condiz com o userID do Request
	userDataRaw, ok := validateResult.Data["user"]
	if !ok {
		ctx.Error(errors.New(http.StatusUnauthorized, "Dados de usuário ausentes no Telegram"))
		return
	}

	if !telegramUserMatches(userDataRaw, req.UserID) {
		ctx.Error(errors.ErrForbidden)
		return
	}
	tgUser, _ := parseTelegramLoginUser(userDataRaw)

	// 2. Determinar Role
	role := auth.RoleUser
	isBlacklisted := false
	if req.UserID == config.OwnerID {
		role = auth.RoleOwner
	} else {
		user, err := ac.container.UserService.GetUserByID(ctx, req.UserID)
		if err != nil || user == nil {
			// Usuário novo: cria a partir dos dados do initData do Telegram.
			// UpsertUser preserva flags de admin/blacklist em conflito.
			{
				newUser := &models.User{
					UserId:    tgUser.ID,
					FirstName: utils.RemoveHTMLTags(tgUser.FirstName),
				}
				if tgUser.Username != "" {
					newUser.Username = "@" + tgUser.Username
				}
				if upErr := ac.container.UserService.UpsertUser(ctx, newUser); upErr != nil {
					// Não bloqueia o login: o usuário pode já estar cadastrado
					// ou o banco pode ter falhado; o fluxo segue abaixo.
					logger.Error("API", "Erro ao criar usuário no login: %v", upErr)
				}
				user, _ = ac.container.UserService.GetUserByID(ctx, req.UserID)
			}
		}
		if user != nil {
			if user.IsAdmin {
				role = auth.RoleAdmin
			}
			if user.IsBlacklisted {
				isBlacklisted = true
			}
		}
	}

	// 3. Gerar Token Seguro
	token, err := auth.GenerateToken(req.UserID, role, 1, 12*time.Hour)
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	// 4. Setar Cookie Seguro.
	// Telegram Mini Apps rodam em WebView/iframe de terceiros: cookies
	// SameSite=Strict são descartados e a primeira autenticação de usuários
	// novos falha. Em prod usamos SameSite=None+Secure (requer HTTPS); em dev,
	// SameSite=Lax para funcionar em HTTP local.
	sameSite := http.SameSiteLaxMode
	if config.AppEnv != "dev" {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		MaxAge:   43200, // 12h
		HttpOnly: true,
		Secure:   config.AppEnv != "dev",
		SameSite: sameSite,
	})

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"role":          role,
		"isBlacklisted": isBlacklisted,
	}, "Login realizado com sucesso"))
}
