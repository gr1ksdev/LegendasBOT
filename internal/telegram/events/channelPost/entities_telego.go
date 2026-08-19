package channelpost

import (
	"encoding/json"

	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

// messageEntityDTO representa uma entidade do Telegram serializavel via JSON.
// Duplicado do handler de captions (my_channel/caption.go) para evitar
// dependencia circular.
type messageEntityDTO struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"customEmojiId,omitempty"`
	UserID        int64  `json:"userId,omitempty"`
}

// entitiesToJSON converte uma slice de telego.MessageEntity para JSON.
func entitiesToJSON(entities []telego.MessageEntity) string {
	if len(entities) == 0 {
		return ""
	}

	dtos := make([]messageEntityDTO, 0, len(entities))
	for _, e := range entities {
		dto := messageEntityDTO{
			Type:          e.Type,
			Offset:        e.Offset,
			Length:        e.Length,
			URL:           e.URL,
			Language:      e.Language,
			CustomEmojiID: e.CustomEmojiID,
		}
		if e.User != nil {
			dto.UserID = e.User.ID
		}
		dtos = append(dtos, dto)
	}

	data, err := json.Marshal(dtos)
	if err != nil {
		return ""
	}
	return string(data)
}

// combineEntityJSONs combina entities do post (telego) com entities da caption
// (JSON DTOs), ajustando os offsets das entities da caption pelo shift.
// Retorna o JSON combinado.
func combineEntityJSONs(postEntities []telego.MessageEntity, captionEntitiesJSON string, shift int) string {
	// Converter post entities para DTOs
	postDTOS := make([]messageEntityDTO, 0, len(postEntities))
	for _, e := range postEntities {
		dto := messageEntityDTO{
			Type:          e.Type,
			Offset:        e.Offset,
			Length:        e.Length,
			URL:           e.URL,
			Language:      e.Language,
			CustomEmojiID: e.CustomEmojiID,
		}
		if e.User != nil {
			dto.UserID = e.User.ID
		}
		postDTOS = append(postDTOS, dto)
	}

	// Converter caption entities de JSON para DTOs
	var captionDTOS []messageEntityDTO
	if captionEntitiesJSON != "" {
		if err := json.Unmarshal([]byte(captionEntitiesJSON), &captionDTOS); err != nil {
			logger.Warn("CHANNELPOST", "erro ao dar unmarshal nas caption entities: %v", err)
		}
	}

	// Combinar, ajustando offsets das caption entities
	combined := make([]messageEntityDTO, 0, len(postDTOS)+len(captionDTOS))
	combined = append(combined, postDTOS...)
	for _, dto := range captionDTOS {
		dto.Offset += shift
		combined = append(combined, dto)
	}

	if len(combined) == 0 {
		return ""
	}

	data, err := json.Marshal(combined)
	if err != nil {
		return ""
	}
	return string(data)
}
