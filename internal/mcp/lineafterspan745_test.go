package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Final pass of #745. The refusal over MCP is built from the range the engine
// resolved, not from the address text: `1,+1` and `/^one$/,/^two$/` are ranges no
// number parser can read, and a pending page for the line after was missed. And
// through an alias the ack that lifts the refusal is one for the range through the
// line after: the alias's own record holds nothing else.
func TestTheLineAfterRefusalOverMCPReadsTheResolvedRange(t *testing.T) {
	const body = "one\ntwo\nthree\nfour\nfive"
	reasonOf := func(t *testing.T, res map[string]any) string {
		t.Helper()
		if res["isError"] != true {
			t.Fatalf("the plan was not refused: %v", res)
		}
		hunks, _ := structured(t, res)["hunks"].([]any)
		for _, h := range hunks {
			if m, _ := h.(map[string]any); m["status"] == "failed" {
				return fmt.Sprint(m["reason"])
			}
		}
		t.Fatalf("no failed hunk: %v", hunks)
		return ""
	}
	for _, c := range []struct{ name, plan string }{
		{"relative_end", "@@ f.txt 1,+1 replace anchor=\"one\"\nx\ny\n"},
		{"pattern_range", "@@ f.txt /^one$/,/^two$/ replace anchor=\"one\"\nx\ny\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
			if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 0 create\n" + body + "\n"}); res["isError"] == true {
				t.Fatalf("create: %v", res)
			}
			reason := reasonOf(t, call(t, root, "mrw_write", map[string]any{"plan": c.plan}))
			if strings.Contains(reason, "mrw read") || !strings.Contains(reason, "mrw_read f.txt lines 1-3") {
				t.Errorf("with nothing served: %s", reason)
			}
			call(t, root, "mrw_read", map[string]any{"specs": []string{"f.txt:3"}})
			reason = reasonOf(t, call(t, root, "mrw_write", map[string]any{"plan": c.plan}))
			if !strings.Contains(reason, "served but never acknowledged") {
				t.Errorf("with line 3 served and not acknowledged: %s", reason)
			}
		})
	}
}

func TestTheLineAfterAckThroughAnAliasNamesTheRangeItNeeds(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ real.go 0 create\nfunc A() {\n\tif x {\n\t}\n}\n"}); res["isError"] == true {
		t.Fatalf("create: %v", res)
	}
	if err := os.Symlink("real.go", filepath.Join(root, "link.go")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	plan := "@@ link.go 1-3 replace anchor=\"func A\"\nfunc A() {\n\ty()\n}\n"
	reasonOf := func() string {
		res := call(t, root, "mrw_write", map[string]any{"plan": plan})
		hunks, _ := structured(t, res)["hunks"].([]any)
		for _, h := range hunks {
			if m, _ := h.(map[string]any); m["status"] == "failed" {
				return fmt.Sprint(m["reason"])
			}
		}
		t.Fatalf("not refused: %v", res)
		return ""
	}
	// Only the line after served through the alias: acking it would leave lines 1-3
	// unshown under link.go, so the hint is not made.
	call(t, root, "mrw_read", map[string]any{"specs": []string{"link.go:4"}})
	if reason := reasonOf(); strings.Contains(reason, "served but never acknowledged") || !strings.Contains(reason, "mrw_read link.go lines 1-4") {
		t.Errorf("with only line 4 served through the alias: %s", reason)
	}
	// The whole range through the line after served: one ack lifts it.
	res := call(t, root, "mrw_read", map[string]any{"specs": []string{"link.go:1-4"}})
	if reason := reasonOf(); !strings.Contains(reason, "served but never acknowledged") {
		t.Errorf("with lines 1-4 served through the alias: %s", reason)
	}
	var acks []string
	for _, id := range checkpointsIn(firstText(t, res)) {
		acks = append(acks, id.(string))
	}
	if res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks}); res["isError"] == true {
		t.Errorf("after the ack the refusal's hint named: %v", res)
	}
}
