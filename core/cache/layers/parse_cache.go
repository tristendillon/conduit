package layers

import (
	"fmt"
	"sync"
	"time"

	"github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/logger"
	coreModels "github.com/tristendillon/conduit/core/models"
)

// ParseCache implements Layer 2: Parsed file data storage
type ParseCache struct {
	entries map[string]*coreModels.ParsedFile
	mutex   sync.RWMutex
	stats   struct {
		hits   int64
		misses int64
	}
}

// NewParseCache creates a new parse cache
func NewParseCache() *ParseCache {
	return &ParseCache{
		entries: make(map[string]*coreModels.ParsedFile),
		mutex:   sync.RWMutex{},
	}
}

// SetParsedFile stores parsed file data
func (pc *ParseCache) SetParsedFile(filePath string, parsed *coreModels.ParsedFile) error {
	if parsed == nil {
		return fmt.Errorf("parsed file cannot be nil")
	}

	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	pc.entries[filePath] = parsed
	logger.Debug("ParseCache: Stored parsed data for %s (methods: %v)", filePath, parsed.Methods)
	return nil
}

// GetParsedFile retrieves parsed file data
func (pc *ParseCache) GetParsedFile(filePath string) (*coreModels.ParsedFile, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	parsed, exists := pc.entries[filePath]
	if exists {
		pc.stats.hits++
		logger.Debug("ParseCache: Hit for %s", filePath)
	} else {
		pc.stats.misses++
		logger.Debug("ParseCache: Miss for %s", filePath)
	}
	return parsed, exists
}

// InvalidateParse removes parsed data for a file
func (pc *ParseCache) InvalidateParse(filePath string) error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	if _, exists := pc.entries[filePath]; exists {
		delete(pc.entries, filePath)
		logger.Debug("ParseCache: Invalidated parsed data for %s", filePath)
	}
	return nil
}

// GetDependencies extracts dependency information from parsed data
func (pc *ParseCache) GetDependencies(filePath string) ([]string, error) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	parsed, exists := pc.entries[filePath]
	if !exists {
		return nil, fmt.Errorf("no parsed data found for %s", filePath)
	}

	var dependencies []string

	// Extract dependencies from parsed file
	if parsed.Dependencies != nil {
		// Add local imports as dependencies
		for _, localImport := range parsed.Dependencies.LocalImports {
			dependencies = append(dependencies, localImport.ImportPath)
		}

		// Add external imports (these might affect generation if templates change)
		dependencies = append(dependencies, parsed.Dependencies.ExternalImports...)

		logger.Debug("ParseCache: Found %d dependencies for %s", len(dependencies), filePath)
	}

	return dependencies, nil
}

// GetStats returns cache statistics
func (pc *ParseCache) GetStats() *models.CacheStats {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	total := pc.stats.hits + pc.stats.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(pc.stats.hits) / float64(total) * 100
	}

	return &models.CacheStats{
		TotalFiles:  len(pc.entries),
		CacheHits:   pc.stats.hits,
		CacheMisses: pc.stats.misses,
		HitRate:     hitRate,
		LastUpdate:  time.Now(),
	}
}

// Clear removes all entries
func (pc *ParseCache) Clear() error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	pc.entries = make(map[string]*coreModels.ParsedFile)
	pc.stats.hits = 0
	pc.stats.misses = 0
	logger.Debug("ParseCache: Cleared all entries")
	return nil
}

// GetAllParsedFiles returns all parsed files (for dependency graph building)
func (pc *ParseCache) GetAllParsedFiles() map[string]*coreModels.ParsedFile {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	// Return a copy to avoid concurrent modification issues
	result := make(map[string]*coreModels.ParsedFile)
	for path, parsed := range pc.entries {
		result[path] = parsed
	}
	return result
}

// GetFilesCount returns the number of cached parsed files
func (pc *ParseCache) GetFilesCount() int {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return len(pc.entries)
}
