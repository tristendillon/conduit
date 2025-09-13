package models

type DiscoveredFile struct {
	Path  string
	Route *RouteInfo
}

type RouteInfo struct {
	PackageName string
	Funcs       []string
	Imports     []string
}
