package generator

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tristendillon/conduit/core/cache"
	cacheModels "github.com/tristendillon/conduit/core/cache/models"
	"github.com/tristendillon/conduit/core/config"
	"github.com/tristendillon/conduit/core/dependency"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
	"github.com/tristendillon/conduit/core/template_engine"
	"github.com/tristendillon/conduit/core/walker"
)

type RouteGenerator struct {
	wd     string
	Walker *walker.RouteWalkerImpl
}

func NewRouteGenerator(wd string) *RouteGenerator {
	walker := walker.NewRouteWalker()
	return &RouteGenerator{wd: wd, Walker: walker}
}

func (rg *RouteGenerator) GenerateRouteTree(logLevel logger.LogLevel) error {
	walker := rg.Walker
	moduleName := rg.getModuleName()
	if _, err := walker.Walk(rg.wd, moduleName); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	walker.RouteTree.PrintTree(logLevel)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	moduleName = rg.getModuleName()
	if err := walker.RouteTree.CalculateOutputPaths(cfg, moduleName); err != nil {
		return fmt.Errorf("failed to calculate output paths: %w", err)
	}

	if err := rg.generatePerRouteFiles(walker.RouteTree.Routes); err != nil {
		return fmt.Errorf("failed to generate per-route files: %w", err)
	}

	// Only generate routes registry if needed
	if rg.needsRegistryRegeneration(walker.RouteTree.Routes) {
		if err := rg.generateRoutesRegistry(walker.RouteTree.Routes, cfg); err != nil {
			return fmt.Errorf("failed to generate routes registry: %w", err)
		}
	} else {
		logger.Debug("Routes registry is up to date, skipping generation")
	}

	cacheManager := cache.GetCacheManager()

	// Log cache statistics
	stats := cacheManager.GetStats()
	for layer, stat := range stats {
		logger.Debug("%s cache stats: %d files, %.1f%% hit rate", layer, stat.TotalFiles, stat.HitRate)
	}

	return nil
}

func (rg *RouteGenerator) getModuleName() string {
	goModPath := filepath.Join(rg.wd, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		logger.Debug("Could not read go.mod, using default module name: %v", err)
		return "app"
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	logger.Debug("No module declaration found in go.mod, using default")
	return "app" // fallback
}

func (rg *RouteGenerator) generatePerRouteFiles(routes []models.Route) error {
	engine := template_engine.NewTemplateEngine()
	moduleName := rg.getModuleName()

	// Load config to get output directory
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config for dependency copying: %w", err)
	}

	// Create dependency copier
	depCopier := dependency.NewDependencyCopier(rg.wd, moduleName, cfg.Codegen.Go.Output)

	for _, route := range routes {
		if !rg.needsRegeneration(route) {
			logger.Debug("Skipping unchanged route: %s", route.FolderPath)
			continue
		}

		// Copy dependencies if they exist
		var copiedDependencies []models.CopiedDependency
		if route.ParsedFile != nil && route.ParsedFile.Dependencies != nil && len(route.ParsedFile.Dependencies.LocalImports) > 0 {
			logger.Debug("Copying dependencies for route %s", route.FolderPath)
			copiedDeps, err := depCopier.CopyDependencies(route.ParsedFile.Dependencies)
			if err != nil {
				logger.Debug("Failed to copy dependencies for route %s: %v", route.FolderPath, err)
			} else {
				copiedDependencies = copiedDeps
				logger.Debug("Successfully copied %d dependencies for route %s", len(copiedDeps), route.FolderPath)
			}
		}

		templateData := struct {
			Route              models.Route
			ModuleName         string
			Timestamp          time.Time
			CopiedDependencies []models.CopiedDependency
		}{
			Route:              route,
			ModuleName:         moduleName,
			Timestamp:          time.Now(),
			CopiedDependencies: copiedDependencies,
		}

		if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.FULL_GEN_ROUTE_GO, route.OutputPath, templateData); err != nil {
			return fmt.Errorf("failed to generate route file %s: %w", route.OutputPath, err)
		}

		// Mark the file as generated in the cache
		cacheManager := cache.GetCacheManager()
		if err := cacheManager.MarkGenerated(route.ParsedFile.Path, route.OutputPath); err != nil {
			logger.Debug("Failed to mark %s as generated: %v", route.ParsedFile.Path, err)
		}

		logger.Debug("Generated %s for route %s with %d dependencies", route.RelativeOutput, route.FolderPath, len(copiedDependencies))
	}

	return nil
}

func (rg *RouteGenerator) generateRoutesRegistry(routes []models.Route, cfg *config.Config) error {
	engine := template_engine.NewTemplateEngine()

	templateData := struct {
		Routes      []models.Route
		PackageName string
		ModuleName  string
		Timestamp   time.Time
	}{
		Routes:      routes,
		PackageName: "generated",
		ModuleName:  rg.getModuleName(),
		Timestamp:   time.Now(),
	}

	registryPath := filepath.Join(cfg.Codegen.Go.Output, "routes_registry.go")
	if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.ROUTES_REGISTRY_GO, registryPath, templateData); err != nil {
		return fmt.Errorf("failed to generate routes registry: %w", err)
	}

	// Update registry signature in cache
	cacheManager := cache.GetCacheManager()
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.FolderPath
	}

	// Create new signature with proper hash calculation
	signature := rg.createRegistrySignature(routePaths)

	if err := cacheManager.SetRegistrySignature(signature); err != nil {
		logger.Debug("Failed to update registry signature: %v", err)
	}

	logger.Debug("Generated routes registry with %d routes", len(routes))
	return nil
}

func (rg *RouteGenerator) needsRegeneration(route models.Route) bool {
	// Check if output file exists
	if _, err := os.Stat(route.OutputPath); os.IsNotExist(err) {
		logger.Debug("Output file does not exist, regeneration needed for route: %s -> %s", route.FolderPath, route.OutputPath)
		return true
	}

	cacheManager := cache.GetCacheManager()

	// Get a regeneration plan for this specific file
	plan, err := cacheManager.GetRegenerationPlan([]string{route.ParsedFile.Path})
	if err != nil {
		logger.Debug("Failed to get regeneration plan for %s: %v, assuming regeneration needed", route.ParsedFile.Path, err)
		return true
	}

	// Check if this route is in the affected files list
	for _, affectedFile := range plan.AffectedFiles {
		if affectedFile == route.ParsedFile.Path {
			reason := plan.Reasons[affectedFile]
			logger.Debug("Regeneration needed for route: %s (source: %s) - %s", route.FolderPath, route.ParsedFile.Path, reason)
			return true
		}
	}

	logger.Debug("No regeneration needed for route: %s (source: %s)", route.FolderPath, route.ParsedFile.Path)
	return false
}

func (rg *RouteGenerator) needsRegistryRegeneration(routes []models.Route) bool {
	cacheManager := cache.GetCacheManager()

	// Extract route paths (the structural information we care about for registry)
	routePaths := make([]string, len(routes))
	for i, route := range routes {
		routePaths[i] = route.FolderPath
	}

	// Check if registry needs regeneration
	needsRegen, err := cacheManager.NeedsRegistryRegeneration(routePaths)
	if err != nil {
		logger.Debug("Failed to check registry regeneration: %v, assuming regeneration needed", err)
		return true
	}

	return needsRegen
}

func (rg *RouteGenerator) createRegistrySignature(routePaths []string) *cacheModels.RegistrySignature {
	// Sort the routes for consistent signature generation
	sortedPaths := make([]string, len(routePaths))
	copy(sortedPaths, routePaths)
	sort.Strings(sortedPaths)

	// Create hash from sorted route paths
	data := strings.Join(sortedPaths, "|")
	hash := md5.Sum([]byte(data))
	signature := fmt.Sprintf("%x", hash)

	return &cacheModels.RegistrySignature{
		RouteCount: len(routePaths),
		RoutePaths: sortedPaths,
		Signature:  signature,
		UpdatedAt:  time.Now(),
	}
}
