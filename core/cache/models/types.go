package models

import (
	"time"
)

// NodeType represents the type of file in the dependency graph
type NodeType int

const (
	SourceFile NodeType = iota
	GeneratedFile
	TemplateFile
	ConfigFile
)

func (nt NodeType) String() string {
	switch nt {
	case SourceFile:
		return "Source"
	case GeneratedFile:
		return "Generated"
	case TemplateFile:
		return "Template"
	case ConfigFile:
		return "Config"
	default:
		return "Unknown"
	}
}

// ContentEntry tracks file content state (Layer 1)
type ContentEntry struct {
	FilePath    string    `json:"file_path"`
	ContentHash string    `json:"content_hash"`
	ModTime     time.Time `json:"mod_time"`
	Size        int64     `json:"size"`
	Exists      bool      `json:"exists"`
}

// DependencyNode represents a node in the dependency graph (Layer 3)
type DependencyNode struct {
	FilePath     string   `json:"file_path"`
	NodeType     NodeType `json:"node_type"`
	Dependencies []string `json:"dependencies"` // files this depends on
	Dependents   []string `json:"dependents"`   // files that depend on this
	ContentHash  string   `json:"content_hash"` // current content hash
}

// GenerationInfo tracks generation state for output files (Layer 4)
type GenerationInfo struct {
	SourcePath      string    `json:"source_path"`
	OutputPath      string    `json:"output_path"`
	SourceHash      string    `json:"source_hash"`      // hash when last generated
	TemplateHash    string    `json:"template_hash"`    // template version used
	DependencyHash  string    `json:"dependency_hash"`  // combined hash of all dependencies
	GeneratedAt     time.Time `json:"generated_at"`
	ConfigHash      string    `json:"config_hash"`      // config state when generated
}

// RegenerationPlan represents what needs to be regenerated
type RegenerationPlan struct {
	ChangedFiles    []string              `json:"changed_files"`    // files that actually changed
	AffectedFiles   []string              `json:"affected_files"`   // files affected by changes
	RegenerationMap map[string][]string   `json:"regeneration_map"` // source -> affected outputs
	Reasons         map[string]string     `json:"reasons"`          // why each file needs regeneration
	Priority        map[string]int        `json:"priority"`         // regeneration priority
}

// CacheStats provides metrics about cache performance
type CacheStats struct {
	TotalFiles       int     `json:"total_files"`
	CacheHits        int64   `json:"cache_hits"`
	CacheMisses      int64   `json:"cache_misses"`
	HitRate          float64 `json:"hit_rate"`
	DependencyNodes  int     `json:"dependency_nodes"`
	GenerationEntries int    `json:"generation_entries"`
	LastUpdate       time.Time `json:"last_update"`
}

// ChangeEvent represents a file system change
type ChangeEvent struct {
	FilePath  string    `json:"file_path"`
	EventType string    `json:"event_type"` // "write", "delete", "create"
	Timestamp time.Time `json:"timestamp"`
	OldHash   string    `json:"old_hash,omitempty"`
	NewHash   string    `json:"new_hash,omitempty"`
}