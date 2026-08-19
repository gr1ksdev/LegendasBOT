package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var localCache *cache.Cache

func init() {
	// Default expiration of 5 minutes, cleanup every 10 minutes
	localCache = cache.New(5*time.Minute, 10*time.Minute)
}

func (s *Service) SetLocal(key string, value interface{}, expiration time.Duration) {
	localCache.Set(key, value, expiration)
}

func (s *Service) GetLocal(key string) (interface{}, bool) {
	return localCache.Get(key)
}

func (s *Service) DeleteLocal(key string) {
	localCache.Delete(key)
}
