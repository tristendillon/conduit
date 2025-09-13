package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
)

type FileCache struct {
	entries map[string]*models.CacheEntry
	config  *CacheConfig
	metrics *CacheMetrics
	mutex   sync.RWMutex
}

var (
	globalCache *FileCache
	cacheOnce   sync.Once
)

func GetCache() *FileCache {
	cacheOnce.Do(func() {
		globalCache = NewFileCache(DefaultCacheConfig())
	})
	return globalCache
}

func NewFileCache(config *CacheConfig) *FileCache {
	cache := &FileCache{
		entries: make(map[string]*models.CacheEntry),
		config:  config,
		metrics: &CacheMetrics{},
		mutex:   sync.RWMutex{},
	}

	logger.Debug("Created new file cache with config: MaxEntries=%d, TTL=%v",
		config.MaxEntries, config.DefaultTTL)

	return cache
}

func (fc *FileCache) ValidateAndGet(filePath string) (*models.ParsedFile, bool) {
	fc.mutex.RLock()
	entry, exists := fc.entries[filePath]
	fc.mutex.RUnlock()

	if !exists {
		fc.incrementMisses()
		logger.Debug("Cache miss for %s - entry not found", filePath)
		return nil, false
	}

	valid, err := entry.IsValid()
	if err != nil {
		logger.Debug("Cache validation error for %s: %v", filePath, err)
		fc.InvalidateFile(filePath)
		fc.incrementMisses()
		return nil, false
	}

	if !valid {
		logger.Debug("Cache miss for %s - file modified", filePath)
		fc.InvalidateFile(filePath)
		fc.incrementMisses()
		return nil, false
	}

	if fc.isExpired(entry) {
		logger.Debug("Cache miss for %s - entry expired", filePath)
		fc.InvalidateFile(filePath)
		fc.incrementMisses()
		return nil, false
	}

	fc.incrementHits()
	logger.Debug("Cache hit for %s", filePath)
	return entry.ParsedFile, true
}

func (fc *FileCache) Set(filePath string, parsedFile *models.ParsedFile) error {
	entry, err := models.NewCacheEntry(filePath, parsedFile)
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if len(fc.entries) >= fc.config.MaxEntries {
		logger.Debug("Cache full, evicting oldest entries")
		fc.evictOldest()
	}

	fc.entries[filePath] = entry
	logger.Debug("Cached parsed data for %s", filePath)
	return nil
}

func (fc *FileCache) InvalidateFile(filePath string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	if _, exists := fc.entries[filePath]; exists {
		delete(fc.entries, filePath)
		fc.metrics.Invalidations++
		logger.Debug("Invalidated cache entry for %s", filePath)
	}
}

func (fc *FileCache) Clear() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	entriesCount := len(fc.entries)
	fc.entries = make(map[string]*models.CacheEntry)
	fc.metrics.Invalidations += int64(entriesCount)
	logger.Info("Cleared entire cache, invalidated %d entries", entriesCount)
}

func (fc *FileCache) GetMetrics() *CacheMetrics {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	metrics := *fc.metrics
	metrics.TotalEntries = len(fc.entries)
	metrics.CalculateHitRate()
	return &metrics
}

func (fc *FileCache) LogStats() {
	metrics := fc.GetMetrics()
	logger.Debug("Cache stats: Hits=%d, Misses=%d, Hit Rate=%.1f%%, Total Entries=%d, Invalidations=%d",
		metrics.Hits, metrics.Misses, metrics.HitRate, metrics.TotalEntries, metrics.Invalidations)
}

func (fc *FileCache) isExpired(entry *models.CacheEntry) bool {
	return time.Since(entry.CreatedAt) > fc.config.DefaultTTL
}

func (fc *FileCache) evictOldest() {
	var oldestPath string
	var oldestTime time.Time

	for path, entry := range fc.entries {
		if oldestPath == "" || entry.CreatedAt.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.CreatedAt
		}
	}

	if oldestPath != "" {
		delete(fc.entries, oldestPath)
		fc.metrics.Invalidations++
		logger.Debug("Evicted oldest cache entry: %s", oldestPath)
	}
}

func (fc *FileCache) incrementHits() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.metrics.Hits++
}

func (fc *FileCache) incrementMisses() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.metrics.Misses++
}
