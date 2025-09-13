package template_refs

import (
	"os"
	"path/filepath"
	"strings"
)

type TemplateNode struct {
	Name     string                   `json:"name"`
	Path     string                   `json:"path"`
	IsDir    bool                     `json:"is_dir"`
	Children map[string]*TemplateNode `json:"children,omitempty"`
	Parent   *TemplateNode            `json:"-"` // Don't serialize parent to avoid cycles
}

type TemplateWalker struct {
	templateDir string
	rootNode    *TemplateNode
}

func NewTemplateWalker(templateDir string) *TemplateWalker {
	return &TemplateWalker{
		templateDir: templateDir,
		rootNode: &TemplateNode{
			Name:     "templates",
			Path:     "",
			IsDir:    true,
			Children: make(map[string]*TemplateNode),
		},
	}
}

func (tw *TemplateWalker) Walk() error {
	return filepath.Walk(tw.templateDir, tw.walkFunc)
}

func (tw *TemplateWalker) walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if path == tw.templateDir {
		return nil
	}

	relPath, err := filepath.Rel(tw.templateDir, path)
	if err != nil {
		return err
	}

	segments := strings.Split(relPath, string(filepath.Separator))

	currentNode := tw.rootNode
	for i, segment := range segments[:len(segments)-1] {
		segmentKey := tw.normalizeKey(segment)
		if currentNode.Children[segmentKey] == nil {
			currentNode.Children[segmentKey] = &TemplateNode{
				Name:     segment,
				Path:     strings.Join(segments[:i+1], "/"),
				IsDir:    true,
				Children: make(map[string]*TemplateNode),
				Parent:   currentNode,
			}
		}
		currentNode = currentNode.Children[segmentKey]
	}

	finalSegment := segments[len(segments)-1]
	finalKey := tw.normalizeKey(finalSegment)

	node := &TemplateNode{
		Name:   finalSegment,
		Path:   relPath,
		IsDir:  info.IsDir(),
		Parent: currentNode,
	}

	if info.IsDir() {
		node.Children = make(map[string]*TemplateNode)
	}

	currentNode.Children[finalKey] = node

	return nil
}

func (tw *TemplateWalker) normalizeKey(name string) string {
	if !strings.Contains(name, "/") {
		if idx := strings.LastIndex(name, "."); idx != -1 {
			name = name[:idx]
		}
	}

	result := strings.ToUpper(name)
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, ".", "_")

	invalidChars := []string{"@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "[", "]", "{", "}", "|", "\\", ":", ";", "\"", "'", "<", ">", ",", "?", "/"}
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	if len(result) > 0 && (result[0] >= '0' && result[0] <= '9') {
		result = "_" + result
	}

	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	result = strings.TrimSuffix(result, "_")

	return result
}
func (tw *TemplateWalker) GetTemplateTree() *TemplateNode {
	return tw.rootNode
}

func (tw *TemplateWalker) GetAllPaths() []string {
	var paths []string
	tw.collectPaths(tw.rootNode, &paths)
	return paths
}

func (tw *TemplateWalker) collectPaths(node *TemplateNode, paths *[]string) {
	if node.Path != "" {
		*paths = append(*paths, node.Path)
	}

	for _, child := range node.Children {
		tw.collectPaths(child, paths)
	}
}

func (tw *TemplateWalker) GetFileNodes() []*TemplateNode {
	var files []*TemplateNode
	tw.collectFileNodes(tw.rootNode, &files)
	return files
}

func (tw *TemplateWalker) collectFileNodes(node *TemplateNode, files *[]*TemplateNode) {
	if !node.IsDir && node.Path != "" {
		*files = append(*files, node)
	}

	for _, child := range node.Children {
		tw.collectFileNodes(child, files)
	}
}

func (tw *TemplateWalker) GetDirectoryNodes() []*TemplateNode {
	var dirs []*TemplateNode
	tw.collectDirectoryNodes(tw.rootNode, &dirs)
	return dirs
}

func (tw *TemplateWalker) collectDirectoryNodes(node *TemplateNode, dirs *[]*TemplateNode) {
	if node.IsDir && node.Path != "" {
		*dirs = append(*dirs, node)
	}

	for _, child := range node.Children {
		tw.collectDirectoryNodes(child, dirs)
	}
}
