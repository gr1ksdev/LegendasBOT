package channelpost

import (
	"strings"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/mymmrac/telego"
)

func CreateInlineKeyboardTelego(buttons []dbmodels.Button, customCaption *dbmodels.CustomCaption, channel *dbmodels.Channel, messageType MessageType) *telego.InlineKeyboardMarkup {
	var finalButtons []dbmodels.Button
	pm := GetPermissionManager()

	if customCaption != nil && len(customCaption.Buttons) > 0 {
		for _, cb := range customCaption.Buttons {
			finalButtons = append(finalButtons, dbmodels.Button{
				NameButton: cb.NameButton,
				ButtonURL:  cb.ButtonURL,
				Style:      cb.Style,
				PositionY:  cb.PositionY,
				PositionX:  cb.PositionX,
			})
		}
	} else {
		perms := pm.CheckPermissions(channel, messageType)
		if !perms.CanAddButtons {
			return nil
		}
		finalButtons = buttons
	}

	if len(finalButtons) == 0 && (channel == nil || channel.Reactions == "") {
		return nil
	}

	rows := map[int][]telego.InlineKeyboardButton{}

	for _, b := range finalButtons {
		buttonURL := utils.NormalizeTelegramURL(b.ButtonURL)
		if b.NameButton == "" || buttonURL == "" || !utils.IsValidButtonURL(buttonURL) {
			continue
		}
		row := b.PositionY
		if row < 0 {
			row = 0
		}
		btn := telego.InlineKeyboardButton{Text: b.NameButton, URL: buttonURL, Style: b.Style}
		rows[row] = append(rows[row], btn)
	}

	perms := pm.CheckPermissions(channel, messageType)
	if channel != nil && channel.Reactions != "" && perms.CanAddReactions {
		reactions := strings.Split(channel.Reactions, ",")
		var reactionRow []telego.InlineKeyboardButton
		for _, r := range reactions {
			emoji := strings.TrimSpace(r)
			if emoji != "" {
				reactionRow = append(reactionRow, telego.InlineKeyboardButton{
					Text:         emoji,
					CallbackData: "vote:" + emoji,
				})
			}
		}
		if len(reactionRow) > 0 {
			row := channel.ReactionPosition
			if row < 0 {
				row = 0
			}

			hasConflict := false
			maxBtnRow := -1
			for _, b := range finalButtons {
				if b.PositionY > maxBtnRow {
					maxBtnRow = b.PositionY
				}
				if b.PositionY == row {
					hasConflict = true
				}
			}

			if hasConflict {
				row = maxBtnRow + 1
			}

			rows[row] = append(rows[row], reactionRow...)
		}
	}

	keyboard := make([][]telego.InlineKeyboardButton, 0)
	maxRow := -1
	for r := range rows {
		if r > maxRow {
			maxRow = r
		}
	}

	for r := 0; r <= maxRow; r++ {
		if line, ok := rows[r]; ok && len(line) > 0 {
			keyboard = append(keyboard, line)
		}
	}

	if len(keyboard) == 0 {
		return nil
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}
