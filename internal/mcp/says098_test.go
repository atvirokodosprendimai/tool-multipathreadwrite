package mcp

import (
	"strings"
	"testing"
)

// ADR-098 T2. The 2026-09-29 audit found sentences the server sends that state
// a behaviour the code does not have. One subtest per row names the row, so a
// correction that goes back says which.

// servedDescription is one tool's description, or one of its inputSchema
// properties' when prop is set.
func servedDescription(t *testing.T, name, prop string) string {
	t.Helper()
	for _, tl := range tools() {
		if tl.Name != name {
			continue
		}
		if prop == "" {
			return tl.Description
		}
		props, _ := tl.InputSchema.(map[string]any)["properties"].(map[string]any)
		p, _ := props[prop].(map[string]any)
		d, _ := p["description"].(string)
		if d == "" {
			t.Fatalf("%s has no described property %q", name, prop)
		}
		return d
	}
	t.Fatalf("no tool %q", name)
	return ""
}

func TestTheServedTextSaysWhatTheCodeDoes(t *testing.T) {
	ins := instructionsText()
	plan := servedDescription(t, "mrw_write", "plan")
	rows := []struct {
		row, text string
		has       []string
		hasNot    []string
	}{
		// next_index is always present and empty on the last page (tools.go
		// matchIndex); next_read really is absent, and stays so.
		{"H7 after", servedDescription(t, "mrw_read", "after"), []string{"`next_index` is empty", "`ast_grep`"}, []string{"is absent"}},
		{"H7 instructions keep next_read", ins, []string{"next_read is absent"}, nil},
		// A dry run reports ok with nothing written (apply.go).
		{"M7 hunks.status", writeDescriptions["hunks.status"], []string{"dry run"}, nil},
		// create, unlink and rename refuse anchor= and lines= (plan.go).
		{"M8 plan guards", plan, []string{"sha=<hex> is checked on every op", "refused on create, unlink and rename"}, []string{"Guards, checked on every op"}},
		{"M8 instructions guards", ins, []string{"refused on create, unlink and rename"}, []string{"Guards, checked on every op"}},
		// exclude without a finder is refused (tools.go).
		{"M9 exclude", servedDescription(t, "mrw_read", "exclude"), []string{"Refused without `grep` or `ast_grep`"}, []string{"Only meaningful"}},
		// With ast_grep, specs are paths to search and carry no range.
		{"M10 specs", servedDescription(t, "mrw_read", "specs"), []string{"`grep` or `ast_grep`", "carry no range"}, nil},
		// The write checks after a code write when a check exists, on both
		// surfaces since ADR-113; check: false turns it off here.
		{"M11 write", servedDescription(t, "mrw_write", ""), []string{"After a write that touches code", "check: false turns it off"}, []string{"this tool runs none"}},
		{"M11 instructions", ins, []string{"--check on prose"}, []string{"a check after code writes (--check)"}},
		// rename: address `-` and one body line, the destination (plan.go).
		{"M12 rename", plan, []string{"rename takes address `-` and one body line"}, nil},
		// ADR-052: a multi-line replace needs the line after its range served,
		// except when the range ends at the last line (apply.go).
		{"M13 served line after End", plan, []string{"the line after its range served", "unless the range ends at the file's last line"}, nil},
		// Elision drops ok and skipped verdicts, then UNWRITTEN file records;
		// failed hunks and written files are kept (tools.go boundedReceipt).
		{"M14 elided", writeDescriptions["elided"], []string{"NOT written", "every WRITTEN file"}, []string{"and file records after them"}},
		{"M14 instructions", ins, []string{"drops the check's tail, then ok and skipped verdicts, then UNWRITTEN files"}, []string{"drops successes then"}},
	}
	for _, r := range rows {
		t.Run(r.row, func(t *testing.T) {
			for _, s := range r.has {
				if !strings.Contains(r.text, s) {
					t.Errorf("%s: missing %q in:\n%s", r.row, s, r.text)
				}
			}
			for _, s := range r.hasNot {
				if strings.Contains(r.text, s) {
					t.Errorf("%s: still says %q in:\n%s", r.row, s, r.text)
				}
			}
		})
	}
	t.Run("M15 one copy of the routing sentence", func(t *testing.T) {
		if n := strings.Count(ins, "With a shell prefer the CLI"); n != 1 {
			t.Errorf("the instructions say %q %d times, want once", "With a shell prefer the CLI", n)
		}
	})
}
