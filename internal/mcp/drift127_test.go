package mcp

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-127, the in-process review of #342. The MCP text names the other
// writes that landed while the check ran, and says nothing when none did.
func TestTheMCPTextNamesOtherWritersDuringTheCheck(t *testing.T) {
	passed := &check.Result{Ran: true, ExitCode: 0, Command: "true"}
	if _, tail := checkReport(passed, nil, nil, 2); !strings.Contains(tail, "drift: 2 other write(s) landed in this checkout while the check ran") {
		t.Errorf("two other writes were not named: %q", tail)
	}
	if _, tail := checkReport(passed, nil, nil, 0); strings.Contains(tail, "other write") {
		t.Errorf("no other write, and the text named some: %q", tail)
	}
}
