package mcp

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"

// LicensedNotShown marks what a --quiet read observed as licensed and not
// shown, the way a --sub plan's lines are (ADR-161, ADR-118): a write to those
// lines is allowed, and a from= copy of them is not, because the caller was
// shown none (Zy, 2026-10-22, ADR-167). The ledger cannot tell a quiet read from
// a shown one, so the two record sites say so here. The input is not changed.
func LicensedNotShown(observed map[string]seen.Observation) map[string]seen.Observation {
	out := make(map[string]seen.Observation, len(observed))
	for path, o := range observed {
		o.Written, o.Shown = true, [][2]int{}
		out[path] = o
	}
	return out
}
