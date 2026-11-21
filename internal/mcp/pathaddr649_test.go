package mcp

import (
	"regexp"
	"strings"
	"testing"
)

// #649, the MCP twin of the CLI test: the text rows of a failed and a skipped path op
// name `-`, not the unresolved bound. The skipped rows are the ones a hunk refused
// after validation gets (refuseFile, clashAt, sameFileAs), which carry the Addr of
// their input; a hunk that merely sat beside a failing one was validated first.
func TestAPathOpRowOfTheMCPAnswerPrintsTheAddressThePlanWrote(t *testing.T) {
	zero := regexp.MustCompile(` 0 (unlink|rename)|^(failed|skipped) \S+ 0 `)
	for _, c := range []struct {
		name, plan string
		want       []string
	}{
		{"a failed unlink", "@@ a.txt - unlink\n@@ nope.txt - unlink\n", []string{"failed nope.txt - ", "skipped a.txt - "}},
		{"a failed rename", "@@ nope.txt - rename\nz.txt\n", []string{"failed nope.txt - "}},
		{"skipped beside a directory spelling", "@@ a.txt/ 1 replace\nx\n@@ a.txt - rename\nz.txt\n", []string{"failed a.txt 1 ", "skipped a.txt - "}},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := licensed(t, map[string]string{"a.txt": "a\n"})
			text := resultText(call(t, root, "mrw_write", map[string]any{"plan": c.plan}))
			for _, want := range c.want {
				if !strings.Contains(text, want) {
					t.Errorf("want a row starting %q:\n%s", want, text)
				}
			}
			for _, row := range strings.Split(text, "\n") {
				if zero.MatchString(row) && !strings.Contains(row, "replace") {
					t.Errorf("a row names address 0: %q", row)
				}
			}
		})
	}
}
