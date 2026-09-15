package adversarial

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// ── Four-leftovers arm 3: drive the built binary ───────────────────────────
//
// Pool enumerated from cmd/mrw/main.go (rg 'two sources of specs|two answers
// to one question|exclude without'): 4 two-source pairs, range+finder, exclude
// without a finder. ast-grep on/off PATH and present-with-[] vs present-with-a
// hit are the UC-2 outcomes. Oracle is ADR-058 Decision 1–2 plus the pre-
// existing two-source table, not the switch in main.go.
//
// Left out on purpose: a hanging ast-grep on PATH. The record does not bound
// that subprocess; encoding today's lack of a timeout as a promise would go
// red if anyone added one.

var (
	fakeAstOnce sync.Once
	fakeAstPath string
	fakeAstErr  error
)

func fakeAstGrepBin(t *testing.T) string {
	t.Helper()
	fakeAstOnce.Do(func() {
		dir, err := os.MkdirTemp("", "mrw-fake-ast-")
		if err != nil {
			fakeAstErr = err
			return
		}
		src := filepath.Join(dir, "fake.go")
		body := `package main
import ("fmt"; "os"; "strconv")
func main() {
	out := os.Getenv("MRW_FAKE_ASTGREP_OUT")
	if out == "" {
		out = "[]"
	}
	fmt.Fprint(os.Stdout, out)
	code := 0
	if s := os.Getenv("MRW_FAKE_ASTGREP_EXIT"); s != "" {
		code, _ = strconv.Atoi(s)
	}
	os.Exit(code)
}
`
		if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
			fakeAstErr = err
			return
		}
		name := "ast-grep"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		fakeAstPath = filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", fakeAstPath, src)
		if out, err := cmd.CombinedOutput(); err != nil {
			fakeAstErr = fmt.Errorf("build fake ast-grep: %v\n%s", err, out)
		}
	})
	if fakeAstErr != nil {
		t.Fatal(fakeAstErr)
	}
	return fakeAstPath
}

func runRead(t *testing.T, state, root string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(mrwBinary(t), append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "XDG_STATE_HOME="+state)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v\n%s", err, out)
		}
		code = ee.ExitCode()
	}
	return string(out), code
}

type findCombo struct {
	grep, ast, filesFrom, exclude, pos, posRange bool
	astPresent                                   bool
	hits                                         bool
}

// oracleFind is ADR-058 D1–D2 plus the existing two-source / exclude table.
func oracleFind(c findCombo) (code int, must, mustNot []string) {
	if c.exclude && !c.grep && !c.ast {
		return 2, []string{"exclude"}, nil
	}
	sources := 0
	if c.grep {
		sources++
	}
	if c.ast {
		sources++
	}
	if c.filesFrom {
		sources++
	}
	if sources >= 2 {
		return 2, []string{"two sources"}, nil
	}
	if c.filesFrom && c.pos {
		return 2, []string{"two sources"}, nil
	}
	if c.posRange && c.ast {
		return 2, []string{"two answers"}, nil
	}
	if c.posRange && c.grep {
		return 2, []string{"two answers"}, nil
	}
	if !c.grep && !c.ast && !c.filesFrom && !c.pos {
		return 2, nil, nil
	}
	if c.ast && !c.astPresent {
		return 2, []string{"ast-grep"}, []string{"unknown flag", "flag provided"}
	}
	if c.ast && c.astPresent && !c.hits {
		return 1, []string{"needle"}, []string{"not found"}
	}
	if c.ast && c.astPresent && c.hits && c.exclude {
		return 1, []string{"needle"}, nil
	}
	if c.ast && c.astPresent && c.hits {
		return 0, []string{"hit.go"}, nil
	}
	if c.grep && !c.posRange {
		return 1, []string{"needle"}, nil
	}
	if c.filesFrom {
		return 0, []string{"hit.go"}, nil
	}
	if c.pos {
		return 0, []string{"hit.go"}, nil
	}
	return 2, nil, nil
}

func TestRandomisedAstGrepFlagMatrixMatchesTheExitOracle(t *testing.T) {
	seed := int64(58)
	if s := os.Getenv("MRW_SEED"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("MRW_SEED: %v", err)
		}
		seed = n
	}
	iters := 120
	if s := os.Getenv("MRW_STRESS_ITERS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("MRW_STRESS_ITERS: %v", err)
		}
		iters = n
	}
	rng := rand.New(rand.NewSource(seed))
	fake := fakeAstGrepBin(t)
	fakeDir := filepath.Dir(fake)
	for i := 0; i < iters; i++ {
		c := findCombo{
			grep:       rng.Intn(2) == 0,
			ast:        rng.Intn(2) == 0,
			filesFrom:  rng.Intn(2) == 0,
			exclude:    rng.Intn(2) == 0,
			pos:        rng.Intn(2) == 0,
			posRange:   rng.Intn(2) == 0,
			astPresent: rng.Intn(2) == 0,
			hits:       rng.Intn(2) == 0,
		}
		if c.posRange {
			c.pos = true
		}
		want, must, mustNot := oracleFind(c)
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\nfunc Target() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		state := t.TempDir()
		var env []string
		if c.ast && c.astPresent {
			env = append(env, "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			if c.hits {
				env = append(env, `MRW_FAKE_ASTGREP_OUT=[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`)
			} else {
				env = append(env, "MRW_FAKE_ASTGREP_OUT=[]")
			}
		} else {
			env = append(env, "PATH="+t.TempDir())
		}
		args := []string{"read"}
		if c.grep {
			args = append(args, "--grep", "needle")
		}
		if c.ast {
			args = append(args, "--ast-grep", "needle")
		}
		if c.filesFrom {
			list := filepath.Join(t.TempDir(), "list")
			if err := os.WriteFile(list, []byte("hit.go\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			args = append(args, "--files-from", list)
		}
		if c.exclude {
			args = append(args, "--exclude", "*.go")
		}
		if c.pos {
			if c.posRange {
				args = append(args, "hit.go:1-2")
			} else {
				args = append(args, "hit.go")
			}
		}
		out, got := runRead(t, state, root, env, args...)
		if got != want {
			t.Fatalf("seed=%d iter=%d combo=%+v: exit %d want %d\n%s", seed, i, c, got, want, out)
		}
		for _, s := range must {
			if s != "" && !strings.Contains(out, s) {
				t.Fatalf("seed=%d iter=%d combo=%+v: missing %q\n%s", seed, i, c, s, out)
			}
		}
		for _, s := range mustNot {
			if s != "" && strings.Contains(out, s) {
				t.Fatalf("seed=%d iter=%d combo=%+v: must not contain %q\n%s", seed, i, c, s, out)
			}
		}
	}
}

func TestDesktopProbeRecipeNamesServedLineVersusReceipt(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "adr", "ADR-023-a-reads-answer-is-the-served-text.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "served line") && !strings.Contains(got, "served text") {
		t.Fatal("the Desktop recipe does not tell the operator to look at the served line")
	}
	if !strings.Contains(got, "receipt") {
		t.Fatal("the Desktop recipe does not name the receipt as the wrong half")
	}
}

func TestOtherHostsEntryDoesNotSayMissRate(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "adr", "BACKLOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	i := strings.Index(got, "ADR-023: other hosts")
	if i < 0 {
		t.Fatal("BACKLOG has no ADR-023: other hosts entry")
	}
	window := got[i:]
	if len(window) > 800 {
		window = window[:800]
	}
	if strings.Contains(strings.ToLower(window), "miss rate") {
		t.Fatal("the other-hosts entry treats an unrun probe as a miss rate")
	}
}
