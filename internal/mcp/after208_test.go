package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-208 T1, over the server: the same refusal on mrw_write, landing once the
// line after was read and acknowledged. A rewrite made by mrw_write itself
// (ast_grep with rewrite) is not judged; the plan mrw_read returns for the same
// arguments, sent later, is.
func TestMrwWriteRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown(t *testing.T) {
	const (
		fGo   = "func A() {\n\tif x {\n\t}\n}\n"
		rep   = "@@ f.go 1-3 replace anchor=\"func A\"\nfunc A() {\n\ty()\n}\n"
		aGo   = "package a\n\nfunc f() {\n\tfoo(1, 2)\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; foo(s, \"ŝ\")\n\t_ = x\n}\n"
		aWant = "package a\n\nfunc f() {\n\tbar(2, 1)\n\tx := bar(4, 3)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; bar(\"ŝ\", s)\n\t_ = x\n}\n"
		// what ast-grep 0.45.3 printed for these arguments on aGo (contract §264)
		stream = `{"text":"foo(1, 2)","range":{"byteOffset":{"start":23,"end":32},"start":{"line":3,"column":1},"end":{"line":3,"column":10}},"file":"a.go","lines":"\tfoo(1, 2)","charCount":{"leading":1,"trailing":0},"replacement":"bar(2, 1)","replacementOffsets":{"start":23,"end":32},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
{"text":"foo(\n\t\t3,\n\t\t4)","range":{"byteOffset":{"start":39,"end":53},"start":{"line":4,"column":6},"end":{"line":6,"column":4}},"file":"a.go","lines":"\tx := foo(\n\t\t3,\n\t\t4)","charCount":{"leading":6,"trailing":0},"replacement":"bar(4, 3)","replacementOffsets":{"start":39,"end":53},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
{"text":"foo(s, \"ŝ\")","range":{"byteOffset":{"start":93,"end":105},"start":{"line":8,"column":15},"end":{"line":8,"column":26}},"file":"a.go","lines":"\ts := \"naïve\"; foo(s, \"ŝ\")","charCount":{"leading":15,"trailing":0},"replacement":"bar(\"ŝ\", s)","replacementOffsets":{"start":93,"end":105},"language":"Go","metaVariables":{"single":{},"multi":{},"transformed":{}}}
`
	)
	create := func(t *testing.T, root, name, body string) {
		t.Helper()
		res := call(t, root, "mrw_write", map[string]any{"plan": "@@ " + name + " 0 create\n" + strings.TrimSuffix(body, "\n") + "\n"})
		if res["isError"] == true {
			t.Fatalf("create %s: %v", name, res)
		}
	}
	slurp := func(t *testing.T, root, name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	failedHunk := func(t *testing.T, res map[string]any, after int) {
		t.Helper()
		if res["isError"] != true {
			t.Fatalf("the plan was not refused: %v", res)
		}
		hunks, _ := structured(t, res)["hunks"].([]any)
		var failed bool
		for _, h := range hunks {
			m, _ := h.(map[string]any)
			if m["status"] == "failed" && strings.Contains(fmt.Sprint(m["reason"]), fmt.Sprintf("needs a served line after %d", after)) {
				failed = true
			}
		}
		if !failed {
			t.Fatalf("no failed hunk names the line after %d: %v", after, hunks)
		}
	}
	readAcks := func(t *testing.T, root, spec string) []string {
		t.Helper()
		res := call(t, root, "mrw_read", map[string]any{"specs": []string{spec}})
		var ids []string
		for _, id := range checkpointsIn(firstText(t, res)) {
			ids = append(ids, id.(string))
		}
		if len(ids) == 0 {
			t.Fatalf("the read of %s served no checkpoint: %v", spec, res)
		}
		return ids
	}

	t.Run("numbered", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
		create(t, root, "f.go", fGo)
		failedHunk(t, call(t, root, "mrw_write", map[string]any{"plan": rep}), 3)
		if got := slurp(t, root, "f.go"); got != fGo {
			t.Errorf("a refused plan changed f.go: %q", got)
		}
		acks := readAcks(t, root, "f.go:4")
		if res := call(t, root, "mrw_write", map[string]any{"plan": rep, "ack": acks}); res["isError"] == true {
			t.Fatalf("after the read of line 4 and its ack: %v", res)
		}
	})
	t.Run("same_call_rewrite", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
		create(t, root, "a.go", aGo)
		installRecordingAstGrep(t, stream)
		if res := call(t, root, "mrw_write", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)"}); res["isError"] == true {
			t.Fatalf("a same-call rewrite: %v", res)
		}
		if got := slurp(t, root, "a.go"); got != aWant {
			t.Errorf("a.go is %q, want the rewrite", got)
		}
	})
	t.Run("later_call_rewrite", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
		create(t, root, "a.go", aGo)
		installRecordingAstGrep(t, stream)
		planRes := call(t, root, "mrw_read", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)"})
		plan := firstText(t, planRes)
		if !strings.Contains(plan, "@@ a.go 4-7 replace") {
			t.Fatalf("not the plan:\n%s", plan)
		}
		failedHunk(t, call(t, root, "mrw_write", map[string]any{"plan": plan}), 7)
		acks := readAcks(t, root, "a.go:8")
		if res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks}); res["isError"] == true {
			t.Fatalf("after the read of line 8 and its ack: %v", res)
		}
		if got := slurp(t, root, "a.go"); got != aWant {
			t.Errorf("a.go is %q, want the rewrite", got)
		}
	})
}
