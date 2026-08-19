package executor

import (
	"encoding/json"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/mymmrac/telego"
)

// MessageEntityDTO representa uma entidade do Telegram serializavel via JSON.
// Mesma estrutura usada no handler de captions (my_channel/caption.go).
type MessageEntityDTO struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"customEmojiId,omitempty"`
	UserID        int64  `json:"userId,omitempty"`
}

// entitiesJSONToGotd converte um JSON de MessageEntityDTOs para uma slice
// de tipos gotd/td MessageEntityClass.
func entitiesJSONToGotd(entitiesJSON string) ([]tg.MessageEntityClass, error) {
	if entitiesJSON == "" {
		return nil, nil
	}

	var dtos []MessageEntityDTO
	if err := json.Unmarshal([]byte(entitiesJSON), &dtos); err != nil {
		return nil, fmt.Errorf("unmarshal entities JSON: %w", err)
	}

	entities := make([]tg.MessageEntityClass, 0, len(dtos))
	for _, dto := range dtos {
		entity, err := dtoToMessageEntity(dto)
		if err != nil {
			return nil, fmt.Errorf("convert entity at offset %d: %w", dto.Offset, err)
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// dtoToMessageEntity converte um MessageEntityDTO para o tipo gotd/td correspondente.
func dtoToMessageEntity(dto MessageEntityDTO) (tg.MessageEntityClass, error) {
	switch dto.Type {
	case "bold":
		return &tg.MessageEntityBold{Offset: dto.Offset, Length: dto.Length}, nil
	case "italic":
		return &tg.MessageEntityItalic{Offset: dto.Offset, Length: dto.Length}, nil
	case "underline":
		return &tg.MessageEntityUnderline{Offset: dto.Offset, Length: dto.Length}, nil
	case "strikethrough":
		return &tg.MessageEntityStrike{Offset: dto.Offset, Length: dto.Length}, nil
	case "spoiler":
		return &tg.MessageEntitySpoiler{Offset: dto.Offset, Length: dto.Length}, nil
	case "code":
		return &tg.MessageEntityCode{Offset: dto.Offset, Length: dto.Length}, nil
	case "pre":
		return &tg.MessageEntityPre{Offset: dto.Offset, Length: dto.Length, Language: dto.Language}, nil
	case "text_link":
		return &tg.MessageEntityTextURL{Offset: dto.Offset, Length: dto.Length, URL: dto.URL}, nil
	case "text_mention":
		return &tg.MessageEntityMentionName{Offset: dto.Offset, Length: dto.Length, UserID: dto.UserID}, nil
	case "url":
		return &tg.MessageEntityURL{Offset: dto.Offset, Length: dto.Length}, nil
	case "email":
		return &tg.MessageEntityEmail{Offset: dto.Offset, Length: dto.Length}, nil
	case "phone":
		return &tg.MessageEntityPhone{Offset: dto.Offset, Length: dto.Length}, nil
	case "mention":
		return &tg.MessageEntityMention{Offset: dto.Offset, Length: dto.Length}, nil
	case "hashtag":
		return &tg.MessageEntityHashtag{Offset: dto.Offset, Length: dto.Length}, nil
	case "cashtag":
		return &tg.MessageEntityCashtag{Offset: dto.Offset, Length: dto.Length}, nil
	case "bot_command":
		return &tg.MessageEntityBotCommand{Offset: dto.Offset, Length: dto.Length}, nil
	case "blockquote":
		return &tg.MessageEntityBlockquote{Offset: dto.Offset, Length: dto.Length}, nil
	case "expandable_blockquote":
		return &tg.MessageEntityBlockquote{Offset: dto.Offset, Length: dto.Length, Collapsed: false}, nil
	case "custom_emoji":
		documentID := parseInt64(dto.CustomEmojiID)
		return &tg.MessageEntityCustomEmoji{Offset: dto.Offset, Length: dto.Length, DocumentID: documentID}, nil
	case "bank_card":
		return &tg.MessageEntityBankCard{Offset: dto.Offset, Length: dto.Length}, nil
	default:
		// Fallback para entidade desconhecida
		return &tg.MessageEntityUnknown{Offset: dto.Offset, Length: dto.Length}, nil
	}
}

// parseInt64 tenta converter string para int64.
// Usado para campos como CustomEmojiID que vem como string no DTO.
func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			break
		}
	}
	return n
}

// entitiesJSONToTelego converte um JSON de MessageEntityDTOs para uma slice
// de telego.MessageEntity (usado na Bot API).
func entitiesJSONToTelego(entitiesJSON string) ([]telego.MessageEntity, error) {
	if entitiesJSON == "" {
		return nil, nil
	}

	var dtos []MessageEntityDTO
	if err := json.Unmarshal([]byte(entitiesJSON), &dtos); err != nil {
		return nil, fmt.Errorf("unmarshal entities JSON: %w", err)
	}

	entities := make([]telego.MessageEntity, 0, len(dtos))
	for _, dto := range dtos {
		entity := telego.MessageEntity{
			Type:     dto.Type,
			Offset:   dto.Offset,
			Length:   dto.Length,
			URL:      dto.URL,
			Language: dto.Language,
		}
		if dto.CustomEmojiID != "" {
			entity.CustomEmojiID = dto.CustomEmojiID
		}
		entities = append(entities, entity)
	}

	return entities, nil
}
