package dependency

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
	astParser "github.com/tristendillon/conduit/core/ast"
)

type DependencyCopier struct {
	projectRoot  string
	moduleName   string
	outputDir    string
	copiedDeps   map[string]*models.CopiedDependency
}

func NewDependencyCopier(projectRoot, moduleName, outputDir string) *DependencyCopier {
	return &DependencyCopier{
		projectRoot: projectRoot,
		moduleName:  moduleName,
		outputDir:   outputDir,
		copiedDeps:  make(map[string]*models.CopiedDependency),
	}
}

// CopyDependencies recursively copies all local dependencies for a route
func (dc *DependencyCopier) CopyDependencies(analysis *models.DependencyAnalysis) ([]models.CopiedDependency, error) {
	var result []models.CopiedDependency

	for _, localDep := range analysis.LocalImports {
		copied, err := dc.copyDependency(localDep)
		if err != nil {
			return nil, fmt.Errorf("failed to copy dependency %s: %w", localDep.ImportPath, err)
		}
		if copied != nil {
			result = append(result, *copied)
		}
	}

	return result, nil
}

func (dc *DependencyCopier) copyDependency(dep models.LocalDependency) (*models.CopiedDependency, error) {
	// Check if already copied
	if existing, exists := dc.copiedDeps[dep.ImportPath]; exists {
		logger.Debug("Dependency %s already copied", dep.ImportPath)
		return existing, nil
	}

	// Determine source path
	sourcePath := filepath.Join(dc.projectRoot, dep.RelativePath)
	logger.Debug("Attempting to copy dependency %s", dep.ImportPath)
	logger.Debug("  Source path: %s", sourcePath)
	logger.Debug("  Relative path: %s", dep.RelativePath)
	logger.Debug("  Project root: %s", dc.projectRoot)

	if !dc.pathExists(sourcePath) {
		return nil, fmt.Errorf("dependency path does not exist: %s", sourcePath)
	}

	// Determine target path in generated tree
	targetPath := filepath.Join(dc.outputDir, "dependencies", dep.RelativePath)
	logger.Debug("  Target path: %s", targetPath)

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy files
	copiedFiles, err := dc.copyPackageFiles(sourcePath, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy package files: %w", err)
	}

	// Analyze transitive dependencies
	transitiveDeps, err := dc.analyzeTransitiveDependencies(targetPath)
	if err != nil {
		logger.Debug("Failed to analyze transitive dependencies for %s: %v", dep.ImportPath, err)
		transitiveDeps = []models.LocalDependency{}
	}

	// Create copied dependency record
	newImportPath := fmt.Sprintf("%s/%s/dependencies/%s", dc.moduleName, strings.TrimPrefix(dc.outputDir, "./"), dep.RelativePath)
	copied := &models.CopiedDependency{
		OriginalPath:  sourcePath,
		GeneratedPath: targetPath,
		ImportPath:    newImportPath,
		Files:         copiedFiles,
		Dependencies:  transitiveDeps,
	}

	// Cache the result
	dc.copiedDeps[dep.ImportPath] = copied

	// Recursively copy transitive dependencies
	for _, transitive := range transitiveDeps {
		_, err := dc.copyDependency(transitive)
		if err != nil {
			logger.Debug("Failed to copy transitive dependency %s: %v", transitive.ImportPath, err)
		}
	}

	logger.Debug("Copied dependency %s to %s", dep.ImportPath, targetPath)
	return copied, nil
}

func (dc *DependencyCopier) copyPackageFiles(sourcePath, targetPath string) ([]string, error) {
	var copiedFiles []string

	logger.Debug("  copyPackageFiles called:")
	logger.Debug("    sourcePath: %s", sourcePath)
	logger.Debug("    targetPath: %s", targetPath)

	// Handle both single file and directory packages
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		logger.Debug("    Failed to stat source path: %v", err)
		return nil, err
	}

	logger.Debug("    Source is directory: %v", sourceInfo.IsDir())

	if sourceInfo.IsDir() {
		// Ensure target directory exists
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			logger.Debug("    Failed to create target directory %s: %v", targetPath, err)
			return nil, fmt.Errorf("failed to create target directory %s: %w", targetPath, err)
		}

		// Copy all .go files in the directory
		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			logger.Debug("    Failed to read source directory: %v", err)
			return nil, err
		}

		logger.Debug("    Found %d entries in source directory", len(entries))

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
				logger.Debug("    Skipping entry: %s (isDir=%v, isGo=%v)", entry.Name(), entry.IsDir(), strings.HasSuffix(entry.Name(), ".go"))
				continue
			}

			sourceFile := filepath.Join(sourcePath, entry.Name())
			targetFile := filepath.Join(targetPath, entry.Name())
			logger.Debug("    Copying file: %s -> %s", sourceFile, targetFile)

			if err := dc.copyAndRewriteFile(sourceFile, targetFile); err != nil {
				return nil, fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
			}
			copiedFiles = append(copiedFiles, targetFile)
		}
	} else {
		// Single file package - ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create target parent directory: %w", err)
		}

		if err := dc.copyAndRewriteFile(sourcePath, targetPath); err != nil {
			return nil, err
		}
		copiedFiles = append(copiedFiles, targetPath)
	}

	logger.Debug("    Successfully copied %d files", len(copiedFiles))
	return copiedFiles, nil
}

func (dc *DependencyCopier) copyAndRewriteFile(sourcePath, targetPath string) error {
	// Read source file
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Parse AST to rewrite imports
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, sourcePath, src, parser.ParseComments)
	if err != nil {
		// If parsing fails, just copy the file as-is
		logger.Debug("Failed to parse %s for import rewriting, copying as-is: %v", sourcePath, err)
		return os.WriteFile(targetPath, src, 0644)
	}

	// TODO: Implement import path rewriting here
	// For now, just copy the file as-is
	// In the future, we'll rewrite import paths to point to generated dependencies

	return os.WriteFile(targetPath, src, 0644)
}

func (dc *DependencyCopier) analyzeTransitiveDependencies(packagePath string) ([]models.LocalDependency, error) {
	var transitiveDeps []models.LocalDependency

	// Read all .go files in the package
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(packagePath, entry.Name())
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			logger.Debug("Failed to parse %s for transitive analysis: %v", filePath, err)
			continue
		}

		analysis, err := astParser.AnalyzeDependencies(f, dc.moduleName)
		if err != nil {
			logger.Debug("Failed to analyze dependencies in %s: %v", filePath, err)
			continue
		}

		// Add local dependencies that we haven't seen yet
		for _, dep := range analysis.LocalImports {
			if !dc.containsLocalDep(transitiveDeps, dep) {
				transitiveDeps = append(transitiveDeps, dep)
			}
		}
	}

	return transitiveDeps, nil
}

func (dc *DependencyCopier) containsLocalDep(deps []models.LocalDependency, target models.LocalDependency) bool {
	for _, dep := range deps {
		if dep.ImportPath == target.ImportPath {
			return true
		}
	}
	return false
}

func (dc *DependencyCopier) pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetCopiedDependencies returns all dependencies that have been copied
func (dc *DependencyCopier) GetCopiedDependencies() map[string]*models.CopiedDependency {
	return dc.copiedDeps
}