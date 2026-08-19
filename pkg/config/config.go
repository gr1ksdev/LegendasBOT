package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

// isTestMode retorna true se o binario foi compilado com testes.
func isTestMode() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

var (
	TelegramBotToken      string
	DatabaseFile          string
	RedisAddr             string
	OwnerID               int64
	SecreteKey            string // Deprecated: use SecretKey
	SecretKey             string
	WebAppURL             string
	WebhookURL            string
	TelegramWebhookSecret string
	AppPort               string
	AppEnv                string
	JWTIssuer             string
	CORSAllowOrigins      []string

	// StarsTestMode define se as invoices usam precos de teste (1 star).
	StarsTestMode bool

	// MTProto
	MTProtoAppID         int
	MTProtoAppHash       string
	MTProtoEncryptionKey string
)

func init() {
	Load()
}

// Load recarrega as variáveis de ambiente para a estrutura global do pacote.
func Load() {
	if os.Getenv("GO_ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			logger.Warn("CONFIG", "⚠️  .env não encontrado — usando variáveis de ambiente do container")
		}
	}

	TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if TelegramBotToken == "" && isTestMode() {
		TelegramBotToken = "test_TELEGRAM_BOT_TOKEN"
	}

	RedisAddr = os.Getenv("REDIS_HOST")
	if RedisAddr == "" && isTestMode() {
		RedisAddr = "test_REDIS_HOST"
	}

	DatabaseFile = os.Getenv("DATABASE_FILE") // opcional em dev (default sqlite local)
	if DatabaseFile == "" && isTestMode() {
		DatabaseFile = "test_database.db"
	}

	ownerStr := os.Getenv("OWNER_ID")
	if ownerStr != "" {
		if id, err := strconv.ParseInt(ownerStr, 10, 64); err == nil {
			OwnerID = id
		}
	} else if isTestMode() {
		OwnerID = 123456789
	}

	AppPort = os.Getenv("APP_PORT")
	if AppPort == "" {
		AppPort = "7000"
	}

	SecretKey = os.Getenv("SECRET_KEY")
	if SecretKey == "" && isTestMode() {
		SecretKey = "test_secret_key_must_be_at_least_32_bytes_long_for_hs256!"
	}
	SecreteKey = SecretKey

	WebAppURL = os.Getenv("WEBAPP_URL")
	if WebAppURL == "" && isTestMode() {
		WebAppURL = "https://example.com"
	}

	WebhookURL = os.Getenv("WEBHOOK_URL")
	TelegramWebhookSecret = os.Getenv("TELEGRAM_WEBHOOK_SECRET")

	AppEnv = os.Getenv("APP_ENV")
	if AppEnv == "" {
		AppEnv = "dev"
	}

	StarsTestMode = os.Getenv("STARS_TEST_MODE") == "true"
	if StarsTestMode && (AppEnv == "prod" || os.Getenv("GO_ENV") == "production") {
		logger.Error("CONFIG", "⛔ STARS_TEST_MODE=true em ambiente de produção! Desativando por segurança.")
		StarsTestMode = false
	}
	if StarsTestMode {
		logger.Bot("🧪 Modo teste Stars ATIVADO — invoices com ?test=true custam 1 star")
	} else {
		logger.Bot("⭐ Modo teste Stars DESATIVADO — assinaturas usam precos reais")
	}

	JWTIssuer = getEnvDefault("JWT_ISSUER", "t.me/legendasbrbot")
	CORSAllowOrigins = parseOrigins(os.Getenv("CORS_ALLOW_ORIGINS"), WebAppURL)

	mtprotoAppIDStr := os.Getenv("MTPROTO_APP_ID")
	MTProtoAppID = 0
	if mtprotoAppIDStr != "" {
		if id, err := strconv.Atoi(mtprotoAppIDStr); err == nil {
			MTProtoAppID = id
		}
	}
	MTProtoAppHash = os.Getenv("MTPROTO_APP_HASH")

	MTProtoEncryptionKey = os.Getenv("MTPROTO_ENCRYPTION_KEY")
	if MTProtoEncryptionKey == "" {
		MTProtoEncryptionKey = SecretKey
		if MTProtoAppID > 0 {
			logger.Warn("CONFIG", "⚠️ MTPROTO_ENCRYPTION_KEY não configurada. Usando SECRET_KEY como fallback para criptografia MTProto.")
		}
	}
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetMTProtoAppID retorna o App ID para MTProto ou 0 se nao configurado.
func GetMTProtoAppID() int {
	return MTProtoAppID
}

// GetMTProtoAppHash retorna o App Hash para MTProto ou string vazia.
func GetMTProtoAppHash() string {
	return MTProtoAppHash
}

func parseOrigins(raw string, fallback string) []string {
	var origins []string
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 && fallback != "" {
		origins = append(origins, fallback)
	}
	return origins
}

func isValidWebhookSecret(secret string) bool {
	if len(secret) < 1 || len(secret) > 256 {
		return false
	}
	for _, r := range secret {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// Validate verifica se todas as configurações obrigatórias foram carregadas corretamente.
func Validate() error {
	var missing []string

	if TelegramBotToken == "" {
		missing = append(missing, "TELEGRAM_BOT_TOKEN")
	}
	if RedisAddr == "" {
		missing = append(missing, "REDIS_HOST")
	}
	if OwnerID <= 0 {
		missing = append(missing, "OWNER_ID (deve ser um inteiro positivo > 0)")
	}
	if SecretKey == "" {
		missing = append(missing, "SECRET_KEY")
	} else if len(SecretKey) < 32 {
		missing = append(missing, "SECRET_KEY (deve ter pelo menos 32 caracteres/bytes)")
	}

	if AppEnv != "dev" && AppEnv != "prod" {
		missing = append(missing, "APP_ENV (deve ser 'dev' ou 'prod')")
	}

	dbDriver := os.Getenv("DATABASE_DRIVER")
	if (AppEnv == "prod" || dbDriver == "postgres") && dbDriver != "sqlite" && DatabaseFile == "" {
		missing = append(missing, "DATABASE_FILE (DSN de conexão com PostgreSQL é obrigatório em produção ou quando DATABASE_DRIVER=postgres)")
	}

	if WebhookURL != "" {
		if TelegramWebhookSecret == "" {
			missing = append(missing, "TELEGRAM_WEBHOOK_SECRET (obrigatório quando WEBHOOK_URL está configurado)")
		} else if len(TelegramWebhookSecret) < 8 || len(TelegramWebhookSecret) > 256 {
			missing = append(missing, "TELEGRAM_WEBHOOK_SECRET (deve ter entre 8 e 256 caracteres)")
		} else if !isValidWebhookSecret(TelegramWebhookSecret) {
			missing = append(missing, "TELEGRAM_WEBHOOK_SECRET (deve conter apenas caracteres permitidos A-Z, a-z, 0-9, _ ou -)")
		}
	}

	if (MTProtoAppID > 0 && MTProtoAppHash == "") || (MTProtoAppID == 0 && MTProtoAppHash != "") {
		missing = append(missing, "MTPROTO_APP_ID e MTPROTO_APP_HASH (ambos devem ser configurados para habilitar MTProto)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("configuração de ambiente inválida: %s", strings.Join(missing, "; "))
	}
	return nil
}
