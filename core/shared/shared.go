package shared

import "strings"

func ToTitle(s string) string {
	first := strings.ToUpper(s[:1])
	rest := s[1:]
	return first + rest
}
