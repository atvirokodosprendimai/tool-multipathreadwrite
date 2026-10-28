package main

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"

// namedFiles names up to five files and counts the rest, as ADR-148's notices do.
func namedFiles(names []string) string {
	return read.NamedFiles(names)
}
