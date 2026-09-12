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

const twoHunkSearchReplace = "a.go\n" +
	"<<<<<<< SEARCH\n" +
	"func A() int { return 1 }\n" +
	"=======\n" +
	"func A() int { return 10 }\n" +
	">>>>>>> REPLACE\n" +
	"<<<<<<< SEARCH\n" +
	"func C() int { return 3 }\n" +
	"=======\n" +
	"func C() int { return 30 }\n" +
	">>>>>>> REPLACE\n"

func TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	observed, _ := read.Run(io.Discard, root,
		[]read.Spec{{Path: "a.go", Ranges: []read.Range{{Start: 3, End: 3}}}}, read.Options{})

	planText, err := CompileSearchReplace(root, []byte(twoHunkSearchReplace))
	if err != nil {
		t.Fatalf("compile refused a locatable two-hunk SEARCH/REPLACE: %v", err)
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

func TestCompileSearchReplaceGoesThroughParse(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	planText, err := CompileSearchReplace(root, []byte(twoHunkSearchReplace))
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
}

func TestANearMissSearchIsACompileRefusal(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	doc := "a.go\n<<<<<<< SEARCH\nfunc A() int { return  1 }\n=======\nfunc A() int { return 10 }\n>>>>>>> REPLACE\n"
	if _, err := CompileSearchReplace(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "matched no lines") {
		t.Fatalf("a near-miss SEARCH compiled (fuzzy apply): %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demo {
		t.Errorf("compile wrote the file:\n%s", got)
	}
}

func TestAnAmbiguousSearchIsACompileRefusal(t *testing.T) {
	root := writeTree(t, map[string]string{"a.txt": "line\nline\nline\n"})
	doc := "a.txt\n<<<<<<< SEARCH\nline\n=======\nLINE\n>>>>>>> REPLACE\n"
	if _, err := CompileSearchReplace(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "matched 3 times") {
		t.Fatalf("ambiguous SEARCH: %v", err)
	}
}

func TestEmptySearchOnExistingFileIsACompileRefusal(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	doc := "a.go\n<<<<<<< SEARCH\n=======\nnew\n>>>>>>> REPLACE\n"
	if _, err := CompileSearchReplace(root, []byte(doc)); err == nil || !strings.Contains(err.Error(), "empty SEARCH") {
		t.Fatalf("empty SEARCH on existing file: %v", err)
	}
}

func TestEmptySearchOnMissingFileIsCreate(t *testing.T) {
	root := writeTree(t, map[string]string{})
	doc := "new.txt\n<<<<<<< SEARCH\n=======\nhello\n>>>>>>> REPLACE\n"
	planText, err := CompileSearchReplace(root, []byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatal(err)
	}
	if len(hunks) != 1 || string(hunks[0].Op) != "create" || strings.Join(hunks[0].Body, "\n") != "hello" {
		t.Fatalf("create = %+v", hunks)
	}
}
