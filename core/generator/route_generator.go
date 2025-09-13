package generator

import (
	"fmt"
	"path/filepath"

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
	walker := rg.Walker
	if _, err := walker.Walk(rg.wd); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	walker.RouteTree.PrintTree(logLevel)

	templateData := struct {
		Routes []models.Route
	}{
		Routes: walker.RouteTree.Routes,
	}

	engine := template_engine.NewTemplateEngine()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	outputPath := filepath.Join(cfg.Codegen.Go.Output, "gen_routes.go")
	if err := engine.GenerateFile(template_engine.TEMPLATES.DEV.GEN_ROUTES_GO, outputPath, templateData); err != nil {
		return fmt.Errorf("failed to generate routes file: %w", err)
	}

	// Log cache statistics after generation
	fileCache := cache.GetCache()
	fileCache.LogStats()

	return nil
}
