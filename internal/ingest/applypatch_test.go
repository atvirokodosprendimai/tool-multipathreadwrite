package ingest

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

const demo = "package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n"

const twoHunkPatch = "*** Begin Patch\n" +
	"*** Update File: a.go\n" +
	"@@\n" +
	"-func A() int { return 1 }\n" +
	"+func A() int { return 10 }\n" +
	"@@\n" +
	"-func C() int { return 3 }\n" +
	"+func C() int { return 30 }\n" +
	"*** End Patch\n"

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func applyCompiled(t *testing.T, root string, planText []byte, observed map[string]apply.Seen) apply.Result {
	t.Helper()
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("compiled plan did not parse: %v\n%s", err, planText)
	}
	in := make([]apply.Input, 0, len(hunks))
	for _, h := range hunks {
		in = append(in, apply.Input{
			Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd,
			CountedBody: h.CountedBody, Body: h.Body, SHA: h.SHA, Lines: h.Lines,
			Anchor: h.Anchor, SrcLine: h.SrcLine, Index: h.Index,
		})
	}
	res, err := apply.Apply(root, in, apply.Options{Seen: observed})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing is ADR-051 T1: compile
// succeeds, Apply refuses the unread sibling, and the file is unchanged.
func TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "a.go", Ranges: []read.Range{{Start: 3, End: 3}}}}, read.Options{})

	planText, err := CompileApplyPatch(root, []byte(twoHunkPatch))
	if err != nil {
		t.Fatalf("compile refused a locatable two-hunk patch: %v", err)
	}
	res := applyCompiled(t, root, planText, observed)
	if res.Failed != 1 {
		t.Fatalf("want 1 failed hunk, got failed=%d applied=%v: %+v", res.Failed, res.Applied, res.Hunks)
	}
	if res.Applied {
		t.Fatal("the run reported applied")
	}
	var sawFail, sawSkip bool
	for _, h := range res.Hunks {
		switch h.Status {
		case apply.StatusFailed:
			sawFail = true
			if !strings.Contains(h.Reason, "has not been read") {
				t.Errorf("the refusal is not the ledger's: %s", h.Reason)
			}
		case apply.StatusSkipped:
			sawSkip = true
		case apply.StatusOK:
			t.Errorf("a sibling reported ok: %+v", h)
		}
	}
	if !sawFail || !sawSkip {
		t.Errorf("want FAIL+skip, got %+v", res.Hunks)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demo {
		t.Errorf("the file was written:\n%s", got)
	}
}

func TestCompileApplyPatchGoesThroughParse(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	planText, err := CompileApplyPatch(root, []byte(twoHunkPatch))
	if err != nil {
		t.Fatal(err)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("plan.Parse refused compiled text: %v\n%s", err, planText)
	}
	if len(hunks) != 2 {
		t.Fatalf("got %d hunks, want 2:\n%s", len(hunks), planText)
	}
	if hunks[0].Path != "a.go" || string(hunks[0].Op) != "replace" || hunks[0].Addr.Start != 3 {
		t.Errorf("first hunk = %+v", hunks[0])
	}
	if hunks[1].Addr.Start != 5 {
		t.Errorf("second hunk start = %d, want 5", hunks[1].Addr.Start)
	}
}

func TestAnAmbiguousOldSideIsACompileRefusal(t *testing.T) {
	root := writeTree(t, map[string]string{"a.txt": "line\nline\nline\n"})
	doc := "*** Begin Patch\n*** Update File: a.txt\n@@\n-line\n+LINE\n*** End Patch\n"
	_, err := CompileApplyPatch(root, []byte(doc))
	if err == nil {
		t.Fatal("an old side that matches three times compiled")
	}
	if !strings.Contains(err.Error(), "matched 3 times") {
		t.Errorf("the refusal does not name the ambiguity: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "line\nline\nline\n" {
		t.Errorf("compile wrote the file:\n%s", got)
	}
}

func TestAGitPatchIsNotAnApplyPatch(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	doc := "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -3,1 +3,1 @@\n-func A() int { return 1 }\n+func A() int { return 10 }\n"
	_, err := CompileApplyPatch(root, []byte(doc))
	if err == nil {
		t.Fatal("a git patch compiled as apply_patch")
	}
	if !strings.Contains(err.Error(), "git patch is not an apply_patch") {
		t.Errorf("the refusal does not name the grammar: %v", err)
	}
}
