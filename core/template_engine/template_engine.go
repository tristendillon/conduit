package template_engine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/shared"
)

type TemplateRef struct {
	Path  string
	IsDir bool
}

func (tr TemplateRef) IsFile() bool {
	return !tr.IsDir
}

func (tr TemplateRef) IsDirectory() bool {
	return tr.IsDir
}

type TemplateEngine struct {
	funcMap template.FuncMap
}

var GlobalFuncMap = template.FuncMap{}

func RegisterGlobalFunc(name string, fn interface{}) {
	GlobalFuncMap[name] = fn
}

func RegisterGlobalFuncs(funcs template.FuncMap) {
	for name, fn := range funcs {
		GlobalFuncMap[name] = fn
	}
}

func getDefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     shared.ToTitle,
		"trim":      strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join":      strings.Join,

		"now":        time.Now,
		"formatTime": func(layout string, t time.Time) string { return t.Format(layout) },
		"date":       func(t time.Time) string { return t.Format("2006-01-02") },
		"datetime":   func(t time.Time) string { return t.Format("2006-01-02 15:04:05") },

		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"env":    os.Getenv,
		"getenv": os.Getenv,
		"add":    func(a, b int) int { return a + b },
		"sub":    func(a, b int) int { return a - b },
		"mul":    func(a, b int) int { return a * b },
		"div":    func(a, b int) int { return a / b },

		"len": func(v interface{}) int { return reflect.ValueOf(v).Len() },
		"first": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice && rv.Len() > 0 {
				return rv.Index(0).Interface()
			}
			return nil
		},
		"last": func(v interface{}) interface{} {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice && rv.Len() > 0 {
				return rv.Index(rv.Len() - 1).Interface()
			}
			return nil
		},
		"not": func(b bool) bool { return !b },
		"and": func(a, b bool) bool { return a && b },
		"or":  func(a, b bool) bool { return a || b },
	}
}

func NewTemplateEngine() *TemplateEngine {
	funcMap := template.FuncMap{}

	for name, fn := range getDefaultFuncMap() {
		funcMap[name] = fn
	}

	for name, fn := range GlobalFuncMap {
		funcMap[name] = fn
	}

	return &TemplateEngine{
		funcMap: funcMap,
	}
}

func NewTemplateEngineWithFuncs(customFuncs template.FuncMap) *TemplateEngine {
	engine := NewTemplateEngine()

	for name, fn := range customFuncs {
		engine.funcMap[name] = fn
	}

	return engine
}
func (te *TemplateEngine) AddFunc(name string, fn interface{}) {
	te.funcMap[name] = fn
}

func (te *TemplateEngine) AddFuncs(funcs template.FuncMap) {
	for name, fn := range funcs {
		te.funcMap[name] = fn
	}
}

func (te *TemplateEngine) GenerateFile(templateRef TemplateRef, outputPath string, data interface{}) error {
	if templateRef.IsDirectory() {
		return fmt.Errorf("cannot generate file from directory reference: %s", templateRef.Path)
	}

	templatePath := filepath.Join("templates", templateRef.Path)
	content, err := TemplateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templateRef.Path)).Funcs(te.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateRef.Path, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateRef.Path, err)
	}

	return nil
}

func (te *TemplateEngine) GenerateFolder(templateRef TemplateRef, outputDir string, data interface{}) error {
	if templateRef.IsFile() {
		return fmt.Errorf("cannot generate folder from file reference: %s", templateRef.Path)
	}

	templateDir := filepath.Join("templates", templateRef.Path)
	logger.Debug("Generating folder from template reference: %s", templateDir)

	return fs.WalkDir(TemplateFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		logger.Debug("Generating file from path: %s", path)
		if err != nil {
			return err
		}

		if path == templateDir {
			return nil
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		outputPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(outputPath, os.ModePerm)
		}

		return te.generateFileFromPath(path, outputPath, data)
	})
}

func (te *TemplateEngine) generateFileFromPath(templatePath, outputPath string, data interface{}) error {
	content, err := TemplateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	if !strings.HasSuffix(templatePath, ".tmpl") {
		if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		return os.WriteFile(outputPath, content, 0644)
	}

	outputPath = strings.TrimSuffix(outputPath, ".tmpl")

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(te.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return nil
}

func (te *TemplateEngine) ListTemplates(templateRef TemplateRef) ([]string, error) {
	if templateRef.IsFile() {
		return []string{templateRef.Path}, nil
	}

	var templates []string
	templateDir := filepath.Join("templates", templateRef.Path)

	err := fs.WalkDir(TemplateFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			relPath, err := filepath.Rel("templates", path)
			if err != nil {
				return err
			}
			templates = append(templates, relPath)
		}

		return nil
	})

	return templates, err
}

func (te *TemplateEngine) ValidateTemplate(templateRef TemplateRef) error {
	templatePath := filepath.Join("templates", templateRef.Path)

	info, err := fs.Stat(TemplateFS, templatePath)
	if err != nil {
		return fmt.Errorf("template not found: %s", templateRef.Path)
	}

	if info.IsDir() != templateRef.IsDirectory() {
		return fmt.Errorf("template reference type mismatch for %s: expected dir=%t, got dir=%t",
			templateRef.Path, templateRef.IsDirectory(), info.IsDir())
	}

	return nil
}
