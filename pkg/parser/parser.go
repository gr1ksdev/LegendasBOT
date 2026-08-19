package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"gopkg.in/yaml.v3"
)

type Button struct {
	Text                         string `yaml:"text"`
	CallbackData                 string `yaml:"callback_data,omitempty"`
	URL                          string `yaml:"url,omitempty"`
	SwitchInlineQuery            string `yaml:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat string `yaml:"switch_inline_query_current_chat,omitempty"`
	IconCustomEmojiID            string `yaml:"custom_emoji,omitempty"`
	Style                        string `yaml:"style,omitempty"`
	WebApp                       string `yaml:"web_app,omitempty"`
}

type Message struct {
	Name        string     `yaml:"name"`
	Text        string     `yaml:"text"`
	Buttons     [][]Button `yaml:"buttons,omitempty"`
	HasVarsText bool       `yaml:"-"`
	VarKeys     []string   `yaml:"-"`
}

var (
	messagesMap = make(map[string]Message)
	loadOnce    sync.Once
	varRegex    = regexp.MustCompile(`\{(\w+)\}`) // Adjusted to match {var} instead of {{var}} based on yml
)

func loadMessages() {
	paths := []string{"config/messages.yml", "../config/messages.yml", "../../config/messages.yml"}
	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		logger.Error("PARSER", "Erro ao carregar arquivo de mensagens: %v", err)
		return
	}

	// Unmarshal into a slice first (!!seq)
	var messages []Message
	if err := yaml.Unmarshal(data, &messages); err != nil {
		logger.Error("PARSER", "Erro ao parsear YAML (list): %v", err)
		return
	}

	// Clear existing map to avoid stale data
	for k := range messagesMap {
		delete(messagesMap, k)
	}

	for _, msg := range messages {
		msg.HasVarsText = varRegex.MatchString(msg.Text)
		if msg.HasVarsText {
			matches := varRegex.FindAllStringSubmatch(msg.Text, -1)
			keys := make([]string, 0, len(matches))
			for _, m := range matches {
				keys = append(keys, m[1])
			}
			msg.VarKeys = keys
		}
		messagesMap[msg.Name] = msg
	}

	logger.Info("PARSER", "✅ %d mensagens carregadas com sucesso", len(messagesMap))
}

func ParseText(text string, vars map[string]string, keys []string) string {
	res := text
	for _, key := range keys {
		val, ok := vars[key]
		if !ok {
			val = fmt.Sprintf("{%s}", key)
		}
		res = strings.ReplaceAll(res, "{"+key+"}", val)
	}
	return res
}

func parseButtons(rows [][]Button, vars map[string]string) [][]Button {
	if len(rows) == 0 {
		return nil
	}

	newRows := make([][]Button, len(rows))
	for i, row := range rows {
		newRow := make([]Button, len(row))
		for j, btn := range row {
			newBtn := btn
			// Parse text
			if varRegex.MatchString(btn.Text) {
				matches := varRegex.FindAllStringSubmatch(btn.Text, -1)
				keys := make([]string, 0, len(matches))
				for _, m := range matches {
					keys = append(keys, m[1])
				}
				newBtn.Text = ParseText(btn.Text, vars, keys)
			}
			// Parse callback_data
			if btn.CallbackData != "" && varRegex.MatchString(btn.CallbackData) {
				matches := varRegex.FindAllStringSubmatch(btn.CallbackData, -1)
				keys := make([]string, 0, len(matches))
				for _, m := range matches {
					keys = append(keys, m[1])
				}
				newBtn.CallbackData = ParseText(btn.CallbackData, vars, keys)
			}
			// Parse URL
			if btn.URL != "" && varRegex.MatchString(btn.URL) {
				matches := varRegex.FindAllStringSubmatch(btn.URL, -1)
				keys := make([]string, 0, len(matches))
				for _, m := range matches {
					keys = append(keys, m[1])
				}
				newBtn.URL = ParseText(btn.URL, vars, keys)
			}
			// Parse switch_inline_query
			if btn.SwitchInlineQuery != "" && varRegex.MatchString(btn.SwitchInlineQuery) {
				matches := varRegex.FindAllStringSubmatch(btn.SwitchInlineQuery, -1)
				keys := make([]string, 0, len(matches))
				for _, m := range matches {
					keys = append(keys, m[1])
				}
				newBtn.SwitchInlineQuery = ParseText(btn.SwitchInlineQuery, vars, keys)
			}
			// Parse WebApp
			if btn.WebApp != "" && varRegex.MatchString(btn.WebApp) {
				matches := varRegex.FindAllStringSubmatch(btn.WebApp, -1)
				keys := make([]string, 0, len(matches))
				for _, m := range matches {
					keys = append(keys, m[1])
				}
				newBtn.WebApp = ParseText(btn.WebApp, vars, keys)
			}

			newRow[j] = newBtn
		}
		newRows[i] = newRow
	}
	return newRows
}

func BuildInlineKeyboardTelego(buttons [][]Button) *telego.InlineKeyboardMarkup {
	if len(buttons) == 0 {
		return nil
	}

	inlineKeyboard := make([][]telego.InlineKeyboardButton, len(buttons))
	for i, row := range buttons {
		btnRow := make([]telego.InlineKeyboardButton, len(row))
		for j, btn := range row {
			btnRow[j] = telego.InlineKeyboardButton{
				Text:              btn.Text,
				CallbackData:      btn.CallbackData,
				URL:               btn.URL,
				IconCustomEmojiID: btn.IconCustomEmojiID,
				Style:             btn.Style,
			}

			if btn.SwitchInlineQuery != "" {
				query := btn.SwitchInlineQuery
				btnRow[j].SwitchInlineQuery = &query
			}
			if btn.SwitchInlineQueryCurrentChat != "" {
				query := btn.SwitchInlineQueryCurrentChat
				btnRow[j].SwitchInlineQueryCurrentChat = &query
			}

			if btn.WebApp != "" {
				btnRow[j].WebApp = &telego.WebAppInfo{URL: btn.WebApp}
			}
		}
		inlineKeyboard[i] = btnRow
	}

	return &telego.InlineKeyboardMarkup{InlineKeyboard: inlineKeyboard}
}

func GetMessageTelego(name string, vars map[string]string) (string, *telego.InlineKeyboardMarkup) {
	loadOnce.Do(loadMessages)

	msg, ok := messagesMap[name]
	if !ok {
		return fmt.Sprintf("Mensagem '%s' não encontrada!", name), nil
	}

	text := msg.Text
	if msg.HasVarsText && len(vars) > 0 {
		text = ParseText(text, vars, msg.VarKeys)
	}

	buttons := parseButtons(msg.Buttons, vars)
	keyboard := BuildInlineKeyboardTelego(buttons)

	return text, keyboard
}
