package channelpost

import (
	"sync"
	"time"
)

type MediaGroupTelego struct {
	Messages           []MediaMessageTelego
	Processed          bool
	Timer              *time.Timer
	MessageEditAllowed bool
	ChatID             int64
	mu                 sync.Mutex
}

type MediaGroupManagerTelego struct {
	groups          sync.Map // string -> *MediaGroupTelego
	processedGroups sync.Map // string -> ProcessedGroup
	newPackChannels sync.Map // int64 -> bool
}

var globalMediaGroupManagerTelego *MediaGroupManagerTelego
var onceTelego sync.Once

func GetMediaGroupManagerTelego() *MediaGroupManagerTelego {
	onceTelego.Do(func() {
		globalMediaGroupManagerTelego = &MediaGroupManagerTelego{}
		go globalMediaGroupManagerTelego.cleanupRoutine()
	})
	return globalMediaGroupManagerTelego
}

func (mgm *MediaGroupManagerTelego) cleanupRoutine() {
	ticker := time.NewTicker(CleanupTimeout)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		mgm.processedGroups.Range(func(key, value interface{}) bool {
			if group, ok := value.(ProcessedGroup); ok {
				if now.Sub(group.Timestamp) > CleanupTimeout {
					mgm.processedGroups.Delete(key)
				}
			}
			return true
		})
	}
}

func (mgm *MediaGroupManagerTelego) GetMediaGroup(groupID string) (*MediaGroupTelego, bool) {
	if value, ok := mgm.groups.Load(groupID); ok {
		return value.(*MediaGroupTelego), true
	}
	return nil, false
}

func (mgm *MediaGroupManagerTelego) SetMediaGroup(groupID string, group *MediaGroupTelego) {
	mgm.groups.Store(groupID, group)
}

func (mgm *MediaGroupManagerTelego) DeleteMediaGroup(groupID string) {
	mgm.groups.Delete(groupID)
}

func (mgm *MediaGroupManagerTelego) IsProcessed(groupID string) bool {
	_, exists := mgm.processedGroups.Load(groupID)
	return exists
}

func (mgm *MediaGroupManagerTelego) MarkProcessed(groupID string) {
	mgm.processedGroups.Store(groupID, ProcessedGroup{Timestamp: time.Now()})
}

func (mgm *MediaGroupManagerTelego) IsNewPackActive(channelID int64) bool {
	value, exists := mgm.newPackChannels.Load(channelID)
	return exists && value.(bool)
}

func (mgm *MediaGroupManagerTelego) SetNewPackActive(channelID int64, active bool) {
	if active {
		mgm.newPackChannels.Store(channelID, true)
	} else {
		mgm.newPackChannels.Delete(channelID)
	}
}
