package layers

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/logger"
)

// ContentCache implements Layer 1: File content tracking
type ContentCache struct {
	entries map[string]*models.ContentEntry
	mutex   sync.RWMutex
	stats   struct {
		hits   int64
		misses int64
	}
}

// NewContentCache creates a new content cache
func NewContentCache() *ContentCache {
	return &ContentCache{
		entries: make(map[string]*models.ContentEntry),
		mutex:   sync.RWMutex{},
	}
}

// UpdateContent checks if file content has changed and updates entry
func (cc *ContentCache) UpdateContent(filePath string) (*models.ContentEntry, bool, error) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Get file info
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted
			if existing, exists := cc.entries[filePath]; exists {
				logger.Debug("ContentCache: File deleted: %s", filePath)
				delete(cc.entries, filePath)
				return existing, true, nil // changed = true because file was deleted
			}
			return nil, false, nil // file doesn't exist and wasn't cached
		}
		return nil, false, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	existing, exists := cc.entries[filePath]

	// If we don't have an entry, create one
	if !exists {
		logger.Debug("ContentCache: New file detected: %s", filePath)
		cc.stats.misses++
		entry, err := cc.createContentEntry(filePath, stat)
		if err != nil {
			return nil, false, err
		}
		cc.entries[filePath] = entry
		return entry, true, nil // changed = true because it's new
	}

	// Quick check: if size and modtime haven't changed, assume content is same
	if stat.Size() == existing.Size && stat.ModTime().Equal(existing.ModTime) {
		logger.Debug("ContentCache: Quick hit for %s (size and modtime unchanged)", filePath)
		cc.stats.hits++
		return existing, false, nil
	}

	// Size or modtime changed, need to check content hash
	newHash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
	}

	// Content actually changed
	if newHash != existing.ContentHash {
		logger.Debug("ContentCache: Content changed for %s (hash: %s -> %s)", filePath, existing.ContentHash[:8], newHash[:8])
		entry := &models.ContentEntry{
			FilePath:    filePath,
			ContentHash: newHash,
			ModTime:     stat.ModTime(),
			Size:        stat.Size(),
			Exists:      true,
		}
		cc.entries[filePath] = entry
		return entry, true, nil
	}

	// Content same, but modtime/size changed (editor save, etc.)
	logger.Debug("ContentCache: Metadata changed but content same for %s", filePath)
	existing.ModTime = stat.ModTime()
	existing.Size = stat.Size()
	cc.stats.hits++
	return existing, false, nil
}

// GetContent retrieves current content entry
func (cc *ContentCache) GetContent(filePath string) (*models.ContentEntry, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	entry, exists := cc.entries[filePath]
	if exists {
		cc.stats.hits++
	} else {
		cc.stats.misses++
	}
	return entry, exists
}

// SetContent manually sets content entry (for testing)
func (cc *ContentCache) SetContent(filePath string, entry *models.ContentEntry) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.entries[filePath] = entry
	logger.Debug("ContentCache: Manually set entry for %s", filePath)
	return nil
}

// RemoveContent removes entry for deleted files
func (cc *ContentCache) RemoveContent(filePath string) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if _, exists := cc.entries[filePath]; exists {
		delete(cc.entries, filePath)
		logger.Debug("ContentCache: Removed entry for %s", filePath)
	}
	return nil
}

// GetStats returns cache statistics
func (cc *ContentCache) GetStats() *models.CacheStats {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	total := cc.stats.hits + cc.stats.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(cc.stats.hits) / float64(total) * 100
	}

	return &models.CacheStats{
		TotalFiles:  len(cc.entries),
		CacheHits:   cc.stats.hits,
		CacheMisses: cc.stats.misses,
		HitRate:     hitRate,
		LastUpdate:  time.Now(),
	}
}

// Clear removes all entries
func (cc *ContentCache) Clear() error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.entries = make(map[string]*models.ContentEntry)
	cc.stats.hits = 0
	cc.stats.misses = 0
	logger.Debug("ContentCache: Cleared all entries")
	return nil
}

// createContentEntry creates a new content entry for a file
func (cc *ContentCache) createContentEntry(filePath string, stat os.FileInfo) (*models.ContentEntry, error) {
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
	}

	return &models.ContentEntry{
		FilePath:    filePath,
		ContentHash: hash,
		ModTime:     stat.ModTime(),
		Size:        stat.Size(),
		Exists:      true,
	}, nil
}

// calculateFileHash computes MD5 hash of file content
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}