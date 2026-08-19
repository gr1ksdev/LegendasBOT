package cache

import "context"

type SessionManager struct {
	cache *Service
}

func NewSessionManager(cache *Service) *SessionManager {
	return &SessionManager{
		cache: cache,
	}
}

func (sm *SessionManager) CreateChannelSession(ctx context.Context, channelID, ownerID int64, title string) (*Session, error) {
	payload := ChannelPayload{
		ChannelID: channelID,
		Title:     title,
		OwnerID:   ownerID,
	}

	return sm.cache.CreateSession(ctx, payload)
}

func (sm *SessionManager) CreateClaimSession(ctx context.Context, channelID, ownerID, newOwnerId int64) (*Session, error) {
	payload := ChannelPayload{
		ChannelID:  channelID,
		OwnerID:    ownerID,
		NewOwnerID: newOwnerId,
	}

	return sm.cache.CreateSession(ctx, payload)
}

func (sm *SessionManager) GetChannelSession(ctx context.Context, key string) (*ChannelPayload, error) {
	return sm.cache.GetSession(ctx, key)
}

func (sm *SessionManager) DeleteChannelSession(ctx context.Context, key string) error {
	return sm.cache.Delete(ctx, key)
}
