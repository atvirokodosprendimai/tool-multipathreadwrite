package mcp

import (
	"path/filepath"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ReadReceipt is the receipt mrw_read carries in content[1] for a read it
// answers whole (ADR-023): what was observed, keyed with "/", the problem
// count, how many of those were the caller's own caps (ADR-173), and skipped, counts and unchanged where the read had them. `mrw read
// --json` builds its receipt with this function too (ADR-165), so the two
// surfaces cannot drift; mrw_read adds next_read, index and next_index on its
// own, since only it pages.
func ReadReceipt(observed map[string]seen.Observation, problems, capped int, skipped read.WalkSkipped, counts map[string]int, unchanged []read.Unchanged, sameAsServed []read.SameAsServed) map[string]any {
	m := withCounts(withSkipped(map[string]any{
		"observed": slashKeys(observed, filepath.Separator),
		"problems": problems,
		"capped":   capped,
	}, skipped), counts)
	if len(unchanged) > 0 {
		m["unchanged"] = unchanged
	}
	if len(sameAsServed) > 0 {
		m["same_as_served"] = sameAsServed
	}
	return m
}

// NoMatchReceipt is the receipt of a walk that matched no file: nothing
// observed, the walk's own problems, matches 0.
func NoMatchReceipt(problems int, skipped read.WalkSkipped) map[string]any {
	return withSkipped(map[string]any{
		"observed": map[string]seen.Observation{},
		"problems": problems,
		"matches":  0,
	}, skipped)
}
