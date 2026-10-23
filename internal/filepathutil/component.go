package filepathutil

import "strings"

// IsComponent reports whether name is a single portable path component.
func IsComponent(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\:\x00")
}
