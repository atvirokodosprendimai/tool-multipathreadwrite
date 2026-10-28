package main

import (
	"fmt"
	"strings"
)

// namedFiles names up to five files and counts the rest, as ADR-148's notices
// do: `a.py, b.py and 2 more`.
func namedFiles(names []string) string {
	if len(names) <= 5 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:5], ", "), len(names)-5)
}
