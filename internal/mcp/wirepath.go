package mcp

import (
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// slashKeys returns m with every key's sep separators spelled "/", for the
// wire. ADR-091: read.Run keys an observation by filepath.Clean, which on
// Windows is backslashed, while a plan, hunks.path and a grep index all spell
// a root-relative path with "/". Only the copy handed to a receipt is
// converted; the ledger and the ack store keep the OS keys read.Run gave them.
// sep is a parameter so a test can drive `\` on any platform.
func slashKeys(m map[string]seen.Observation, sep rune) map[string]seen.Observation {
	out := make(map[string]seen.Observation, len(m))
	for k, v := range m {
		out[strings.ReplaceAll(k, string(sep), "/")] = v
	}
	return out
}
