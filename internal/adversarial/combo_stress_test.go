package adversarial

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ── v1.21.0 kitchen-sink arm 3 ─────────────────────────────────────────────
//
// Pools enumerated from source on 2026-09-15, not from memory:
//
//	python3 -c '… Name: "…" in cmd/mrw/main.go'  → 9 subcommands
//	  (mcp, version, instructions, stats, seen, read, write, iter, check)
//	read flags (readCmd Flags):  stat, no-numbers, context, max-lines,
//	  grep, ast-grep, exclude, files-from
//	write flags (writeCmd Flags): dry-run, force, json, quiet, check,
//	  no-check, format, echo-pad, strict-balance
//	plan ops (internal/plan/plan.go Op* constants): replace, insert-after,
//	  insert-before, delete, create, unlink, rename  — 7, all kept
//
// Left out and why:
//	bare `mrw mcp` — starts a server; the pool only uses --max-result-chars -1
//	  which is usage and returns.
//	a hanging ast-grep — 2 s each; Decision 5 is cmd/mrw/astgrep_timeout_test.go
//	  and contract §111, not this matrix.
//	LLM served-size curve — docs/model-benches.md; not a binary exit oracle.
//
// Oracle is ADR-001 (exit 1 writes nothing), ADR-002 (unread without --force
// is 1), ADR-015 D4 (two English-word UNREADABLE → shell-split), ADR-058 D1–D2
// (two sources / two answers / missing binary / zero hits), ADR-054 (check
// flag contradictions), write --format=git (a git patch is not an apply_patch).
// When several usage errors are possible, only the exit (2) is asserted —
// preemption order is not a promise.

func comboSeed(t *testing.T, fallback int64) (*rand.Rand, int64) {
	t.Helper()
	seed := fallback
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("MRW_SEED: %v", err)
		}
		seed = v
	}
	return rand.New(rand.NewSource(seed)), seed
}

func comboIters(t *testing.T, fallback int) int {
	t.Helper()
	n := fallback
	if s := os.Getenv("MRW_STRESS_ITERS"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("MRW_STRESS_ITERS: %v", err)
		}
		n = v
	}
	return n
}

func comboExitOK(code int) bool {
	return code == 0 || code == 1 || code == 2 || code == 3
}

func TestRandomisedReadKitchenSinkMatchesTheUsageOracle(t *testing.T) {
	r, seed := comboSeed(t, 121)
	iters := comboIters(t, 80)
	fake := fakeAstGrepBin(t)
	fakeDir := filepath.Dir(fake)

	for i := 0; i < iters; i++ {
		c := struct {
			grep, ast, filesFrom, exclude, pos, posRange, astPresent, hits bool
			negContext, negMaxLines, badExclude, emptyFilesFrom            bool
			stat, noNumbers                                                bool
			englishPair                                                    bool
			hostileJSON                                                    bool
		}{
			grep:           r.Intn(2) == 0,
			ast:            r.Intn(2) == 0,
			filesFrom:      r.Intn(2) == 0,
			exclude:        r.Intn(2) == 0,
			pos:            r.Intn(2) == 0,
			posRange:       r.Intn(2) == 0,
			astPresent:     r.Intn(2) == 0,
			hits:           r.Intn(2) == 0,
			negContext:     r.Intn(8) == 0,
			negMaxLines:    r.Intn(8) == 0,
			badExclude:     r.Intn(8) == 0,
			emptyFilesFrom: r.Intn(8) == 0,
			stat:           r.Intn(4) == 0,
			noNumbers:      r.Intn(4) == 0,
			englishPair:    r.Intn(4) == 0,
			hostileJSON:    r.Intn(6) == 0,
		}
		if c.posRange {
			c.pos = true
		}

		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "hit.go"), []byte("package hit\nfunc Target() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		state := t.TempDir()
		var env []string
		if c.ast && c.astPresent {
			env = append(env, "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			switch {
			case c.hostileJSON:
				env = append(env, `MRW_FAKE_ASTGREP_OUT={"hits":[]}`)
			case c.hits:
				env = append(env, `MRW_FAKE_ASTGREP_OUT=[{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`)
			default:
				env = append(env, "MRW_FAKE_ASTGREP_OUT=[]")
			}
		} else {
			env = append(env, "PATH="+t.TempDir())
		}

		args := []string{"read"}
		if c.negContext {
			args = append(args, "-C", "-1")
		}
		if c.negMaxLines {
			args = append(args, "--max-lines", "-3")
		}
		if c.stat {
			args = append(args, "--stat")
		}
		if c.noNumbers {
			args = append(args, "--no-numbers")
		}
		if c.grep {
			args = append(args, "--grep", "needle")
		}
		if c.ast {
			args = append(args, "--ast-grep", "needle")
		}
		if c.filesFrom {
			if c.emptyFilesFrom {
				args = append(args, "--files-from", "")
			} else {
				list := filepath.Join(t.TempDir(), "list")
				if err := os.WriteFile(list, []byte("hit.go\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--files-from", list)
			}
		}
		if c.exclude {
			g := "*.go"
			if c.badExclude {
				g = "["
			}
			args = append(args, "--exclude", g)
		}
		if c.englishPair && !c.grep && !c.ast && !c.filesFrom {
			args = append(args, "rules", "that")
			c.pos = true
			c.posRange = false
		} else if c.pos {
			if c.posRange {
				args = append(args, "hit.go:1-2")
			} else {
				args = append(args, "hit.go")
			}
		}

		out, got := runRead(t, state, root, env, args...)
		label := fmt.Sprintf("seed=%d iter=%d combo=%+v\n%s", seed, i, c, out)

		usage := false
		switch {
		case c.negContext, c.negMaxLines:
			usage = true
		case c.exclude && !c.grep && !c.ast:
			usage = true
		case c.grep && c.ast, c.ast && c.filesFrom, c.grep && c.filesFrom:
			usage = true
		case c.filesFrom && c.pos && !(c.englishPair && !c.grep && !c.ast && !c.filesFrom):
			// files-from + positional paths. englishPair forces pos without filesFrom above.
			usage = true
		case c.filesFrom && c.emptyFilesFrom:
			usage = true
		case c.posRange && (c.ast || c.grep):
			usage = true
		case !c.grep && !c.ast && !c.filesFrom && !c.pos:
			usage = true
		case c.ast && !c.astPresent:
			usage = true
		case c.exclude && c.badExclude && (c.grep || c.ast):
			usage = true
		case c.ast && c.astPresent && c.hostileJSON:
			usage = true
		}
		if usage {
			if got != 2 {
				t.Fatalf("%s: usage path exited %d want 2", label, got)
			}
			if strings.Contains(out, "unknown flag") {
				t.Fatalf("%s: a known flag was reported unknown", label)
			}
			continue
		}

		if !comboExitOK(got) {
			t.Fatalf("%s: exit %d is not a documented status", label, got)
		}
		if c.ast && c.astPresent && !c.hits && !c.hostileJSON {
			if got != 1 {
				t.Fatalf("%s: zero ast-grep hits exited %d want 1", label, got)
			}
			if strings.Contains(out, "not found") || strings.Contains(out, "PATH") {
				t.Fatalf("%s: zero hits reported as a missing binary", label)
			}
		}
		if c.englishPair && !c.grep && !c.ast && !c.filesFrom {
			if got != 1 {
				t.Fatalf("%s: two missing English-word paths exited %d want 1", label, got)
			}
			if !strings.Contains(out, "shell-split") {
				t.Fatalf("%s: ADR-015 D4 hint did not name shell-split", label)
			}
		}
	}
}

func TestRandomisedAstGrepMatrixExtraSeedsStillMatchTheOracle(t *testing.T) {
	if os.Getenv("MRW_STRESS_WIDE") == "" {
		t.Skip("set MRW_STRESS_WIDE=1 to replay leftover matrix seeds 1,7,13,99")
	}
	orig := os.Getenv("MRW_SEED")
	origI := os.Getenv("MRW_STRESS_ITERS")
	t.Cleanup(func() {
		_ = os.Setenv("MRW_SEED", orig)
		_ = os.Setenv("MRW_STRESS_ITERS", origI)
	})
	if err := os.Setenv("MRW_STRESS_ITERS", "40"); err != nil {
		t.Fatal(err)
	}
	for _, seed := range []int64{1, 7, 13, 99} {
		if err := os.Setenv("MRW_SEED", strconv.FormatInt(seed, 10)); err != nil {
			t.Fatal(err)
		}
		t.Run(strconv.FormatInt(seed, 10), func(t *testing.T) {
			TestRandomisedAstGrepFlagMatrixMatchesTheExitOracle(t)
		})
	}
}

func TestRandomisedWriteKitchenSinkMatchesTheFlagOracle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the declared checks are sh -c scripts")
	}
	r, seed := comboSeed(t, 221)
	iters := comboIters(t, 80)

	formats := []string{"plan", "plan", "plan", "git", "nope"}
	ops := []plan.Op{
		plan.OpReplace, plan.OpInsertAfter, plan.OpInsertBefore, plan.OpDelete,
		plan.OpCreate, plan.OpUnlink, plan.OpRename,
	}

	for i := 0; i < iters; i++ {
		root := tree(t, map[string]string{"a.go": "line one\nline two {\nline three\n"})
		state := t.TempDir()
		op := ops[r.Intn(len(ops))]
		format := formats[r.Intn(len(formats))]
		force := r.Intn(4) == 0
		skipRead := r.Intn(4) == 0
		echoNeg := r.Intn(8) == 0
		demand := r.Intn(5) == 0
		optOut := r.Intn(5) == 0
		dry := r.Intn(5) == 0
		js := r.Intn(4) == 0
		quiet := r.Intn(4) == 0
		echoPad := r.Intn(3)
		badAnchor := op == plan.OpReplace && r.Intn(3) == 0

		if !skipRead {
			if _, code := run(t, state, root, "read", "a.go"); code != 0 {
				t.Fatalf("seed=%d iter=%d: pre-read failed", seed, i)
			}
		}

		var body strings.Builder
		switch op {
		case plan.OpReplace:
			anchor := "line two"
			if badAnchor {
				anchor = "NOPE"
			}
			fmt.Fprintf(&body, "@@ a.go 2 replace anchor=%q\nline two }\n", anchor)
		case plan.OpInsertAfter:
			fmt.Fprintf(&body, "@@ a.go 2 insert-after\n{ inserted\n")
		case plan.OpInsertBefore:
			fmt.Fprintf(&body, "@@ a.go 2 insert-before\n{ inserted\n")
		case plan.OpDelete:
			fmt.Fprintf(&body, "@@ a.go 2 delete\n")
		case plan.OpCreate:
			fmt.Fprintf(&body, "@@ new.go 0 create\npackage n\n")
		case plan.OpUnlink:
			fmt.Fprintf(&body, "@@ a.go - unlink\n")
		case plan.OpRename:
			fmt.Fprintf(&body, "@@ a.go - rename\nb.go\n")
		}
		planFile := filepath.Join(t.TempDir(), "p.mrw")
		if err := os.WriteFile(planFile, []byte(body.String()), 0o644); err != nil {
			t.Fatal(err)
		}

		args := []string{"write", "--format", format, "--echo-pad", strconv.Itoa(echoPad)}
		if echoNeg {
			args[len(args)-1] = "-1"
		}
		if force {
			args = append(args, "--force")
		}
		if demand {
			args = append(args, "--check")
		}
		if optOut {
			args = append(args, "--no-check")
		}
		if dry {
			args = append(args, "--dry-run")
		}
		if js {
			args = append(args, "--json")
		}
		if quiet {
			args = append(args, "--quiet")
		}
		args = append(args, planFile)

		before := readFile(t, root, "a.go")
		out, code := run(t, state, root, args...)
		afterBytes, afterErr := os.ReadFile(filepath.Join(root, "a.go"))
		after := ""
		if afterErr == nil {
			after = string(afterBytes)
		}
		unchanged := afterErr == nil && after == before
		label := fmt.Sprintf("seed=%d iter=%d op=%s format=%s force=%v skipRead=%v echoNeg=%v demand=%v optOut=%v dry=%v badAnchor=%v\n%s",
			seed, i, op, format, force, skipRead, echoNeg, demand, optOut, dry, badAnchor, out)

		if !comboExitOK(code) {
			t.Fatalf("%s: exit %d is not a documented status", label, code)
		}

		usage := echoNeg || format == "git" || format == "nope" || (demand && optOut) || (demand && dry)
		if usage {
			if code != 2 {
				t.Fatalf("%s: usage path exited %d want 2", label, code)
			}
			if !unchanged {
				t.Fatalf("%s: usage wrote the tree", label)
			}
			continue
		}

		// create names a path that does not exist yet; ADR-002 is per line of
		// a file that is already there. unlink/rename/replace still need a read.
		if skipRead && !force && op != plan.OpCreate {
			if code != 1 {
				t.Fatalf("%s: unread without --force exited %d want 1", label, code)
			}
			if !unchanged {
				t.Fatalf("%s: unread write mutated the tree", label)
			}
			continue
		}

		if badAnchor {
			if code != 1 {
				t.Fatalf("%s: a false anchor exited %d want 1", label, code)
			}
			if !unchanged {
				t.Fatalf("%s: a false anchor wrote the tree", label)
			}
			continue
		}

		if code == 1 {
			if !unchanged {
				t.Fatalf("%s: ADR-001: exit 1 wrote the tree", label)
			}
		}
	}
}

func TestRandomisedSubcommandUsageStaysExitTwo(t *testing.T) {
	r, seed := comboSeed(t, 321)
	iters := comboIters(t, 60)
	root := tree(t, map[string]string{"a.go": "package a\n"})
	state := t.TempDir()

	type caseFn func() []string
	cases := []caseFn{
		func() []string { return []string{"version", "extra"} },
		func() []string { return []string{"instructions", "extra"} },
		func() []string { return []string{"not-a-command"} },
		func() []string { return []string{"seen", "--dry-run"} },
		func() []string { return []string{"mcp", "--max-result-chars", "-1"} },
		func() []string { return []string{"iter", "add"} },
		func() []string { return []string{"write", "one.mrw", "two.mrw"} },
		func() []string { return []string{"read", "--max-lines", "-1", "a.go"} },
		func() []string { return []string{"read", "-C", "-2", "a.go"} },
		func() []string { return []string{"write", "--format", "git", "-"} },
	}

	for i := 0; i < iters; i++ {
		args := cases[r.Intn(len(cases))]()
		out, code := run(t, state, root, args...)
		if code != 2 {
			t.Fatalf("seed=%d iter=%d args=%v: exit %d want 2 (usage, never check-failed 3)\n%s", seed, i, args, code, out)
		}
	}
}

func TestRandomFileBytesOnReadStayInsideDocumentedExits(t *testing.T) {
	r, seed := comboSeed(t, 421)
	iters := comboIters(t, 40)
	for i := 0; i < iters; i++ {
		n := r.Intn(256)
		buf := make([]byte, n)
		for j := range buf {
			buf[j] = byte(r.Intn(256))
		}
		if r.Intn(4) == 0 && n > 2 {
			buf = append([]byte("a\r\n"), buf...)
		}
		root := t.TempDir()
		name := "blob.txt"
		if r.Intn(3) == 0 {
			name = "blob.go"
		}
		if err := os.WriteFile(filepath.Join(root, name), buf, 0o644); err != nil {
			t.Fatal(err)
		}
		state := t.TempDir()
		out, code := run(t, state, root, "read", name)
		if !comboExitOK(code) || code == 3 {
			t.Fatalf("seed=%d iter=%d bytes=%d: exit %d\n%s", seed, i, len(buf), code, out)
		}
	}
}

func TestASingleEnglishUnreadableDoesNotNameShellSplit(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	out, code := run(t, state, root, "read", "rules")
	if code != 1 {
		t.Fatalf("one missing English-word path exited %d want 1\n%s", code, out)
	}
	if strings.Contains(out, "shell-split") {
		t.Fatalf("one UNREADABLE must not fire the two-or-more hint:\n%s", out)
	}
}

func TestTwoEnglishUnreadablesNameShellSplit(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	out, code := run(t, state, root, "read", "rules", "that")
	if code != 1 {
		t.Fatalf("two missing English-word paths exited %d want 1\n%s", code, out)
	}
	if !strings.Contains(out, "shell-split") {
		t.Fatalf("ADR-015 D4: two English-word UNREADABLE paths must name shell-split:\n%s", out)
	}
	if strings.Contains(out, "glob your shell did not expand") {
		t.Fatalf("the glob hint must not fire on English-word paths:\n%s", out)
	}
}
