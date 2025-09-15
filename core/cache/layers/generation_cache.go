package layers

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/logger"
)

// GenerationCache implements Layer 4: Generation state tracking
type GenerationCache struct {
	entries map[string]*models.GenerationInfo
	mutex   sync.RWMutex
}

// NewGenerationCache creates a new generation cache
func NewGenerationCache() *GenerationCache {
	return &GenerationCache{
		entries: make(map[string]*models.GenerationInfo),
		mutex:   sync.RWMutex{},
	}
}

// MarkGenerated records successful generation
func (gc *GenerationCache) MarkGenerated(sourcePath, outputPath, sourceHash, templateHash, configHash string, dependencies []string) error {
	if sourcePath == "" || outputPath == "" {
		return fmt.Errorf("source path and output path cannot be empty")
	}

	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	// Calculate combined dependency hash
	depHash := gc.calculateDependencyHash(dependencies)

	entry := &models.GenerationInfo{
		SourcePath:     sourcePath,
		OutputPath:     outputPath,
		SourceHash:     sourceHash,
		TemplateHash:   templateHash,
		DependencyHash: depHash,
		GeneratedAt:    time.Now(),
		ConfigHash:     configHash,
	}

	gc.entries[sourcePath] = entry
	logger.Debug("GenerationCache: Marked %s as generated (output: %s)", sourcePath, outputPath)
	return nil
}

// NeedsRegeneration checks if file needs regeneration
func (gc *GenerationCache) NeedsRegeneration(sourcePath string, currentHash string, dependencies []string) (bool, string, error) {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	entry, exists := gc.entries[sourcePath]
	if !exists {
		return true, "no generation record found", nil
	}

	// Check if source content changed
	if entry.SourceHash != currentHash {
		return true, fmt.Sprintf("source content changed (hash: %s -> %s)",
			entry.SourceHash[:8], currentHash[:8]), nil
	}

	// Check if dependencies changed
	currentDepHash := gc.calculateDependencyHash(dependencies)
	if entry.DependencyHash != currentDepHash {
		return true, "dependencies changed", nil
	}

	// TODO: In a real implementation, you'd also check:
	// - Template version changes
	// - Config changes
	// - Output file existence/modification

	logger.Debug("GenerationCache: %s does not need regeneration", sourcePath)
	return false, "", nil
}

// GetGenerationInfo retrieves generation metadata
func (gc *GenerationCache) GetGenerationInfo(sourcePath string) (*models.GenerationInfo, bool) {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	entry, exists := gc.entries[sourcePath]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent modification
	entryCopy := &models.GenerationInfo{
		SourcePath:     entry.SourcePath,
		OutputPath:     entry.OutputPath,
		SourceHash:     entry.SourceHash,
		TemplateHash:   entry.TemplateHash,
		DependencyHash: entry.DependencyHash,
		GeneratedAt:    entry.GeneratedAt,
		ConfigHash:     entry.ConfigHash,
	}

	return entryCopy, true
}

// InvalidateGeneration marks file as needing regeneration
func (gc *GenerationCache) InvalidateGeneration(sourcePath string) error {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if _, exists := gc.entries[sourcePath]; exists {
		delete(gc.entries, sourcePath)
		logger.Debug("GenerationCache: Invalidated generation record for %s", sourcePath)
	}
	return nil
}

// GetOutdatedFiles returns all files needing regeneration
// This is a simplified implementation - in reality, you'd need to check against actual file system state
func (gc *GenerationCache) GetOutdatedFiles() ([]string, error) {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	var outdated []string

	// For this implementation, we'll just return files that are older than a certain threshold
	// In a real implementation, you'd check against actual file modification times, etc.
	threshold := time.Now().Add(-24 * time.Hour) // Files older than 24 hours

	for sourcePath, entry := range gc.entries {
		if entry.GeneratedAt.Before(threshold) {
			outdated = append(outdated, sourcePath)
		}
	}

	sort.Strings(outdated)
	logger.Debug("GenerationCache: Found %d outdated files", len(outdated))
	return outdated, nil
}

// GetStats returns cache statistics
func (gc *GenerationCache) GetStats() *models.CacheStats {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	return &models.CacheStats{
		TotalFiles:        len(gc.entries),
		GenerationEntries: len(gc.entries),
		LastUpdate:        time.Now(),
	}
}

// Clear removes all entries
func (gc *GenerationCache) Clear() error {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.entries = make(map[string]*models.GenerationInfo)
	logger.Debug("GenerationCache: Cleared all entries")
	return nil
}

// GetGeneratedFiles returns all files that have generation records
func (gc *GenerationCache) GetGeneratedFiles() []string {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	var files []string
	for sourcePath := range gc.entries {
		files = append(files, sourcePath)
	}
	sort.Strings(files)
	return files
}

// GetGenerationAge returns how long ago a file was generated
func (gc *GenerationCache) GetGenerationAge(sourcePath string) (time.Duration, error) {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	entry, exists := gc.entries[sourcePath]
	if !exists {
		return 0, fmt.Errorf("no generation record found for %s", sourcePath)
	}

	return time.Since(entry.GeneratedAt), nil
}

// Helper methods

// calculateDependencyHash creates a stable hash from dependency list
func (gc *GenerationCache) calculateDependencyHash(dependencies []string) string {
	if len(dependencies) == 0 {
		return ""
	}

	// Sort dependencies to ensure consistent hash
	sorted := make([]string, len(dependencies))
	copy(sorted, dependencies)
	sort.Strings(sorted)

	// Create hash from sorted dependencies
	combined := strings.Join(sorted, "|")
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

// UpdateTemplateHash updates the template hash for all entries
// This would be called when templates change
func (gc *GenerationCache) UpdateTemplateHash(newTemplateHash string) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	for _, entry := range gc.entries {
		entry.TemplateHash = newTemplateHash
	}
	logger.Debug("GenerationCache: Updated template hash for %d entries", len(gc.entries))
}

// UpdateConfigHash updates the config hash for all entries
// This would be called when config changes
func (gc *GenerationCache) UpdateConfigHash(newConfigHash string) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	for _, entry := range gc.entries {
		entry.ConfigHash = newConfigHash
	}
	logger.Debug("GenerationCache: Updated config hash for %d entries", len(gc.entries))
}

// GetFilesGeneratedAfter returns files generated after a specific time
func (gc *GenerationCache) GetFilesGeneratedAfter(after time.Time) []string {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	var files []string
	for sourcePath, entry := range gc.entries {
		if entry.GeneratedAt.After(after) {
			files = append(files, sourcePath)
		}
	}
	sort.Strings(files)
	return files
}