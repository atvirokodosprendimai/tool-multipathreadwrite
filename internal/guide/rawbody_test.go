package guide

import (
	"strings"
	"testing"
)

// The CLI guide said nothing of the one silent case: a body line that is itself
// a valid hunk header starts a new hunk, so a body holding one needs body=N and
// raw=true. Only the MCP instructions said so (Codex review of PR #502,
// 2026-10-22), so an agent with only the binary could not learn it.
func TestCLITeachesThatAHeaderInABodyNeedsACount(t *testing.T) {
	got := CLI()
	for _, must := range []string{"raw=true", "starts a new hunk"} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not say %q", must)
		}
	}
}
