package mcp

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-152. The mrw_write text carries the check's hint, and says nothing when
// there is none.
func TestTheMCPTextPrintsTheSlowCheckHint(t *testing.T) {
	hinted := &check.Result{Ran: true, ExitCode: 0, Command: "true", Hint: `declare "scoped_check" here`}
	if _, tail := checkReport(hinted, nil, nil, 0); !strings.Contains(tail, "hint: declare \"scoped_check\" here\n") {
		t.Errorf("the hint is not in the text: %q", tail)
	}
	plain := &check.Result{Ran: true, ExitCode: 0, Command: "true"}
	if _, tail := checkReport(plain, nil, nil, 0); strings.Contains(tail, "hint:") {
		t.Errorf("no hint, and the text has one: %q", tail)
	}
}
