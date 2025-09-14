package models

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristendillon/conduit/core/config"
	"github.com/tristendillon/conduit/core/logger"
)

type RouteSegment struct {
	Name      string
	APIName   string
	IsParam   bool
	ParamName string
}

type RouteNode struct {
	Segment    RouteSegment
	Children   map[string]*RouteNode
	Parent     *RouteNode
	FullPath   string
	FolderPath string
	Depth      int
	Methods    []string
	ParsedFile *ParsedFile
}

type Route struct {
	APIPath    string
	FolderPath string
	Segments   []RouteSegment
	Parameters []string
	IsLeaf     bool
	Methods    []string
	ParsedFile *ParsedFile

	// Per-route generation fields
	OutputPath     string // Full path to generated file
	ImportPath     string // Go import path for this route
	RelativeOutput string // Relative path within output directory
	PackageAlias   string // Safe package alias for imports (e.g., "users_route")
}

type RouteTree struct {
	Root   *RouteNode
	Routes []Route
}

func NewRouteTree() *RouteTree {
	return &RouteTree{
		Root: &RouteNode{
			Segment:    RouteSegment{Name: "", APIName: ""},
			Children:   make(map[string]*RouteNode),
			FullPath:   "",
			FolderPath: "",
			Depth:      0,
			Methods:    []string{},
			ParsedFile: nil,
		},
		Routes: []Route{},
	}
}

func (rt *RouteTree) Reset() {
	rt.Root = &RouteNode{
		Segment:    RouteSegment{Name: "", APIName: ""},
		Children:   make(map[string]*RouteNode),
		FullPath:   "",
		FolderPath: "",
		Depth:      0,
		Methods:    []string{},
		ParsedFile: nil,
	}
	rt.Routes = []Route{}
}

func ParseSegment(folderName string) RouteSegment {
	segment := RouteSegment{Name: folderName}
	if strings.HasSuffix(folderName, "_") {
		segment.IsParam = true
		segment.ParamName = strings.TrimSuffix(folderName, "_")
		segment.APIName = ":" + segment.ParamName
	} else {
		segment.APIName = folderName
	}
	return segment
}

func (rt *RouteTree) AddRoute(parsed *ParsedFile) {
	cleanPath := filepath.Clean(parsed.RelPath)
	parts := strings.Split(cleanPath, string(filepath.Separator))

	var validParts []string
	for _, part := range parts {
		if part != "" && part != "." {
			validParts = append(validParts, part)
		}
	}
	logger.Debug("Valid parts: %v", validParts)
	if len(validParts) == 0 {
		return
	}

	current := rt.Root
	var apiParts []RouteSegment
	var parameters []string

	for i, part := range validParts {
		segment := ParseSegment(part)
		apiParts = append(apiParts, segment)

		if segment.IsParam {
			parameters = append(parameters, segment.ParamName)
		}

		if child, exists := current.Children[part]; exists {
			current = child
		} else {
			apiPath := make([]string, len(apiParts))
			for j, s := range apiParts {
				apiPath[j] = s.APIName
			}

			newNode := &RouteNode{
				Segment:    segment,
				Children:   make(map[string]*RouteNode),
				Parent:     current,
				FullPath:   strings.Join(apiPath, "/"),
				FolderPath: strings.Join(validParts[:i+1], "/"),
				Depth:      i + 1,
				Methods:    []string{},
				ParsedFile: nil,
			}
			current.Children[part] = newNode
			current = newNode
		}
	}

	current.ParsedFile = parsed
	current.Methods = append(current.Methods, parsed.Methods...)

	route := Route{
		APIPath:    current.FullPath,
		FolderPath: parsed.RelPath,
		Segments:   apiParts,
		Parameters: parameters,
		IsLeaf:     len(current.Children) == 0,
		Methods:    parsed.Methods,
		ParsedFile: parsed,
	}

	rt.Routes = append(rt.Routes, route)
}

func (rt *RouteTree) CalculateOutputPaths(cfg *config.Config, moduleName string) error {
	for i, route := range rt.Routes {
		// Calculate paths based on route structure
		rt.Routes[i].RelativeOutput = filepath.Join("routes", route.FolderPath, "gen_route.go")
		rt.Routes[i].OutputPath = filepath.Join(cfg.Codegen.Go.Output, rt.Routes[i].RelativeOutput)
		rt.Routes[i].ImportPath = fmt.Sprintf("%s/%s/routes/%s", moduleName, cfg.Codegen.Go.Output, route.FolderPath)
		rt.Routes[i].PackageAlias = rt.generatePackageAlias(route.FolderPath)
	}
	return nil
}

func (rt *RouteTree) generatePackageAlias(folderPath string) string {
	// Convert "api/v1/users" to "api_v1_users_route"
	// Replace slashes and other problematic characters with underscores
	alias := strings.ReplaceAll(folderPath, "/", "_")
	alias = strings.ReplaceAll(alias, "-", "_")
	alias = strings.ReplaceAll(alias, " ", "_")
	return alias + "_route"
}

func (rt *RouteTree) PrintTree(level logger.LogLevel) {
	rt.printNode(rt.Root, "", level)
}

func (rt *RouteTree) printNode(node *RouteNode, prefix string, level logger.LogLevel) {
	if node != rt.Root {
		paramInfo := ""
		if node.Segment.IsParam {
			paramInfo = fmt.Sprintf(" (param: %s)", node.Segment.ParamName)
		}
		methodsInfo := ""
		if len(node.Methods) > 0 {
			methods := make([]string, len(node.Methods))
			copy(methods, node.Methods)
			sort.Strings(methods)
			methodsInfo = fmt.Sprintf(" [%s]", strings.Join(methods, ", "))
		}
		logger.GetLogFromLevel(level)("%s%s -> %s%s%s", prefix, node.Segment.Name, node.FullPath, paramInfo, methodsInfo)

	}

	keys := make([]string, 0, len(node.Children))
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		rt.printNode(node.Children[key], prefix+"  ", level)
	}
}
