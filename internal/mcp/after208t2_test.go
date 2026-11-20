package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-208 Decision 1a over the server: the lines a search_replace write produced
// are licensed and not shown, so a replace ending above them is refused with the
// line-after kind until a read of the line after is acknowledged.
func TestMrwWriteRefusesAReplaceEndingAboveLinesASearchReplaceWrote(t *testing.T) {
	const (
		five    = "package a\n\nfunc A() {}\n\nvar z = 1"
		block   = "f.go\n<<<<<<< SEARCH\nvar z = 1\n=======\nvar z = 2\nfunc B() {\n\tif y {\n\t}\n}\n>>>>>>> REPLACE\n"
		replace = "@@ f.go 6-8 replace anchor=\"func B\"\nfunc B() {\n\tz()\n}\n"
	)
	fixture := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
		if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.go 0 create\n" + five + "\n"}); res["isError"] == true {
			t.Fatalf("create: %v", res)
		}
		if res := call(t, root, "mrw_write", map[string]any{"format": "search_replace", "plan": block}); res["isError"] == true {
			t.Fatalf("search_replace: %v", res)
		}
		return root
	}
	failed := func(t *testing.T, res map[string]any) map[string]any {
		t.Helper()
		if res["isError"] != true {
			t.Fatalf("the plan was not refused: %v", res)
		}
		hunks, _ := structured(t, res)["hunks"].([]any)
		for _, h := range hunks {
			if m, _ := h.(map[string]any); m["status"] == "failed" {
				return m
			}
		}
		t.Fatalf("no failed hunk: %v", hunks)
		return nil
	}
	t.Run("refused", func(t *testing.T) {
		root := fixture(t)
		h := failed(t, call(t, root, "mrw_write", map[string]any{"plan": replace}))
		if !strings.Contains(fmt.Sprint(h["reason"]), "needs a served line after 8") || !strings.Contains(fmt.Sprint(h["reason"]), "mrw_read f.go lines 6-9") {
			t.Errorf("the reason is %q, want the line after and the mrw_read to make", h["reason"])
		}
	})
	t.Run("before_ack", func(t *testing.T) {
		root := fixture(t)
		call(t, root, "mrw_read", map[string]any{"specs": []string{"f.go:6-9"}})
		h := failed(t, call(t, root, "mrw_write", map[string]any{"plan": replace}))
		if !strings.Contains(fmt.Sprint(h["reason"]), "never acknowledged") {
			t.Errorf("a page served and not acknowledged: %q, want the ack named", h["reason"])
		}
	})
	t.Run("after_ack", func(t *testing.T) {
		root := fixture(t)
		res := call(t, root, "mrw_read", map[string]any{"specs": []string{"f.go:6-9"}})
		var acks []string
		for _, id := range checkpointsIn(firstText(t, res)) {
			acks = append(acks, id.(string))
		}
		if len(acks) == 0 {
			t.Fatalf("the read served no checkpoint: %v", res)
		}
		if res := call(t, root, "mrw_write", map[string]any{"plan": replace, "ack": acks}); res["isError"] == true {
			t.Errorf("after the ack: %v", res)
		}
	})
}

// Deep review of #761. After a search_replace write the plan's own record is
// written-shaped and partial: it licenses the lines the write produced. A page of the
// line after that was served and not acknowledged is then all the ack must add, so the
// refusal names the ack instead of asking again for lines the record already licenses.
func TestTheLineAfterRefusalNamesAPendingAckOnAPartialWrittenRecord(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 0 create\nseed\n"}); res["isError"] == true {
		t.Fatalf("create: %v", res)
	}
	if res := call(t, root, "mrw_write", map[string]any{"format": "search_replace", "plan": "f.txt\n<<<<<<< SEARCH\nseed\n=======\na\nb\nc\n>>>>>>> REPLACE\n"}); res["isError"] == true {
		t.Fatalf("search_replace: %v", res)
	}
	read := call(t, root, "mrw_read", map[string]any{"specs": []string{"f.txt:3"}})
	plan := "@@ f.txt 1-2 replace anchor=\"a\"\nA\nB\n"
	res := call(t, root, "mrw_write", map[string]any{"plan": plan})
	if res["isError"] != true {
		t.Fatalf("the plan was not refused: %v", res)
	}
	// The page already served covers everything missing, so the remedy leads with the ack and asks for no read.
	if reason := fmt.Sprint(structured(t, res)["hunks"]); !strings.Contains(reason, "served but never acknowledged") || strings.Contains(reason, "mrw_read") {
		t.Errorf("a pending page of line 3 over a partial written record: want the ack named and no read asked for:\n%s", reason)
	}
	var acks []string
	for _, id := range checkpointsIn(firstText(t, read)) {
		acks = append(acks, id.(string))
	}
	if res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks}); res["isError"] == true {
		t.Errorf("after the ack the plan did not land: %v", res)
	}
}

// Deep review of #761, final pass. The record under the plan's own spelling may be
// read-shaped (a read through a link) and still cover the hunk's lines: then only the
// line after is missing, and a pending page of it is the whole remedy. A link whose
// own record covers nothing still needs the whole range.
func TestTheLineAfterRefusalThroughALinkNamesAPendingAckWhenItsOwnReadCoversTheHunk(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ f.txt 0 create\nseed\n"}); res["isError"] == true {
		t.Fatalf("create: %v", res)
	}
	if res := call(t, root, "mrw_write", map[string]any{"format": "search_replace", "plan": "f.txt\n<<<<<<< SEARCH\nseed\n=======\na\nb\nc\n>>>>>>> REPLACE\n"}); res["isError"] == true {
		t.Fatalf("search_replace: %v", res)
	}
	if err := os.Symlink("f.txt", filepath.Join(root, "g.txt")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	ackOf := func(res map[string]any) []string {
		var acks []string
		for _, id := range checkpointsIn(firstText(t, res)) {
			acks = append(acks, id.(string))
		}
		return acks
	}
	plan := "@@ g.txt 1-2 replace anchor=\"a\"\nA\nB\n"
	reason := func() string {
		res := call(t, root, "mrw_write", map[string]any{"plan": plan})
		if res["isError"] != true {
			t.Fatalf("the plan was not refused: %v", res)
		}
		return fmt.Sprint(structured(t, res)["hunks"])
	}
	// A link that has shown nothing: the whole range is the read to make.
	call(t, root, "mrw_read", map[string]any{"specs": []string{"g.txt:3"}})
	if r := reason(); strings.Contains(r, "never acknowledged") || !strings.Contains(r, "mrw_read g.txt lines 1-3") {
		t.Errorf("a link that has shown nothing and has line 3 pending: %s", r)
	}
	// Lines 1-2 read and acknowledged under the link (the ack rides on a harmless write); line 3 still pending.
	first := call(t, root, "mrw_read", map[string]any{"specs": []string{"g.txt:1"}})
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ x.txt 0 create\nx\n", "ack": ackOf(first)}); res["isError"] == true {
		t.Fatalf("ack of g.txt:1: %v", res)
	}
	third := call(t, root, "mrw_read", map[string]any{"specs": []string{"g.txt:3"}})
	if r := reason(); !strings.Contains(r, "served but never acknowledged") || strings.Contains(r, "mrw_read") {
		t.Errorf("lines 1-2 acknowledged through the link and line 3 pending: want the ack alone:\n%s", r)
	}
	if res := call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": ackOf(third)}); res["isError"] == true {
		t.Errorf("after acking line 3 alone the plan did not land: %v", res)
	}
}
