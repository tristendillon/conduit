package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
)

func ExtractRouteInfo(file *ast.File) *models.RouteInfo {
	info := &models.RouteInfo{
		PackageName: file.Name.Name,
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			info.Funcs = append(info.Funcs, fn.Name.Name)
		}
	}

	for _, imp := range file.Imports {
		info.Imports = append(info.Imports, imp.Path.Value)
	}

	return info
}

func ParseRoute(path, relPath string) (*models.ParsedFile, error) {
	fset := token.NewFileSet()

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Handle empty files gracefully
	srcStr := strings.TrimSpace(string(src))
	if srcStr == "" {
		logger.Debug("Empty route file %s, skipping parsing", relPath)
		return &models.ParsedFile{
			Path:        path,
			PackageName: "",
			Methods:     []string{},
			RelPath:     relPath,
		}, nil
	}

	// Check for minimum valid Go content (at least package declaration)
	if !strings.Contains(srcStr, "package ") {
		logger.Debug("Route file %s missing package declaration, skipping parsing", relPath)
		return &models.ParsedFile{
			Path:        path,
			PackageName: "",
			Methods:     []string{},
			RelPath:     relPath,
		}, nil
	}

	f, err := parser.ParseFile(fset, path, src, parser.AllErrors)
	if err != nil {
		logger.Debug("Failed to parse route file %s: %v - treating as empty", relPath, err)
		return &models.ParsedFile{
			Path:        path,
			PackageName: "",
			Methods:     []string{},
			RelPath:     relPath,
		}, nil
	}

	methods := []string{}

	logger.Debug("Parsing %s with methods %v already existing", relPath, methods)

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}

		name := fn.Name.Name
		upper := strings.ToUpper(name)

		switch upper {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD":
			methods = append(methods, upper)
			logger.Debug("Found method %s in %s", upper, relPath)
		}
	}

	packageName := ""
	if f.Name != nil {
		packageName = f.Name.Name
	}

	parsed := &models.ParsedFile{
		Path:        path,
		PackageName: packageName,
		Methods:     methods,
		RelPath:     relPath,
	}

	return parsed, nil
}
