package channelpost

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/mymmrac/telego"
)

func GetMessageTypeTelego(post *telego.Message) MessageType {
	if post == nil {
		return ""
	}
	switch {
	case post.Text != "":
		return MessageTypeText
	case post.Audio != nil:
		return MessageTypeAudio
	case post.Sticker != nil:
		return MessageTypeSticker
	case post.Photo != nil:
		return MessageTypePhoto
	case post.Video != nil:
		return MessageTypeVideo
	case post.Animation != nil:
		return MessageTypeAnimation
	case post.Document != nil:
		return MessageTypeDocument
	default:
		return ""
	}
}

var (
	retryAfterRegex = regexp.MustCompile(`retry after (\d+)`)
	hashtagRegex    = regexp.MustCompile(`#(\w+)`)
	prefixRegex     = regexp.MustCompile(`!(\w+)`)
)

func extractRetryAfter(errStr string) int {
	matches := retryAfterRegex.FindStringSubmatch(errStr)
	if len(matches) > 1 {
		retryAfter, _ := strconv.Atoi(matches[1])
		return retryAfter
	}
	return 0
}

func extractHashtag(text string) string {
	matches := hashtagRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractPrefix(text string) string {
	matches := prefixRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func removePrefix(text, prefix string) string {
	return strings.TrimSpace(strings.Replace(text, "!"+prefix, "", 1))
}

func removeHashtag(text, hashtag string) string {
	return strings.TrimSpace(strings.Replace(text, "#"+hashtag, "", 1))
}

func findCustomCaption(channel *dbmodels.Channel, hashtag string) *dbmodels.CustomCaption {
	if channel == nil || hashtag == "" {
		return nil
	}
	for _, cc := range channel.CustomCaptions {
		if strings.EqualFold(cc.Code, hashtag) {
			return &cc
		}
	}
	return nil
}

func composeMessage(o, d, sep, order string) string {
	if o == "" && d == "" {
		return ""
	}
	if o == "" {
		return d
	}
	if d == "" {
		return o
	}
	if sep == "" {
		sep = "\n\n"
	}
	if order == "prepend" {
		return d + sep + o
	}
	return o + sep + d
}

func IsMarkdown(text string) bool {
	// Verifica se contém marcadores de Markdown (suporta básicos do Telegram)
	mdChars := []string{"*", "_", "`", "["}
	for _, char := range mdChars {
		if strings.Contains(text, char) {
			return true
		}
	}
	return false
}

func DetectParseMode(text string) string {
	if text == "" {
		return ""
	}

	// Escapar caracteres HTML ANTES de aplicar as regexes de Markdown
	// Isso garante que '&' vire '&amp;', mas as tags que inserirmos depois (<b>, <i>) fiquem intactas.
	res := html.EscapeString(text)
	res, protectedLinks := utils.ProtectMarkdownLinks(res, "channelpost.DetectParseMode")

	// Bold: *text* -> <b>text</b>
	boldRegex := regexp.MustCompile(`\*([^\*\n]+)\*`)
	res = boldRegex.ReplaceAllString(res, "<b>$1</b>")

	// Italic: _text_ -> <i>text</i>
	italicRegex := regexp.MustCompile(`_([^\_\n]+)_`)
	res = italicRegex.ReplaceAllString(res, "<i>$1</i>")

	// Code: `text` -> <code>text</code>
	codeRegex := regexp.MustCompile("`([^`\\n]+)`")
	res = codeRegex.ReplaceAllString(res, "<code>$1</code>")

	res = utils.RestoreProtectedMarkdownLinks(res, protectedLinks)

	return res
}
func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func ExtractDynamicLinks(text string) ([]dbmodels.Button, string) {
	var buttons []dbmodels.Button
	cleanText := text
	linkPattern := `(?:https?://[^\s<>")]+|tg://[^\s<>")]+|@[^\s<>")]+|(?:t\.me|telegram\.me)/[^\s<>")]+)`

	appendButton := func(name, rawURL, rawMatch string) {
		name = strings.TrimSpace(utils.RemoveHTMLTags(name))
		buttonURL := utils.NormalizeTelegramURL(rawURL)
		if name == "" || !utils.IsValidButtonURL(buttonURL) {
			return
		}

		buttons = append(buttons, dbmodels.Button{
			NameButton: name,
			ButtonURL:  buttonURL,
			PositionY:  len(buttons),
		})
		cleanText = strings.Replace(cleanText, rawMatch, "", 1)
	}

	// 1. Bang style: !Name \n !URL (at the start of a line)
	bangRegex := regexp.MustCompile(`(?m)^!\s*(.+)\s*\n\s*!\s*(` + linkPattern + `)`)
	matchesBang := bangRegex.FindAllStringSubmatch(cleanText, -1)
	for _, match := range matchesBang {
		if len(match) == 3 {
			appendButton(match[1], match[2], match[0])
		}
	}

	// 2. Markdown style: [Name](URL)
	linkRegex := regexp.MustCompile(`\[(.*?)\]\((` + linkPattern + `)\)`)
	matchesLink := linkRegex.FindAllStringSubmatch(cleanText, -1)
	for _, match := range matchesLink {
		if len(match) == 3 {
			appendButton(match[1], match[2], match[0])
		}
	}

	// 3. HTML style: <a href="URL">Name</a> (often from processed entities)
	htmlRegex := regexp.MustCompile(`<a\s+href="(` + linkPattern + `)">([^<]+)</a>`)
	matchesHTML := htmlRegex.FindAllStringSubmatch(cleanText, -1)
	for _, match := range matchesHTML {
		if len(match) == 3 {
			appendButton(match[2], match[1], match[0])
		}
	}

	return buttons, strings.TrimSpace(cleanText)
}
func convertUserTemplateButtons(buttons []dbmodels.UserCaptionTemplateButton) []dbmodels.Button {
	btns := make([]dbmodels.Button, len(buttons))
	for i, b := range buttons {
		btns[i] = dbmodels.Button{
			ButtonID:   b.ButtonID,
			NameButton: b.NameButton,
			ButtonURL:  b.ButtonURL,
			Style:      b.Style,
			PositionX:  b.PositionX,
			PositionY:  b.PositionY,
		}
	}
	return btns
}

func convertCustomButtons(cbs []dbmodels.CustomCaptionButton) []dbmodels.Button {
	btns := make([]dbmodels.Button, len(cbs))
	for i, cb := range cbs {
		btns[i] = dbmodels.Button{
			ButtonID:   cb.ButtonID,
			NameButton: cb.NameButton,
			ButtonURL:  cb.ButtonURL,
			Style:      cb.Style,
			PositionX:  cb.PositionX,
			PositionY:  cb.PositionY,
		}
	}
	return btns
}
