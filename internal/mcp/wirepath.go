package mcp

import (
	"slices"
	"strings"

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
	return strings.ReplaceAll(p, string(sep), "/")
}

// slashKeys returns m with every key slash-spelled, for mrw_read's observed.
func slashKeys(m map[string]seen.Observation, sep rune) map[string]seen.Observation {
	out := make(map[string]seen.Observation, len(m))
	for k, v := range m {
		out[slash(k, sep)] = v
	}
	return out
}

// slashResult returns res with every root-relative path slash-spelled, for
// mrw_write's receipt: hunk and file paths, a rename's destination, a
// symlink's target and the directories a plan made. The slices are copied, so
// the engine's result is not changed. Root stays an OS path: it is absolute,
// and a caller hands it to its own filesystem.
func slashResult(res apply.Result, sep rune) apply.Result {
	res.Hunks = slices.Clone(res.Hunks)
	for i := range res.Hunks {
		res.Hunks[i].Path = slash(res.Hunks[i].Path, sep)
	}
	res.Files = slices.Clone(res.Files)
	for i := range res.Files {
		f := &res.Files[i]
		f.Path, f.RenamedTo, f.Target = slash(f.Path, sep), slash(f.RenamedTo, sep), slash(f.Target, sep)
	}
	res.DirsCreated = slices.Clone(res.DirsCreated)
	for i, d := range res.DirsCreated {
		res.DirsCreated[i] = slash(d, sep)
	}
	return res
}
