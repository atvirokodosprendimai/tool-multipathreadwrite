package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// ADR-206 T2: M and N are two builds of one tag that differ in one refusal.
// The test builds both with bench/stage-9/build.sh and drives them from
// identical fresh trees and state homes, row by row of the task's Goal table.

type s9Run struct {
	code        int
	stdout, err string
}

// s9Exec runs bin with a minimal environment. The state home is the only
// thing the caller varies.
func s9Exec(t *testing.T, bin, root, state, stdin string, args ...string) s9Run {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + filepath.Dir(root), "XDG_STATE_HOME=" + state}
	cmd.Stdin = strings.NewReader(stdin)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("%s %v: %v", bin, args, err)
		}
		code = ee.ExitCode()
	}
	return s9Run{code, o.String(), e.String()}
}

const s9Text = "line1\nline2\nline3\nline4\nline5\n"

// s9Sandbox makes the same fresh root and state home at the same paths for
// whichever binary is driven next, so their outputs compare byte for byte.
func s9Sandbox(t *testing.T, base string) (root, state string) {
	t.Helper()
	if err := os.RemoveAll(base); err != nil {
		t.Fatal(err)
	}
	root, state = filepath.Join(base, "root"), filepath.Join(base, "state")
	for _, d := range []string{root, state} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, state
}

// s9Written makes a.txt through the binary itself, so the ledger holds it as a
// file mrw wrote and no line of it as shown (ADR-203's case).
func s9Written(t *testing.T, bin, root, state string) {
	t.Helper()
	if r := s9Exec(t, bin, root, state, s9Text, "--root", root, "write", "--create", "a.txt"); r.code != 0 {
		t.Fatalf("create a.txt: %+v", r)
	}
}

type s9Event struct {
	Path         string `json:"path"`
	Op           string `json:"op"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	FirstUnshown int    `json:"first_unshown"`
	SHABefore    string `json:"sha_before"`
}

// s9Log reads the kept log: whether it exists, and its objects.
func s9Log(t *testing.T, state string) (bool, []s9Event) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(state, "mrw", "stage9-exposed.jsonl"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var evs []s9Event
	for _, ln := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if ln == "" {
			continue
		}
		var e s9Event
		dec := json.NewDecoder(strings.NewReader(ln))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&e); err != nil {
			t.Fatalf("log line %q: %v", ln, err)
		}
		evs = append(evs, e)
	}
	return true, evs
}

func s9Sum(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func s9File(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// s9Tag is the newest v* tag reachable from HEAD, or "" with the reason.
func s9Tag(t *testing.T) (repo, tag, skip string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", "git is not on PATH"
	}
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", "not a git checkout"
	}
	repo = strings.TrimSpace(string(top))
	out, err := exec.Command("git", "-C", repo, "tag", "--merged", "HEAD", "--list", "v*", "--sort=-v:refname").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return repo, "", "no v* tag is reachable in this clone (a shallow checkout has none)"
	}
	tag = strings.Fields(string(out))[0]
	// The build needs the module cache; offline, a missing module is a skip.
	chk := exec.Command("go", "list", "-mod=mod", "-m", "all")
	chk.Dir = repo
	chk.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=mod")
	if err := chk.Run(); err != nil {
		return repo, "", "the module cache lacks a dependency of " + tag + " (go list with GOPROXY=off failed)"
	}
	return repo, tag, ""
}

var s9Sha = regexp.MustCompile(`(?m)^([mn]) sha256: ([0-9a-f]{64})$`)

// s9Kept hides the name of the plan file a refusal keeps, which holds a clock
// reading and a pid: the one thing two runs of one refusal do not share.
var s9Kept = regexp.MustCompile(`\S*mrw-refused/\S+\.plan`)

func s9Same(r s9Run) s9Run {
	r.stdout, r.err = s9Kept.ReplaceAllString(r.stdout, "KEPT"), s9Kept.ReplaceAllString(r.err, "KEPT")
	return r
}

func TestArmNDiffersFromMInOneRefusal(t *testing.T) {
	repo, tag, skip := s9Tag(t)
	if skip != "" {
		t.Skip(skip)
	}
	t.Logf("tag built: %s", tag)
	out := t.TempDir()
	script := filepath.Join(repo, "bench", "stage-9", "build.sh")
	cmd := exec.Command("sh", script, tag, out)
	cmd.Dir = repo
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build.sh %s %s: %v\n%s", tag, out, err, b)
	} else {
		sums := map[string]string{}
		for _, m := range s9Sha.FindAllStringSubmatch(string(b), -1) {
			sums[m[1]] = m[2]
		}
		for arm, name := range map[string]string{"m": "mrw-m", "n": "mrw-n"} {
			bin, err := os.ReadFile(filepath.Join(out, name))
			if err != nil {
				t.Fatal(err)
			}
			if sums[arm] != s9Sum(string(bin)) {
				t.Fatalf("build.sh printed %q for %s, the file hashes to %s", sums[arm], name, s9Sum(string(bin)))
			}
		}
		if sums["m"] == sums["n"] {
			t.Fatal("M and N are the same binary")
		}
	}
	m, n := filepath.Join(out, "mrw-m"), filepath.Join(out, "mrw-n")
	work := t.TempDir()

	const stale = "no recorded showing in this version"
	// both runs a plan against fresh identical sandboxes after a create of a.txt.
	both := func(t *testing.T, plan string, extra ...string) (rm, rn s9Run, rootM, rootN, stateN string) {
		t.Helper()
		for i, bin := range []string{m, n} {
			root, state := s9Sandbox(t, filepath.Join(work, "case"))
			s9Written(t, bin, root, state)
			args := append([]string{"--root", root, "write"}, extra...)
			r := s9Exec(t, bin, root, state, plan, append(args, "-")...)
			if i == 0 {
				rm, rootM = r, s9File(t, root, "a.txt")
			} else {
				rn, rootN, stateN = r, s9File(t, root, "a.txt"), state
			}
		}
		return
	}

	t.Run("a number alone on a line of a file mrw wrote: M refuses, N lands and logs", func(t *testing.T) {
		rm, rn, fm, fn, state := both(t, "@@ a.txt 3 replace\nX\n")
		if rm.code != 1 || !strings.Contains(rm.stdout+rm.err, stale) || fm != s9Text {
			t.Fatalf("M: %+v", rm)
		}
		if rn.code != 0 || fn != "line1\nline2\nX\nline4\nline5\n" {
			t.Fatalf("N: %+v file %q", rn, fn)
		}
		ex, evs := s9Log(t, state)
		want := s9Event{"a.txt", "replace", 3, 3, 3, s9Sum(s9Text)}
		if !ex || len(evs) != 1 || evs[0] != want {
			t.Fatalf("log %v %+v, want one %+v", ex, evs, want)
		}
	})
	for _, c := range []struct {
		name, plan string
		want       s9Event
		file       string
	}{
		{"numbered delete of two lines", "@@ a.txt 2-3 delete\n", s9Event{"a.txt", "delete", 2, 3, 2, s9Sum(s9Text)}, "line1\nline4\nline5\n"},
		{"insert-after", "@@ a.txt 2 insert-after\nNEW\n", s9Event{"a.txt", "insert-after", 2, 2, 2, s9Sum(s9Text)}, "line1\nline2\nNEW\nline3\nline4\nline5\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rm, rn, _, fn, state := both(t, c.plan)
			if rm.code != 1 || !strings.Contains(rm.stdout+rm.err, stale) {
				t.Fatalf("M: %+v", rm)
			}
			if rn.code != 0 || fn != c.file {
				t.Fatalf("N: %+v file %q", rn, fn)
			}
			if ex, evs := s9Log(t, state); !ex || len(evs) != 1 || evs[0] != c.want {
				t.Fatalf("log %v %+v, want one %+v", ex, evs, c.want)
			}
		})
	}

	t.Run("a plan that commits one file and fails on the other logs the hunk that landed", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("the commit failure is made with chflags uchg, which only macOS has")
		}
		root, state := s9Sandbox(t, filepath.Join(work, "partial"))
		s9Written(t, n, root, state)
		bp := filepath.Join(root, "b.txt")
		if err := os.WriteFile(bp, []byte(s9Text), 0o644); err != nil {
			t.Fatal(err)
		}
		if r := s9Exec(t, n, root, state, "", "--root", root, "read", "b.txt"); r.code != 0 {
			t.Fatalf("read b.txt: %+v", r)
		}
		if err := exec.Command("chflags", "uchg", bp).Run(); err != nil {
			t.Skip("chflags uchg is not available: " + err.Error())
		}
		t.Cleanup(func() { _ = exec.Command("chflags", "nouchg", bp).Run() })
		r := s9Exec(t, n, root, state, "@@ a.txt 3 replace\nX\n@@ b.txt 3 replace\nY\n", "--root", root, "write", "-")
		if !strings.Contains(r.stdout+r.err, "PARTIALLY APPLIED") {
			t.Fatalf("the plan did not commit partially: %+v", r)
		}
		if got := s9File(t, root, "a.txt"); got != "line1\nline2\nX\nline4\nline5\n" {
			t.Fatalf("a.txt %q", got)
		}
		ex, evs := s9Log(t, state)
		want := s9Event{"a.txt", "replace", 3, 3, 3, s9Sum(s9Text)}
		if !ex || len(evs) != 1 || evs[0] != want {
			t.Fatalf("log %v %+v, want one %+v", ex, evs, want)
		}
	})

	t.Run("a dry run and a plan whose other hunk fails land and log nothing", func(t *testing.T) {
		for _, c := range []struct {
			name, plan string
			extra      []string
		}{
			{"dry run", "@@ a.txt 3 replace\nX\n", []string{"--dry-run"}},
			{"failing sibling", "@@ a.txt 3 replace\nX\n@@ a.txt /no such line/ replace\nY\n", nil},
		} {
			rm, rn, fm, fn, state := both(t, c.plan, c.extra...)
			if fm != s9Text || fn != s9Text {
				t.Fatalf("%s: a file changed: %q %q", c.name, fm, fn)
			}
			if rm.code != 1 {
				t.Fatalf("%s: M %+v", c.name, rm)
			}
			if c.name == "failing sibling" && rn.code != 1 {
				t.Fatalf("%s: N %+v", c.name, rn)
			}
			if c.name == "dry run" && rn.code != 0 {
				t.Fatalf("%s: N %+v", c.name, rn)
			}
			if ex, evs := s9Log(t, state); !ex || len(evs) != 0 {
				t.Fatalf("%s: log exists=%v %+v, want it empty", c.name, ex, evs)
			}
		}
	})

	t.Run("the same edit with anchor= lands in both and logs nothing", func(t *testing.T) {
		rm, rn, fm, fn, state := both(t, "@@ a.txt 3 replace anchor=\"line3\"\nX\n")
		if rm.code != 0 || rn.code != 0 || fm != fn {
			t.Fatalf("M %+v N %+v files %q %q", rm, rn, fm, fn)
		}
		if ex, evs := s9Log(t, state); !ex || len(evs) != 0 {
			t.Fatalf("log exists=%v %+v, want it present and empty", ex, evs)
		}
	})

	t.Run("a line no read has served, in a file mrw did not write, is refused by both alike", func(t *testing.T) {
		var runs [2]s9Run
		for i, bin := range []string{m, n} {
			root, state := s9Sandbox(t, filepath.Join(work, "unread"))
			if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(s9Text), 0o644); err != nil {
				t.Fatal(err)
			}
			runs[i] = s9Same(s9Exec(t, bin, root, state, "@@ a.txt 2 replace\nX\n", "--root", root, "write", "-"))
			if ex, evs := s9Log(t, state); i == 1 && (len(evs) != 0 || ex) {
				t.Fatalf("a refused plan logged: %v %+v", ex, evs)
			}
		}
		if runs[0].code != 1 || runs[0] != runs[1] || runs[0].stdout+runs[0].err == "" {
			t.Fatalf("M %+v N %+v", runs[0], runs[1])
		}
	})

	t.Run("version, instructions and skill print what M prints", func(t *testing.T) {
		for _, args := range [][]string{{"version"}, {"instructions"}, {"instructions", "--core"}, {"skill"}} {
			var runs [2]s9Run
			for i, bin := range []string{m, n} {
				root, state := s9Sandbox(t, filepath.Join(work, "same"))
				runs[i] = s9Exec(t, bin, root, state, "", args...)
			}
			if runs[0].code != 0 || runs[0] != runs[1] || runs[0].stdout == "" {
				t.Fatalf("%v: M %+v N %+v", args, runs[0], runs[1])
			}
		}
	})

	t.Run("a log path that is a directory changes nothing an agent sees", func(t *testing.T) {
		var runs [2]s9Run
		for i := range runs {
			root, state := s9Sandbox(t, filepath.Join(work, "dirlog"))
			s9Written(t, n, root, state)
			if i == 1 {
				p := filepath.Join(state, "mrw", "stage9-exposed.jsonl")
				if err := os.RemoveAll(p); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(p, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			runs[i] = s9Exec(t, n, root, state, "@@ a.txt 3 replace\nX\n", "--root", root, "write", "-")
			if got := s9File(t, root, "a.txt"); got != "line1\nline2\nX\nline4\nline5\n" {
				t.Fatalf("a.txt %q", got)
			}
		}
		if runs[0].code != 0 || runs[0] != runs[1] {
			t.Fatalf("writable %+v, directory %+v", runs[0], runs[1])
		}
	})
}
