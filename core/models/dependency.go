package models

type LocalDependency struct {
	ImportPath    string // Full import path: "my-app/api/v1/users/user_repo"
	RelativePath  string // Relative path: "api/v1/users/user_repo"
	Alias         string // Import alias if any
}

type DependencyAnalysis struct {
	StandardLibImports []string
	ExternalImports    []string
	LocalImports       []LocalDependency
}

type CopiedDependency struct {
	OriginalPath   string // Original source path
	GeneratedPath  string // Path in generated tree
	ImportPath     string // New import path for generated code
	Files          []string // List of copied files
	Dependencies   []LocalDependency // Transitive dependencies
}