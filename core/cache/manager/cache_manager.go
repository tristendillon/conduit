package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristendillon/conduit/core/cache/layers"
	"github.com/tristendillon/conduit/core/cache/models"
	coreModels "github.com/tristendillon/conduit/core/models"
	"github.com/tristendillon/conduit/core/logger"
)

// CacheManager coordinates all cache layers and provides unified interface
type CacheManager struct {
	content    models.ContentCacheInterface
	parse      models.ParseCacheInterface
	deps       models.DependencyGraphInterface
	generation models.GenerationCacheInterface
}

// NewCacheManager creates a new cache manager with default implementations
func NewCacheManager() *CacheManager {
	return &CacheManager{
		content:    layers.NewContentCache(),
		parse:      layers.NewParseCache(),
		deps:       layers.NewDependencyGraph(),
		generation: layers.NewGenerationCache(),
	}
}

// NewCacheManagerWithLayers creates a cache manager with custom layer implementations
func NewCacheManagerWithLayers(
	content models.ContentCacheInterface,
	parse models.ParseCacheInterface,
	deps models.DependencyGraphInterface,
	generation models.GenerationCacheInterface,
) *CacheManager {
	return &CacheManager{
		content:    content,
		parse:      parse,
		deps:       deps,
		generation: generation,
	}
}

// HandleFileChange processes a file system change event
func (cm *CacheManager) HandleFileChange(event *models.ChangeEvent) (*models.RegenerationPlan, error) {
	logger.Debug("CacheManager: Handling file change: %s (%s)", event.FilePath, event.EventType)

	plan := &models.RegenerationPlan{
		ChangedFiles:    []string{event.FilePath},
		AffectedFiles:   []string{},
		RegenerationMap: make(map[string][]string),
		Reasons:         make(map[string]string),
		Priority:        make(map[string]int),
	}

	switch event.EventType {
	case "delete":
		return cm.handleFileDelete(event, plan)
	case "write", "create":
		return cm.handleFileChange(event, plan)
	default:
		return plan, fmt.Errorf("unknown event type: %s", event.EventType)
	}
}

// GetParsedFile retrieves parsed file (checks content, then parse cache)
func (cm *CacheManager) GetParsedFile(filePath string) (*coreModels.ParsedFile, bool, error) {
	// First check if content has changed
	contentEntry, contentChanged, err := cm.content.UpdateContent(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check content for %s: %w", filePath, err)
	}

	if contentEntry == nil || !contentEntry.Exists {
		// File doesn't exist
		cm.parse.InvalidateParse(filePath)
		return nil, false, nil
	}

	// If content changed, invalidate parse cache
	if contentChanged {
		logger.Debug("CacheManager: Content changed for %s, invalidating parse cache", filePath)
		cm.parse.InvalidateParse(filePath)
	}

	// Try to get from parse cache
	parsed, exists := cm.parse.GetParsedFile(filePath)
	return parsed, exists, nil
}

// SetParsedFile stores parsed file and updates dependency graph
func (cm *CacheManager) SetParsedFile(filePath string, parsed *coreModels.ParsedFile) error {
	// Store in parse cache
	if err := cm.parse.SetParsedFile(filePath, parsed); err != nil {
		return fmt.Errorf("failed to store parsed file: %w", err)
	}

	// Extract dependencies and update dependency graph
	dependencies, err := cm.parse.GetDependencies(filePath)
	if err != nil {
		logger.Debug("CacheManager: Failed to get dependencies for %s: %v", filePath, err)
		dependencies = []string{} // Continue with empty dependencies
	}

	if err := cm.deps.UpdateNode(filePath, dependencies); err != nil {
		return fmt.Errorf("failed to update dependency graph: %w", err)
	}

	// Update content hash in dependency graph if we have content info
	if contentEntry, exists := cm.content.GetContent(filePath); exists {
		if node, nodeExists := cm.deps.GetNode(filePath); nodeExists {
			node.ContentHash = contentEntry.ContentHash
		}
	}

	logger.Debug("CacheManager: Stored parsed file and updated dependencies for %s", filePath)
	return nil
}

// MarkGenerated records successful generation
func (cm *CacheManager) MarkGenerated(sourcePath, outputPath string) error {
	// Get current content hash
	contentEntry, exists := cm.content.GetContent(sourcePath)
	if !exists {
		return fmt.Errorf("no content entry found for source file: %s", sourcePath)
	}

	// Get dependencies
	dependencies, err := cm.deps.GetDependencies(sourcePath)
	if err != nil {
		logger.Debug("CacheManager: Failed to get dependencies for %s: %v", sourcePath, err)
		dependencies = []string{}
	}

	// TODO: In a real implementation, you'd get actual template and config hashes
	templateHash := "template-v1" // Placeholder
	configHash := "config-v1"     // Placeholder

	return cm.generation.MarkGenerated(sourcePath, outputPath, contentEntry.ContentHash, templateHash, configHash, dependencies)
}

// GetRegenerationPlan returns what needs to be regenerated
func (cm *CacheManager) GetRegenerationPlan(changedFiles []string) (*models.RegenerationPlan, error) {
	plan := &models.RegenerationPlan{
		ChangedFiles:    changedFiles,
		AffectedFiles:   []string{},
		RegenerationMap: make(map[string][]string),
		Reasons:         make(map[string]string),
		Priority:        make(map[string]int),
	}

	allAffected := make(map[string]bool)

	// For each changed file, find all affected files
	for _, changedFile := range changedFiles {
		affected, err := cm.deps.GetAffectedFiles(changedFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get affected files for %s: %w", changedFile, err)
		}

		// Add to regeneration map
		plan.RegenerationMap[changedFile] = affected

		// Collect all affected files
		for _, affectedFile := range affected {
			if !allAffected[affectedFile] {
				allAffected[affectedFile] = true
				plan.AffectedFiles = append(plan.AffectedFiles, affectedFile)
				plan.Reasons[affectedFile] = fmt.Sprintf("depends on changed file: %s", changedFile)
				plan.Priority[affectedFile] = 1 // Default priority
			}
		}
	}

	// Also check generation cache for files that need regeneration
	for _, changedFile := range changedFiles {
		if contentEntry, exists := cm.content.GetContent(changedFile); exists {
			dependencies, _ := cm.deps.GetDependencies(changedFile)
			needsRegen, reason, err := cm.generation.NeedsRegeneration(changedFile, contentEntry.ContentHash, dependencies)
			if err != nil {
				logger.Debug("CacheManager: Error checking regeneration for %s: %v", changedFile, err)
				continue
			}

			if needsRegen {
				if !allAffected[changedFile] {
					allAffected[changedFile] = true
					plan.AffectedFiles = append(plan.AffectedFiles, changedFile)
				}
				plan.Reasons[changedFile] = reason
				plan.Priority[changedFile] = 2 // Higher priority for direct changes
			}
		}
	}

	logger.Debug("CacheManager: Generated regeneration plan - %d changed files affect %d total files",
		len(changedFiles), len(plan.AffectedFiles))

	return plan, nil
}

// GetAffectedFiles returns files affected by changes
func (cm *CacheManager) GetAffectedFiles(changedFile string) ([]string, error) {
	return cm.deps.GetAffectedFiles(changedFile)
}

// ValidateIntegrity checks cache consistency across layers
func (cm *CacheManager) ValidateIntegrity() error {
	// Check that all parsed files have corresponding content entries
	parsedFiles := cm.parse.(*layers.ParseCache).GetAllParsedFiles()
	for filePath := range parsedFiles {
		if _, exists := cm.content.GetContent(filePath); !exists {
			logger.Debug("CacheManager: Warning - parsed file %s has no content entry", filePath)
		}
	}

	// Check for dependency cycles
	cycles, err := cm.deps.DetectCycles()
	if err != nil {
		return fmt.Errorf("failed to detect cycles: %w", err)
	}

	if len(cycles) > 0 {
		logger.Debug("CacheManager: Warning - detected %d dependency cycles", len(cycles))
		for i, cycle := range cycles {
			logger.Debug("  Cycle %d: %v", i+1, cycle)
		}
	}

	logger.Debug("CacheManager: Cache integrity validation completed")
	return nil
}

// GetStats returns comprehensive cache statistics
func (cm *CacheManager) GetStats() map[string]*models.CacheStats {
	return map[string]*models.CacheStats{
		"content":    cm.content.GetStats(),
		"parse":      cm.parse.GetStats(),
		"dependency": cm.deps.GetStats(),
		"generation": cm.generation.GetStats(),
	}
}

// WarmCache initializes cache from file system
func (cm *CacheManager) WarmCache(rootDir string, excludePaths []string) error {
	logger.Debug("CacheManager: Warming cache from directory: %s", rootDir)
	startTime := time.Now()

	var fileCount int
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if path should be excluded
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		for _, exclude := range excludePaths {
			if strings.Contains(relPath, exclude) {
				return nil
			}
		}

		// Only process route.go files for now
		if !strings.HasSuffix(path, "route.go") {
			return nil
		}

		// Update content cache
		_, _, err = cm.content.UpdateContent(path)
		if err != nil {
			logger.Debug("CacheManager: Failed to cache content for %s: %v", path, err)
			return nil // Continue with other files
		}

		fileCount++
		return nil
	})

	duration := time.Since(startTime)
	logger.Debug("CacheManager: Cache warming completed in %v - processed %d files", duration, fileCount)
	return err
}

// Clear resets all cache layers
func (cm *CacheManager) Clear() error {
	if err := cm.content.Clear(); err != nil {
		return fmt.Errorf("failed to clear content cache: %w", err)
	}
	if err := cm.parse.Clear(); err != nil {
		return fmt.Errorf("failed to clear parse cache: %w", err)
	}
	if err := cm.deps.Clear(); err != nil {
		return fmt.Errorf("failed to clear dependency graph: %w", err)
	}
	if err := cm.generation.Clear(); err != nil {
		return fmt.Errorf("failed to clear generation cache: %w", err)
	}

	logger.Debug("CacheManager: Cleared all cache layers")
	return nil
}

// Helper methods for internal use

// handleFileDelete processes file deletion
func (cm *CacheManager) handleFileDelete(event *models.ChangeEvent, plan *models.RegenerationPlan) (*models.RegenerationPlan, error) {
	// Remove from all caches
	cm.content.RemoveContent(event.FilePath)
	cm.parse.InvalidateParse(event.FilePath)
	cm.deps.RemoveNode(event.FilePath)
	cm.generation.InvalidateGeneration(event.FilePath)

	// Find files that depended on this file
	dependents, err := cm.deps.GetDependents(event.FilePath)
	if err == nil {
		plan.AffectedFiles = dependents
		for _, dependent := range dependents {
			plan.Reasons[dependent] = fmt.Sprintf("dependency deleted: %s", event.FilePath)
			plan.Priority[dependent] = 3 // High priority for deleted dependencies
		}
	}

	return plan, nil
}

// handleFileChange processes file modification/creation
func (cm *CacheManager) handleFileChange(event *models.ChangeEvent, plan *models.RegenerationPlan) (*models.RegenerationPlan, error) {
	// Update content cache
	_, contentChanged, err := cm.content.UpdateContent(event.FilePath)
	if err != nil {
		return plan, fmt.Errorf("failed to update content cache: %w", err)
	}

	if contentChanged {
		// Invalidate parse cache
		cm.parse.InvalidateParse(event.FilePath)

		// Find affected files
		affected, err := cm.deps.GetAffectedFiles(event.FilePath)
		if err == nil {
			plan.AffectedFiles = affected
			for _, affectedFile := range affected {
				plan.Reasons[affectedFile] = fmt.Sprintf("dependency changed: %s", event.FilePath)
				plan.Priority[affectedFile] = 1
			}
		}

		// The changed file itself needs regeneration
		plan.AffectedFiles = append(plan.AffectedFiles, event.FilePath)
		plan.Reasons[event.FilePath] = "file content changed"
		plan.Priority[event.FilePath] = 2
	}

	return plan, nil
}