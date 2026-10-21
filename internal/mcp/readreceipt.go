package mcp

import (
	"path/filepath"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ReadReceipt is the receipt mrw_read carries in content[1] for a read it
// answers whole (ADR-023): what was observed, keyed with "/", the problem
// count, and skipped and counts where the read had them. `mrw read --json`
// builds its receipt with this function too (ADR-165), so the two surfaces
// cannot drift; mrw_read adds next_read, index and next_index on its own,
// since only it pages.
func ReadReceipt(observed map[string]seen.Observation, problems int, skipped read.WalkSkipped, counts map[string]int) map[string]any {
	return withCounts(withSkipped(map[string]any{
		"observed": slashKeys(observed, filepath.Separator),
		"problems": problems,
	}, skipped), counts)
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
