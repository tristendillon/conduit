package walker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tristendillon/conduit/core/ast"
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
	var discovered []models.DiscoveredFile

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
			parsed, err := ast.ParseRoute(routeFile, relPath)
			if err != nil {
				return err
			}

			w.RouteTree.AddRoute(parsed)
			logger.Debug("Registered route: %s (methods: %v)", relPath, parsed.Methods)
		}

		return nil
	})

	return discovered, err
}
