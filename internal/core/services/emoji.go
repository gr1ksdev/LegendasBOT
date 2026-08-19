package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

// ── Emoji Service ──

type EmojiService struct {
	repo       *repositories.EmojiRepository
	bot        *telego.Bot
	client     *http.Client
	downloadMu sync.Map // map[string]*sync.Mutex — um mutex por emojiID
}

func NewEmojiService(repo *repositories.EmojiRepository, bot *telego.Bot) *EmojiService {
	return &EmojiService{
		repo: repo,
		bot:  bot,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

const maxEmojiSize = 5 * 1024 * 1024 // 5MB limit per emoji

// Regex para extrair emoji-ids de texto HTML
var tgEmojiRe = regexp.MustCompile(`<tg-emoji\s+emoji-id="(\d+)"`)

// ExtractEmojiIDs extrai todos os emoji-ids únicos de um texto HTML.
func (s *EmojiService) ExtractEmojiIDs(text string) []string {
	matches := tgEmojiRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		id := m[1]
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

// TextContainsEmoji verifica rapidamente se um texto tem tags tg-emoji.
func (s *EmojiService) TextContainsEmoji(text string) bool {
	return tgEmojiRe.MatchString(text)
}

// FetchForUser é o método principal para o endpoint público.
// Concede acesso ao usuário, baixa via Bot API se não existir, e retorna o emoji.
func (s *EmojiService) FetchForUser(ctx context.Context, userID int64, emojiID string) (*models.CustomEmoji, error) {
	// 1. Concede acesso (idempotente — OnConflict DoNothing)
	if err := s.repo.GrantAccess(ctx, userID, emojiID); err != nil {
		return nil, fmt.Errorf("grant access: %w", err)
	}

	// 2. Busca no banco
	emoji, err := s.repo.GetEmoji(ctx, emojiID)
	if err != nil {
		return nil, fmt.Errorf("get emoji: %w", err)
	}
	if emoji != nil {
		return emoji, nil
	}

	// 3. Não existe — baixar via Bot API com mutex por emojiID
	mu, _ := s.downloadMu.LoadOrStore(emojiID, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	// Double-check dentro do lock
	existing, _ := s.repo.GetEmoji(ctx, emojiID)
	if existing != nil {
		return existing, nil
	}

	download, err := s.downloadAndSave(ctx, emojiID)
	if err != nil {
		return nil, fmt.Errorf("download emoji %s: %w", emojiID, err)
	}

	return download, nil
}

// EnsureEmojisForText garante que todos os emojis mencionados no texto
// estejam baixados e que o usuário tenha acesso a eles.
func (s *EmojiService) EnsureEmojisForText(ctx context.Context, userID int64, text string) error {
	ids := s.ExtractEmojiIDs(text)
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		if err := s.repo.GrantAccess(ctx, userID, id); err != nil {
			logger.Warn("EMOJI", "GrantAccess error (user=%d, emoji=%s): %v", userID, id, err)
			continue
		}

		existing, err := s.repo.GetEmoji(ctx, id)
		if err != nil {
			logger.Warn("EMOJI", "GetEmoji error (emoji=%s): %v", id, err)
			continue
		}
		if existing != nil {
			continue
		}

		// Download em background
		go func(eid string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if _, err := s.downloadAndSave(bgCtx, eid); err != nil {
				logger.Warn("EMOJI", "Background download failed (emoji=%s): %v", eid, err)
			} else {
				logger.Info("EMOJI", "Downloaded emoji %s via Bot API", eid)
			}
		}(id)
	}

	return nil
}

// ListAccessedEmojiIDs retorna os IDs dos emojis que o usuário já usou,
// ordenados do mais recente para o mais antigo (último acesso primeiro).
func (s *EmojiService) ListAccessedEmojiIDs(ctx context.Context, userID int64) ([]string, error) {
	return s.repo.GetAccessedEmojiIDs(ctx, userID)
}

// ── Download via Bot API ──

// downloadAndSave baixa o emoji via BotAPI e salva no banco.
// Fluxo: getCustomEmojiStickers → getFile → download dos bytes
func (s *EmojiService) downloadAndSave(ctx context.Context, emojiID string) (*models.CustomEmoji, error) {
	// 1. Busca info do sticker via Bot API
	stickers, err := s.bot.GetCustomEmojiStickers(ctx, &telego.GetCustomEmojiStickersParams{
		CustomEmojiIDs: []string{emojiID},
	})
	if err != nil {
		return nil, fmt.Errorf("getCustomEmojiStickers: %w", err)
	}
	if len(stickers) == 0 {
		return nil, fmt.Errorf("no sticker found for emoji %s", emojiID)
	}

	sticker := stickers[0]

	// 2. Determinar o tipo de arquivo
	// is_video → .webm (video sticker)
	// is_animated → .tgs (Lottie animated)
	// neither → .webp (static)
	var ext string
	switch {
	case sticker.IsVideo:
		ext = ".webm"
	case sticker.IsAnimated:
		ext = ".tgs"
	default:
		ext = ".webp"
	}
	logger.Info("EMOJI", "Emoji %s: type=%s animated=%v video=%v ext=%s",
		emojiID, sticker.Type, sticker.IsAnimated, sticker.IsVideo, ext)

	// 3. Obtém o file_path via getFile
	fileInfo, err := s.bot.GetFile(ctx, &telego.GetFileParams{
		FileID: sticker.FileID,
	})
	if err != nil {
		return nil, fmt.Errorf("getFile: %w", err)
	}
	if fileInfo == nil || fileInfo.FilePath == "" {
		return nil, fmt.Errorf("empty file path for emoji %s", emojiID)
	}

	// 4. Download dos bytes
	// URL: https://api.telegram.org/file/bot<token>/<file_path>
	data, err := s.downloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	// 5. Salvar no banco
	emoji := &models.CustomEmoji{
		EmojiID:  emojiID,
		FileData: data,
		FileType: ext,
	}

	if err := s.repo.UpsertEmoji(ctx, emoji); err != nil {
		return nil, fmt.Errorf("save emoji %s: %w", emojiID, err)
	}

	return emoji, nil
}

// downloadFile baixa o arquivo binário do Telegram.
func (s *EmojiService) downloadFile(ctx context.Context, filePath string) ([]byte, error) {
	// Usamos a URL pública: https://api.telegram.org/file/bot<token>/<file_path>
	// Logar apenas o filePath, nunca a URL completa (que contém o token)
	logger.Info("EMOJI", "Baixando arquivo: %s", filePath)

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", s.bot.Token(), filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// User-Agent realista para evitar bloqueios
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TelegramBot)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d ao baixar arquivo", resp.StatusCode)
	}

	if resp.ContentLength > maxEmojiSize {
		return nil, fmt.Errorf("arquivo de emoji excede %d bytes", maxEmojiSize)
	}
	limited := io.LimitReader(resp.Body, maxEmojiSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxEmojiSize {
		return nil, fmt.Errorf("arquivo de emoji excede %d bytes", maxEmojiSize)
	}

	return data, nil
}
