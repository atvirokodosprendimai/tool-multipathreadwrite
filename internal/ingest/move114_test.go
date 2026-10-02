package ingest

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// ADR-114 T2. Codex apply_patch moves and edits a file in one section —
// Update File, Move to, hunks — and the compiler refused it. It now emits
// the hunks against the source and one rename, which the engine applies in
// one plan, whether Move to comes before the hunks or after them.
func TestAMoveWithHunksCompiles(t *testing.T) {
	for name, doc := range map[string]string{
		"move first": "*** Begin Patch\n*** Update File: a.go\n*** Move to: b.go\n@@\n-func A() int { return 1 }\n+func A() int { return 10 }\n*** End Patch\n",
		"move after": "*** Begin Patch\n*** Update File: a.go\n@@\n-func A() int { return 1 }\n+func A() int { return 10 }\n*** Move to: b.go\n*** End Patch\n",
	} {
		root := writeTree(t, map[string]string{"a.go": demo})
		observed, _ := read.Run(io.Discard, root, []read.Spec{{Path: "a.go"}}, read.Options{})
		planText, err := CompileApplyPatch(root, []byte(doc))
		if err != nil {
			t.Fatalf("%s: compile refused a move with a hunk: %v", name, err)
		}
		res := applyCompiled(t, root, planText, observed)
		if !res.Applied {
			t.Fatalf("%s: the compiled plan did not apply: %+v\n%s", name, res.Hunks, planText)
		}
		got, err := os.ReadFile(filepath.Join(root, "b.go"))
		if err != nil || !strings.Contains(string(got), "return 10") {
			t.Errorf("%s: b.go = %q (%v), want the edit", name, got, err)
		}
		if _, err := os.Lstat(filepath.Join(root, "a.go")); err == nil {
			t.Errorf("%s: a.go is still there", name)
		}
	}
	if _, err := CompileApplyPatch(t.TempDir(), []byte("*** Begin Patch\n*** Move to: b.go\n*** End Patch\n")); err == nil {
		t.Error("a Move to with no Update File compiled")
	}
	twice := "*** Begin Patch\n*** Update File: a.go\n*** Move to: b.go\n*** Move to: c.go\n*** End Patch\n"
	if _, err := CompileApplyPatch(writeTree(t, map[string]string{"a.go": demo}), []byte(twice)); err == nil {
		t.Error("two Move to lines in one section compiled")
	}
}
