package walker

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristendillon/conduit/core/ast"
	"github.com/tristendillon/conduit/core/cache"
	"github.com/tristendillon/conduit/core/config"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
)

type RouteWalker interface {
	Walk(root string) ([]models.DiscoveredFile, error)
}

type RouteWalkerImpl struct {
	RouteTree *models.RouteTree
	Exclude   []string
}

func getExcludePaths() []string {
	cfg, err := config.Load()
	if err != nil {
		logger.Debug("Failed to load config: %v", err)
	}
	return []string{
		".git", "node_modules", "vendor", ".next",
		"build", "dist", "__pycache__", ".DS_Store",
		".conduit", // default output directory for conduit
		cfg.Codegen.Go.Output,
		cfg.Codegen.Typescript.Output,
	}
}

func NewRouteWalker() *RouteWalkerImpl {
	exclude := getExcludePaths()
	return &RouteWalkerImpl{
		RouteTree: models.NewRouteTree(),
		Exclude:   exclude,
	}
}

func (w *RouteWalkerImpl) Walk(root string) ([]models.DiscoveredFile, error) {
	startTime := time.Now()
	w.RouteTree.Reset()
	var discovered []models.DiscoveredFile
	fileCache := cache.GetCache()

	var cacheHits, cacheMisses int

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		for _, ex := range w.Exclude {
			if strings.Contains(relPath, ex) {
				return nil
			}
		}

		routeFile := filepath.Join(path, "route.go")
		if _, err := os.Stat(routeFile); err == nil {
			if cachedParsed, found := fileCache.ValidateAndGet(routeFile); found {
				w.RouteTree.AddRoute(cachedParsed)
				logger.Debug("Using cached route: %s (methods: %v)", relPath, cachedParsed.Methods)
				cacheHits++
			} else {
				parsed, err := ast.ParseRoute(routeFile, relPath)
				if err != nil {
					logger.Debug("Failed to parse route %s: %v", routeFile, err)
					return err
				}

				if err := fileCache.Set(routeFile, parsed); err != nil {
					logger.Debug("Failed to cache parsed route %s: %v", routeFile, err)
				}

				w.RouteTree.AddRoute(parsed)
				logger.Debug("Parsed and registered route: %s (methods: %v)", relPath, parsed.Methods)
				cacheMisses++
			}
		}

		return nil
	})

	walkDuration := time.Since(startTime)
	totalRoutes := cacheHits + cacheMisses
	logger.Debug("Walk completed in %v: %d routes (%d cached, %d parsed)",
		walkDuration, totalRoutes, cacheHits, cacheMisses)

	if totalRoutes > 0 {
		cacheHitRate := float64(cacheHits) / float64(totalRoutes) * 100
		logger.Debug("Cache performance: %.1f%% hit rate", cacheHitRate)
	}

	return discovered, err
}
