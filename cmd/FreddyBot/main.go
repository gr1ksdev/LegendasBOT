package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/leirbagxis/FreddyBot/internal/api"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database"
	"github.com/leirbagxis/FreddyBot/internal/telegram"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

func main() {
	if err := config.Validate(); err != nil {
		logger.Error("APP", "Configuração inválida: %v", err)
		os.Exit(1)
	}

	db, err := database.InitDB()
	if err != nil {
		logger.Error("APP", "Erro ao inicializar banco de dados: %v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	webhookHandler, _, app, cleanupBot, err := telegram.StartBot(ctx, db)
	if err != nil {
		logger.Error("APP", "Erro ao iniciar bot: %v", err)
		os.Exit(1)
	}

	app.StartBackground(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := api.StartApi(ctx, app, webhookHandler); err != nil {
			logger.Error("APP", "Erro no servidor de API: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("APP", "🧹 Encerrando app com segurança...")

	// 1. Para recebimento de novos updates/webhooks do Telegram
	if cleanupBot != nil {
		cleanupBot()
	}

	// 2. Aguarda o encerramento gracioso do servidor HTTP (srv.Shutdown com 5s timeout)
	wg.Wait()

	// 3. Fecha conexões com o Redis
	if err := cache.CloseRedis(); err != nil {
		logger.Error("APP", "Erro ao fechar Redis: %v", err)
	}

	// 4. Fecha conexões do banco de dados por último
	sqlDB, err := db.DB()
	if err == nil && sqlDB != nil {
		_ = sqlDB.Close()
		logger.Info("APP", "✅ Conexão com banco de dados encerrada")
	}

	logger.Info("APP", "✨ FreddyBot encerrado com sucesso!")
}
