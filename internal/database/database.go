package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	customLogger "github.com/leirbagxis/FreddyBot/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultGlobalDefaultCaption = "🐈‍⠀៹ [t.me/legendasbot](https://t.me/{usernameBot})  ‹"

const defaultGlobalNewPackCaption = " `୨ৎ`  pack ˚₊· [click here to add]($link) ·₊˚\n\n `✧˖°`  $count stickers ˖°✧\n\n🐈⠀៹ [t.me/legendasbot](https://t.me/{usernameBot})  ‹"

const defaultFixedPostBuilderPayload = `{"media_type":"photo","media_file_id":"AgACAgEAAxkBAAIN1GoO7mINPlBGs_ydPnmkDPdxeQ8eAAKoC2sbf_d4RIZ9nu_0BSIiAQADAgADeAADOwQ","menu_message_id":0,"prompt_message_id":0,"title":"","body":"<tg-emoji emoji-id=\"5373026167722876724\">🤩</tg-emoji> Cansado de perder tempo editando postagens?\nO <a href=\"http://t.me/LegendasBrBot?start=start\">LegendasBOT</a> resolve isso pra você de forma simples e eficiente <tg-emoji emoji-id=\"5445284980978621387\">🚀</tg-emoji>","footer":"","reactions":"","buttons":[{"text":"🤖 Legendas BOT","url":"http://t.me/LegendasBrBot?start=start","custom_emoji_id":"5296447931627352804"},{"text":"📺 Central de Novidades","url":"https://t.me/LegendasBOTTopic","custom_emoji_id":"5373330964372004748"}],"step":""}`

func DefaultFixedPostBuilderPayload() string {
	return defaultFixedPostBuilderPayload
}

func validFixedPostBuilderPayload(payload string) bool {
	if strings.TrimSpace(payload) == "" {
		return false
	}
	var raw map[string]any
	return json.Unmarshal([]byte(payload), &raw) == nil
}

func InitDB() (*gorm.DB, error) {
	var dialector gorm.Dialector
	isSQLite := false

	// DatabaseDriver permite forçar postgres/sqlite independente do AppEnv.
	// Valores: "postgres", "sqlite" (ou vazio = usa AppEnv).
	dbDriver := os.Getenv("DATABASE_DRIVER")
	switch dbDriver {
	case "postgres":
		customLogger.DB("🐘 Usando banco de dados PostgreSQL (forçado por DATABASE_DRIVER)")
		dialector = postgres.Open(config.DatabaseFile)
	case "sqlite":
		isSQLite = true
		customLogger.DB("📦 Usando banco de dados SQLite (forçado por DATABASE_DRIVER)")
		dialector = sqlite.Open(config.DatabaseFile)
	default:
		if config.AppEnv == "dev" {
			isSQLite = true
			customLogger.DB("📦 Usando banco de dados SQLite (modo dev)")
			dialector = sqlite.Open(config.DatabaseFile)
		} else {
			customLogger.DB("🐘 Usando banco de dados PostgreSQL (modo prod)")
			dialector = postgres.Open(config.DatabaseFile)
		}
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.Config.Logger = logger.Default.LogMode(logger.Silent)

	// Habilitar Foreign Keys no SQLite
	if isSQLite {
		if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
			customLogger.Error("DATABASE", "Erro ao habilitar foreign keys no SQLite: %v", err)
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
	}

	// Configurar Pool de Conexões (Crucial para produção)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	customLogger.DB("⚙️ Pool de conexões configurado (Idle: 10, Open: 100)")

	if !isSQLite {
		customLogger.DB("⚙️ Executando migrações DDL manuais do PostgreSQL...")
		// Forçar recriação de índices que mudaram de estrutura
		if err := db.Exec("DROP INDEX IF EXISTS idx_vote_user").Error; err != nil {
			return nil, fmt.Errorf("drop index idx_vote_user: %w", err)
		}

		// Migrações na tabela scheduled_posts (apenas se a tabela já existir)
		if err := db.Exec(`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='scheduled_posts') THEN
				IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='scheduled_posts' AND column_name='id' AND data_type='uuid') THEN
					ALTER TABLE scheduled_posts ALTER COLUMN id TYPE text;
				END IF;

				IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='scheduled_posts' AND column_name='pin_message') THEN
					ALTER TABLE scheduled_posts ADD COLUMN pin_message boolean NOT NULL DEFAULT false;
				END IF;
			END IF;
		END $$;`).Error; err != nil {
			return nil, fmt.Errorf("migrate scheduled_posts DDL: %w", err)
		}
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Subscription{},
		&models.PaymentIntent{},
		&models.ServerConfig{},
		&models.Channel{},
		&models.ChannelEvent{},
		&models.DefaultCaption{},
		&models.MessagePermission{},
		&models.ButtonsPermission{},
		&models.Button{},
		&models.Separator{},
		&models.CustomCaption{},
		&models.CustomCaptionButton{},
		&models.Vote{},
		&models.ConnectedAccount{},
		&models.ConnectedAccountChannel{},
		&models.CustomEmoji{},
		&models.UserEmojiAccess{},
		&models.AdminMTProtoAccount{},
		&models.PremiumFeature{},
		&models.Refund{},
		&models.ScheduledPost{},
		&models.AutoDeletePost{},
		&models.UserPostTemplate{},
		&models.UserCaptionTemplate{},
		&models.UserCaptionTemplateButton{},
	)
	if err != nil {
		return nil, err
	}

	if err := initServerConfig(db); err != nil {
		return nil, err
	}

	if err := seedPremiumFeatures(db); err != nil {
		return nil, err
	}

	return db, nil
}

func initServerConfig(db *gorm.DB) error {
	config := models.ServerConfig{
		ID:                      1,
		Maintence:               false,
		ForceJoin:               false,
		GlobalDefaultCaption:    defaultGlobalDefaultCaption,
		FixedPostBuilderEnabled: true,
		FixedPostBuilderKey:     "legendasbot",
		FixedPostBuilderPayload: defaultFixedPostBuilderPayload,
		GlobalNewPackCaption:    defaultGlobalNewPackCaption,
		LogRetentionDays:        30,
	}

	if err := db.WithContext(context.Background()).FirstOrCreate(&config, models.ServerConfig{ID: 1}).Error; err != nil {
		return err
	}

	changed := false
	if strings.TrimSpace(config.GlobalDefaultCaption) == "" {
		config.GlobalDefaultCaption = defaultGlobalDefaultCaption
		changed = true
	}
	if strings.TrimSpace(config.GlobalNewPackCaption) == "" {
		config.GlobalNewPackCaption = defaultGlobalNewPackCaption
		changed = true
	}
	if config.FixedPostBuilderKey == "" {
		config.FixedPostBuilderKey = "legendasbot"
		changed = true
	}
	if !validFixedPostBuilderPayload(config.FixedPostBuilderPayload) {
		config.FixedPostBuilderPayload = defaultFixedPostBuilderPayload
		config.FixedPostBuilderEnabled = true
		changed = true
	}
	if config.LogRetentionDays <= 0 {
		config.LogRetentionDays = 30
		changed = true
	}
	if changed {
		if err := db.WithContext(context.Background()).Save(&config).Error; err != nil {
			return err
		}
	}

	customLogger.DB("✔️ ServerConfig iniciado criadas com sucesso.")
	return nil
}

// seedPremiumFeatures cria as features premium padrao se nao existirem.
func seedPremiumFeatures(db *gorm.DB) error {
	defaults := []models.PremiumFeature{
		{
			Key:         "managed_premium_account",
			Name:        "Conta Telegram Gerenciada",
			Description: "Usa uma conta Telegram gerenciada pelo admin como executor MTProto para edicoes avançadas.",
			Enabled:     true,
			Price:       80,
		},
		{
			Key:         "connected_account",
			Name:        "Conta Telegram Pessoal",
			Description: "Permite ao usuario conectar sua propria conta Telegram via MTProto para recursos exclusivos.",
			Enabled:     true,
			Price:       0,
		},
		{
			Key:         "custom_emojis",
			Name:        "Emojis Customizados",
			Description: "Permite o uso de emojis customizados (Premium) nas legendas dos posts.",
			Enabled:     true,
			Price:       0,
		},
		{
			Key:         "extra_channels",
			Name:        "Canais Extras",
			Description: "Permite adicionar canais adicionais alem do limite padrao. Preco por canal extra.",
			Enabled:     true,
			Price:       35,
		},
	}

	for _, f := range defaults {
		var existing models.PremiumFeature
		err := db.WithContext(context.Background()).
			Where("key = ?", f.Key).
			First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.WithContext(context.Background()).Create(&f).Error; err != nil {
				customLogger.Error("DATABASE", "Erro ao criar feature premium %s: %v", f.Key, err)
				return err
			}
			customLogger.DB("🌟 Feature premium criada: %s (%d stars)", f.Key, f.Price)
		} else if err != nil {
			return err
		}
	}

	return nil
}
