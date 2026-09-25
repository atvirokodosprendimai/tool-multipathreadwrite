package mcp

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-070 T1. The handshake's worked plan is the one example an MCP host is
// shown; it counts its body on the header, and still parses.
func TestTheHandshakeShowsBodyOnAHeader(t *testing.T) {
	header := regexp.MustCompile(`(?m)^@@ \S+ \S+ \S+ .*\bbody=\d+`)
	if !header.MatchString(examplePlan) {
		t.Errorf("the handshake's worked plan counts no body on a header:\n%s", examplePlan)
	}
	if _, err := plan.Parse(bytes.NewReader([]byte(examplePlan))); err != nil {
		t.Errorf("the worked plan does not parse: %v", err)
	}
}
