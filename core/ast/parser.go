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

func ParseRouteWithFunctions(path, relPath string) (*models.ParsedFile, error) {
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
			Functions:   []models.ExtractedFunction{},
			Imports:     []string{},
		}, nil
	}

	if !strings.Contains(srcStr, "package ") {
		logger.Debug("Route file %s missing package declaration, skipping parsing", relPath)
		return &models.ParsedFile{
			Path:        path,
			PackageName: "",
			Methods:     []string{},
			RelPath:     relPath,
			Functions:   []models.ExtractedFunction{},
			Imports:     []string{},
		}, nil
	}

	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		logger.Debug("Failed to parse route file %s: %v - treating as empty", relPath, err)
		return &models.ParsedFile{
			Path:        path,
			PackageName: "",
			Methods:     []string{},
			RelPath:     relPath,
			Functions:   []models.ExtractedFunction{},
			Imports:     []string{},
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

	parsed := &models.ParsedFile{
		Path:        path,
		PackageName: packageName,
		Methods:     methods,
		RelPath:     relPath,
		Functions:   functions,
		Imports:     imports,
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
