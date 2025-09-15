package models

import (
	"github.com/tristendillon/conduit/core/models"
)

// ContentCacheInterface manages file content tracking (Layer 1)
type ContentCacheInterface interface {
	// UpdateContent checks if file content has changed and updates entry
	UpdateContent(filePath string) (*ContentEntry, bool, error) // entry, changed, error

	// GetContent retrieves current content entry
	GetContent(filePath string) (*ContentEntry, bool) // entry, exists

	// SetContent manually sets content entry (for testing)
	SetContent(filePath string, entry *ContentEntry) error

	// RemoveContent removes entry for deleted files
	RemoveContent(filePath string) error

	// GetStats returns cache statistics
	GetStats() *CacheStats

	// Clear removes all entries
	Clear() error
}

// ParseCacheInterface manages parsed file data (Layer 2)
type ParseCacheInterface interface {
	// SetParsedFile stores parsed file data
	SetParsedFile(filePath string, parsed *models.ParsedFile) error

	// GetParsedFile retrieves parsed file data
	GetParsedFile(filePath string) (*models.ParsedFile, bool) // parsed, exists

	// InvalidateParse removes parsed data for a file
	InvalidateParse(filePath string) error

	// GetDependencies extracts dependency information from parsed data
	GetDependencies(filePath string) ([]string, error)

	// GetStats returns cache statistics
	GetStats() *CacheStats

	// Clear removes all entries
	Clear() error
}

// DependencyGraphInterface manages dependency relationships (Layer 3)
type DependencyGraphInterface interface {
	// BuildGraph constructs dependency graph from parsed files
	BuildGraph(parsedFiles map[string]*models.ParsedFile) error

	// UpdateNode updates a single node in the graph
	UpdateNode(filePath string, dependencies []string) error

	// GetAffectedFiles returns all files affected by a change
	GetAffectedFiles(changedFile string) ([]string, error)

	// GetDependencies returns direct dependencies of a file
	GetDependencies(filePath string) ([]string, error)

	// GetDependents returns files that depend on this file
	GetDependents(filePath string) ([]string, error)

	// GetNode retrieves a dependency node
	GetNode(filePath string) (*DependencyNode, bool)

	// RemoveNode removes a node and updates dependent relationships
	RemoveNode(filePath string) error

	// DetectCycles finds circular dependencies
	DetectCycles() ([][]string, error)

	// GetTopologicalOrder returns files in dependency order
	GetTopologicalOrder() ([]string, error)

	// GetStats returns graph statistics
	GetStats() *CacheStats

	// Clear removes all nodes
	Clear() error
}

// GenerationCacheInterface manages generation state (Layer 4)
type GenerationCacheInterface interface {
	// MarkGenerated records successful generation
	MarkGenerated(sourcePath, outputPath, sourceHash, templateHash, configHash string, dependencies []string) error

	// NeedsRegeneration checks if file needs regeneration
	NeedsRegeneration(sourcePath string, currentHash string, dependencies []string) (bool, string, error) // needs, reason, error

	// GetGenerationInfo retrieves generation metadata
	GetGenerationInfo(sourcePath string) (*GenerationInfo, bool)

	// InvalidateGeneration marks file as needing regeneration
	InvalidateGeneration(sourcePath string) error

	// GetOutdatedFiles returns all files needing regeneration
	GetOutdatedFiles() ([]string, error)

	// GetStats returns cache statistics
	GetStats() *CacheStats

	// Clear removes all entries
	Clear() error
}

// CacheManagerInterface provides unified cache coordination
type CacheManagerInterface interface {
	// HandleFileChange processes a file system change event
	HandleFileChange(event *ChangeEvent) (*RegenerationPlan, error)

	// GetParsedFile retrieves parsed file (checks content, then parse cache)
	GetParsedFile(filePath string) (*models.ParsedFile, bool, error)

	// SetParsedFile stores parsed file and updates dependency graph
	SetParsedFile(filePath string, parsed *models.ParsedFile) error

	// MarkGenerated records successful generation
	MarkGenerated(sourcePath, outputPath string) error

	// GetRegenerationPlan returns what needs to be regenerated
	GetRegenerationPlan(changedFiles []string) (*RegenerationPlan, error)

	// GetAffectedFiles returns files affected by changes
	GetAffectedFiles(changedFile string) ([]string, error)

	// ValidateIntegrity checks cache consistency across layers
	ValidateIntegrity() error

	// GetStats returns comprehensive cache statistics
	GetStats() map[string]*CacheStats

	// WarmCache initializes cache from file system
	WarmCache(rootDir string, excludePaths []string) error

	// GetRegistrySignature gets cached registry signature
	GetRegistrySignature() (*RegistrySignature, bool)

	// SetRegistrySignature stores registry signature
	SetRegistrySignature(signature *RegistrySignature) error

	// NeedsRegistryRegeneration checks if registry needs regeneration
	NeedsRegistryRegeneration(currentRoutes []string) (bool, error)

	// Clear resets all cache layers
	Clear() error
}