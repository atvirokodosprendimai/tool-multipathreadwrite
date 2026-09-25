package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-067 T1. An index is the answer a read too large to serve degrades to,
// and it used to carry only the COUNT of the walk's problems — so a path the
// walk could not use, or a CR-only file ADR-065 refuses, was never named. The
// served answer names each one as `-- <path>: <reason>`; the index now does too.

// indexOf fails the test unless res reads as an index, and returns its text.
func indexOf(t *testing.T, res map[string]any, what string) string {
	t.Helper()
	text := served0(t, res)
	if !strings.Contains(text, "-- INDEX:") {
		t.Fatalf("%s did not degrade to an index:\n%.600s", what, text)
	}
	return text
}

// withCeiling lowers the ceiling for one test.
func withCeiling(t *testing.T, n int) {
	t.Helper()
	restore := MaxResultChars
	MaxResultChars = n
	t.Cleanup(func() { MaxResultChars = restore })
}

// Reaches the raw-overflow return (tools.go:348): the report alone is past
// the ceiling. At 20,000 characters the 1,500 index entries are too many to
// fit either, so the trim loop runs — and the problem line must survive it.
func TestAnIndexNamesEveryPathTheWalkCouldNotUse(t *testing.T) {
	root := grepTree(t, 1500, 1)
	withCeiling(t, 20_000)

	res := call(t, root, "mrw_read", map[string]any{"specs": []any{"no-such-dir", "."}, "grep": "NEEDLE"})
	text := indexOf(t, res, "an oversized grep beside a missing path")
	if !strings.Contains(text, "-- no-such-dir:") {
		t.Errorf("the index does not name the path the walk could not use:\n%.800s", text)
	}
	if !strings.Contains(text, "Send the SAME grep again") {
		t.Errorf("the fixture did not make the trim loop run, so it cannot show the line survives it:\n%.800s", text)
	}
	if got := receipt(t, res)["problems"]; got != float64(1) {
		t.Errorf("problems = %v, want 1", got)
	}

	clean := call(t, root, "mrw_read", map[string]any{"specs": []any{"."}, "grep": "NEEDLE"})
	if strings.Contains(indexOf(t, clean, "the same grep without the missing path"), "-- no-such-dir") {
		t.Error("a grep that named no missing path still printed one — the line is not coming from the problem")
	}
}

// bands serves a small-file grep beside a missing path at the default ceiling,
// then derives the two sizes that separate the later index returns: E, the
// encoded answer WITHOUT checkpoint markers (tools.go:414 compares it), and M,
// the answer WITH them (tools.go:446). Both are computed from what was served,
// so a ceiling between them lands in exactly one band. r, the length of the report itself, is the floor of the receipt band.
func bands(t *testing.T) (root string, args map[string]any, r, e, m int) {
	t.Helper()
	root, _ = manyFiles(t, 200)
	args = map[string]any{"specs": []any{"no-such-dir", "."}, "grep": "x"}
	res := call(t, root, "mrw_read", args)
	marked := served0(t, res)
	if strings.Contains(marked, "-- INDEX:") {
		t.Fatal("the fixture overflowed at the default ceiling; the bands cannot be measured")
	}
	blocks := res["content"].([]any)
	rec := blocks[1].(map[string]any)["text"].(string)

	var unmarked strings.Builder
	for _, l := range strings.SplitAfter(marked, "\n") {
		if strings.HasPrefix(l, "-- ck ") || strings.HasPrefix(l, "-- This serve licenses NOTHING") || l == "-- "+AckRule+"\n" {
			continue
		}
		unmarked.WriteString(l)
	}
	e = encodedSize(callToolResult{Content: []contentBlock{{Type: "text", Text: unmarked.String()}, {Type: "text", Text: rec}}})
	m = encodedSize(callToolResult{Content: []contentBlock{{Type: "text", Text: marked}, {Type: "text", Text: rec}}})
	if !(e < m) {
		t.Fatalf("markers added nothing (E=%d, M=%d); the checkpoint band does not exist for this fixture", e, m)
	}
	if !strings.Contains(unmarked.String(), "-- no-such-dir:") {
		t.Fatalf("the served answer does not name the missing path, so the fixture is wrong:\n%.600s", unmarked.String())
	}
	r = len(unmarked.String())
	return root, args, r, e, m
}

// Reaches the receipt-overflow return (tools.go:425): the report fits and the
// receipt carries the answer past the ceiling.
func TestAnIndexFromAnOverflowingReceiptNamesTheProblem(t *testing.T) {
	// The ceiling sits in the MIDDLE of the receipt band, not one under E:
	// interleave may add a trailing newline, so a reconstructed E can sit a
	// byte or two above the real one, and E-1 then lands in the checkpoint
	// band instead. Found by the mutant at :425 surviving.
	root, args, r, e, _ := bands(t)
	c := r + (e-r)/2
	if !(r < c && c < e) {
		t.Fatalf("no ceiling separates the report (%d) from the encoded answer (%d)", r, e)
	}
	withCeiling(t, c)
	text := indexOf(t, call(t, root, "mrw_read", args), fmt.Sprintf("a read whose receipt overflows a %d ceiling", c))
	if !strings.Contains(text, "-- no-such-dir:") {
		t.Errorf("the receipt-overflow index does not name the missing path:\n%.800s", text)
	}
}

// Reaches the checkpoint-overflow return (tools.go:448): the unmarked answer
// fits and the marked one does not.
func TestAnIndexFromOverflowingCheckpointsNamesTheProblem(t *testing.T) {
	root, args, _, e, m := bands(t)
	if e > m-1 {
		t.Fatalf("no ceiling separates E=%d and M=%d", e, m)
	}
	withCeiling(t, m-1)
	text := indexOf(t, call(t, root, "mrw_read", args), fmt.Sprintf("a read whose checkpoints overflow a %d ceiling", m-1))
	if !strings.Contains(text, "-- no-such-dir:") {
		t.Errorf("the checkpoint-overflow index does not name the missing path:\n%.800s", text)
	}
}

// installFakeAstGrep puts an ast-grep on PATH that prints stdout — the helper
// internal/read's tests use, copied because test helpers do not cross packages.
func installFakeAstGrep(t *testing.T, stdout string) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "fake.go")
	body := fmt.Sprintf("package main\nimport (\"fmt\"; \"os\")\nfunc main() { fmt.Fprint(os.Stdout, %q); os.Exit(0) }\n", stdout)
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ast-grep")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, src)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake ast-grep: %v\n%s", err, out)
	}
	// Run it once, untimed, before any test times it: the first start of a
	// freshly built .exe on a Windows runner can spend the whole 2 s bound.
	_ = exec.Command(bin).Run()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// The BACKLOG P2 this task closes: an ast_grep answer too large to serve
// named no CR-only file ADR-065 had refused.
func TestAnAstGrepIndexNamesACROnlyFileItRefused(t *testing.T) {
	root := grepTree(t, 60, 400)
	if err := os.WriteFile(filepath.Join(root, "cr.go"), []byte("one\rtwo\rthree\r"), 0o644); err != nil {
		t.Fatal(err)
	}
	var hits []map[string]any
	for i := 0; i < 60; i++ {
		hits = append(hits, map[string]any{"file": fmt.Sprintf("document%05d.csv", i),
			"range": map[string]any{"start": map[string]any{"line": 0}, "end": map[string]any{"line": 400}}})
	}
	hits = append(hits, map[string]any{"file": "cr.go",
		"range": map[string]any{"start": map[string]any{"line": 1}, "end": map[string]any{"line": 1}}})
	raw, err := json.Marshal(hits)
	if err != nil {
		t.Fatal(err)
	}
	installFakeAstGrep(t, string(raw))

	text := indexOf(t, call(t, root, "mrw_read", map[string]any{"ast_grep": "x"}), "an oversized ast_grep")
	if !strings.Contains(text, "-- cr.go:") {
		t.Errorf("the ast_grep index does not name the CR-only file it refused:\n%.800s", text)
	}
}

// Problem lines are never trimmed, so when they alone cannot fit the answer is
// the ceiling's refusal — never an index that silently dropped some of them.
func TestProblemLinesTooLargeForTheCeilingAreRefusedLegibly(t *testing.T) {
	root := grepTree(t, 60, 400)
	specs := []any{"."}
	for i := 0; i < 400; i++ {
		specs = append(specs, fmt.Sprintf("missing-%04d-%s", i, strings.Repeat("p", 60)))
	}
	withCeiling(t, 20_000)

	raw := rawResult(t, root, "mrw_read", map[string]any{"specs": specs, "grep": "NEEDLE"})
	if len(raw) > 20_000 {
		t.Fatalf("the answer is %d bytes against a 20,000 ceiling", len(raw))
	}
	var res map[string]any
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(served0(t, res), "-- INDEX:") {
		t.Fatalf("an index was served although its problem lines cannot all fit:\n%.600s", served0(t, res))
	}
	if res["isError"] != true {
		t.Errorf("the refusal is not flagged isError: %v", res)
	}
}
