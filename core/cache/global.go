package cache

import (
	"sync"

	"github.com/tristendillon/conduit/core/cache/manager"
	"github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/logger"
)

var (
	globalCacheManager models.CacheManagerInterface
	cacheOnce          sync.Once
)

// GetCacheManager returns the global cache manager instance
// This provides backward compatibility with the old cache.GetCache() pattern
func GetCacheManager() models.CacheManagerInterface {
	cacheOnce.Do(func() {
		globalCacheManager = manager.NewCacheManager()
		logger.Debug("Initialized global cache manager")
	})
	return globalCacheManager
}

// SetCacheManager allows setting a custom cache manager (useful for testing)
func SetCacheManager(cm models.CacheManagerInterface) {
	globalCacheManager = cm
	logger.Debug("Set custom cache manager")
}

// ClearGlobalCache clears the global cache manager
func ClearGlobalCache() error {
	if globalCacheManager != nil {
		return globalCacheManager.Clear()
	}
	return nil
}

// GetGlobalCacheStats returns statistics for all cache layers
func GetGlobalCacheStats() map[string]*models.CacheStats {
	if globalCacheManager != nil {
		return globalCacheManager.GetStats()
	}
	return make(map[string]*models.CacheStats)
}

// ValidateGlobalCacheIntegrity checks cache consistency
func ValidateGlobalCacheIntegrity() error {
	if globalCacheManager != nil {
		return globalCacheManager.ValidateIntegrity()
	}
	return nil
}