package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/controllers"
	admincontroller "github.com/leirbagxis/FreddyBot/internal/api/controllers/adminController"
	"github.com/leirbagxis/FreddyBot/internal/api/handlers"
	"github.com/leirbagxis/FreddyBot/internal/api/middleware"
	"github.com/leirbagxis/FreddyBot/internal/container"
)

func RegisterRoutes(r *gin.Engine, c *container.AppContainer) {
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

	api := r.Group("/api")
	api.Use(middleware.BodyLimit(middleware.MaxAPIRequestBodyBytes))
	api.Use(middleware.RateLimit(120, time.Minute))

	// --- Health & Readiness Check Endpoints (Sem Auth) ---
	healthController := controllers.NewHealthController(c)
	r.GET("/healthz", healthController.Healthz)
	r.GET("/readyz", healthController.Readyz)

	// Controladores
	authController := controllers.NewAuthController(c)
	captionController := controllers.NewCaptionController(c)
	ButtonsController := controllers.NewButtonsController(c)
	permissionsController := controllers.NewPermissionController(c)
	customCaptionController := controllers.NewCustomCaptionController(c)
	userController := controllers.NewUserController(c)
	channelController := controllers.NewChannelController(c)
	getALlUsers := admincontroller.NewUsersAdminController(c)
	configController := admincontroller.NewConfigController(c)
	mediaController := admincontroller.NewMediaController(c)
	auditController := admincontroller.NewAuditController(c)
	channelEventsController := admincontroller.NewChannelEventsController(c)
	accountController := controllers.NewAccountController(c)
	emojiController := controllers.NewEmojiController(c)
	captionTemplateController := controllers.NewCaptionTemplateController(c)
	userCaptionTemplateController := controllers.NewUserCaptionTemplateController(c)
	adminAccountController := admincontroller.NewAdminAccountController(c)
	subscriptionController := controllers.NewSubscriptionController(c.SubscriptionService)
	adminSubscriptionController := admincontroller.NewAdminSubscriptionController(c.SubscriptionService)
	schedulerController := controllers.NewSchedulerController(c)

	// --- Rota de Login Unificada ---
	api.POST("/login", middleware.RateLimitStrict(15, time.Minute, true), authController.Login)

	// --- Log de erros do frontend (sem auth - captura erros antes do login) ---
	api.POST("/log/client-error", handlers.ClientErrorHandler(c))

	// --- Rotas Protegidas ---
	api.Use(auth.AuthMiddlewareJWT(c))
	{
		api.GET("/ping", handlers.PingHandler(c))
		api.GET("/me/channels", userController.GetUserChannelsController)
		api.GET("/user/info/:userParams", userController.GetUserInfo)
		api.POST("/channel/transfer", userController.TransferChannelController)
		api.GET("/emoji/history", emojiController.ListEmojiHistory)
		api.GET("/emoji/:id", emojiController.ServeEmoji)

		// Rotas de Assinatura Premium
		api.GET("/subscription", subscriptionController.GetSubscription)
		api.POST("/subscription/create", subscriptionController.CreateInvoice)
		api.POST("/subscription/cancel", subscriptionController.Cancel)
		api.POST("/subscription/channels/add-invoice", subscriptionController.CreateExtraChannelInvoice)
		api.POST("/subscription/channels/remove", subscriptionController.RemoveExtraChannel)

		// Rotas de Templates de Legenda (nível de usuário)
		api.GET("/me/templates", userCaptionTemplateController.List)
		api.POST("/me/templates", userCaptionTemplateController.Create)
		api.GET("/me/templates/:id", userCaptionTemplateController.Get)
		api.PUT("/me/templates/:id", userCaptionTemplateController.Update)
		api.POST("/me/templates/:id/buttons", userCaptionTemplateController.CreateButton)
		api.PUT("/me/templates/:id/buttons/:buttonId", userCaptionTemplateController.UpdateButton)
		api.DELETE("/me/templates/:id/buttons/:buttonId", userCaptionTemplateController.DeleteButton)
		api.PUT("/me/templates/:id/layout", userCaptionTemplateController.UpdateLayout)
		api.DELETE("/me/templates/:id", userCaptionTemplateController.Delete)

		// Rotas de Rascunhos do PostBuilder (Post Templates)
		postTemplateController := controllers.NewPostTemplateController(c)
		api.GET("/me/post-templates", postTemplateController.ListTemplates)
		api.POST("/me/post-templates", postTemplateController.SaveCurrentTemplate)
		api.DELETE("/me/post-templates/:id", postTemplateController.DeleteTemplate)
		api.POST("/me/post-templates/:id/load", postTemplateController.LoadTemplate)

		// Rotas de Configuração do Bot

		// Rotas de Agendamento
		api.GET("/schedule", schedulerController.GetMySchedules)
		api.POST("/schedule", schedulerController.CreateSchedule)
		api.GET("/schedule/:id", schedulerController.GetScheduleByID)
		api.PUT("/schedule/:id/status", schedulerController.UpdateStatus)
		api.DELETE("/schedule/:id", schedulerController.DeleteSchedule)
		api.PATCH("/schedule/:id", schedulerController.EditSchedule)

		// Rotas específicas de Canal (Com verificação de autorização)
		channelRoutes := api.Group("/channel/:channelId")
		channelRoutes.Use(auth.AuthorizeChannel(c))
		{
			channelRoutes.GET("", channelController.GetChannelByIDController)
			channelRoutes.GET("/photo", captionController.GetChannelPhotoController)
			channelRoutes.DELETE("", channelController.DisconectChannel)
			channelRoutes.PUT("/caption", captionController.UpdateDefaultCaptionController)
			channelRoutes.PUT("/newpackcaption", captionController.UpdateNewPackCaptionController)
			channelRoutes.PUT("/reactions", captionController.UpdateReactionsController)
			channelRoutes.PUT("/reactions/active", permissionsController.UpdateReactionsActiveController)
			channelRoutes.PUT("/reactions/position", captionController.UpdateReactionPositionController)
			channelRoutes.PUT("/native-reactions", captionController.UpdateNativeReactionsController)
			channelRoutes.PUT("/native-reactions/mode", captionController.UpdateNativeReactionModeController)
			channelRoutes.PUT("/native-reactions/enabled", captionController.UpdateNativeReactionsEnabledController)
			channelRoutes.PUT("/dynamic-links", permissionsController.UpdateDynamicLinksController)
			channelRoutes.PUT("/caption/permissions", permissionsController.UpdateMessagePermissionController)
			channelRoutes.PUT("/buttons/permissions", permissionsController.UpdateButtonsPermissionController)

			channelRoutes.POST("/buttons", ButtonsController.CreateDefaultButtonController)
			channelRoutes.DELETE("/buttons/:buttonId", ButtonsController.DeleteDefaultButtonController)
			channelRoutes.PUT("/buttons/:buttonId", ButtonsController.UpdateDefaultButtonController)
			channelRoutes.PUT("/buttons/layout", ButtonsController.UpdateLayoutDefaultButtons)

			channelRoutes.POST("/custom-captions", customCaptionController.CreateCustomCaptionController)
			channelRoutes.POST("/custom-captions/:captionId/buttons", customCaptionController.CreateCustomCaptionButtonController)
			channelRoutes.PUT("/custom-captions/:captionId", customCaptionController.UpdateCustomCaptionController)
			channelRoutes.PUT("/custom-captions/:captionId/layout", customCaptionController.UpdateCustomCaptionLayoutController)
			channelRoutes.PUT("/custom-captions/:captionId/buttons/:buttonId", customCaptionController.UpdateCustomCaptionButtonController)
			channelRoutes.DELETE("/custom-captions/:captionId", customCaptionController.DeleteCustomCaptionController)
			channelRoutes.DELETE("/custom-captions/:captionId/buttons/:buttonId", customCaptionController.DeleteCustomCaptionButtonController)

			channelRoutes.GET("/caption-templates", captionTemplateController.List)
			channelRoutes.POST("/caption-templates", captionTemplateController.Save)
			channelRoutes.GET("/caption-templates/:templateId", captionTemplateController.Get)
			channelRoutes.POST("/caption-templates/:templateId/apply", captionTemplateController.Apply)
			channelRoutes.DELETE("/caption-templates/:templateId", captionTemplateController.Delete)

			channelRoutes.GET("/separator/:separatorId", channelController.GetSeparator)
			channelRoutes.GET("/separator", channelController.GetSeparatorByChannel)
			channelRoutes.PUT("/separator", channelController.UpdateSeparator)
			channelRoutes.DELETE("/separator", channelController.DeleteSeparator)
		}
	}

	// --- Rotas Administrativas (Apenas Admin/Owner) ---
	adminRoute := api.Group("/admin")
	adminRoute.Use(auth.RequireRole(auth.RoleAdmin, auth.RoleOwner))
	{
		adminRoute.GET("/overview", getALlUsers.GetAdminOverview)
		adminRoute.GET("/users", getALlUsers.GetAllUsersAdminController)
		adminRoute.GET("/channels", channelController.GetAllChannelsController)
		adminRoute.POST("/notice", getALlUsers.SendNoticeAdminController)

		adminRoute.GET("/config", configController.GetConfig)
		adminRoute.PUT("/config", configController.UpdateConfig)

		adminRoute.GET("/media-proxy/:fileId", mediaController.GetMediaPreview)
		adminRoute.GET("/audit/checkbot", auditController.GetCheckBotAudit)
		adminRoute.GET("/logs", channelEventsController.List)
		adminRoute.DELETE("/logs", channelEventsController.DeleteAll)
		adminRoute.POST("/audit/bulk-delete", auditController.BulkDeleteUserChannels)

		adminRoute.POST("/users/:userId/admin", getALlUsers.UpdateUserAdminController)
		adminRoute.POST("/users/:userId/blacklist", getALlUsers.UpdateUserBlacklistController)

		adminRoute.GET("/accounts", adminAccountController.ListAccounts)
		adminRoute.POST("/accounts/connect", adminAccountController.ConnectAccount)
		adminRoute.POST("/accounts/verify", adminAccountController.VerifyCode)
		adminRoute.POST("/accounts/password", adminAccountController.SendPassword)
		adminRoute.DELETE("/accounts/:id", adminAccountController.DeleteAccount)
		adminRoute.POST("/accounts/:id/toggle", adminAccountController.ToggleAccount)

		// Premium Features
		premiumFeaturesController := admincontroller.NewPremiumFeaturesController(c)
		adminRoute.GET("/premium/features", premiumFeaturesController.ListFeatures)
		adminRoute.PUT("/premium/features/:key", premiumFeaturesController.UpdateFeature)
		adminRoute.POST("/premium/features/:key/toggle", premiumFeaturesController.ToggleFeature)

		// Admin Subscriptions
		adminRoute.GET("/subscriptions", adminSubscriptionController.ListSubscriptions)
		adminRoute.POST("/subscriptions/cancel", adminSubscriptionController.Cancel)
		adminRoute.POST("/subscriptions/refund", adminSubscriptionController.Refund)
	}

	// --- Rotas de Conta Conectada MTProto ---
	accountRoutes := api.Group("/account")
	{
		accountRoutes.GET("", accountController.GetAccountStatus)
		accountRoutes.GET("/status", accountController.GetAuthStatus)
		accountRoutes.POST("/connect", accountController.ConnectAccount)
		accountRoutes.POST("/verify", accountController.VerifyCode)
		accountRoutes.POST("/password", accountController.SendPassword)
		accountRoutes.DELETE("", accountController.DisconnectAccount)
	}

}
