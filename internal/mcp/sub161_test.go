package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// tree161 makes a checkout holding files, none of them read by mrw.
func tree161(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
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

func file161(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// stream161 is ast-grep --json=stream for the given matches of file in body.
func stream161(t *testing.T, file, body string, matches ...[2]string) string {
	t.Helper()
	var sb strings.Builder
	for _, m := range matches {
		start := strings.Index(body, m[0])
		if start < 0 {
			t.Fatalf("%q is not in %q", m[0], body)
		}
		b, err := json.Marshal(map[string]any{
			"text":        m[0],
			"range":       map[string]any{"byteOffset": map[string]int{"start": start, "end": start + len(m[0])}},
			"file":        file,
			"replacement": m[1],
		})
		if err != nil {
			t.Fatal(err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

const goSrc161 = "package a\n\nfunc f() {\n\tfoo(1, 2)\n}\n"

// ADR-161 Decision 12. mrw_write builds its plan from a walk — grep and sub, or
// ast_grep and rewrite — and applies it as one plan: every site and no
// neighbour, a dry run writes nothing, and a file the walk could not read
// applies nothing at all.
func TestAnMCPSubWriteAppliesThePlanInOneCall(t *testing.T) {
	root := tree161(t, map[string]string{
		"a.txt": "Old\nOldX\nxOld\nkeep Old\n",
		"b.txt": "Old here\n",
		"c.txt": "nothing\n",
	})
	dry := call(t, root, "mrw_write", map[string]any{"grep": `\bOld\b`, "sub": "New", "dry_run": true, "check": false})
	if dry["isError"] == true || structured(t, dry)["dry_run"] != true {
		t.Fatalf("the dry run: %v", dry)
	}
	if got := file161(t, root, "a.txt"); got != "Old\nOldX\nxOld\nkeep Old\n" {
		t.Fatalf("a dry run changed a.txt to %q", got)
	}

	res := call(t, root, "mrw_write", map[string]any{"grep": `\bOld\b`, "sub": "New", "check": false})
	sc := structured(t, res)
	if res["isError"] == true || sc["applied"] != true || sc["failed"] != float64(0) {
		t.Fatalf("the sub write: %v", res)
	}
	if got := file161(t, root, "a.txt"); got != "New\nOldX\nxOld\nkeep New\n" {
		t.Errorf("a.txt is %q: every Old and neither OldX nor xOld changes", got)
	}
	if got := file161(t, root, "b.txt"); got != "New here\n" {
		t.Errorf("b.txt is %q", got)
	}
	if got := file161(t, root, "c.txt"); got != "nothing\n" {
		t.Errorf("c.txt is %q: it held no match", got)
	}

	// an empty replacement is a real one
	if res := call(t, root, "mrw_write", map[string]any{"grep": "^keep ", "sub": "", "check": false}); res["isError"] == true || file161(t, root, "a.txt") != "New\nOldX\nxOld\nNew\n" {
		t.Errorf("an empty sub: %v, a.txt %q", res, file161(t, root, "a.txt"))
	}

	t.Run("an unreadable matching file applies nothing", func(t *testing.T) {
		skipUnlessPermissionsBind(t)
		root := tree161(t, map[string]string{"a.txt": "Old\n", "b.txt": "Old\n"})
		if err := os.Chmod(filepath.Join(root, "b.txt"), 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "b.txt"), 0o644) })
		res := call(t, root, "mrw_write", map[string]any{"grep": "Old", "sub": "New", "check": false})
		if res["isError"] != true {
			t.Errorf("a partial plan is not an error: %v", res)
		}
		if got := file161(t, root, "a.txt"); got != "Old\n" {
			t.Errorf("the readable file was edited to %q: a partial edit must never land", got)
		}
	})

	t.Run("ast_grep and rewrite apply ast-grep's own rewrite", func(t *testing.T) {
		root := tree161(t, map[string]string{"a.go": goSrc161})
		installRecordingAstGrep(t, stream161(t, "a.go", goSrc161, [2]string{"foo(1, 2)", "bar(2, 1)"}))
		res := call(t, root, "mrw_write", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)", "check": false})
		if res["isError"] == true || structured(t, res)["applied"] != true {
			t.Fatalf("the rewrite write: %v", res)
		}
		if got := file161(t, root, "a.go"); got != strings.Replace(goSrc161, "foo(1, 2)", "bar(2, 1)", 1) {
			t.Errorf("a.go is %q", got)
		}
	})

	t.Run("overlapping rewrites are refused and nothing is written", func(t *testing.T) {
		src := "package a\n\nfunc f() {\n\t_ = foo(foo(5, 6), 7)\n}\n"
		root := tree161(t, map[string]string{"a.go": src})
		installRecordingAstGrep(t, stream161(t, "a.go", src,
			[2]string{"foo(foo(5, 6), 7)", "bar(7, foo(5, 6))"}, [2]string{"foo(5, 6)", "bar(6, 5)"}))
		res := call(t, root, "mrw_write", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)", "check": false})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "overlap") {
			t.Errorf("overlapping matches: %v", res)
		}
		if got := file161(t, root, "a.go"); got != src {
			t.Errorf("an overlap changed a.go to %q", got)
		}
	})
}

// The CLI's refusals, by name, on both tools; an argument no tool declares is
// refused as before.
func TestAnMCPSubRefusesWhatTheCLIRefuses(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "Old\nsame\n"})
	const hunk = "@@ a.txt 1 replace\nX\n"
	for _, c := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"mrw_write", map[string]any{"grep": "Old"}, "go together"},
		{"mrw_write", map[string]any{"sub": "New"}, "go together"},
		{"mrw_write", map[string]any{"ast_grep": "p"}, "go together"},
		{"mrw_write", map[string]any{"rewrite": "r"}, "go together"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "New", "plan": hunk}, "makes its own plan"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "New", "format": "plan"}, "makes its own plan"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "New", "ast_grep": "p", "rewrite": "r"}, "two ways to make a plan"},
		{"mrw_write", map[string]any{"plan": hunk, "exclude": []any{"x"}}, "shape the walk"},
		{"mrw_write", map[string]any{"plan": hunk, "no_ignore": true}, "shape the walk"},
		{"mrw_write", map[string]any{"grep": "(", "sub": "x"}, "--grep"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "a\nb"}, "line break"},
		{"mrw_write", map[string]any{"grep": "nothing-matches", "sub": "x"}, "no match"},
		{"mrw_write", map[string]any{"grep": "same", "sub": "same"}, "changed no line"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "New", "exclude": []any{"["}}, "exclude"},
		{"mrw_write", map[string]any{"grep": "Old", "sub": "New", "bogus": 1}, "bogus"},
		{"mrw_write", map[string]any{}, "needs a plan"},
		{"mrw_read", map[string]any{"sub": "New"}, "grep"},
		{"mrw_read", map[string]any{"grep": "Old", "ast_grep": "p", "sub": "New"}, "two sources"},
		{"mrw_read", map[string]any{"rewrite": "r"}, "ast_grep"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "rewrite": "r"}, "ast_grep"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "a\nb"}, "line break"},
		{"mrw_read", map[string]any{"specs": []any{"a.txt:1-2"}, "grep": "Old", "sub": "New"}, "two answers"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "count": true}, "count"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "stat": true}, "stat"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "quiet": true}, "quiet"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "max_cols": 10}, "max_cols"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "max_lines": 3}, "max_lines"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "files_from": "x"}, "files_from"},
		{"mrw_read", map[string]any{"grep": "Old", "sub": "New", "after": "a.txt"}, "after"},
		{"mrw_read", map[string]any{"grep": "Old", "quiet": true, "count": true}, "quiet"},
		{"mrw_read", map[string]any{"grep": "Old", "quiet": true, "stat": true}, "quiet"},
	} {
		res := call(t, root, c.tool, c.args)
		if res["isError"] != true || !strings.Contains(firstText(t, res), c.want) {
			t.Errorf("%s %v: want a refusal naming %q, got %v", c.tool, c.args, c.want, res)
		}
	}
	if got := file161(t, root, "a.txt"); got != "Old\nsame\n" {
		t.Errorf("a refusal changed a.txt to %q", got)
	}
}

// mrw_read with sub or rewrite returns the plan as its text, exactly, and
// licenses the planned lines and no others, with no ack: nothing was shown, so
// the checkpoints, which certify seen lines, have nothing to certify.
func TestAnMCPSubReadReturnsAPlanAndLicensesItsLines(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "Old\nOldX\nkeep Old\n"})
	res := call(t, root, "mrw_read", map[string]any{"grep": `\bOld\b`, "sub": "New"})
	text := served0(t, res)
	if res["isError"] == true || strings.Contains(text, "-- ck ") || strings.Contains(text, "| ") {
		t.Fatalf("a plan carries no checkpoint and no served line: %v", res)
	}
	hunks, err := plan.Parse(strings.NewReader(text))
	if err != nil || len(hunks) != 2 || hunks[0].Addr.Start != 1 || hunks[1].Addr.Start != 3 {
		t.Fatalf("the text is not exactly the plan for lines 1 and 3 (%v): %q", err, text)
	}
	// licensed, not shown: an unplanned line stays unwritable
	if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 2 replace\nTWO\n", "check": false}); w["isError"] != true || !strings.Contains(firstText(t, w), "has not been read") {
		t.Errorf("the unplanned line 2 was writable: %v", w)
	}
	w := call(t, root, "mrw_write", map[string]any{"plan": text, "check": false})
	if w["isError"] == true || structured(t, w)["applied"] != true || file161(t, root, "a.txt") != "New\nOldX\nkeep New\n" {
		t.Errorf("the plan the read returned did not apply: %v, a.txt %q", w, file161(t, root, "a.txt"))
	}

	t.Run("rewrite", func(t *testing.T) {
		root := tree161(t, map[string]string{"a.go": goSrc161})
		installRecordingAstGrep(t, stream161(t, "a.go", goSrc161, [2]string{"foo(1, 2)", "bar(2, 1)"}))
		res := call(t, root, "mrw_read", map[string]any{"ast_grep": "foo($A, $B)", "rewrite": "bar($B, $A)"})
		text := served0(t, res)
		if res["isError"] == true || !strings.HasPrefix(text, "@@ a.go 4 replace sha=") {
			t.Fatalf("the rewrite plan: %v", res)
		}
		if got := file161(t, root, "a.go"); got != goSrc161 {
			t.Errorf("a read changed a.go to %q", got)
		}
		if w := call(t, root, "mrw_write", map[string]any{"plan": text, "check": false}); w["isError"] == true || !strings.Contains(file161(t, root, "a.go"), "bar(2, 1)") {
			t.Errorf("the rewrite plan did not apply: %v", w)
		}
	})

	t.Run("a plan too large for one answer is refused and licenses nothing", func(t *testing.T) {
		root := tree161(t, map[string]string{"a.txt": strings.Repeat("Old\n", 400)})
		withCeiling(t, 4000)
		res := call(t, root, "mrw_read", map[string]any{"grep": "Old", "sub": "New"})
		if res["isError"] != true || strings.Contains(served0(t, res), "@@ a.txt") {
			t.Errorf("an oversized plan was returned in part: %.300s", firstText(t, res))
		}
		withCeiling(t, 200000)
		if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nNew\n", "check": false}); w["isError"] != true {
			t.Errorf("a refused plan licensed line 1: %v", w)
		}
	})
}

// quiet licenses what the read serves and returns no line. It needs no ack
// because nothing is shown for a host to cut; the caller asserts it knows the
// lines, as the CLI's --quiet does.
func TestAnMCPQuietReadLicensesAndReturnsNoLine(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "l1\nl2\nl3\nl4\nl5\nl6\n"})
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt:2-3"}, "quiet": true})
	text := served0(t, res)
	if res["isError"] == true || strings.Contains(text, "l2") || strings.Contains(text, "| ") || strings.Contains(text, "-- ck ") {
		t.Fatalf("a quiet read returned a line or a checkpoint: %v", res)
	}
	if !strings.Contains(text, "-- licensed 3 line(s) in 1 file(s)") {
		t.Errorf("a quiet read does not count what it licensed (2-3 and the line after):\n%s", text)
	}
	// licensed with no ack: a multi-line replace of the lines it was told, line 4 as its next line
	if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 2-3 replace anchor=\"l2\"\nTWO\nTHREE\n", "check": false}); w["isError"] == true || structured(t, w)["applied"] != true {
		t.Errorf("a quiet read did not license its range: %v", w)
	}
	// and nothing it did not serve
	root = tree161(t, map[string]string{"a.txt": "l1\nl2\nl3\nl4\nl5\nl6\n"})
	call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt:2-3"}, "quiet": true})
	if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 6 replace\nSIX\n", "check": false}); w["isError"] != true {
		t.Errorf("a quiet read of 2-3 licensed line 6: %v", w)
	}
	// the pair: the same read without quiet licenses nothing until it is acknowledged
	root = tree161(t, map[string]string{"a.txt": "l1\nl2\nl3\nl4\nl5\nl6\n"})
	call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt:2-3"}})
	if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 2 replace\nTWO\n", "check": false}); w["isError"] != true {
		t.Errorf("an unacknowledged serve licensed line 2: %v", w)
	}
}

var followRe161 = regexp.MustCompile(`open lines (\d+)-(\d+) \((\d+) lines follow\)`)

// A ranged read serves the line after the range inside the checkpoint that
// brackets the range, and that checkpoint counts it (ADR-161 Decision 10, ADR-108).
func TestAnMCPRangeServesTheLineAfterInsideItsCheckpoint(t *testing.T) {
	body := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\n"
	root := tree161(t, map[string]string{"a.txt": body})
	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt:3-5"}})
	text := served0(t, res)
	m := followRe161.FindStringSubmatch(text)
	if m == nil || m[1] != "3" || m[2] != "6" || m[3] != "4" {
		t.Fatalf("the checkpoint should bracket lines 3-6 and say 4 lines follow:\n%s", text)
	}
	if !strings.Contains(text, "6| l6") || strings.Contains(text, "-- note: a multi-line replace") {
		t.Errorf("line 6 is not served, or ADR-136's note remains:\n%s", text)
	}
	// acknowledged, a multi-line replace of exactly the lines read is not refused
	w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 3-5 replace anchor=\"l3\"\nA\nB\n", "ack": checkpointsIn(text), "check": false})
	if w["isError"] == true || structured(t, w)["applied"] != true {
		t.Errorf("a replace of exactly the lines read was refused: %v", w)
	}

	// a range that ends on the last line, and a whole file, gain nothing
	root = tree161(t, map[string]string{"a.txt": body})
	if m := followRe161.FindStringSubmatch(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt:8-10"}}))); m == nil || m[2] != "10" || m[3] != "3" {
		t.Errorf("a range ending on the last line: %v", m)
	}
	if m := followRe161.FindStringSubmatch(served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{"a.txt"}}))); m == nil || m[3] != "10" {
		t.Errorf("a whole file: %v", m)
	}

	// a grep match is a range of one line and gets its next line
	g := served0(t, call(t, root, "mrw_read", map[string]any{"grep": "^l4$"}))
	if m := followRe161.FindStringSubmatch(g); m == nil || m[1] != "4" || m[2] != "5" || m[3] != "2" {
		t.Errorf("a grep match and the line after it:\n%s", g)
	}
}

// A page is a range, and a page must not repeat a line: the continuation serves
// the line a page ends on. Following next_read to its end from a range's
// start reassembles the rest of the file exactly, whatever the page size.
func TestAnMCPPageContinuationNeverRepeatsOrDropsALine(t *testing.T) {
	const lines = 6000
	root, path := bigCheckout(t, lines)
	var b strings.Builder
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(&b, "line %d padding padding padding padding padding\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, path), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	var got []string
	spec := path + ":100-"
	for page := 0; ; page++ {
		if page > 20 {
			t.Fatalf("still paging after %d pages", page)
		}
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{spec}})
		got = append(got, numberedLines(t, served0(t, res))...)
		next := nextOf(t, res)
		if next == "" {
			break
		}
		spec = next
	}
	if len(got) != lines-99 {
		t.Fatalf("reassembled %d lines, want %d", len(got), lines-99)
	}
	for i, l := range got {
		if want := fmt.Sprintf("line %d padding padding padding padding padding", 100+i); l != want {
			t.Fatalf("line %d of the reassembly is %q, want %q", i, l, want)
		}
	}
}

// The handshake carries the recipes in their MCP form.
func TestTheHandshakeCarriesTheRecipesInTheirMCPForm(t *testing.T) {
	got := instructionsText()
	for _, must := range []string{
		"Recipes",
		"mrw_write with grep and sub",
		"mrw_write with ast_grep and rewrite",
		"dry_run",
		"mrw_read with quiet",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("the handshake lacks %q", must)
		}
	}
	if len(got) > maxInstructionsChars {
		t.Errorf("instructions are %d bytes, over %d", len(got), maxInstructionsChars)
	}
}

// A path narrows the walk of an mrw_write plan: a matching file outside it is
// untouched. A range in a path, a path outside the root, and paths without a
// walk are refused, and nothing is written.
func TestAnMCPSubWriteWalksOnlyThePathsItIsGiven(t *testing.T) {
	root := tree161(t, map[string]string{"in/a.txt": "Old\n", "out/b.txt": "Old\n"})
	res := call(t, root, "mrw_write", map[string]any{"grep": "Old", "sub": "New", "paths": []any{"in"}, "check": false})
	if res["isError"] == true || structured(t, res)["applied"] != true {
		t.Fatalf("the narrowed write: %v", res)
	}
	if file161(t, root, "in/a.txt") != "New\n" || file161(t, root, "out/b.txt") != "Old\n" {
		t.Errorf("in/a.txt %q, out/b.txt %q: only the file under the path may change", file161(t, root, "in/a.txt"), file161(t, root, "out/b.txt"))
	}
	for name, args := range map[string]map[string]any{
		"a range":     {"grep": "Old", "sub": "New", "paths": []any{"out/b.txt:1-1"}},
		"outside":     {"grep": "Old", "sub": "New", "paths": []any{"../x"}},
		"without one": {"plan": "@@ out/b.txt 1 replace\nX\n", "paths": []any{"out"}},
	} {
		args["check"] = false
		if r := call(t, root, "mrw_write", args); r["isError"] != true {
			t.Errorf("%s: not refused: %v", name, r)
		}
	}
	if file161(t, root, "out/b.txt") != "Old\n" {
		t.Errorf("a refusal changed out/b.txt to %q", file161(t, root, "out/b.txt"))
	}
}

// A plan crosses JSON as a string, and JSON replaces a byte that is not valid
// UTF-8: a plan read back from mrw_read would write U+FFFD where the file had
// \xe9 while the original line stayed licensed (ADR-108). mrw_read refuses such
// a plan and licenses nothing. mrw_write keeps its plan in memory and never
// serialises it, so it applies the same plan and keeps the byte.
func TestAnMCPSubReadRefusesAPlanJSONWouldCorrupt(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "caf\xe9 Old\n"})
	res := call(t, root, "mrw_read", map[string]any{"grep": "Old", "sub": "New"})
	txt := firstText(t, res)
	if res["isError"] != true || strings.Contains(txt, "@@ ") || !strings.Contains(txt, "a.txt") || !strings.Contains(txt, "CLI") {
		t.Fatalf("a plan holding an invalid UTF-8 byte was returned, or the refusal does not name the file and the CLI: %v", res)
	}
	if w := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nX\n", "check": false}); w["isError"] != true {
		t.Errorf("a refused plan licensed line 1: %v", w)
	}
	if got := file161(t, root, "a.txt"); got != "caf\xe9 Old\n" {
		t.Fatalf("a read changed a.txt to %q", got)
	}
	w := call(t, root, "mrw_write", map[string]any{"grep": "Old", "sub": "New", "check": false})
	if w["isError"] == true || file161(t, root, "a.txt") != "caf\xe9 New\n" {
		t.Errorf("mrw_write did not apply the in-memory plan byte for byte: %v, a.txt %q", w, file161(t, root, "a.txt"))
	}
}

// An empty pattern is a pattern and an empty replacement a replacement: the
// CLI tells presence (IsSet) from value, and so do both MCP tools.
func TestAnMCPSubTakesAnEmptyPatternAndAnEmptyReplacement(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "a\nb\n"})
	res := call(t, root, "mrw_read", map[string]any{"grep": "", "sub": "X"})
	if res["isError"] == true || !strings.Contains(served0(t, res), "XaX") {
		t.Fatalf("read with an empty pattern: %v", res)
	}
	if got := file161(t, root, "a.txt"); got != "a\nb\n" {
		t.Fatalf("a read changed a.txt to %q", got)
	}
	if res := call(t, root, "mrw_read", map[string]any{"grep": "a", "sub": ""}); res["isError"] == true || !strings.Contains(served0(t, res), "@@ a.txt 1 replace") {
		t.Errorf("read with an empty replacement: %v", res)
	}
	if res := call(t, root, "mrw_write", map[string]any{"grep": "", "sub": "X", "check": false}); res["isError"] == true || file161(t, root, "a.txt") != "XaX\nXbX\n" {
		t.Errorf("write with an empty pattern: %v, a.txt %q", res, file161(t, root, "a.txt"))
	}
	if res := call(t, root, "mrw_write", map[string]any{"grep": "b", "sub": "", "check": false}); res["isError"] == true || file161(t, root, "a.txt") != "XaX\nXX\n" {
		t.Errorf("write with an empty replacement: %v, a.txt %q", res, file161(t, root, "a.txt"))
	}
	// the pair: a missing half is still refused
	if res := call(t, root, "mrw_write", map[string]any{"grep": "", "check": false}); res["isError"] != true {
		t.Errorf("an empty grep without sub was not refused: %v", res)
	}
}

// A key sent twice whose last value is null is missing: the struct decoder keeps
// the earlier value, so presence comes from the raw keys alone (the Codex review
// of T9). Neither tool plans from it, and nothing is written or licensed.
func TestAnMCPPlanCallTakesTheLastValueOfADuplicateKey(t *testing.T) {
	root := tree161(t, map[string]string{"a.txt": "Old\n"})
	for _, params := range []string{
		`{"name":"mrw_write","arguments":{"grep":"Old","grep":null,"sub":"New","check":false}}`,
		`{"name":"mrw_read","arguments":{"grep":"Old","grep":null,"sub":"New"}}`,
	} {
		res, rerr := callTool(root, json.RawMessage(params), false, nil)
		if rerr == nil && !res.IsError {
			t.Errorf("%s: answered, want a refusal: %+v", params, res)
		}
		if got := file161(t, root, "a.txt"); got != "Old\n" {
			t.Fatalf("%s: a.txt became %q", params, got)
		}
	}
	if res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt 1 replace\nNew\n", "check": false}); res["isError"] != true {
		t.Errorf("a refused plan call licensed a.txt: %v", res)
	}
}
