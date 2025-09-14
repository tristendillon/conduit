package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristendillon/conduit/core/cache"
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

	if err := rg.generateRoutesRegistry(walker.RouteTree.Routes, cfg); err != nil {
		return fmt.Errorf("failed to generate routes registry: %w", err)
	}

	fileCache := cache.GetCache()
	fileCache.LogStats()

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
			Route               models.Route
			ModuleName          string
			Timestamp           time.Time
			CopiedDependencies  []models.CopiedDependency
		}{
			Route:               route,
			ModuleName:          moduleName,
			Timestamp:           time.Now(),
			CopiedDependencies:  copiedDependencies,
		}

		if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.FULL_GEN_ROUTE_GO, route.OutputPath, templateData); err != nil {
			return fmt.Errorf("failed to generate route file %s: %w", route.OutputPath, err)
		}

		logger.Info("Generated %s for route %s with %d dependencies", route.RelativeOutput, route.FolderPath, len(copiedDependencies))
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

	logger.Info("Generated routes registry with %d routes", len(routes))
	return nil
}

func (rg *RouteGenerator) needsRegeneration(route models.Route) bool {
	if _, err := os.Stat(route.OutputPath); os.IsNotExist(err) {
		return true
	}

	fileCache := cache.GetCache()
	return fileCache.HasContentChanged(route.ParsedFile.Path)
}
