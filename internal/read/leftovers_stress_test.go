package read

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// ── Four-leftovers stress (UC-1 hint, UC-2 mapper) ─────────────────────────
//
// Oracles are transcribed from spec F-10 / F-2 / ADR-058 Decision 2, not from
// isEnglishWordToken or the mapper's if-ladders. A first green on the unit
// tests is not this suite.
//
// Token class (F-10): /^[A-Za-z][A-Za-z0-9_-]*$/ and no *?[ .
// English hint (F-2, Decision 4): fires iff one Run reports two or more
// UNREADABLE paths in that class; names the shell-split class. A glob *?[
// path keeps the glob wording and does not count.
// Line mapping (ADR-058 D2): ast-grep JSON lines are 0-based; served ranges
// are 1-based; a hit outside the root is a Problem, not a Spec.

// specEnglishToken is the F-10 regex, compiled here, not the production var.
var specEnglishToken = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func specIsEnglishWord(path string) bool {
	if strings.ContainsAny(path, "*?[") {
		return false
	}
	return specEnglishToken.MatchString(path)
}

func englishHintFired(out string) bool {
	// "quot" also sits in the glob hint ("quoting keeps the star"). The
	// English-word class is named shell-split in F-2.
	return strings.Contains(out, "shell-split")
}

func globHintFired(out string) bool {
	return strings.Contains(out, "glob your shell did not expand")
}

func TestEnglishWordTokenMatchesTheSpecRegex(t *testing.T) {
	corpus := []struct {
		in   string
		want bool
	}{
		{"rules", true},
		{"that", true},
		{"will", true},
		{"A", true},
		{"foo-bar", true},
		{"baz_qux", true},
		{"HTML", true},
		{"nope.go", false},
		{"9foo", false},
		{"foo*", false},
		{"*.go", false},
		{"src/foo", false},
		{"café", false},
		{"", false},
		{"foo bar", false},
		{"-foo", false},
		{"_foo", false},
	}
	for _, c := range corpus {
		if got := isEnglishWordToken(c.in); got != c.want || got != specIsEnglishWord(c.in) {
			t.Errorf("%q: impl=%v spec=%v want=%v", c.in, isEnglishWordToken(c.in), specIsEnglishWord(c.in), c.want)
		}
	}
}

func FuzzEnglishWordToken(f *testing.F) {
	for _, s := range []string{"rules", "nope.go", "a", "9x", "foo-bar", "a*b", "café", "src/x"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if isEnglishWordToken(s) != specIsEnglishWord(s) {
			t.Fatalf("impl=%v spec=%v for %q", isEnglishWordToken(s), specIsEnglishWord(s), s)
		}
	})
}

func TestRandomisedEnglishHintFollowsTheSpecOracle(t *testing.T) {
	seed := int64(15)
	if s := os.Getenv("MRW_SEED"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("MRW_SEED: %v", err)
		}
		seed = n
	}
	iters := 400
	if s := os.Getenv("MRW_STRESS_ITERS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("MRW_STRESS_ITERS: %v", err)
		}
		iters = n
	}
	// Pools enumerated from F-10's character class and F-14's glob set *?[
	// (not from memory of the unit tests). Left out: colon-bearing specs —
	// ParseSpec owns the range split, which is not this Decision.
	english := []string{"rules", "that", "will", "foo", "Bar", "a", "Z", "foo-bar", "baz_qux", "HTML"}
	dotted := []string{"nope.go", "foo.txt", "a.c", "README.md"}
	glob := []string{"*.go", "sub/*.go", "foo?", "a[b]", "*"}
	slash := []string{"src/foo", "a/b"}
	digit := []string{"9foo", "1", "0x"}
	uni := []string{"café", "naïve"}
	present := "present.txt"
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < iters; i++ {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, present), []byte("ok\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		n := 1 + rng.Intn(6)
		args := make([]string, n)
		wantEnglish := 0
		wantGlob := false
		for j := 0; j < n; j++ {
			var p string
			switch rng.Intn(8) {
			case 0:
				p = english[rng.Intn(len(english))]
			case 1:
				p = dotted[rng.Intn(len(dotted))]
			case 2:
				p = glob[rng.Intn(len(glob))]
			case 3:
				p = slash[rng.Intn(len(slash))]
			case 4:
				p = digit[rng.Intn(len(digit))]
			case 5:
				p = uni[rng.Intn(len(uni))]
			case 6:
				p = present
			default:
				p = english[rng.Intn(len(english))]
			}
			args[j] = p
			if p == present {
				continue
			}
			if strings.ContainsAny(p, "*?[") {
				wantGlob = true
			}
			if specIsEnglishWord(p) {
				wantEnglish++
			}
		}
		specs := make([]Spec, 0, len(args))
		for _, a := range args {
			sp, err := ParseSpec(a)
			if err != nil {
				t.Fatalf("seed=%d iter=%d ParseSpec(%q): %v — generator produced a range", seed, i, a, err)
			}
			specs = append(specs, sp)
		}
		var b strings.Builder
		Run(&b, root, specs, Options{})
		got := b.String()
		wantHint := wantEnglish >= 2
		if englishHintFired(got) != wantHint {
			t.Fatalf("seed=%d iter=%d args=%q english=%d glob=%v: hint=%v want %v\n%s",
				seed, i, args, wantEnglish, wantGlob, englishHintFired(got), wantHint, got)
		}
		if globHintFired(got) != wantGlob {
			t.Fatalf("seed=%d iter=%d args=%q: glob hint=%v want %v\n%s",
				seed, i, args, globHintFired(got), wantGlob, got)
		}
	}
}

// Named attacks: the shapes a caller (or a shell) would use to make the hint
// lie — fire on an ordinary miss, stay silent on a split regex, or rewrite
// the glob wording.

func TestTwoMissingDottedFilesStaySilent(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "nope.go", "missing.go"), Options{})
	got := b.String()
	if englishHintFired(got) {
		t.Fatalf("two dotted misses are not the shell-split class:\n%s", got)
	}
	if n := strings.Count(got, "UNREADABLE"); n != 2 {
		t.Fatalf("want 2 UNREADABLE, got %d:\n%s", n, got)
	}
}

func TestExactlyTwoEnglishTokensFireTheHint(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "alpha", "beta"), Options{})
	got := b.String()
	if !englishHintFired(got) {
		t.Fatalf("two English-word misses must hint:\n%s", got)
	}
	if strings.Count(got, "shell-split") != 1 {
		t.Fatalf("the hint is one extra block, not per path:\n%s", got)
	}
}

func TestOneEnglishAndOneGlobDoesNotFireTheEnglishHint(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "rules", "*.go"), Options{})
	got := b.String()
	if englishHintFired(got) {
		t.Fatalf("one English token is below F-10's floor:\n%s", got)
	}
	if !globHintFired(got) {
		t.Fatalf("the glob path lost Decision 2's wording:\n%s", got)
	}
}

func TestDigitLeadingTokensStaySilent(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "9foo", "8bar"), Options{})
	if englishHintFired(b.String()) {
		t.Fatalf("digit-leading tokens are outside F-10:\n%s", b.String())
	}
}

func TestASlashPathIsNotAnEnglishToken(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "src/foo", "pkg/bar"), Options{})
	if englishHintFired(b.String()) {
		t.Fatalf("slash paths are not the shell-split class:\n%s", b.String())
	}
}

func TestDuplicateEnglishWordStillCountsTwice(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "rules", "rules"), Options{})
	if !englishHintFired(b.String()) {
		t.Fatalf("the same token twice is still two UNREADABLE args:\n%s", b.String())
	}
}

func TestAnExistingFileDoesNotPadTheEnglishCount(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "present.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "rules", "present.txt"), Options{})
	if englishHintFired(b.String()) {
		t.Fatalf("one miss plus a hit is not two UNREADABLE tokens:\n%s", b.String())
	}
}

func TestAReadableEnglishNamedFileDoesNotHint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "rules"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "that"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	Run(&b, root, leftoverSpecs(t, "rules", "that"), Options{})
	got := b.String()
	if strings.Contains(got, "UNREADABLE") {
		t.Fatalf("existing files were UNREADABLE:\n%s", got)
	}
	if englishHintFired(got) {
		t.Fatalf("served files must not pick up the split-regex hint:\n%s", got)
	}
}

func leftoverSpecs(t *testing.T, paths ...string) []Spec {
	t.Helper()
	out := make([]Spec, 0, len(paths))
	for _, p := range paths {
		sp, err := ParseSpec(p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sp)
	}
	return out
}

func FuzzParseAstGrepJSON(f *testing.F) {
	for _, s := range []string{
		"", "[]", "{}", "[{}]", "null", "not json",
		`[{"file":"a.go","range":{"start":{"line":0},"end":{"line":2}}}]`,
		`[{"path":"a.go","range":{"start":{"line":1},"end":{"line":1}}}]`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("parseAstGrepJSON panicked on %q: %v", b, rec)
			}
		}()
		hits, err := parseAstGrepJSON(b)
		s := strings.TrimSpace(string(b))
		if s == "" {
			if err != nil || hits != nil {
				t.Fatalf("empty input: hits=%v err=%v", hits, err)
			}
			return
		}
		var probe []json.RawMessage
		if json.Unmarshal([]byte(s), &probe) != nil {
			if err == nil {
				t.Fatalf("non-array JSON parsed as hits: %q → %#v", b, hits)
			}
		}
	})
}

func TestAstGrepJSONObjectIsNotZeroHits(t *testing.T) {
	hits, err := parseAstGrepJSON([]byte(`{}`))
	if err == nil {
		t.Fatalf("a JSON object parsed as hits %#v — a hostile binary would look like zero matches", hits)
	}
}

func TestAstGrepZeroBasedLineZeroServesLineOne(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\nfunc T() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"hit.go","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
	specs, problems, err := AstGrep(root, nil, "package hit", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if len(specs) != 1 || len(specs[0].Ranges) != 1 || specs[0].Ranges[0].Start != 1 || specs[0].Ranges[0].End != 1 {
		t.Fatalf("0-based line 0 must serve 1-1, got %#v", specs)
	}
}

func TestAstGrepZeroBasedLineOneServesLineTwo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\nfunc T() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	specs, problems, err := AstGrep(root, nil, "func T", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if len(specs) != 1 || specs[0].Ranges[0].Start != 2 || specs[0].Ranges[0].End != 2 {
		t.Fatalf("0-based line 1 must serve 2-2, got %#v", specs)
	}
}

func TestAstGrepEndBeforeStartCollapsesToAPoint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"hit.go","range":{"start":{"line":4},"end":{"line":1}}}]`, 0)
	specs, _, err := AstGrep(root, nil, "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].Ranges[0].Start != 5 || specs[0].Ranges[0].End != 5 {
		t.Fatalf("end < start must collapse to the start line (5), got %#v", specs)
	}
}

func TestAstGrepPathKeyIsAcceptedLikeFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"path":"hit.go","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
	specs, _, err := AstGrep(root, nil, "package", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].Path != "hit.go" {
		t.Fatalf("path key should map like file, got %#v", specs)
	}
}

func TestAstGrepHitOutsideTheRootIsAProblemNotASpec(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "..", "outside.go")
	installFakeAstGrepJSON(t, fmt.Sprintf(`[{"file":%q,"range":{"start":{"line":0},"end":{"line":0}}}]`, outside), 0)
	specs, problems, err := AstGrep(root, nil, "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Fatalf("an outside hit was served: %#v", specs)
	}
	if len(problems) == 0 {
		t.Fatal("an outside hit produced no Problem")
	}
	cleaned := filepath.Clean(outside)
	if rooted.Contains(filepath.Clean(root), cleaned) {
		t.Fatal("fixture is not outside the root")
	}
}

func TestAstGrepEmptyFileFieldIsNotServed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
	specs, problems, err := AstGrep(root, nil, "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Fatalf("empty file name was served: %#v", specs)
	}
	if len(problems) == 0 {
		t.Fatal("empty file name produced no Problem")
	}
}

func TestAstGrepExcludeDropsAHit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"hit.go","range":{"start":{"line":0},"end":{"line":0}}}]`, 0)
	specs, _, err := AstGrep(root, nil, "x", []string{"*.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Fatalf("--exclude must drop the hit, got %#v", specs)
	}
}

func TestWalkSourceNamesNoAstGrep(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller file")
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "walk.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if strings.Contains(src, "ast-grep") || strings.Contains(src, "AstGrep") {
		t.Fatal("Walk gained an ast-grep branch; ADR-007 forbade a fifth matching rule")
	}
}

func installFakeAstGrepJSON(t *testing.T, stdout string, exit int) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "fake.go")
	body := fmt.Sprintf("package main\nimport (\"fmt\"; \"os\")\nfunc main() { fmt.Fprint(os.Stdout, %q); os.Exit(%d) }\n", stdout, exit)
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
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestAstGrepPresentBinaryExitOneWithEmptyArrayIsZeroHits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[]`, 1)
	specs, problems, err := AstGrep(root, nil, "zzz-absent", nil)
	if err != nil {
		t.Fatalf("exit 1 plus [] must not be the missing-binary path: %v", err)
	}
	if len(specs) != 0 || len(problems) != 0 {
		t.Fatalf("want zero hits, got specs=%#v problems=%v", specs, problems)
	}
}

func TestAstGrepNeverRecords(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\nfunc T() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	installFakeAstGrepJSON(t, `[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	var buf bytes.Buffer
	specs, _, err := AstGrep(root, nil, "func T", nil)
	if err != nil {
		t.Fatal(err)
	}
	observed, _ := Run(&buf, root, specs, Options{Numbers: true})
	if _, ok := observed["miss.go"]; ok {
		t.Fatal("a file the finder did not hit was observed")
	}
	for _, spans := range observed {
		_ = spans
	}
	if !strings.Contains(buf.String(), "hit.go") {
		t.Fatalf("hit was not served:\n%s", buf.String())
	}
}
