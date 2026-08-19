package dto

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
)

// entityDTO representa uma entidade do Telegram para desserialização.
type entityDTO struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"customEmojiId,omitempty"`
	UserID        int64  `json:"userId,omitempty"`
}

// entityEvent marca a abertura ou fechamento de uma entidade em determinada posição.
type entityEvent struct {
	pos    int
	entity entityDTO
	isOpen bool
}

func ToUserDTO(u *models.User) UserDTO {
	if u == nil {
		return UserDTO{}
	}
	dto := UserDTO{
		ID:            u.UserId,
		FirstName:     u.FirstName,
		Username:      u.Username,
		IsAdmin:       u.IsAdmin,
		IsBlacklisted: u.IsBlacklisted,
		IsContribute:  u.IsContribute,
	}

	if len(u.Channels) > 0 {
		for _, c := range u.Channels {
			dto.Channels = append(dto.Channels, ToChannelDTO(&c))
		}
	}

	return dto
}

func ToUserLookupDTO(u *models.User) UserLookupDTO {
	if u == nil {
		return UserLookupDTO{}
	}
	return UserLookupDTO{
		ID:        u.UserId,
		FirstName: u.FirstName,
		Username:  u.Username,
	}
}

func ToChannelDTO(c *models.Channel) ChannelDTO {
	if c == nil {
		return ChannelDTO{}
	}
	dto := ChannelDTO{
		ID:                     c.ID,
		Title:                  c.Title,
		NewPackCaption:         c.NewPackCaption,
		NewPackMessageButtons:  boolValueOrDefault(c.NewPackMessageButtons, true),
		NewPackStickerButtons:  boolValueOrDefault(c.NewPackStickerButtons, true),
		NewPackMessagePosition: stringValueOrDefault(c.NewPackMessagePosition, "above"),
		NewPackReplyToSticker:  boolValueOrDefault(c.NewPackReplyToSticker, false),
		InviteURL:              c.InviteURL,
		OwnerID:                c.OwnerID,
		Reactions:              c.Reactions,
		ReactionPosition:       c.ReactionPosition,
		DynamicLinks:           c.DynamicLinks,
		DLBotButtons:           c.DLBotButtons,
		DLBotCaptions:          c.DLBotCaptions,
		DLBotReactions:         c.DLBotReactions,
		NativeReactionsEnabled: c.NativeReactionsEnabled,
		NativeReactions:        c.NativeReactions,
		NativeReactionMode:     c.NativeReactionMode,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
	}

	if c.DefaultCaption != nil {
		dto.DefaultCaption = ToDefaultCaptionDTO(c.DefaultCaption)
	}

	if len(c.Buttons) > 0 {
		for _, b := range c.Buttons {
			dto.Buttons = append(dto.Buttons, ToButtonDTO(&b))
		}
	}

	if len(c.CustomCaptions) > 0 {
		for _, cc := range c.CustomCaptions {
			dto.CustomCaptions = append(dto.CustomCaptions, ToCustomCaptionDTO(&cc))
		}
	}

	return dto
}

func boolValueOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func stringValueOrDefault(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func ToDefaultCaptionDTO(dc *models.DefaultCaption) *DefaultCaptionDTO {
	dto := &DefaultCaptionDTO{
		CaptionID: dc.CaptionID,
		CreatedAt: dc.CreatedAt,
	}

	// Se entities estiverem presentes, converter para markdown
	// para exibição no dashboard (CaptionPreview espera markdown)
	if dc.UseEntities && dc.Entities != "" {
		dto.Caption = entitiesToMarkdown(dc.Caption, dc.Entities)
	} else {
		dto.Caption = dc.Caption
	}

	if dc.MessagePermission != nil {
		dto.MessagePermission = &PermissionDTO{
			LinkPreview: dc.MessagePermission.LinkPreview,
			Message:     dc.MessagePermission.Message,
			Audio:       dc.MessagePermission.Audio,
			Video:       dc.MessagePermission.Video,
			Photo:       dc.MessagePermission.Photo,
			Document:    dc.MessagePermission.Document,
			Sticker:     dc.MessagePermission.Sticker,
			GIF:         dc.MessagePermission.GIF,
			Reactions:   dc.MessagePermission.Reactions,
		}
	}

	if dc.ButtonsPermission != nil {
		dto.ButtonsPermission = &PermissionDTO{
			Message:  dc.ButtonsPermission.Message,
			Audio:    dc.ButtonsPermission.Audio,
			Video:    dc.ButtonsPermission.Video,
			Photo:    dc.ButtonsPermission.Photo,
			Document: dc.ButtonsPermission.Document,
			Sticker:  dc.ButtonsPermission.Sticker,
			GIF:      dc.ButtonsPermission.GIF,
		}
	}

	return dto
}

func ToButtonDTO(b *models.Button) ButtonDTO {
	return ButtonDTO{
		ButtonID:  b.ButtonID,
		Name:      b.NameButton,
		URL:       b.ButtonURL,
		Style:     b.Style,
		PositionX: b.PositionX,
		PositionY: b.PositionY,
	}
}

func ToCustomCaptionButtonDTO(b *models.CustomCaptionButton) ButtonDTO {
	return ButtonDTO{
		ButtonID:  b.ButtonID,
		Name:      b.NameButton,
		URL:       b.ButtonURL,
		Style:     b.Style,
		PositionX: b.PositionX,
		PositionY: b.PositionY,
	}
}

func ToCustomCaptionDTO(cc *models.CustomCaption) CustomCaptionDTO {
	dto := CustomCaptionDTO{
		CaptionID:   cc.CaptionID,
		Code:        cc.Code,
		Caption:     cc.Caption,
		LinkPreview: cc.LinkPreview,
		CreatedAt:   cc.CreatedAt,
	}

	if len(cc.Buttons) > 0 {
		for _, b := range cc.Buttons {
			dto.Buttons = append(dto.Buttons, ToCustomCaptionButtonDTO(&b))
		}
	}

	return dto
}

// ── Conversão de Telegram Entities para Markdown ──

// entitiesToMarkdown converte texto cru + entities JSON para markdown
// que o CaptionPreview do dashboard entende (**, __, ~~, ||, `, etc.)
// Usa UTF-16 code units para alinhar com os offsets do Telegram Bot API.
func entitiesToMarkdown(text string, entitiesJSON string) string {
	var entities []entityDTO
	if err := json.Unmarshal([]byte(entitiesJSON), &entities); err != nil {
		return text
	}
	if len(entities) == 0 {
		return text
	}

	// Converter para UTF-16 (Telegram usa offsets em UTF-16 code units)
	runes := []rune(text)
	utf16Units := utf16.Encode(runes)

	// Construir lista de eventos (abertura/fechamento de entidades) em UTF-16
	var events []entityEvent
	for _, e := range entities {
		start := e.Offset
		if start < 0 {
			start = 0
		}
		end := e.Offset + e.Length
		if end > len(utf16Units) {
			end = len(utf16Units)
		}
		if start >= end {
			continue
		}
		events = append(events, entityEvent{pos: start, entity: e, isOpen: true})
		events = append(events, entityEvent{pos: end, entity: e, isOpen: false})
	}

	// Ordenar por posição UTF-16
	sort.Slice(events, func(i, j int) bool {
		if events[i].pos != events[j].pos {
			return events[i].pos < events[j].pos
		}
		// Mesma posição: fechamentos antes de aberturas
		if events[i].isOpen != events[j].isOpen {
			return !events[i].isOpen && events[j].isOpen
		}
		// Ambos fechamento: entidade interna (offset maior) primeiro
		if !events[i].isOpen && !events[j].isOpen {
			return events[i].entity.Offset > events[j].entity.Offset
		}
		// Ambos abertura: entidade externa (offset menor) primeiro
		return events[i].entity.Offset < events[j].entity.Offset
	})

	var result strings.Builder
	utf16Pos := 0
	activeStack := make([]entityDTO, 0) // pilha de entidades abertas

	for _, evt := range events {
		// Escrever texto da posição UTF-16 atual até o evento
		if evt.pos > utf16Pos {
			// Se estivermos dentro de um custom_emoji, não escreve o
			// caractere placeholder original — só um espaço
			if isInsideCustomEmoji(activeStack) {
				result.WriteString(" ")
			} else {
				segment := string(utf16.Decode(utf16Units[utf16Pos:evt.pos]))
				result.WriteString(segment)
			}
			utf16Pos = evt.pos
		}

		if evt.isOpen {
			activeStack = append(activeStack, evt.entity)
			result.WriteString(openTag(evt.entity))
		} else {
			// Fechar entidade: procurar na pilha (LIFO)
			for i := len(activeStack) - 1; i >= 0; i-- {
				e := activeStack[i]
				if e.Offset == evt.entity.Offset && e.Offset+e.Length == evt.entity.Offset+evt.entity.Length {
					result.WriteString(closeTag(e))
					activeStack = append(activeStack[:i], activeStack[i+1:]...)
					break
				}
			}
		}
	}

	// Texto restante
	if utf16Pos < len(utf16Units) {
		result.WriteString(string(utf16.Decode(utf16Units[utf16Pos:])))
	}

	return result.String()
}

func openTag(e entityDTO) string {
	switch e.Type {
	case "bold":
		return "**"
	case "italic":
		return "__"
	case "underline":
		return "<u>"
	case "strikethrough":
		return "~~"
	case "spoiler":
		return "||"
	case "code":
		return "`"
	case "pre":
		return "```"
	case "text_link":
		return "["
	case "custom_emoji":
		return fmt.Sprintf(`<tg-emoji emoji-id="%s">`, e.CustomEmojiID)
	case "blockquote":
		return ""
	default:
		return ""
	}
}

func closeTag(e entityDTO) string {
	switch e.Type {
	case "bold":
		return "**"
	case "italic":
		return "__"
	case "underline":
		return "</u>"
	case "strikethrough":
		return "~~"
	case "spoiler":
		return "||"
	case "code":
		return "`"
	case "pre":
		return "```"
	case "text_link":
		if strings.HasPrefix(e.URL, "http://") || strings.HasPrefix(e.URL, "https://") {
			return fmt.Sprintf("](%s)", e.URL)
		}
		return "]()"
	case "custom_emoji":
		return "</tg-emoji>"
	case "blockquote":
		return ""
	default:
		return ""
	}
}

// isInsideCustomEmoji verifica se há alguma entidade custom_emoji ativa na pilha.
func isInsideCustomEmoji(stack []entityDTO) bool {
	for _, e := range stack {
		if e.Type == "custom_emoji" {
			return true
		}
	}
	return false
}
