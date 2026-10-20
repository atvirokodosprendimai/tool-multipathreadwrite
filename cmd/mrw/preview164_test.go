package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// previewHunks is the hunks of a --json receipt, with the two keys ADR-164
// Decision 2 adds.
func previewHunks(t *testing.T, out string) []struct {
	Path   string   `json:"path"`
	Before []string `json:"before"`
	After  []string `json:"after"`
} {
	t.Helper()
	var rec struct {
		Hunks []struct {
			Path   string   `json:"path"`
			Before []string `json:"before"`
			After  []string `json:"after"`
		} `json:"hunks"`
	}
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("not one JSON receipt: %v\n%s", err, out)
	}
	return rec.Hunks
}

// ADR-164 Decision 2. A `--grep --sub` dry run prints, under each ok row, the
// line with the change marked in place; --json carries the replaced and the
// new lines; a CRLF file and a file with no final newline show the same
// lines, and a dry run writes nothing.
func TestASubDryRunShowsEachChange(t *testing.T) {
	root, run := subRun(t)
	files := map[string]string{
		"a.css":    ".a { color: red; }\n.b { background: red; }\n",
		"crlf.css": ".c { color: red; }\r\nkeep\r\n",
		"last.css": ".d { color: red; }",
		"twice.go": "red and red\n",
	}
	subTree(t, root, files)

	out, errOut, code := run("", "write", "--dry-run", "--grep", `\bred\b`, "--sub", "blue", ".")
	if code != 0 {
		t.Fatalf("dry run: exit %d:\n%s%s", code, out, errOut)
	}
	for _, want := range []string{
		"   ~ 1| .a { color: [-red-]{+blue+}; }",
		"   ~ 2| .b { background: [-red-]{+blue+}; }",
		"   ~ 1| .c { color: [-red-]{+blue+}; }\n", // no \r: line endings are not shown
		"   ~ 1| .d { color: [-red-]{+blue+}; }\n", // no final newline
		"   ~ 1| [-red and red-]{+blue and blue+}", // both matches inside one span
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the receipt lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\r") {
		t.Errorf("the receipt shows a line ending:\n%q", out)
	}
	// the row sits under its ok row, before the summary
	if ok, row := strings.Index(out, "ok   a.css 1 replace"), strings.Index(out, "   ~ 1| .a {"); ok < 0 || row < ok {
		t.Errorf("the ~ row is not under the ok row (%d, %d):\n%s", ok, row, out)
	}
	for name, body := range files {
		if got := subFile(t, root, name); got != body {
			t.Errorf("a dry run changed %s to %q", name, got)
		}
	}

	out, errOut, code = run("", "write", "--dry-run", "--json", "--grep", `\bred\b`, "--sub", "blue", ".")
	if code != 0 {
		t.Fatalf("dry run --json: exit %d:\n%s%s", code, out, errOut)
	}
	found := map[string]bool{}
	for _, h := range previewHunks(t, out) {
		if len(h.Before) != 1 || len(h.After) != 1 {
			t.Errorf("%s: before %q after %q, want one line each", h.Path, h.Before, h.After)
			continue
		}
		found[h.Path] = true
		if strings.ContainsAny(h.Before[0]+h.After[0], "\r\n") || !strings.Contains(h.Before[0], "red") || !strings.Contains(h.After[0], "blue") {
			t.Errorf("%s: before %q after %q", h.Path, h.Before, h.After)
		}
	}
	if len(found) != 4 {
		t.Errorf("hunks with before/after for %v, want all four files", found)
	}
}

// ADR-164 Decision 2. The same for `--ast-grep --rewrite`, over a span of
// three lines that becomes one: the old lines under `-`, the new under `+`,
// the paired first line marked.
func TestARewriteDryRunShowsEachChange(t *testing.T) {
	const a = "package a\n\nfunc f() {\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = x\n}\n"
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.go": a})
	installRecordingAstGrep(t, rwStream(t, "a.go", a, [2]string{"foo(\n\t\t3,\n\t\t4)", "bar(4, 3)"}))

	out, errOut, code := run("", "write", "--dry-run", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	if code != 0 {
		t.Fatalf("dry run: exit %d:\n%s%s", code, out, errOut)
	}
	for _, want := range []string{"   - 4| \tx := [-foo(-]\n", "   - 5| \t\t3,\n", "   - 6| \t\t4)\n", "   + 4| \tx := {+bar(4, 3)+}\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("the receipt lacks %q:\n%s", want, out)
		}
	}
	if got := subFile(t, root, "a.go"); got != a {
		t.Errorf("a dry run changed a.go to %q", got)
	}

	out, errOut, code = run("", "write", "--dry-run", "--json", "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", ".")
	hs := previewHunks(t, out)
	if code != 0 || len(hs) != 1 || len(hs[0].Before) != 3 || len(hs[0].After) != 1 || hs[0].After[0] != "\tx := bar(4, 3)" {
		t.Errorf("--json: exit %d, hunks %+v:\n%s%s", code, hs, out, errOut)
	}
}

// ADR-164 Decision 2. A hand-written plan's dry run and a real `--sub` write
// print no `~`, `-` or `+` row and carry no before or after.
func TestAPlainDryRunShowsNoPreview(t *testing.T) {
	root, run := subRun(t)
	subTree(t, root, map[string]string{"a.css": ".a { color: red; }\n", "b.css": ".b { color: red; }\n"})
	if _, errOut, code := run("", "read", "a.css"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, errOut)
	}
	plan := "@@ a.css 1 replace\n.a { color: blue; }\n"

	hasRow := func(out string) bool {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(l, "   ~ ") || strings.HasPrefix(l, "   - ") || strings.HasPrefix(l, "   + ") {
				return true
			}
		}
		return false
	}
	out, errOut, code := run(plan, "write", "--dry-run", "-")
	if code != 0 || hasRow(out) || !strings.Contains(out, "1 hunk(s), 1 file(s), 0 failed") {
		t.Errorf("a plain dry run: exit %d, want 0 and no preview row:\n%s%s", code, out, errOut)
	}
	out, errOut, code = run(plan, "write", "--dry-run", "--json", "-")
	if code != 0 || strings.Contains(out, `"before"`) || strings.Contains(out, `"after"`) {
		t.Errorf("a plain dry run --json: exit %d, want no before or after:\n%s%s", code, out, errOut)
	}

	out, errOut, code = run("", "write", "--no-check", "--grep", `\bred\b`, "--sub", "blue", ".")
	if code != 0 || hasRow(out) || !strings.Contains(out, "2 hunk(s), 2 file(s), 0 failed") || subFile(t, root, "b.css") != ".b { color: blue; }\n" {
		t.Errorf("a real --sub write: exit %d, want 0, the write made and no preview row:\n%s%s", code, out, errOut)
	}
}
