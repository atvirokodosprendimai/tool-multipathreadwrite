package writer

import (
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// StopWord is the headline word for a check something stopped before it gave
// a verdict (ADR-132): INTERRUPTED, TIMED OUT, or "" for a check that ran to
// its exit or could not start. Such a check was headed FAIL while no process
// had exited with one. The CLI and mrw_write head the same check alike.
func StopWord(skipped string) string {
	switch {
	case skipped == check.Interrupted:
		return "INTERRUPTED"
	case strings.HasPrefix(skipped, "timed out"):
		return "TIMED OUT"
	}
	return ""
}
