package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fileProperty returns the description of one property of a row of mrw_write's
// `files`, read from the output schema tools/list serves.
func fileProperty(t *testing.T, name string) string {
	t.Helper()
	for _, tl := range tools() {
		schema, ok := tl.OutputSchema.(map[string]any)
		if !ok || tl.Name != "mrw_write" {
			continue
		}
		props, _ := schema["properties"].(map[string]any)
		files, _ := props["files"].(map[string]any)
		items, _ := files["items"].(map[string]any)
		fp, _ := items["properties"].(map[string]any)
		p, _ := fp[name].(map[string]any)
		d, _ := p["description"].(string)
		return d
	}
	t.Fatal("mrw_write declares no output schema")
	return ""
}

// ADR-212. A moved symbolic link's record carries no content (#725), and the
// schema, which is what a model reads, says so for the two keys it affects.
func TestTheWriteSchemaSaysAMovedLinkCarriesNoContent(t *testing.T) {
	const sha = "Absent for a removed file and for a moved symbolic link: the receipt is built without reading what the link leads to."
	const lines = "0 for a removed file and for a moved symbolic link, whose receipt is built without reading what the link leads to: on a written row it is a count only when sha_after is present."
	if d := fileProperty(t, "sha_after"); !strings.Contains(d, sha) {
		t.Errorf("files.sha_after's description lacks %q:\n%s", sha, d)
	}
	if d := fileProperty(t, "lines_after"); !strings.Contains(d, lines) {
		t.Errorf("files.lines_after's description lacks %q:\n%s", lines, d)
	}
}

// ADR-212. Over MCP a relative symlink read and moved to another directory has a
// destination row with written true, lines_after 0 and no sha_after; the same plan
// as a dry run names the source row with renamed_to and written false, and no row
// for the destination.
func TestTheMCPReceiptOfAMovedLinkGivesNoContent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	for _, d := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a", "payload.txt"), []byte("A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b", "payload.txt"), []byte("B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("payload.txt", filepath.Join(root, "a", "link.txt")); err != nil {
		t.Skipf("this OS cannot make a symbolic link here: %v", err)
	}
	const plan = "@@ a/link.txt - rename\nb/link.txt\n"
	rows := func(dry bool) map[string]map[string]any {
		t.Helper()
		rd := call(t, root, "mrw_read", map[string]any{"specs": []any{"a/link.txt"}})
		args := map[string]any{"plan": plan, "ack": checkpointsIn(served0(t, rd))}
		if dry {
			args["dry_run"] = true
		}
		w := call(t, root, "mrw_write", args)
		if w["isError"] == true {
			t.Fatalf("the write was refused: %v", w["content"])
		}
		out := map[string]map[string]any{}
		fs, _ := structured(t, w)["files"].([]any)
		for _, f := range fs {
			row, ok := f.(map[string]any)
			if !ok {
				t.Fatalf("a files row is %T, want an object: %v", f, f)
			}
			p, ok := row["path"].(string)
			if !ok {
				t.Fatalf("a files row has path %v (%T), want a string", row["path"], row["path"])
			}
			// Counted before indexing: a second row for one path is a defect
			// the map below would hide.
			if _, dup := out[p]; dup {
				t.Fatalf("two files rows for %s: %v", p, fs)
			}
			out[p] = row
		}
		return out
	}

	dry := rows(true)
	src, ok := dry["a/link.txt"]
	if !ok {
		t.Fatalf("a dry run gave rows %v, want the source a/link.txt", dry)
	}
	if src["renamed_to"] != "b/link.txt" {
		t.Errorf("the dry-run source row's renamed_to is %v, want b/link.txt", src["renamed_to"])
	}
	if w, ok := src["written"].(bool); !ok || w {
		t.Errorf("the dry-run source row's written is %v (%T), want present and false", src["written"], src["written"])
	}
	if _, ok := dry["b/link.txt"]; ok || len(dry) != 1 {
		t.Errorf("a dry run gave rows %v, want only the source", dry)
	}

	applied := rows(false)
	dst, ok := applied["b/link.txt"]
	if !ok {
		t.Fatalf("the applied rows %v have none for b/link.txt", applied)
	}
	if w, ok := dst["written"].(bool); !ok || !w {
		t.Errorf("the destination row's written is %v (%T), want present and true", dst["written"], dst["written"])
	}
	if n, ok := dst["lines_after"].(float64); !ok || n != 0 {
		t.Errorf("the destination row's lines_after is %v (%T), want the number 0", dst["lines_after"], dst["lines_after"])
	}
	if _, has := dst["sha_after"]; has {
		t.Errorf("the destination row carries sha_after %v: mrw does not read a moved link's target", dst["sha_after"])
	}
}
