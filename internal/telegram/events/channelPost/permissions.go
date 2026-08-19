package channelpost

import (
	"fmt"
	"sync"
	"time"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type PermissionManager struct {
	cache sync.Map // map[string]*PermissionCheckResult
}

var globalPermissionManager *PermissionManager
var oncePM sync.Once

func GetPermissionManager() *PermissionManager {
	oncePM.Do(func() {
		globalPermissionManager = &PermissionManager{}
	})
	return globalPermissionManager
}

func (pm *PermissionManager) CheckPermissions(channel *dbmodels.Channel, messageType MessageType) *PermissionCheckResult {
	cacheKey := fmt.Sprintf("%d:%s:%d", channel.ID, messageType, channel.TokenVersion)

	if val, ok := pm.cache.Load(cacheKey); ok {
		res := val.(*PermissionCheckResult)
		return res
	}

	result := pm.computePermissions(channel, messageType)
	pm.cache.Store(cacheKey, result)

	// Iniciar cleanup assíncrono para esta entrada
	time.AfterFunc(CacheTTL, func() {
		pm.cache.Delete(cacheKey)
	})

	return result
}

func (pm *PermissionManager) computePermissions(channel *dbmodels.Channel, messageType MessageType) *PermissionCheckResult {
	if channel == nil {
		return &PermissionCheckResult{Reason: "Canal nulo"}
	}

	// 1. Verificar permissões de mensagem (Edição/Legenda)
	canEdit := true
	useLinkPreview := true
	canAddReactions := true

	if channel.DefaultCaption != nil && channel.DefaultCaption.MessagePermission != nil {
		perm := channel.DefaultCaption.MessagePermission
		canAddReactions = perm.Reactions

		switch messageType {
		case MessageTypeText:
			canEdit = perm.Message
			useLinkPreview = perm.LinkPreview
		case MessageTypeAudio:
			canEdit = perm.Audio
		case MessageTypeVideo:
			canEdit = perm.Video
		case MessageTypePhoto:
			canEdit = perm.Photo
		case MessageTypeDocument:
			canEdit = perm.Document
		case MessageTypeSticker:
			canEdit = perm.Sticker
		case MessageTypeAnimation:
			canEdit = perm.GIF
		}
	}

	// 2. Verificar permissões de botões
	canAddButtons := true
	if channel.DefaultCaption != nil && channel.DefaultCaption.ButtonsPermission != nil {
		perm := channel.DefaultCaption.ButtonsPermission
		switch messageType {
		case MessageTypeText:
			canAddButtons = perm.Message
		case MessageTypeAudio:
			canAddButtons = perm.Audio
		case MessageTypeVideo:
			canAddButtons = perm.Video
		case MessageTypePhoto:
			canAddButtons = perm.Photo
		case MessageTypeDocument:
			canAddButtons = perm.Document
		case MessageTypeSticker:
			canAddButtons = perm.Sticker
		case MessageTypeAnimation:
			canAddButtons = perm.GIF
		}
	}

	return &PermissionCheckResult{
		CanEdit:           canEdit,
		CanAddButtons:     canAddButtons,
		CanEditButtons:    canAddButtons, // Por enquanto igual
		CanAddReactions:   canAddReactions,
		CanUseLinkPreview: useLinkPreview,
	}
}

func (pm *PermissionManager) InvalidateCache(channelID int64) {
	// Como a chave contém o TokenVersion, basta incrementar o TokenVersion no banco
	// Mas podemos limpar prefixos se necessário.
	logger.Bot("Invalidating permission cache for %d", channelID)
}
