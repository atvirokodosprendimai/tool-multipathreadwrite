package mcp

import (
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-091: a receipt names a root-relative path the way a plan does, with
// "/". read.Run and the engine key a file by filepath.Clean, which on Windows
// is backslashed; only the copies handed to a receipt are converted, so the
// ledger, the ack store and the engine keep the OS keys they were given. Both
// helpers take sep so a test can drive `\` on any platform.

// slash spells p's sep separators as "/".
func slash(p string, sep rune) string {
	return apply.Slash(p, sep)
}

// slashKeys returns m with every key slash-spelled, for mrw_read's observed.
func slashKeys(m map[string]seen.Observation, sep rune) map[string]seen.Observation {
	out := make(map[string]seen.Observation, len(m))
	for k, v := range m {
		out[slash(k, sep)] = v
	}
	return out
}
