package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// BrazilTZ returns the Brazil timezone (America/Sao_Paulo).
// Brazil has not observed DST since 2019, so this is effectively fixed UTC-3.
func BrazilTZ() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Fallback: fixed UTC-3
		return time.FixedZone("BRT", -3*60*60)
	}
	return loc
}

func GenerateRSAKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func NormalizeTelegramURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}

	lower := strings.ToLower(u)
	switch {
	case strings.HasPrefix(u, "@"):
		return "https://t.me/" + strings.TrimLeft(strings.TrimSpace(u[1:]), "/")
	case strings.HasPrefix(lower, "t.me/"):
		return "https://t.me/" + strings.TrimLeft(strings.TrimSpace(u[5:]), "/")
	case strings.HasPrefix(lower, "telegram.me/"):
		return "https://t.me/" + strings.TrimLeft(strings.TrimSpace(u[len("telegram.me/"):]), "/")
	case !strings.Contains(u, "://") && strings.Contains(u, "."):
		return "https://" + u
	default:
		return u
	}
}

func IsValidButtonURL(raw string) bool {
	u := NormalizeTelegramURL(raw)
	if u == "" {
		return false
	}

	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme == "" {
		return false
	}

	if parsed.Scheme == "tg" {
		return true
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if (parsed.Host == "t.me" || parsed.Host == "telegram.me") && strings.Trim(parsed.Path, "/") == "" {
		return false
	}

	return true
}

var (
	htmlTagRegex      = regexp.MustCompile(`<[^>]*>`)
	markdownLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func RemoveHTMLTags(input string) string {
	return htmlTagRegex.ReplaceAllString(input, "")
}

func HasMarkdownLink(text string) bool {
	return markdownLinkRegex.MatchString(text)
}

func markdownLinkHTML(label, rawURL, source string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	normalizedURL := NormalizeTelegramURL(rawURL)
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(normalizedURL), label)
}

func NormalizeMarkdownLinks(text, source string) string {
	if text == "" {
		return ""
	}

	return markdownLinkRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := markdownLinkRegex.FindStringSubmatch(match)
		if len(matches) != 3 {
			return match
		}

		linkHTML := markdownLinkHTML(matches[1], matches[2], source)
		if linkHTML == "" {
			return match
		}
		return linkHTML
	})
}

func ProtectMarkdownLinks(text, source string) (string, map[string]string) {
	links := make(map[string]string)
	if text == "" {
		return text, links
	}

	idx := 0
	protected := markdownLinkRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := markdownLinkRegex.FindStringSubmatch(match)
		if len(matches) != 3 {
			return match
		}

		linkHTML := markdownLinkHTML(matches[1], matches[2], source)
		if linkHTML == "" {
			return match
		}

		placeholder := fmt.Sprintf("FBMDLINKTOKEN%dTOKEN", idx)
		idx++
		links[placeholder] = linkHTML
		return placeholder
	})

	return protected, links
}

func RestoreProtectedMarkdownLinks(text string, links map[string]string) string {
	for placeholder, linkHTML := range links {
		text = strings.ReplaceAll(text, placeholder, linkHTML)
	}
	return text
}

func NormalizePort(p string) string {
	if p == "" {
		return ":7000"
	}
	if !strings.HasPrefix(p, ":") {
		return ":" + p
	}
	return p
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func MarkdownToTelegramHTML(text string) string {
	// Bloco de código ``` \n texto \n ``` (processado primeiro para evitar conflitos)
	blockRegex := regexp.MustCompile("```([\\s\\S]*?)```")
	text = blockRegex.ReplaceAllString(text, "<pre><code>$1</code></pre>")

	// Código monoespaçado `texto`
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllString(text, "<code>$1</code>")

	// Negrito **texto**
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = boldRegex.ReplaceAllString(text, "<b>$1</b>")

	// Itálico __texto__
	italicRegex := regexp.MustCompile(`__(.*?)__`)
	text = italicRegex.ReplaceAllString(text, "<i>$1</i>")

	// Tachado ~~texto~~
	strikeRegex := regexp.MustCompile(`~~(.*?)~~`)
	text = strikeRegex.ReplaceAllString(text, "<s>$1</s>")

	// Spoiler ||texto||
	spoilerRegex := regexp.MustCompile(`\|\|(.*?)\|\|`)
	text = spoilerRegex.ReplaceAllString(text, "<tg-spoiler>$1</tg-spoiler>")

	// Sublinhado <u>texto</u> já é HTML, mantemos

	// Links [texto](url)
	text = NormalizeMarkdownLinks(text, "utils.MarkdownToTelegramHTML")

	return text
}

func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-2:]
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
