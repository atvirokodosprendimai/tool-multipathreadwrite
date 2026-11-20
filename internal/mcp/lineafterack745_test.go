package mcp

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// Deep review of #745. Over the server the read a line-after refusal names is not
// a command to run: it is mrw_read of the range, and, when the line after was
// already served and nobody acknowledged it, the checkpoint ids that are waiting.
func TestTheLineAfterRefusalOverMCPNamesTheReadAndTheAck(t *testing.T) {
	const (
		fGo = "func A() {\n\tif x {\n\t}\n}\n"
		rep = "@@ f.go 1-3 replace anchor=\"func A\"\nfunc A() {\n\ty()\n}\n"
	)
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
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.go 0 create\n" + strings.TrimSuffix(fGo, "\n") + "\n"}); res["isError"] == true {
		t.Fatalf("create: %v", res)
	}
	// Nothing served yet: the read is mrw_read of lines 1-4, not a CLI command.
	reason := reasonOf(t, call(t, root, "mrw_write", map[string]any{"plan": rep}))
	if strings.Contains(reason, "mrw read") || !strings.Contains(reason, "mrw_read f.go lines 1-4") || strings.Contains(reason, "never acknowledged") {
		t.Errorf("with nothing served: %s", reason)
	}
	// The line after served and not acknowledged: the refusal says to send the ack.
	call(t, root, "mrw_read", map[string]any{"specs": []string{"f.go:4"}})
	reason = reasonOf(t, call(t, root, "mrw_write", map[string]any{"plan": rep}))
	if !strings.Contains(reason, "served but never acknowledged") || !strings.Contains(reason, "ack") || strings.Contains(reason, "mrw read") {
		t.Errorf("with line 4 served and not acknowledged: %s", reason)
	}
}
