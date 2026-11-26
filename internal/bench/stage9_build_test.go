package bench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
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
	// A hung binary (a write blocked on a FIFO) is a failure, not a stuck test.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + filepath.Dir(root), "XDG_STATE_HOME=" + state}
	if runtime.GOOS == "windows" {
		// Go's os.UserHomeDir reads USERPROFILE on Windows, not HOME (#802).
		cmd.Env = append(cmd.Env, "USERPROFILE="+filepath.Dir(root))
	}
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

// s9Log reads the kept log: whether it exists, and its objects. The log is
// stage9-exposed.jsonl in the parent of the state home (ADR-211 Decision 3).
func s9Log(t *testing.T, state string) (bool, []s9Event) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(state), "stage9-exposed.jsonl"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var evs []s9Event
	if len(b) == 0 {
		return true, nil
	}
	for _, ln := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		if ln == "" {
			t.Fatalf("the log holds a blank line: %q", b)
		}
		var e s9Event
		dec := json.NewDecoder(strings.NewReader(ln))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&e); err != nil {
			t.Fatalf("log line %q: %v", ln, err)
		}
		if _, err := dec.Token(); err != io.EOF {
			t.Fatalf("log line %q is not exactly one JSON object", ln)
		}
		evs = append(evs, e)
	}
	return true, evs
}

func s9Sum(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// s9Exe is the path an arm runs from. build.sh writes the extensionless
// mrw-m and mrw-n that builds.md records and hashes; Windows starts only a
// name with an executable extension, so there each arm runs through a copy
// named .exe whose bytes hash to what build.sh printed (#802). The build
// outputs themselves are left as they are.
func s9Exe(t *testing.T, path, sum string) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		return path
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exe := path + ".exe"
	if err := os.WriteFile(exe, b, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(exe); err != nil || s9Sum(string(got)) != sum {
		t.Fatalf("%s does not hash to %s, which build.sh printed for %s: %v", exe, sum, path, err)
	}
	return exe
}

func s9File(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// s9Tag is the newest v* tag reachable from HEAD, or "" with the reason.
func s9Tag(t *testing.T) (repo, tag string, doc []byte, skip string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", nil, "git is not on PATH"
	}
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", nil, "not a git checkout"
	}
	repo = strings.TrimSpace(string(top))
	// The pinned tag is the one builds.md names, never the newest.
	doc, err = os.ReadFile(filepath.Join(repo, "bench", "stage-9", "builds.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := s9TagLine.FindSubmatch(doc)
	if m == nil {
		t.Fatal("bench/stage-9/builds.md has no `- Tag: `vX` line")
	}
	tag = string(m[1])
	if exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag+"^{commit}").Run() != nil {
		return repo, tag, doc, "the pinned tag " + tag + " is not in this clone (a shallow checkout has none)"
	}
	return repo, tag, doc, ""
}

var (
	s9TagLine  = regexp.MustCompile("(?m)^- Tag: `([^`]+)`")
	s9Platform = regexp.MustCompile("(?m)^- Platform: `([^`]+)`")
)

func s9Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var s9Sha = regexp.MustCompile(`(?m)^([mn]) sha256: ([0-9a-f]{64})$`)

// s9Kept is the generated name of the plan file a refusal keeps, which holds a
// clock reading and a pid: the one thing two runs of one refusal do not share.
// Only that last component is hidden; the directory before it stays in the
// comparison, which is the same for both arms because each reuses one sandbox
// path. The separator is the platform's own: on Unix a backslash is part of a
// name (#802).
var (
	s9Sep  = regexp.QuoteMeta(string(filepath.Separator))
	s9Kept = regexp.MustCompile(`mrw-refused` + s9Sep + `[^\s` + s9Sep + `]+\.plan`)
)

func s9Same(r s9Run) s9Run {
	kept := "mrw-refused" + string(filepath.Separator) + "KEPT.plan"
	r.stdout, r.err = s9Kept.ReplaceAllLiteralString(r.stdout, kept), s9Kept.ReplaceAllLiteralString(r.err, kept)
	return r
}

func TestArmNDiffersFromMInOneRefusal(t *testing.T) {
	repo, tag, doc, skip := s9Tag(t)
	if skip != "" {
		t.Skip(skip)
	}
	t.Logf("tag built: %s", tag)
	out := t.TempDir()
	script := filepath.Join(repo, "bench", "stage-9", "build.sh")
	cmd := exec.Command("sh", script, tag, out)
	cmd.Dir = repo
	built, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build.sh %s %s: %v\n%s", tag, out, err, built)
	}
	sums := map[string]string{}
	for _, f := range s9Sha.FindAllStringSubmatch(string(built), -1) {
		sums[f[1]] = f[2]
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
	// The hashes are per platform: compare them where builds.md recorded them.
	pinned := map[string]string{}
	for _, f := range s9Sha.FindAllStringSubmatch(string(doc), -1) {
		pinned[f[1]] = f[2]
	}
	plat := s9Platform.FindSubmatch(doc)
	if len(pinned) != 2 || plat == nil {
		t.Fatalf("bench/stage-9/builds.md records hashes %v and platform %q", pinned, plat)
	}
	if here := s9Env("GOOS", runtime.GOOS) + "/" + s9Env("GOARCH", runtime.GOARCH); here == string(plat[1]) {
		for arm := range pinned {
			if sums[arm] != pinned[arm] {
				t.Fatalf("%s built to %s, builds.md records %s", arm, sums[arm], pinned[arm])
			}
		}
	} else {
		t.Logf("built on %s, builds.md records %s: the hashes are not compared", here, plat[1])
	}
	m, n := s9Exe(t, filepath.Join(out, "mrw-m"), sums["m"]), s9Exe(t, filepath.Join(out, "mrw-n"), sums["n"])
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
		// b.txt is made through the binary too, so its line 3 is exposed as a.txt's
		// is; the log must still hold only the file that landed.
		bp := filepath.Join(root, "b.txt")
		if r := s9Exec(t, n, root, state, s9Text, "--root", root, "write", "--create", "b.txt"); r.code != 0 {
			t.Fatalf("create b.txt: %+v", r)
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
			if ex, evs := s9Log(t, state); ex || len(evs) != 0 {
				t.Fatalf("%s: log exists=%v %+v, want none", c.name, ex, evs)
			}
		}
	})

	t.Run("the same edit with anchor= lands in both and leaves no log", func(t *testing.T) {
		rm, rn, fm, fn, state := both(t, "@@ a.txt 3 replace anchor=\"line3\"\nX\n")
		if rm.code != 0 || rn.code != 0 || fm != fn {
			t.Fatalf("M %+v N %+v files %q %q", rm, rn, fm, fn)
		}
		if ex, evs := s9Log(t, state); ex || len(evs) != 0 {
			t.Fatalf("log exists=%v %+v, want no file at all: a write with no exposed hunk touches nothing", ex, evs)
		}
		s9NoLogInside(t, state)
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

	t.Run("a log path that is a directory, or a FIFO with no reader, changes nothing an agent sees but one stderr line", func(t *testing.T) {
		for _, kind := range []string{"directory", "FIFO"} {
			// The directory case needs no tool: only the FIFO case is skipped without mkfifo.
			if kind == "FIFO" {
				// Git Bash puts an mkfifo on PATH whose FIFO is no named pipe to a
				// native Windows program: an append to it succeeds (#802).
				if runtime.GOOS == "windows" {
					t.Skip("an MSYS FIFO is not a named pipe to a native program: the directory case ran, the FIFO case did not")
				}
				if _, err := exec.LookPath("mkfifo"); err != nil {
					t.Skip("mkfifo is not on PATH: the directory case ran, the FIFO case did not")
				}
			}
			var runs [2]s9Run
			for i := range runs {
				root, state := s9Sandbox(t, filepath.Join(work, "badlog"))
				s9Written(t, n, root, state)
				if i == 1 {
					p := filepath.Join(filepath.Dir(state), "stage9-exposed.jsonl")
					var err error
					if kind == "directory" {
						err = os.MkdirAll(p, 0o755)
					} else {
						err = exec.Command("mkfifo", p).Run()
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				runs[i] = s9Exec(t, n, root, state, "@@ a.txt 3 replace\nX\n", "--root", root, "write", "-")
				if got := s9File(t, root, "a.txt"); got != "line1\nline2\nX\nline4\nline5\n" {
					t.Fatalf("%s: a.txt %q", kind, got)
				}
			}
			rest, lines := s9StripLogLine(runs[1].err)
			if runs[0].code != 0 || runs[1].code != 0 || runs[0].stdout != runs[1].stdout || lines != 1 || rest != runs[0].err {
				t.Fatalf("%s: writable %+v, bad %+v (stderr lines %d, rest %q): want the same result and exactly one mrw: stage9 log: line", kind, runs[0], runs[1], lines, rest)
			}
			if _, lines := s9StripLogLine(runs[0].err); lines != 0 {
				t.Fatalf("a writable log printed a stage9 log line: %q", runs[0].err)
			}
		}
	})

	t.Run("an event logged by one write survives a later write whose append fails", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("a read-only log is writable to root")
		}
		root, state := s9Sandbox(t, filepath.Join(work, "later"))
		s9Written(t, n, root, state)
		first := s9Exec(t, n, root, state, "@@ a.txt 3 replace\nX\n", "--root", root, "write", "-")
		if first.code != 0 || strings.Contains(first.err, "stage9 log") {
			t.Fatalf("first: %+v", first)
		}
		log := filepath.Join(filepath.Dir(state), "stage9-exposed.jsonl")
		before, err := os.ReadFile(log)
		if err != nil || strings.Count(string(before), "\n") != 1 {
			t.Fatalf("the first write's log is %q (%v), want one event", before, err)
		}
		if err := os.Chmod(log, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(log, 0o644) })
		second := s9Exec(t, n, root, state, "@@ a.txt 1 replace\nY\n", "--root", root, "write", "-")
		if got := s9File(t, root, "a.txt"); second.code != 0 || got != "Y\nline2\nX\nline4\nline5\n" {
			t.Fatalf("second: %+v file %q, want the write to land all the same", second, got)
		}
		if _, lines := s9StripLogLine(second.err); lines != 1 {
			t.Errorf("second write's stderr %q, want exactly one mrw: stage9 log: line", second.err)
		}
		if after, _ := os.ReadFile(log); string(after) != string(before) {
			t.Errorf("the log changed: %q, want the first event kept as it was", after)
		}
	})

	// A relative XDG_STATE_HOME never reaches the log: mrw itself refuses to run on it.
	t.Run("with XDG_STATE_HOME unset the write is unchanged and says so once", func(t *testing.T) {
		for _, home := range []string{""} {
			root, _ := s9Sandbox(t, filepath.Join(work, "nohome"))
			s9Written(t, n, root, home)
			r := s9Exec(t, n, root, home, "@@ a.txt 3 replace\nX\n", "--root", root, "write", "-")
			if _, lines := s9StripLogLine(r.err); r.code != 0 || lines != 1 || s9File(t, root, "a.txt") != "line1\nline2\nX\nline4\nline5\n" {
				t.Errorf("XDG_STATE_HOME=%q: %+v, want the write landed and one mrw: stage9 log: line", home, r)
			}
		}
	})
}

// s9StripLogLine removes the lines that start with the log's failure prefix and
// says how many there were.
func s9StripLogLine(stderr string) (rest string, n int) {
	var keep []string
	for _, ln := range strings.SplitAfter(stderr, "\n") {
		if strings.HasPrefix(ln, "mrw: stage9 log: ") {
			n++
			continue
		}
		keep = append(keep, ln)
	}
	return strings.Join(keep, ""), n
}

// s9NoLogInside fails when the state home holds a stage9-exposed.jsonl anywhere:
// the log is beside it, in its parent.
func s9NoLogInside(t *testing.T, state string) {
	t.Helper()
	_ = filepath.WalkDir(state, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Name() == "stage9-exposed.jsonl" {
			t.Errorf("the log is inside the state home: %s", p)
		}
		return nil
	})
}
