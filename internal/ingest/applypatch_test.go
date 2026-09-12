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

// TestCompileApplyPatchRules binds the remaining compile-rule facts. Each
// subtest is a hole the existing T1 tests did not name.
func TestCompileApplyPatchRules(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})

	t.Run("envelope rejects text before and after", func(t *testing.T) {
		before := "note\n" + twoHunkPatch
		if _, err := CompileApplyPatch(root, []byte(before)); err == nil || !strings.Contains(err.Error(), "text before") {
			t.Fatalf("text before Begin Patch: %v", err)
		}
		after := strings.TrimSuffix(twoHunkPatch, "\n") + "\ntrailer\n"
		if _, err := CompileApplyPatch(root, []byte(after)); err == nil || !strings.Contains(err.Error(), "text after") {
			t.Fatalf("text after End Patch: %v", err)
		}
	})

	t.Run("delete and move are compile refusals", func(t *testing.T) {
		del := "*** Begin Patch\n*** Delete File: a.go\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(del)); err == nil || !strings.Contains(err.Error(), "Delete File") {
			t.Fatalf("Delete File: %v", err)
		}
		mov := "*** Begin Patch\n*** Update File: a.go\n*** Move to: b.go\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(mov)); err == nil || !strings.Contains(err.Error(), "Move to") {
			t.Fatalf("Move to: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(root, "a.go"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != demo {
			t.Errorf("delete/move wrote the file:\n%s", got)
		}
	})

	t.Run("end of file is not a marker", func(t *testing.T) {
		doc := "*** Begin Patch\n*** End of File\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(doc)); err == nil {
			t.Fatal("*** End of File compiled")
		}
	})

	t.Run("change context after at-at is discarded", func(t *testing.T) {
		doc := "*** Begin Patch\n*** Update File: a.go\n@@ func A\n-func A() int { return 1 }\n+func A() int { return 10 }\n*** End Patch\n"
		planText, err := CompileApplyPatch(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(bytes.NewReader(planText))
		if err != nil {
			t.Fatal(err)
		}
		if len(hunks) != 1 || hunks[0].Addr.Start != 3 {
			t.Fatalf("trailer was used as location: %+v\n%s", hunks, planText)
		}
	})

	t.Run("zero matches and empty old side refuse", func(t *testing.T) {
		miss := "*** Begin Patch\n*** Update File: a.go\n@@\n-func Z() int { return 0 }\n+func Z() int { return 1 }\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(miss)); err == nil || !strings.Contains(err.Error(), "matched no lines") {
			t.Fatalf("zero match: %v", err)
		}
		empty := "*** Begin Patch\n*** Update File: a.go\n@@\n+func Z() int { return 1 }\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(empty)); err == nil || !strings.Contains(err.Error(), "no old side") {
			t.Fatalf("empty old side: %v", err)
		}
	})

	t.Run("document crlf becomes lf and file crlf does not", func(t *testing.T) {
		crlfDoc := strings.ReplaceAll(twoHunkPatch, "\n", "\r\n")
		planText, err := CompileApplyPatch(root, []byte(crlfDoc))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(planText, []byte("\r")) {
			t.Fatalf("compiled plan kept CR:\n%q", planText)
		}
		crRoot := writeTree(t, map[string]string{"a.go": strings.ReplaceAll(demo, "\n", "\r\n")})
		if _, err := CompileApplyPatch(crRoot, []byte(twoHunkPatch)); err == nil || !strings.Contains(err.Error(), "matched no lines") {
			t.Fatalf("LF old side against CR LF file: %v", err)
		}
	})

	t.Run("backslash hunk lines are skipped", func(t *testing.T) {
		doc := "*** Begin Patch\n*** Update File: a.go\n@@\n-func A() int { return 1 }\n\\ No newline at end of file\n+func A() int { return 10 }\n\\ No newline at end of file\n*** End Patch\n"
		planText, err := CompileApplyPatch(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(planText), `\ No newline`) {
			t.Fatalf("backslash marker leaked into the plan:\n%s", planText)
		}
	})

	t.Run("empty add is create body zero", func(t *testing.T) {
		doc := "*** Begin Patch\n*** Add File: empty.txt\n*** End Patch\n"
		planText, err := CompileApplyPatch(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(bytes.NewReader(planText))
		if err != nil {
			t.Fatalf("empty add did not parse: %v\n%s", err, planText)
		}
		if len(hunks) != 1 || string(hunks[0].Op) != "create" || hunks[0].Path != "empty.txt" || len(hunks[0].Body) != 0 {
			t.Fatalf("empty add = %+v\n%s", hunks, planText)
		}
	})

	t.Run("add file plus lines and multi-line anchor", func(t *testing.T) {
		add := "*** Begin Patch\n*** Add File: new.txt\n+hello\n+world\n*** End Patch\n"
		planText, err := CompileApplyPatch(root, []byte(add))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(bytes.NewReader(planText))
		if err != nil {
			t.Fatal(err)
		}
		if len(hunks) != 1 || string(hunks[0].Op) != "create" || strings.Join(hunks[0].Body, "\n") != "hello\nworld" {
			t.Fatalf("add = %+v body=%q", hunks, hunks[0].Body)
		}
		multi := "*** Begin Patch\n*** Update File: a.go\n@@\n-func A() int { return 1 }\n-func B() int { return 2 }\n+func A() int { return 10 }\n+func B() int { return 20 }\n*** End Patch\n"
		planText, err = CompileApplyPatch(root, []byte(multi))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err = plan.Parse(bytes.NewReader(planText))
		if err != nil {
			t.Fatal(err)
		}
		if len(hunks) != 1 || hunks[0].Anchor != "func A() int { return 1 }" {
			t.Fatalf("multi-line replace missing first-line anchor: %+v\n%s", hunks, planText)
		}
		if hunks[0].Addr.Start != 3 || hunks[0].Addr.End != 4 {
			t.Fatalf("multi-line address = %d-%d", hunks[0].Addr.Start, hunks[0].Addr.End)
		}
	})

	t.Run("rooted path is refused", func(t *testing.T) {
		doc := "*** Begin Patch\n*** Update File: /abs.go\n@@\n-x\n+y\n*** End Patch\n"
		if _, err := CompileApplyPatch(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "not relative") {
			t.Fatalf("rooted path: %v", err)
		}
	})

	t.Run("blank line flushes a hunk", func(t *testing.T) {
		doc := "*** Begin Patch\n*** Update File: a.go\n-func A() int { return 1 }\n+func A() int { return 10 }\n\n-func C() int { return 3 }\n+func C() int { return 30 }\n*** End Patch\n"
		planText, err := CompileApplyPatch(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(bytes.NewReader(planText))
		if err != nil {
			t.Fatal(err)
		}
		if len(hunks) != 2 {
			t.Fatalf("blank flush got %d hunks:\n%s", len(hunks), planText)
		}
	})
}
