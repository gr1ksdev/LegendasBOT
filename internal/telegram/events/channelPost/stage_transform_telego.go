package channelpost

import (
	"context"
	"strings"
	"unicode/utf16"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

func StageTransformTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		post := pCtx.Update.ChannelPost
		if post == nil {
			return nil
		}

		// 1. Determine original text and entities
		var baseText string
		var entities []telego.MessageEntity
		if pCtx.IsMediaGroup {
			for _, m := range pCtx.GroupMessages {
				if m.HasCaption {
					baseText = m.Caption
					entities = m.CaptionEntities
					break
				}
			}
		} else {
			if pCtx.MessageType == MessageTypeText {
				baseText = post.Text
				entities = post.Entities
			} else {
				baseText = post.Caption
				entities = post.CaptionEntities
			}
		}
		pCtx.OriginalCaption = baseText

		// Store post entities as JSON for entity-based caption combination (MTProto path)
		pCtx.PostEntitiesJSON = entitiesToJSON(entities)

		// 2. Format base text
		formattedBase := ProcessTextWithFormattingTelego(baseText, entities)

		// 2.1 Dynamic Links
		extractedDynLinks := false
		if pCtx.Channel.DynamicLinks {
			dynButtons, cleanBase := ExtractDynamicLinks(formattedBase)
			if len(dynButtons) > 0 {
				logger.Bot("🔗 Extraídos %d botões dinâmicos do conteúdo original", len(dynButtons))
				recordChannelPostEvent(c, pCtx, "dynamic_links_extracted", services.ChannelEventStatusInfo, map[string]any{"count": len(dynButtons)}, nil)
				formattedBase = cleanBase
				extractedDynLinks = true
				pCtx.FinalButtons = append(pCtx.FinalButtons, dynButtons...)
			}
		}

		// 3. Extract user template prefix (!prefix) or channel custom caption (#hashtag)
		prefix := extractPrefix(formattedBase)
		hashtag := extractHashtag(formattedBase)
		var dbCaption string
		var finalButtons []dbmodels.Button = pCtx.Channel.Buttons
		var custom *dbmodels.CustomCaption

		if extractedDynLinks && !pCtx.Channel.DLBotButtons {
			finalButtons = []dbmodels.Button{}
		}

		// 3a. Try user-level template (!prefix) first
		userTplFound := false
		if prefix != "" {
			userTpl, _ := c.UserCaptionTemplateService.GetByUserAndCode(context.Background(), pCtx.Channel.OwnerID, prefix)
			if userTpl != nil {
				userTplFound = true
				formattedBase = removePrefix(formattedBase, prefix)
				dbCaption = DetectParseMode(userTpl.Caption)

				if len(userTpl.Buttons) > 0 {
					finalButtons = convertUserTemplateButtons(userTpl.Buttons)
				}
			}
		}

		// 3b. Fallback to channel custom caption (#hashtag)
		if !userTplFound && hashtag != "" {
			custom = findCustomCaption(pCtx.Channel, hashtag)
			if custom != nil {
				formattedBase = removeHashtag(formattedBase, hashtag)
				dbCaption = DetectParseMode(custom.Caption)

				if len(custom.Buttons) > 0 {
					finalButtons = convertCustomButtons(custom.Buttons)
				}

				if pCtx.MessageType == MessageTypeText && !custom.LinkPreview {
					pCtx.DisableLinkPreview = true
				}
			}
		}

		// 4. Fallback to Default
		useEntities := false
		if !userTplFound && custom == nil && pCtx.Channel.DefaultCaption != nil {
			if pCtx.Channel.DefaultCaption.UseEntities && pCtx.Channel.DefaultCaption.Entities != "" {
				// Verificar se o usuario tem conta conectada para usar entities
				ownerID := pCtx.Channel.OwnerID
				if ownerID != 0 && c.ConnectedAccountService.HasActiveAccount(context.Background(), ownerID) {
					useEntities = true
					dbCaption = pCtx.Channel.DefaultCaption.Caption // raw text (nao HTML)
				} else {
					// Fallback para HTML se nao tiver conta conectada
					dbCaption = DetectParseMode(pCtx.Channel.DefaultCaption.Caption)
				}
			} else {
				dbCaption = DetectParseMode(pCtx.Channel.DefaultCaption.Caption)
			}
		}

		if extractedDynLinks && !pCtx.Channel.DLBotCaptions {
			dbCaption = ""
			useEntities = false
		}

		// Se o post original nao tem texto/legenda, remover \n iniciais da dbCaption
		// (o usuario colocou \n\n para separar da legenda original, mas sem ela fica feio)
		if pCtx.OriginalCaption == "" && dbCaption != "" {
			dbCaption = strings.TrimLeft(dbCaption, "\n")
		}

		// 5. Final Assembly
		if useEntities && dbCaption != "" {
			if pCtx.MessageType == MessageTypeAudio {
				// Audio: substitui caption completamente (so a configurada, sem texto original)
				// Usa as entities da configuracao diretamente (sem combinar com as do post original)
				pCtx.FormattedText = dbCaption
				pCtx.FinalEntities = pCtx.Channel.DefaultCaption.Entities
			} else {
				// Path MTProto com entities: compor texto cru + entities combinadas
				rawPostText := pCtx.OriginalCaption
				var shift int
				var combinedText string
				if rawPostText != "" {
					// So adiciona separador \n\n quando ha texto original do usuario
					// Senao fica "\n\nlegenda" — feio
					separator := "\n\n"
					combinedText = rawPostText + separator + dbCaption
					shift = len(utf16.Encode([]rune(rawPostText))) + len(utf16.Encode([]rune(separator)))
				} else {
					combinedText = dbCaption
					shift = 0
				}
				pCtx.FinalEntities = combineEntityJSONs(entities, pCtx.Channel.DefaultCaption.Entities, shift)
				pCtx.FormattedText = combinedText
			}
		} else if pCtx.MessageType == MessageTypeText {
			pCtx.FormattedText = composeMessage(formattedBase, dbCaption, "\n\n", "append")
		} else if pCtx.MessageType == MessageTypeAudio {
			if dbCaption != "" {
				pCtx.FormattedText = dbCaption
			} else {
				pCtx.FormattedText = formattedBase
			}
		} else {
			if dbCaption != "" {
				pCtx.FormattedText = composeMessage(formattedBase, dbCaption, "\n\n", "append")
			} else {
				pCtx.FormattedText = formattedBase
			}
		}

		pCtx.FinalButtons = append(finalButtons, pCtx.FinalButtons...)

		if dbCaption != "" {
			recordChannelPostEvent(c, pCtx, "caption_applied", services.ChannelEventStatusInfo, map[string]any{"custom_caption": custom != nil, "user_template": userTplFound, "message_type": pCtx.MessageType}, nil)
		}

		if extractedDynLinks && !pCtx.Channel.DLBotReactions {
			pCtx.Permissions.CanAddReactions = false
		}

		if !pCtx.Permissions.CanUseLinkPreview {
			pCtx.DisableLinkPreview = true
		}

		return nil
	}
}
