package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-132 Decision 5. ADR-128 cut the --force clause from mrw_write's read
// refusal and left "Run `mrw read X` first" — the CLI's command, on a surface
// whose caller may have no shell.
func TestAnMCPRefusalNamesMrwRead(t *testing.T) {
	root := checkTree113(t, "")
	if err := os.WriteFile(filepath.Join(root, "unread.md"), []byte("never served\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := firstText(t, call(t, root, "mrw_write", map[string]any{"plan": "@@ unread.md 1 replace\nx\n"}))
	if !strings.Contains(text, "has not been read") || !strings.Contains(text, "mrw_read") || strings.Contains(text, "`mrw read") {
		t.Errorf("the MCP read refusal does not name mrw_read, or names the CLI's mrw read:\n%s", text)
	}
}

// ADR-132 Decision 6, on mrw_write: a stopped check was headed FAILED.
func TestAStoppedCheckIsHeadedByWhatStoppedItOverMCP(t *testing.T) {
	for _, tc := range []struct {
		r    check.Result
		want string
	}{
		{check.Result{Ran: true, Command: "sleep 30", Skipped: check.Interrupted, ExitCode: -1}, "check INTERRUPTED"},
		{check.Result{Ran: true, Command: "sleep 30", Skipped: "timed out after 5s", ExitCode: -1}, "check TIMED OUT"},
		{check.Result{Ran: false, Command: "sleep 30", Skipped: check.TimedOutBeforeStart, ExitCode: -1}, "check TIMED OUT"},
	} {
		lead, _ := checkReport(&tc.r, nil, nil, 0)
		if !strings.Contains(lead, tc.want) || strings.Contains(lead, "FAILED") || strings.Contains(lead, "DID NOT RUN") {
			t.Errorf("%q: want %q, got %q", tc.r.Skipped, tc.want, lead)
		}
	}
	if lead, _ := checkReport(&check.Result{Ran: true, Command: "false", ExitCode: 1}, nil, nil, 0); !strings.Contains(lead, "check FAILED") {
		t.Errorf("the pair: a failed check is no longer headed FAILED: %q", lead)
	}
}
