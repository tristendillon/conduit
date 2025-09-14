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

func extractFunctionBody(fset *token.FileSet, fn *ast.FuncDecl, src []byte) (string, error) {
	if fn.Body == nil {
		return "", nil
	}

	start := fset.Position(fn.Body.Lbrace).Offset + 1
	end := fset.Position(fn.Body.Rbrace).Offset

	if start < 0 || end < 0 || start >= len(src) || end > len(src) || start > end {
		return "", nil
	}

	bodyBytes := src[start:end]
	return strings.TrimSpace(string(bodyBytes)), nil
}

func extractImportsFromFile(f *ast.File) []string {
	templateProvidedImports := map[string]bool{
		"net/http": true,
	}

	var imports []string
	importSet := make(map[string]bool)

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		if templateProvidedImports[importPath] {
			continue
		}

		var importStatement string
		if imp.Name != nil {
			importStatement = imp.Name.Name + " \"" + importPath + "\""
		} else {
			importStatement = "\"" + importPath + "\""
		}

		if !importSet[importStatement] {
			importSet[importStatement] = true
			imports = append(imports, importStatement)
		}
	}
	return imports
}

func AnalyzeDependencies(f *ast.File, moduleName string) (*models.DependencyAnalysis, error) {
	analysis := &models.DependencyAnalysis{
		StandardLibImports: []string{},
		ExternalImports:    []string{},
		LocalImports:       []models.LocalDependency{},
	}

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		// Skip template-provided imports
		if importPath == "net/http" {
			continue
		}

		if isStandardLibrary(importPath) {
			analysis.StandardLibImports = append(analysis.StandardLibImports, importPath)
		} else if strings.HasPrefix(importPath, moduleName+"/") {
			// This is a local import within our module
			localDep := models.LocalDependency{
				ImportPath:    importPath,
				RelativePath:  strings.TrimPrefix(importPath, moduleName+"/"),
				Alias:         "",
			}
			if imp.Name != nil {
				localDep.Alias = imp.Name.Name
			}
			analysis.LocalImports = append(analysis.LocalImports, localDep)
		} else {
			// External dependency (third-party)
			analysis.ExternalImports = append(analysis.ExternalImports, importPath)
		}
	}

	return analysis, nil
}

func isStandardLibrary(importPath string) bool {
	// Standard library packages don't contain dots or are well-known stdlib packages
	stdLibPrefixes := []string{
		"bufio", "bytes", "context", "crypto", "database", "encoding", "errors",
		"fmt", "go", "hash", "html", "image", "io", "log", "math", "net",
		"os", "path", "reflect", "regexp", "runtime", "sort", "strconv",
		"strings", "sync", "syscall", "testing", "text", "time", "unicode",
	}

	// If it contains a dot, it's likely external (github.com/..., etc)
	if strings.Contains(importPath, ".") {
		return false
	}

	// Check if it starts with known stdlib prefixes
	for _, prefix := range stdLibPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}

	// If it's a simple name without dots, it's likely stdlib
	return !strings.Contains(importPath, "/") || len(strings.Split(importPath, "/")) <= 2
}

func extractFunctionSignature(fset *token.FileSet, fn *ast.FuncDecl, src []byte) string {
	if fn.Type == nil {
		return ""
	}

	start := fset.Position(fn.Name.Pos()).Offset
	var end int
	if fn.Body != nil {
		end = fset.Position(fn.Body.Lbrace).Offset
	} else {
		end = fset.Position(fn.End()).Offset
	}

	if start < 0 || end < 0 || start >= len(src) || end > len(src) || start > end {
		return ""
	}

	sigBytes := src[start:end]
	return strings.TrimSpace(string(sigBytes))
}

func ParseRouteWithFunctions(path, relPath, moduleName string) (*models.ParsedFile, error) {
	fset := token.NewFileSet()

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	srcStr := strings.TrimSpace(string(src))
	if srcStr == "" {
		logger.Debug("Empty route file %s, skipping parsing", relPath)
		return &models.ParsedFile{
			Path:         path,
			PackageName:  "",
			Methods:      []string{},
			RelPath:      relPath,
			Functions:    []models.ExtractedFunction{},
			Imports:      []string{},
			Dependencies: &models.DependencyAnalysis{},
		}, nil
	}

	if !strings.Contains(srcStr, "package ") {
		logger.Debug("Route file %s missing package declaration, skipping parsing", relPath)
		return &models.ParsedFile{
			Path:         path,
			PackageName:  "",
			Methods:      []string{},
			RelPath:      relPath,
			Functions:    []models.ExtractedFunction{},
			Imports:      []string{},
			Dependencies: &models.DependencyAnalysis{},
		}, nil
	}

	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		logger.Debug("Failed to parse route file %s: %v - treating as empty", relPath, err)
		return &models.ParsedFile{
			Path:         path,
			PackageName:  "",
			Methods:      []string{},
			RelPath:      relPath,
			Functions:    []models.ExtractedFunction{},
			Imports:      []string{},
			Dependencies: &models.DependencyAnalysis{},
		}, nil
	}

	var methods []string
	var functions []models.ExtractedFunction
	imports := extractImportsFromFile(f)

	logger.Debug("Parsing %s for function extraction", relPath)

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

			signature := extractFunctionSignature(fset, fn, src)
			body, bodyErr := extractFunctionBody(fset, fn, src)
			if bodyErr != nil {
				logger.Debug("Failed to extract body for %s: %v", name, bodyErr)
				continue
			}

			functions = append(functions, models.ExtractedFunction{
				Name:      name,
				Method:    upper,
				Signature: signature,
				Body:      body,
			})
		}
	}

	packageName := ""
	if f.Name != nil {
		packageName = f.Name.Name
	}

	// Perform dependency analysis
	dependencies, err := AnalyzeDependencies(f, moduleName)
	if err != nil {
		logger.Debug("Failed to analyze dependencies for %s: %v", relPath, err)
		dependencies = &models.DependencyAnalysis{}
	}

	parsed := &models.ParsedFile{
		Path:         path,
		PackageName:  packageName,
		Methods:      methods,
		RelPath:      relPath,
		Functions:    functions,
		Imports:      imports,
		Dependencies: dependencies,
	}

	return parsed, nil
}

func ParseRoute(path, relPath string) (*models.ParsedFile, error) {
	fset := token.NewFileSet()

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
