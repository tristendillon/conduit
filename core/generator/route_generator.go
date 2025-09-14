package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristendillon/conduit/core/cache"
	"github.com/tristendillon/conduit/core/config"
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
	// 1. Walk and build route tree (existing logic)
	walker := rg.Walker
	if _, err := walker.Walk(rg.wd); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	walker.RouteTree.PrintTree(logLevel)

	// 2. Calculate output paths for all routes
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	moduleName := rg.getModuleName()
	if err := walker.RouteTree.CalculateOutputPaths(cfg, moduleName); err != nil {
		return fmt.Errorf("failed to calculate output paths: %w", err)
	}

	// 3. Generate per-route files
	if err := rg.generatePerRouteFiles(walker.RouteTree.Routes); err != nil {
		return fmt.Errorf("failed to generate per-route files: %w", err)
	}

	// 4. Generate registry aggregation file
	if err := rg.generateRoutesRegistry(walker.RouteTree.Routes, cfg); err != nil {
		return fmt.Errorf("failed to generate routes registry: %w", err)
	}

	// 5. Log cache statistics after generation
	fileCache := cache.GetCache()
	fileCache.LogStats()

	return nil
}

func (rg *RouteGenerator) getModuleName() string {
	// Extract module name from go.mod file
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

	for _, route := range routes {
		// Check if generation needed (leverage existing cache system)
		if !rg.needsRegeneration(route) {
			logger.Debug("Skipping unchanged route: %s", route.FolderPath)
			continue
		}

		// Prepare template data
		templateData := struct {
			Route      models.Route
			ModuleName string
			Timestamp  time.Time
		}{
			Route:      route,
			ModuleName: rg.getModuleName(),
			Timestamp:  time.Now(),
		}

		// Generate using template reference system
		if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.GEN_ROUTE_GO, route.OutputPath, templateData); err != nil {
			return fmt.Errorf("failed to generate route file %s: %w", route.OutputPath, err)
		}

		logger.Info("Generated %s for route %s", route.RelativeOutput, route.FolderPath)
	}

	return nil
}

func (rg *RouteGenerator) generateRoutesRegistry(routes []models.Route, cfg *config.Config) error {
	engine := template_engine.NewTemplateEngine()

	// Prepare registry template data
	templateData := struct {
		Routes      []models.Route
		PackageName string
		ModuleName  string
		Timestamp   time.Time
	}{
		Routes:      routes,
		PackageName: "generated", // Or extract from config
		ModuleName:  rg.getModuleName(),
		Timestamp:   time.Now(),
	}

	// Generate registry file
	registryPath := filepath.Join(cfg.Codegen.Go.Output, "routes_registry.go")
	if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.ROUTES_REGISTRY_GO, registryPath, templateData); err != nil {
		return fmt.Errorf("failed to generate routes registry: %w", err)
	}

	logger.Info("Generated routes registry with %d routes", len(routes))
	return nil
}

func (rg *RouteGenerator) needsRegeneration(route models.Route) bool {
	// Check if output file exists
	if _, err := os.Stat(route.OutputPath); os.IsNotExist(err) {
		return true // Output doesn't exist, need to generate
	}

	// Check if source file changed since last generation
	fileCache := cache.GetCache()
	if fileCache.HasContentChanged(route.ParsedFile.Path) {
		return true // Source changed, need to regenerate
	}

	return false // No changes needed
}
