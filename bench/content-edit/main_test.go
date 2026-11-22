package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for path, data := range files {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func wantText(t *testing.T, root, path, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, path))
	if err != nil || string(got) != want {
		t.Fatalf("%s = %q, err %v; want %q", path, got, err, want)
	}
}

func TestContentEditBatch(t *testing.T) {
	root := fixture(t, map[string]string{"a.txt": "shifted\r\n\r\nold\r\nkeep", "b.txt": "erase\nlast\n"})
	doc := `[
		{"target":"a.txt","old":["","old"],"replacement":["new",""]},
		{"target":"b.txt","old":["erase"],"replacement":[]},
		{"target":"a.txt","old":["keep"],"replacement":["last"]}
	]`
	if _, err := execute(root, strings.NewReader(doc)); err != nil {
		t.Fatal(err)
	}
	wantText(t, root, "a.txt", "shifted\r\nnew\r\n\r\nlast")
	wantText(t, root, "b.txt", "last\n")
	ledger, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ledger); !os.IsNotExist(err) {
		t.Fatalf("unexpected ledger after content edit: %v", err)
	}
}

func TestContentEditConflicts(t *testing.T) {
	root := fixture(t, map[string]string{"a": "same\nsame\n", "b": "ready\n"})
	doc := `[
		{"target":"b","old":["ready"],"replacement":[]},
		{"target":"a","old":["same"],"replacement":["x"]},
		{"target":"a","old":["absent"],"replacement":[]},
		{"target":"missing","old":["old"],"replacement":[]}
	]`
	_, err := execute(root, strings.NewReader(doc))
	if err == nil {
		t.Fatal("conflicted batch succeeded")
	}
	for _, want := range []string{"edit 2", "edit 3", "edit 4", "ambiguous", "no match"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %v", want, err)
		}
	}
	wantText(t, root, "a", "same\nsame\n")
	wantText(t, root, "b", "ready\n")
}

func TestContentEditInvalid(t *testing.T) {
	root := fixture(t, map[string]string{"a": "one\ntwo\n"})
	for _, doc := range []string{
		`[]`, `null`, `[] []`,
		`[{"target":"a","old":["one"]}]`,
		`[{"target":"a","old":["one"],"replacement":null}]`,
		`[{"target":"a","old":["one"],"replacement":[null]}]`,
		`[{"target":"a","old":["one"],"replacement":["x\ny"]}]`,
		`[{"target":"a","old":[],"replacement":["x"]}]`,
		`[{"target":"../a","old":["one"],"replacement":[]}]`,
		`[{"target":"a","old":["one"],"replacement":[],"file":"b"}]`,
		`[{"target":"missing","target":"a","old":["one"],"replacement":[]}]`,
		`[{"target":"a","old":["one","two"],"replacement":[]},{"target":"a","old":["two"],"replacement":["three"]}]`,
	} {
		if _, err := execute(root, strings.NewReader(doc)); err == nil {
			t.Errorf("accepted invalid input %s", doc)
		}
		wantText(t, root, "a", "one\ntwo\n")
	}
}

func TestContentEditComparisonBudget(t *testing.T) {
	a, b := strings.Repeat("a", 2047), strings.Repeat("b", 2047)
	original := strings.Repeat(a+"\n", 512) + strings.Repeat(b+"\n", 512)
	root := fixture(t, map[string]string{"a": original})
	var batch []map[string]any
	for range 34 {
		batch = append(batch, map[string]any{"target": "a", "old": []string{a, b}, "replacement": []string{}})
	}
	doc, err := json.Marshal(batch)
	if err != nil {
		t.Fatal(err)
	}
	_, err = execute(root, bytes.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "comparison budget") {
		t.Fatal("expected the comparison budget to stop preflight")
	}
	wantText(t, root, "a", original)
}

func TestContentEditMixedEndings(t *testing.T) {
	root := fixture(t, map[string]string{"a": "first\r\nlast\n"})
	doc := `[{"target":"a","old":["last"],"replacement":[]}]`
	_, err := execute(root, strings.NewReader(doc))
	if err == nil || !strings.Contains(err.Error(), "line endings") {
		t.Fatalf("expected a line endings refusal, got %v", err)
	}
	wantText(t, root, "a", "first\r\nlast\n")
}

func TestContentEditAfterShift(t *testing.T) {
	root := fixture(t, map[string]string{"a": "header\nkeep\nchosen\n"})
	first := `[{"target":"a","old":["header"],"replacement":["new","header"]}]`
	if _, err := execute(root, strings.NewReader(first)); err != nil {
		t.Fatal(err)
	}
	wantText(t, root, "a", "new\nheader\nkeep\nchosen\n")
	second := `[{"target":"a","old":["chosen"],"replacement":["done"]}]`
	if _, err := execute(root, strings.NewReader(second)); err != nil {
		t.Fatal(err)
	}
	wantText(t, root, "a", "new\nheader\nkeep\ndone\n")
}

func TestContentEditGeneratedBatch(t *testing.T) {
	const count = 500
	var old, want strings.Builder
	var edits []map[string]any
	for i := range count {
		line := fmt.Sprintf("slot %04d", i)
		updated := fmt.Sprintf("value %04d", i*i)
		fmt.Fprintln(&old, line)
		fmt.Fprintln(&want, updated)
		edits = append(edits, map[string]any{"target": "items", "old": []string{line}, "replacement": []string{updated}})
	}
	root := fixture(t, map[string]string{"items": old.String(), "untouched": "keep"})
	request, err := json.Marshal(edits)
	if err != nil {
		t.Fatal(err)
	}
	n, err := execute(root, bytes.NewReader(request))
	if err != nil || n != count {
		t.Fatalf("completed %d edits, err %v", n, err)
	}
	wantText(t, root, "items", want.String())
	wantText(t, root, "untouched", "keep")
}
