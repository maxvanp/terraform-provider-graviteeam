package importid

import "strings"

// Valid reports whether every parsed import ID component contains non-space
// content.
func Valid(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}
