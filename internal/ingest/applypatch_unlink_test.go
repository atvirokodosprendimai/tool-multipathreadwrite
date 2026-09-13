package ingest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

func TestCompileDeleteFileIsUnlink(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	doc := "*** Begin Patch\n*** Delete File: a.go\n*** End Patch\n"
	planText, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("Delete File compile refused: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demo {
		t.Errorf("compile wrote the file:\n%s", got)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("compiled unlink did not parse: %v\n%s", err, planText)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1:\n%s", len(hunks), planText)
	}
	h := hunks[0]
	if h.Path != "a.go" || h.Op != plan.OpUnlink {
		t.Fatalf("hunk = %+v, want path a.go op unlink\n%s", h, planText)
	}
	if h.Addr != (plan.Addr{Start: 0, End: 0}) {
		t.Errorf("addr = %+v, want -", h.Addr)
	}
	if len(h.Body) != 0 {
		t.Errorf("unlink body = %q, want empty", h.Body)
	}
	if !strings.Contains(string(planText), "@@ a.go - unlink") {
		t.Errorf("compiled text is not the native unlink hunk:\n%s", planText)
	}
}

func TestCompileMoveToIsRename(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	doc := "*** Begin Patch\n*** Update File: a.go\n*** Move to: b.go\n*** End Patch\n"
	planText, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("Move to compile refused: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demo {
		t.Errorf("compile wrote the file:\n%s", got)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("compiled rename did not parse: %v\n%s", err, planText)
	}
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks, want 1:\n%s", len(hunks), planText)
	}
	h := hunks[0]
	if h.Path != "a.go" || h.Op != plan.OpRename {
		t.Fatalf("hunk = %+v, want path a.go op rename\n%s", h, planText)
	}
	if len(h.Body) != 1 || h.Body[0] != "b.go" {
		t.Fatalf("body = %q, want one dest b.go\n%s", h.Body, planText)
	}
	if !strings.Contains(string(planText), "@@ a.go - rename") {
		t.Errorf("compiled text is not the native rename hunk:\n%s", planText)
	}

	withHunks := "*** Begin Patch\n*** Update File: a.go\n*** Move to: b.go\n@@\n-func A() int { return 1 }\n+func A() int { return 10 }\n*** End Patch\n"
	if _, err := CompileApplyPatch(root, []byte(withHunks)); err == nil {
		t.Fatal("Move to with extra @@ hunks compiled")
	} else if !strings.Contains(err.Error(), "hunks") {
		t.Errorf("Move to with hunks refusal does not name hunks: %v", err)
	}
}
