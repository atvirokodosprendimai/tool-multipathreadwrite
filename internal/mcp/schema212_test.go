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
	const sha = "Absent for a removed file and for a moved symbolic link, whose target mrw does not read."
	const lines = "0 for a removed file and for a moved symbolic link, whose target mrw does not read: on a written row it is a count only when sha_after is present."
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
			row, _ := f.(map[string]any)
			out[row["path"].(string)] = row
		}
		return out
	}

	dry := rows(true)
	src, ok := dry["a/link.txt"]
	if !ok || src["renamed_to"] != "b/link.txt" || src["written"] == true {
		t.Errorf("the dry-run source row is %v, want renamed_to b/link.txt and written false", src)
	}
	if _, ok := dry["b/link.txt"]; ok || len(dry) != 1 {
		t.Errorf("a dry run gave rows %v, want only the source", dry)
	}

	applied := rows(false)
	dst, ok := applied["b/link.txt"]
	if !ok || dst["written"] != true || dst["lines_after"].(float64) != 0 {
		t.Fatalf("the destination row is %v, want written true and lines_after 0", dst)
	}
	if _, has := dst["sha_after"]; has {
		t.Errorf("the destination row carries sha_after %v: mrw does not read a moved link's target", dst["sha_after"])
	}
}
