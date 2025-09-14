package models

type ExtractedFunction struct {
	Name      string
	Method    string
	Signature string
	Body      string
}

type ParsedFile struct {
	Path        string
	RelPath     string
	PackageName string
	Methods     []string
	Functions   []ExtractedFunction
	Imports     []string
}
