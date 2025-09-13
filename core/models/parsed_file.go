package models

type ParsedFile struct {
	Path        string
	RelPath     string
	PackageName string
	Methods     []string
}
