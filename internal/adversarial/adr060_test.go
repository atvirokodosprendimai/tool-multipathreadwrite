package adversarial

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// ── ADR-060 under stress ───────────────────────────────────────────────────
//
// Oracle is the Decision, not plan.go / main.go:
//  1 leftover body= names declared N vs extra non-empty M; one error per hunk
//    still carrying "is not part of any hunk"
//  2 --dry-run (human, not --json) prints one parsed: line per hunk with body=N
//  3 unquoted anchor= containing " is refused with a quote instruction;
//    quoted and ADR-040 spaced-unquoted forms still parse
//  4 body=@path loads root-relative lines; empty file ≡ empty body; rooted
//    paths refuse
//  5 failing check prints check last: immediately above full output:; PASS
//    does not
//
// Op pool (v2), measured 2026-09-14:
//   grep -E 'Op[A-Za-z]+ +Op = "' internal/plan/plan.go | sort -u
//   → 7: replace, insert-after, insert-before, delete, create, unlink, rename
// Leftover/body=@ headers use replace and create (ops that take a counted
// body). The other five sit in the trailing hunk when leftover extra is 0, so
// a clean counted body still parses beside every native op.

var bodyOptRe = regexp.MustCompile(`(?:^|[\t ])body=(\S+)`)

type leftoverHit struct {
	declared int
	extra    int
}

// leftoverOracle walks remaining-count, not plan.Parse. Blank lines after a
// satisfied count are not extra (Decision 1: non-empty).
func leftoverOracle(doc string) []leftoverHit {
	var hits []leftoverHit
	if doc == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(doc, "\n"), "\n")
	remaining := -1
	declared := 0
	extra := 0
	counted := false
	flush := func() {
		if counted && extra > 0 {
			hits = append(hits, leftoverHit{declared, extra})
		}
	}
	for _, line := range lines {
		hdr := strings.TrimPrefix(line, "\ufeff")
		if strings.HasPrefix(hdr, "@@ ") {
			flush()
			declared, remaining, counted = headerBodyCount(hdr)
			extra = 0
			continue
		}
		if remaining > 0 {
			remaining--
			continue
		}
		if counted && remaining == 0 && strings.TrimSpace(line) != "" {
			extra++
		}
	}
	flush()
	return hits
}

func headerBodyCount(hdr string) (declared, remaining int, counted bool) {
	m := bodyOptRe.FindStringSubmatch(hdr)
	if m == nil {
		return 0, -1, false
	}
	v := m[1]
	if strings.HasPrefix(v, "@") {
		return 0, 0, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, -1, false
	}
	return n, n, true
}

func adr060Seed(t *testing.T, fallback int64) (*rand.Rand, int64) {
	t.Helper()
	seed := fallback
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	return rand.New(rand.NewSource(seed)), seed
}

// Trailing native ops that are not the leftover header. Count is 5 of 7.
var adr060SiblingOps = []string{
	"insert-after", "insert-before", "delete", "unlink", "rename",
}

func TestRandomLeftoverBodyCountMatchesTheOracle(t *testing.T) {
	r, seed := adr060Seed(t, 60)
	const iterations = 400
	for i := 0; i < iterations; i++ {
		declared := r.Intn(5)
		blanks := r.Intn(3)
		extra := r.Intn(5)
		var b strings.Builder
		op := "replace"
		addr := "1"
		if r.Intn(2) == 0 {
			op = "create"
			addr = "0"
		} else if declared == 0 {
			// replace with an empty body is a different refusal (ADR-006).
			declared = 1 + r.Intn(4)
		}
		fmt.Fprintf(&b, "@@ a.go %s %s body=%d\n", addr, op, declared)
		for n := 0; n < declared; n++ {
			fmt.Fprintf(&b, "BODY-%d-%d\n", i, n)
		}
		for n := 0; n < blanks; n++ {
			b.WriteString("   \n")
		}
		for n := 0; n < extra; n++ {
			fmt.Fprintf(&b, "STRAY-%d-%d\n", i, n)
		}
		if extra == 0 {
			switch adr060SiblingOps[r.Intn(len(adr060SiblingOps))] {
			case "insert-after":
				b.WriteString("@@ b.go 1 insert-after\ninserted\n")
			case "insert-before":
				b.WriteString("@@ b.go 1 insert-before\ninserted\n")
			case "delete":
				b.WriteString("@@ b.go 1 delete\n")
			case "unlink":
				b.WriteString("@@ gone.txt - unlink\n")
			case "rename":
				b.WriteString("@@ old.txt - rename\nnew.txt\n")
			}
		}
		doc := b.String()
		hits := leftoverOracle(doc)
		hunks, err := plan.Parse(strings.NewReader(doc))
		label := fmt.Sprintf("seed=%d iter=%d declared=%d blanks=%d extra=%d op=%s\n%s",
			seed, i, declared, blanks, extra, op, doc)
		if extra > 0 {
			if err == nil {
				t.Fatalf("%s\noracle leftover extra=%d but Parse succeeded (%d hunks)", label, extra, len(hunks))
			}
			msg := err.Error()
			if len(hits) != 1 || hits[0].declared != declared || hits[0].extra != extra {
				t.Fatalf("%s\noracle hits=%v want one declared=%d extra=%d", label, hits, declared, extra)
			}
			if !strings.Contains(msg, "body="+strconv.Itoa(declared)) {
				t.Fatalf("%s\nerror does not name declared body=%d:\n%s", label, declared, msg)
			}
			if !strings.Contains(msg, strconv.Itoa(extra)+" extra") {
				t.Fatalf("%s\nerror does not name %d extra:\n%s", label, extra, msg)
			}
			if n := strings.Count(msg, "is not part of any hunk"); n != 1 {
				t.Fatalf("%s\nwant 1 leftover error, got %d:\n%s", label, n, msg)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s\noracle leftover extra=0 but Parse failed: %v", label, err)
		}
		if len(hits) != 0 {
			t.Fatalf("%s\noracle hits=%v on a clean plan", label, hits)
		}
	}
}

func TestRandomUnquotedAnchorQuoteMatchesTheOracle(t *testing.T) {
	r, seed := adr060Seed(t, 60)
	const iterations = 200
	for i := 0; i < iterations; i++ {
		form := r.Intn(3) // 0 unquoted-with-quote, 1 quoted, 2 ADR-040 spaced
		var header string
		switch form {
		case 0:
			header = `@@ a.go 1 replace anchor=from "vitest"`
		case 1:
			header = `@@ a.go 1 replace anchor="from \"vitest\""`
		default:
			header = `@@ a.go 1 replace anchor=func openTestStore body=1`
		}
		doc := header + "\nnew line\n"
		_, err := plan.Parse(strings.NewReader(doc))
		label := fmt.Sprintf("seed=%d iter=%d form=%d\n%s", seed, i, form, doc)
		switch form {
		case 0:
			if err == nil {
				t.Fatalf("%s\nunquoted anchor= with \" parsed; Decision 3 refuses it", label)
			}
			if !strings.Contains(err.Error(), `anchor="`) {
				t.Fatalf("%s\nrefusal does not tell them to write anchor=\"…\":\n%s", label, err)
			}
		default:
			if err != nil {
				t.Fatalf("%s\nquoted / ADR-040 form must still parse: %v", label, err)
			}
		}
	}
}

func fileLinesOracle(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func TestRandomBodyAtPathMatchesTheFileOracle(t *testing.T) {
	r, seed := adr060Seed(t, 60)
	const iterations = 80
	for i := 0; i < iterations; i++ {
		root := t.TempDir()
		n := r.Intn(12) // 0..11 lines; 0 is empty ≡ body=0
		var raw []byte
		if n > 0 {
			var b strings.Builder
			for k := 0; k < n; k++ {
				fmt.Fprintf(&b, "L-%d-%d\n", i, k)
			}
			raw = []byte(b.String())
		}
		src := filepath.Join(root, "src.txt")
		if err := os.WriteFile(src, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(strings.NewReader("@@ dest.txt 0 create body=@src.txt\n"))
		if err != nil {
			t.Fatalf("seed=%d iter=%d parse: %v", seed, i, err)
		}
		if err := plan.LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("seed=%d iter=%d LoadBodyFiles: %v", seed, i, err)
		}
		want := fileLinesOracle(raw)
		got := hunks[0].Body
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("seed=%d iter=%d Body=%q want %q", seed, i, got, want)
		}
		if n == 0 && got != nil && len(got) != 0 {
			t.Fatalf("seed=%d iter=%d empty file Body=%q want empty", seed, i, got)
		}
	}

	t.Run("rooted", func(t *testing.T) {
		root := t.TempDir()
		hunks, err := plan.Parse(strings.NewReader("@@ dest.txt 0 create body=@/etc/hosts\n"))
		if err != nil {
			t.Fatal(err)
		}
		err = plan.LoadBodyFiles(root, hunks)
		if err == nil {
			t.Fatal("rooted body=@/etc/hosts loaded")
		}
		if !strings.Contains(err.Error(), "/etc/hosts") {
			t.Fatalf("rooted refusal does not name the path: %v", err)
		}
		if !rooted.IsRooted("/etc/hosts") {
			t.Fatal("oracle: /etc/hosts must be rooted (never filepath.IsAbs)")
		}
	})
}

func TestADR060BinaryMatrixMatchesTheExitCodeOracle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("declared checks are sh -c scripts")
	}
	r, seed := adr060Seed(t, 60)
	const iterations = 80
	for i := 0; i < iterations; i++ {
		kind := i % 7 // every shape every 7 iters, so a mutant cannot hide in an unrolled column
		state := t.TempDir()
		label := fmt.Sprintf("seed=%d iter=%d kind=%d", seed, i, kind)
		switch kind {
		case 0:
			adr060LeftoverBinary(t, label, state)
		case 1:
			adr060DryRunParsed(t, label, state, false)
		case 2:
			adr060DryRunParsed(t, label, state, true)
		case 3:
			adr060UnquotedBinary(t, label, state)
		case 4:
			adr060BodyAtPathBinary(t, label, state, r)
		case 5:
			adr060NeighbourHint(t, label, state, r)
		default:
			adr060CheckLast(t, label, state, i, r.Intn(2) == 0)
		}
	}
}

func adr060LeftoverBinary(t *testing.T, label, state string) {
	t.Helper()
	root := tree(t, map[string]string{"a.txt": "orig\n"})
	planFile := filepath.Join(t.TempDir(), "p.mrw")
	doc := "@@ a.txt 1 replace body=1\nnew\nSTRAY\nSTRAY2\n"
	if err := os.WriteFile(planFile, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", "--no-check", planFile)
	if code != 2 {
		t.Fatalf("%s leftover write exit %d want 2\n%s", label, code, out)
	}
	if !strings.Contains(out, "body=1") || !strings.Contains(out, "2 extra") {
		t.Fatalf("%s leftover does not name body=1 vs 2 extra\n%s", label, out)
	}
	if readFile(t, root, "a.txt") != "orig\n" {
		t.Fatalf("%s leftover write mutated the tree", label)
	}
}

func adr060DryRunParsed(t *testing.T, label, state string, json bool) {
	t.Helper()
	root := tree(t, map[string]string{"a.txt": "line one\nline two\n"})
	if _, code := run(t, state, root, "read", "a.txt"); code != 0 {
		t.Fatalf("%s read failed", label)
	}
	planFile := filepath.Join(t.TempDir(), "p.mrw")
	doc := "@@ a.txt 2 replace anchor=\"line two\" body=1\nchanged\n"
	if err := os.WriteFile(planFile, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"write", "--dry-run", "--no-check"}
	if json {
		args = append(args, "--json")
	}
	out, code := run(t, state, root, append(args, planFile)...)
	if code != 0 {
		t.Fatalf("%s dry-run exit %d\n%s", label, code, out)
	}
	if readFile(t, root, "a.txt") != "line one\nline two\n" {
		t.Fatalf("%s dry-run mutated the tree", label)
	}
	if json {
		return // Decision 1: parsed: is human, not --json
	}
	if !strings.Contains(out, "parsed:") || !strings.Contains(out, "body=1") {
		t.Fatalf("%s dry-run missing parsed body=1\n%s", label, out)
	}
	if !strings.Contains(out, "a.txt") || !strings.Contains(out, "replace") {
		t.Fatalf("%s parsed: line missing path/op\n%s", label, out)
	}
}

func adr060UnquotedBinary(t *testing.T, label, state string) {
	t.Helper()
	root := tree(t, map[string]string{"a.txt": "from \"vitest\"\n"})
	planFile := filepath.Join(t.TempDir(), "p.mrw")
	doc := "@@ a.txt 1 replace anchor=from \"vitest\"\nnew\n"
	if err := os.WriteFile(planFile, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", "--no-check", planFile)
	if code != 2 {
		t.Fatalf("%s unquoted quote write exit %d want 2\n%s", label, code, out)
	}
	if !strings.Contains(out, `anchor="`) {
		t.Fatalf("%s unquoted quote refusal missing quote instruction\n%s", label, out)
	}
	if readFile(t, root, "a.txt") != "from \"vitest\"\n" {
		t.Fatalf("%s unquoted quote write mutated the tree", label)
	}
}

func adr060BodyAtPathBinary(t *testing.T, label, state string, r *rand.Rand) {
	t.Helper()
	if r.Intn(4) == 0 {
		root := tree(t, map[string]string{})
		planFile := filepath.Join(t.TempDir(), "p.mrw")
		if err := os.WriteFile(planFile, []byte("@@ dest.txt 0 create body=@/etc/hosts\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, code := run(t, state, root, "write", "--no-check", planFile)
		if code != 2 {
			t.Fatalf("%s rooted body=@ exit %d want 2\n%s", label, code, out)
		}
		if fileExists(root, "dest.txt") {
			t.Fatalf("%s rooted body=@ created dest.txt", label)
		}
		return
	}
	body := "alpha\nbeta\n"
	if r.Intn(3) == 0 {
		body = ""
	}
	root := tree(t, map[string]string{"src.txt": body})
	planFile := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(planFile, []byte("@@ dest.txt 0 create body=@src.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", "--no-check", planFile)
	if code != 0 {
		t.Fatalf("%s body=@ create exit %d\n%s", label, code, out)
	}
	got := readFile(t, root, "dest.txt")
	if body == "" {
		if got != "" {
			t.Fatalf("%s empty body=@ wrote %q", label, got)
		}
		return
	}
	if got != body && got != strings.TrimSuffix(body, "\n") {
		t.Fatalf("%s dest.txt=%q want %q", label, got, body)
	}
}

func adr060NeighbourHint(t *testing.T, label, state string, r *rand.Rand) {
	t.Helper()
	root := tree(t, map[string]string{"f.txt": "one\ntwo\nthree\nfour\n"})
	shape := r.Intn(5)
	var args []string
	wantNote := false
	switch shape {
	case 0:
		args = []string{"read", "f.txt:1-2"}
		wantNote = true
	case 1:
		args = []string{"read", "f.txt"}
	case 2:
		args = []string{"read", "f.txt:2"}
	case 3:
		args = []string{"read", "f.txt:3-4"} // End is last line; licence skipped
	default:
		args = []string{"read", "--stat", "f.txt:1-2"}
	}
	out, code := run(t, state, root, args...)
	if code != 0 {
		t.Fatalf("%s read %v exit %d\n%s", label, args, code, out)
	}
	has := strings.Contains(out, "needs a served line after")
	if wantNote && !has {
		t.Fatalf("%s %v missing neighbour note\n%s", label, args, out)
	}
	if !wantNote && has {
		t.Fatalf("%s %v printed neighbour note\n%s", label, args, out)
	}
}

func adr060CheckLast(t *testing.T, label, state string, iter int, fail bool) {
	t.Helper()
	token := fmt.Sprintf("unique-last-%d", iter)
	check := "true"
	if fail {
		check = "sh -c 'echo " + token + "; exit 1'"
	}
	root := tree(t, map[string]string{
		"a.go":                  "package a\nfunc A() {}\n",
		".quality-harness.json": `{"check":"` + check + `"}`,
	})
	if _, code := run(t, state, root, "read", "a.go"); code != 0 {
		t.Fatalf("%s read a.go failed", label)
	}
	planFile := filepath.Join(t.TempDir(), "p.mrw")
	doc := "@@ a.go 2 replace anchor=\"func A\"\nfunc A() { _ = 1 }\n"
	if err := os.WriteFile(planFile, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", planFile)
	if fail {
		if code != 3 {
			t.Fatalf("%s failing check exit %d want 3\n%s", label, code, out)
		}
		last := strings.Index(out, "check last: "+token)
		full := strings.Index(out, "full output:")
		if last < 0 || full < 0 || last > full {
			t.Fatalf("%s check last: must sit above full output:\n%s", label, out)
		}
		return
	}
	if code != 0 {
		t.Fatalf("%s passing check exit %d\n%s", label, code, out)
	}
	if strings.Contains(out, "check last:") {
		t.Fatalf("%s PASS printed check last:\n%s", label, out)
	}
}
