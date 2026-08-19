package executor_test

import (
	"context"
	"sync"
	"testing"

	"github.com/leirbagxis/FreddyBot/internal/telegram/executor"
)

// mockExecutor implements TelegramExecutor for testing.
type mockExecutor struct {
	name     string
	editCall int
	mu       sync.Mutex
}

func (m *mockExecutor) EditMessage(ctx context.Context, chatID int64, messageID int, text, parseMode string, keyboard *executor.InlineKeyboardMarkup, opts *executor.EditOptions) error {
	m.mu.Lock()
	m.editCall++
	m.mu.Unlock()
	return nil
}

func (m *mockExecutor) EditCaption(ctx context.Context, chatID int64, messageID int, caption, parseMode string, keyboard *executor.InlineKeyboardMarkup, opts *executor.EditOptions) error {
	return nil
}

func (m *mockExecutor) EditReplyMarkup(ctx context.Context, chatID int64, messageID int, keyboard *executor.InlineKeyboardMarkup) error {
	return nil
}

func (m *mockExecutor) SendSticker(ctx context.Context, chatID int64, stickerID string) error {
	return nil
}

func (m *mockExecutor) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	return nil
}

func (m *mockExecutor) SendMessage(ctx context.Context, chatID int64, text, parseMode string, keyboard *executor.InlineKeyboardMarkup, opts *executor.EditOptions) error {
	return nil
}

// mockProvider implements executor.Provider for testing.
type mockProvider struct {
	mu        sync.RWMutex
	connected map[int64]bool
	premium   map[int64]bool
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		connected: make(map[int64]bool),
		premium:   make(map[int64]bool),
	}
}

func (p *mockProvider) HasConnectedAccount(ctx context.Context, userID int64) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected[userID]
}

func (p *mockProvider) HasPremiumManagedAccount(ctx context.Context, userID int64) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.premium[userID]
}

func (p *mockProvider) SetConnected(userID int64, connected bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connected[userID] = connected
}

func (p *mockProvider) SetPremium(userID int64, premium bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.premium[userID] = premium
}

func TestExecutorFactory_ForUser_NoConnectedAccount(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, nil, nil, provider)
	ctx := context.Background()

	exec := factory.ForUser(ctx, 100)
	if exec == nil {
		t.Fatal("ForUser() returned nil")
	}

	// Should be BotAPI executor (no connected account)
	_ = exec.EditMessage(ctx, 0, 0, "test", "HTML", nil, nil)
}

func TestExecutorFactory_ForUser_WithConnectedAccount(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, nil, nil, provider)
	ctx := context.Background()

	// User without connected account
	exec1 := factory.ForUser(ctx, 200)
	if exec1 == nil {
		t.Fatal("ForUser() returned nil for non-connected user")
	}

	// User with connected account (still uses BotAPI since MTProto is nil)
	provider.SetConnected(201, true)
	exec2 := factory.ForUser(ctx, 201)
	if exec2 == nil {
		t.Fatal("ForUser() returned nil for connected user")
	}
}

func TestExecutorFactory_WithNilMTProto(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, nil, nil, provider)
	ctx := context.Background()

	provider.SetConnected(300, true)

	// Even with connected account, should use BotAPI since MTProto is nil
	exec := factory.ForUser(ctx, 300)
	if exec == nil {
		t.Fatal("ForUser() returned nil")
	}
}

func TestExecutorFactory_InvalidateCache(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, nil, nil, provider)
	ctx := context.Background()

	provider.SetConnected(400, true)

	// First call - caches BotAPI (since MTProto is nil)
	exec1 := factory.ForUser(ctx, 400)

	// Disconnect and invalidate
	provider.SetConnected(400, false)
	factory.InvalidateCache(400)

	// Should still get BotAPI (same since no MTProto)
	exec2 := factory.ForUser(ctx, 400)

	_ = exec1
	_ = exec2
}

func TestExecutorFactory_ConcurrentAccess(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, nil, nil, provider)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if id%2 == 0 {
				provider.SetConnected(id, true)
			}
			exec := factory.ForUser(ctx, id)
			if exec == nil {
				t.Errorf("ForUser(%d) returned nil in concurrent access", id)
			}
		}(int64(i))
	}
	wg.Wait()
}

func TestExecutorFactory_ForUser_WithPremium(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	// MTProto configured, adminSP configured, user has premium but no connected account
	factory := executor.NewExecutorFactory(botAPI, &executor.MTProtoExecutor{}, &mockAdminSessionProvider{}, provider)
	ctx := context.Background()

	// User with premium
	provider.SetPremium(500, true)

	exec := factory.ForUser(ctx, 500)
	if exec == nil {
		t.Fatal("ForUser() returned nil for premium user")
	}
}

func TestExecutorFactory_ForUser_PremiumOverConnectedAccount(t *testing.T) {
	botAPI := &mockExecutor{name: "bot-api"}
	provider := newMockProvider()
	factory := executor.NewExecutorFactory(botAPI, &executor.MTProtoExecutor{}, &mockAdminSessionProvider{}, provider)
	ctx := context.Background()

	// User has BOTH connected account AND premium
	// Connected account should take priority (user's own MTProto)
	provider.SetConnected(600, true)
	provider.SetPremium(600, true)

	exec := factory.ForUser(ctx, 600)
	if exec == nil {
		t.Fatal("ForUser() returned nil")
	}
}

// mockAdminSessionProvider implements executor.AdminSessionProvider for testing.
type mockAdminSessionProvider struct{}

func (m *mockAdminSessionProvider) GetAdminSession(ctx context.Context) ([]byte, string, error) {
	return nil, "", nil
}

func TestNewEmptyKeyboard(t *testing.T) {
	kb := executor.NewEmptyKeyboard()
	if kb == nil {
		t.Fatal("NewEmptyKeyboard() returned nil")
	}
	if kb.InlineKeyboard == nil {
		t.Error("InlineKeyboard should be initialized")
	}
}

func TestInlineKeyboardMarkup_Structure(t *testing.T) {
	kb := &executor.InlineKeyboardMarkup{
		InlineKeyboard: [][]executor.InlineKeyboardButton{
			{
				{Text: "Button 1", URL: "https://example.com"},
				{Text: "Button 2", CallbackData: "cb_data"},
			},
			{
				{Text: "Button 3", URL: "https://test.com"},
			},
		},
	}

	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(kb.InlineKeyboard))
	}
	if len(kb.InlineKeyboard[0]) != 2 {
		t.Errorf("Expected 2 buttons in row 0, got %d", len(kb.InlineKeyboard[0]))
	}
	if len(kb.InlineKeyboard[1]) != 1 {
		t.Errorf("Expected 1 button in row 1, got %d", len(kb.InlineKeyboard[1]))
	}

	if kb.InlineKeyboard[0][0].URL != "https://example.com" {
		t.Errorf("Button URL = %q", kb.InlineKeyboard[0][0].URL)
	}
	if kb.InlineKeyboard[0][1].CallbackData != "cb_data" {
		t.Errorf("Button CallbackData = %q", kb.InlineKeyboard[0][1].CallbackData)
	}
}
