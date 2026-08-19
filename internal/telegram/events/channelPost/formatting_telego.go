package channelpost

import (
	"html"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/mymmrac/telego"
)

// ProcessTextWithFormattingTelego is the telego version of ProcessTextWithFormatting
func ProcessTextWithFormattingTelego(text string, entities []telego.MessageEntity) string {
	if text == "" {
		return ""
	}

	hasManualEntities := false
	if len(entities) > 0 {
		for _, e := range entities {
			// Ignora entidades automáticas na verificação
			if e.Type != "url" && e.Type != "email" && e.Type != "phone_number" && e.Type != "hashtag" && e.Type != "cashtag" && e.Type != "bot_command" {
				hasManualEntities = true
				break
			}
		}
	}

	var result string
	// Se o usuário formatou algo explicitamente via UI do Telegram (negrito, itálico, links embutidos, etc)
	if hasManualEntities {
		result = ProcessEntitiesOnlyTelego(text, entities)
	} else if IsMarkdown(text) {
		// Se usou sintaxe Markdown crua (ex: [texto](url))
		result = DetectParseMode(text)
	} else if len(entities) > 0 {
		// Fallback para entidades automáticas (URLs puras que o Telegram detectou sozinho)
		result = ProcessEntitiesOnlyTelego(text, entities)
	} else {
		result = DetectParseMode(text)
	}

	return result
}

// ProcessEntitiesOnlyTelego is the telego version of ProcessEntitiesOnly
func ProcessEntitiesOnlyTelego(text string, entities []telego.MessageEntity) string {
	if text == "" {
		return ""
	}
	if len(entities) == 0 {
		return html.EscapeString(text)
	}

	r := []rune(text)
	u := utf16.Encode(r)

	type ent struct {
		t     string
		o     int
		end   int
		url   string
		lang  string
		emoji string
		uid   int64
	}

	es := make([]ent, 0, len(entities))
	for _, e := range entities {
		o := e.Offset
		end := e.Offset + e.Length
		if o < 0 {
			o = 0
		}
		if end > len(u) {
			end = len(u)
		}
		if o >= end {
			continue
		}
		var uid int64
		if e.User != nil {
			uid = e.User.ID
		}
		es = append(es, ent{
			t:     e.Type,
			o:     o,
			end:   end,
			url:   e.URL,
			lang:  e.Language,
			emoji: e.CustomEmojiID,
			uid:   uid,
		})
	}
	if len(es) == 0 {
		return html.EscapeString(text)
	}

	bset := map[int]struct{}{0: {}, len(u): {}}
	for _, e := range es {
		bset[e.o] = struct{}{}
		bset[e.end] = struct{}{}
	}
	bounds := make([]int, 0, len(bset))
	for b := range bset {
		bounds = append(bounds, b)
	}
	sort.Ints(bounds)

	pri := func(t string) int {
		switch t {
		case "code", "pre":
			return 0
		case "blockquote":
			return 1
		case "text_link", "text_mention":
			return 2
		case "url", "email", "phone_number":
			return 2
		case "bold":
			return 3
		case "italic":
			return 4
		case "underline":
			return 5
		case "strikethrough":
			return 6
		case "spoiler":
			return 7
		case "custom_emoji":
			return 8
		default:
			return 9
		}
	}

	spanHTML := func(seg string, active []ent) string {
		for _, a := range active {
			if a.t == "code" {
				return "<code>" + html.EscapeString(seg) + "</code>"
			}
			if a.t == "pre" {
				langClass := ""
				if a.lang != "" {
					langClass = ` class="language-` + html.EscapeString(a.lang) + `"`
				}
				return "<pre><code" + langClass + ">" + html.EscapeString(seg) + "</code></pre>"
			}
		}

		inner := html.EscapeString(seg)

		for _, a := range active {
			if a.t == "custom_emoji" && a.emoji != "" {
				inner = `<tg-emoji emoji-id="` + html.EscapeString(a.emoji) + `">` + inner + "</tg-emoji>"
				break
			}
		}

		wrapIf := func(t, open, close string) {
			for _, a := range active {
				if a.t == t {
					inner = open + inner + close
					break
				}
			}
		}
		wrapIf("bold", "<b>", "</b>")
		wrapIf("italic", "<i>", "</i>")
		wrapIf("underline", "<u>", "</u>")
		wrapIf("strikethrough", "<s>", "</s>")
		for _, a := range active {
			if a.t == "spoiler" {
				inner = "<tg-spoiler>" + inner + "</tg-spoiler>"
				break
			}
		}

		for _, a := range active {
			if a.t == "text_link" && a.url != "" {
				inner = `<a href="` + html.EscapeString(a.url) + `">` + inner + "</a>"
				break
			}
		}
		for _, a := range active {
			if a.t == "text_mention" && a.uid != 0 {
				inner = `<a href="tg://user?id=` + html.EscapeString(int64ToStr(a.uid)) + `">` + inner + "</a>"
				break
			}
		}
		for _, a := range active {
			switch a.t {
			case "url":
				inner = `<a href="` + inner + `">` + inner + "</a>"
			case "email":
				inner = `<a href="mailto:` + inner + `">` + inner + "</a>"
			case "phone_number":
				inner = `<a href="tel:` + inner + `">` + inner + "</a>"
			}
		}

		for _, a := range active {
			if a.t == "blockquote" {
				inner = "<blockquote>" + inner + "</blockquote>"
				break
			}
		}

		return inner
	}

	var b strings.Builder
	for i := 0; i < len(bounds)-1; i++ {
		s, e := bounds[i], bounds[i+1]
		if e <= s {
			continue
		}
		seg := string(utf16.Decode(u[s:e]))

		active := make([]ent, 0, 6)
		for _, en := range es {
			if en.o <= s && en.end >= e {
				active = append(active, en)
			}
		}
		if len(active) == 0 {
			b.WriteString(html.EscapeString(seg))
			continue
		}

		sort.SliceStable(active, func(i, j int) bool {
			pi, pj := pri(active[i].t), pri(active[j].t)
			if pi != pj {
				return pi < pj
			}
			di, dj := (active[i].end - active[i].o), (active[j].end - active[j].o)
			if di != dj {
				return di > dj
			}
			if active[i].o != active[j].o {
				return active[i].o < active[j].o
			}
			return active[i].t < active[j].t
		})

		b.WriteString(spanHTML(seg, active))
	}
	return b.String()
}
