package layers

import (
	"fmt"
	"sync"
	"time"

	"github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/logger"
	coreModels "github.com/tristendillon/conduit/core/models"
)

// DependencyGraph implements Layer 3: Dependency relationship management
type DependencyGraph struct {
	nodes map[string]*models.DependencyNode
	mutex sync.RWMutex
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*models.DependencyNode),
		mutex: sync.RWMutex{},
	}
}

// BuildGraph constructs dependency graph from parsed files
func (dg *DependencyGraph) BuildGraph(parsedFiles map[string]*coreModels.ParsedFile) error {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	// Clear existing graph
	dg.nodes = make(map[string]*models.DependencyNode)

	// First pass: create all nodes
	for filePath := range parsedFiles {
		node := &models.DependencyNode{
			FilePath:     filePath,
			NodeType:     models.SourceFile,
			Dependencies: []string{},
			Dependents:   []string{},
			ContentHash:  "", // Will be filled by cache manager
		}
		dg.nodes[filePath] = node
	}

	// Second pass: build dependency relationships
	for filePath, parsed := range parsedFiles {
		if parsed.Dependencies != nil {
			var dependencies []string

			// Add local imports as dependencies
			for _, localImport := range parsed.Dependencies.LocalImports {
				dependencies = append(dependencies, localImport.ImportPath)
				// Add this file as a dependent of the imported file
				dg.addDependentRelationship(localImport.ImportPath, filePath)
			}

			// Update node with dependencies
			if node, exists := dg.nodes[filePath]; exists {
				node.Dependencies = dependencies
			}
		}
	}

	logger.Debug("DependencyGraph: Built graph with %d nodes", len(dg.nodes))
	return nil
}

// UpdateNode updates a single node in the graph
func (dg *DependencyGraph) UpdateNode(filePath string, dependencies []string) error {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	node, exists := dg.nodes[filePath]
	if !exists {
		// Create new node
		node = &models.DependencyNode{
			FilePath:     filePath,
			NodeType:     models.SourceFile,
			Dependencies: []string{},
			Dependents:   []string{},
		}
		dg.nodes[filePath] = node
	}

	// Remove old dependency relationships
	for _, oldDep := range node.Dependencies {
		dg.removeDependentRelationship(oldDep, filePath)
	}

	// Add new dependency relationships
	node.Dependencies = dependencies
	for _, newDep := range dependencies {
		dg.addDependentRelationship(newDep, filePath)
	}

	logger.Debug("DependencyGraph: Updated node %s with %d dependencies", filePath, len(dependencies))
	return nil
}

// GetAffectedFiles returns all files affected by a change
func (dg *DependencyGraph) GetAffectedFiles(changedFile string) ([]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	visited := make(map[string]bool)
	var affected []string

	// Use DFS to find all dependents
	dg.dfsVisitDependents(changedFile, visited, &affected)

	logger.Debug("DependencyGraph: File %s affects %d files: %v", changedFile, len(affected), affected)
	return affected, nil
}

// GetDependencies returns direct dependencies of a file
func (dg *DependencyGraph) GetDependencies(filePath string) ([]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	node, exists := dg.nodes[filePath]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to avoid concurrent modification
	dependencies := make([]string, len(node.Dependencies))
	copy(dependencies, node.Dependencies)
	return dependencies, nil
}

// GetDependents returns files that depend on this file
func (dg *DependencyGraph) GetDependents(filePath string) ([]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	node, exists := dg.nodes[filePath]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to avoid concurrent modification
	dependents := make([]string, len(node.Dependents))
	copy(dependents, node.Dependents)
	return dependents, nil
}

// GetNode retrieves a dependency node
func (dg *DependencyGraph) GetNode(filePath string) (*models.DependencyNode, bool) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	node, exists := dg.nodes[filePath]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid concurrent modification
	nodeCopy := &models.DependencyNode{
		FilePath:     node.FilePath,
		NodeType:     node.NodeType,
		Dependencies: make([]string, len(node.Dependencies)),
		Dependents:   make([]string, len(node.Dependents)),
		ContentHash:  node.ContentHash,
	}
	copy(nodeCopy.Dependencies, node.Dependencies)
	copy(nodeCopy.Dependents, node.Dependents)

	return nodeCopy, true
}

// RemoveNode removes a node and updates dependent relationships
func (dg *DependencyGraph) RemoveNode(filePath string) error {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	node, exists := dg.nodes[filePath]
	if !exists {
		return nil // Node doesn't exist, nothing to do
	}

	// Remove this node from all its dependencies' dependent lists
	for _, dep := range node.Dependencies {
		dg.removeDependentRelationship(dep, filePath)
	}

	// Remove this node from all its dependents' dependency lists
	for _, dependent := range node.Dependents {
		if depNode, exists := dg.nodes[dependent]; exists {
			depNode.Dependencies = removeFromSlice(depNode.Dependencies, filePath)
		}
	}

	delete(dg.nodes, filePath)
	logger.Debug("DependencyGraph: Removed node %s", filePath)
	return nil
}

// DetectCycles finds circular dependencies
func (dg *DependencyGraph) DetectCycles() ([][]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	var cycles [][]string
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)
	path := []string{}

	for filePath := range dg.nodes {
		if !visited[filePath] {
			if cyclePath := dg.dfsFindCycles(filePath, visited, recursionStack, path); cyclePath != nil {
				cycles = append(cycles, cyclePath)
			}
		}
	}

	if len(cycles) > 0 {
		logger.Debug("DependencyGraph: Detected %d cycles", len(cycles))
	}
	return cycles, nil
}

// GetTopologicalOrder returns files in dependency order
func (dg *DependencyGraph) GetTopologicalOrder() ([]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)
	queue := []string{}
	result := []string{}

	// Calculate in-degrees
	for filePath, node := range dg.nodes {
		inDegree[filePath] = len(node.Dependencies)
		if len(node.Dependencies) == 0 {
			queue = append(queue, filePath)
		}
	}

	// Process queue
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree of dependents
		if node, exists := dg.nodes[current]; exists {
			for _, dependent := range node.Dependents {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					queue = append(queue, dependent)
				}
			}
		}
	}

	// Check for cycles
	if len(result) != len(dg.nodes) {
		return nil, fmt.Errorf("dependency graph contains cycles")
	}

	return result, nil
}

// GetStats returns graph statistics
func (dg *DependencyGraph) GetStats() *models.CacheStats {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	return &models.CacheStats{
		TotalFiles:      len(dg.nodes),
		DependencyNodes: len(dg.nodes),
		LastUpdate:      time.Now(),
	}
}

// Clear removes all nodes
func (dg *DependencyGraph) Clear() error {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	dg.nodes = make(map[string]*models.DependencyNode)
	logger.Debug("DependencyGraph: Cleared all nodes")
	return nil
}

// Helper methods

// addDependentRelationship adds a dependent relationship (not thread-safe, caller must lock)
func (dg *DependencyGraph) addDependentRelationship(dependencyPath, dependentPath string) {
	// Create dependency node if it doesn't exist
	if _, exists := dg.nodes[dependencyPath]; !exists {
		dg.nodes[dependencyPath] = &models.DependencyNode{
			FilePath:     dependencyPath,
			NodeType:     models.SourceFile,
			Dependencies: []string{},
			Dependents:   []string{},
		}
	}

	// Add dependent if not already present
	depNode := dg.nodes[dependencyPath]
	for _, existing := range depNode.Dependents {
		if existing == dependentPath {
			return // Already exists
		}
	}
	depNode.Dependents = append(depNode.Dependents, dependentPath)
}

// removeDependentRelationship removes a dependent relationship (not thread-safe, caller must lock)
func (dg *DependencyGraph) removeDependentRelationship(dependencyPath, dependentPath string) {
	if depNode, exists := dg.nodes[dependencyPath]; exists {
		depNode.Dependents = removeFromSlice(depNode.Dependents, dependentPath)
	}
}

// dfsVisitDependents performs DFS to find all affected files
func (dg *DependencyGraph) dfsVisitDependents(filePath string, visited map[string]bool, affected *[]string) {
	if visited[filePath] {
		return
	}
	visited[filePath] = true

	node, exists := dg.nodes[filePath]
	if !exists {
		return
	}

	// Visit all dependents
	for _, dependent := range node.Dependents {
		*affected = append(*affected, dependent)
		dg.dfsVisitDependents(dependent, visited, affected)
	}
}

// dfsFindCycles performs DFS to detect cycles
func (dg *DependencyGraph) dfsFindCycles(filePath string, visited, recursionStack map[string]bool, path []string) []string {
	visited[filePath] = true
	recursionStack[filePath] = true
	path = append(path, filePath)

	node, exists := dg.nodes[filePath]
	if !exists {
		recursionStack[filePath] = false
		return nil
	}

	for _, dep := range node.Dependencies {
		if !visited[dep] {
			if cycle := dg.dfsFindCycles(dep, visited, recursionStack, path); cycle != nil {
				return cycle
			}
		} else if recursionStack[dep] {
			// Found a cycle, extract the cycle path
			cycleStart := -1
			for i, p := range path {
				if p == dep {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				return cycle
			}
		}
	}

	recursionStack[filePath] = false
	return nil
}

// removeFromSlice removes a string from a slice
func removeFromSlice(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
