package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// keptRig builds mrw once and runs it over one root and one state home, with
// extra environment per call. kept lists the plans in <state home>/mrw-refused.
type keptRig struct {
	t           *testing.T
	root, state string
	bin         string
}

func newKeptRig(t *testing.T) *keptRig {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mrw.exe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return &keptRig{t: t, root: t.TempDir(), state: t.TempDir(), bin: bin}
}

func (k *keptRig) run(stdin string, env []string, args ...string) (string, string, int) {
	k.t.Helper()
	c := exec.Command(k.bin, append([]string{"--root", k.root}, args...)...)
	c.Env = append(append(os.Environ(), "XDG_STATE_HOME="+k.state), env...)
	c.Stdin = strings.NewReader(stdin)
	var o, e strings.Builder
	c.Stdout, c.Stderr = &o, &e
	err := c.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		return o.String(), e.String(), ee.ExitCode()
	} else if err != nil {
		k.t.Fatal(err)
	}
	return o.String(), e.String(), 0
}

func (k *keptRig) dir() string { return filepath.Join(k.state, "mrw-refused") }

func (k *keptRig) kept() []string {
	k.t.Helper()
	ents, err := os.ReadDir(k.dir())
	if err != nil && !os.IsNotExist(err) {
		k.t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, filepath.Join(k.dir(), e.Name()))
	}
	return names
}

func (k *keptRig) reset() {
	k.t.Helper()
	if err := os.RemoveAll(k.dir()); err != nil {
		k.t.Fatal(err)
	}
}

// onlyKept is the one kept plan, its bytes, and fails the test when there are not exactly one.
func (k *keptRig) onlyKept(why string) (string, string) {
	k.t.Helper()
	names := k.kept()
	if len(names) != 1 {
		k.t.Fatalf("%s: %d kept plans %v, want 1", why, len(names), names)
	}
	b, err := os.ReadFile(names[0])
	if err != nil {
		k.t.Fatal(err)
	}
	return names[0], string(b)
}

func (k *keptRig) file(name, body string) {
	k.t.Helper()
	p := filepath.Join(k.root, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		k.t.Fatal(err)
	}
}

func (k *keptRig) read(name string) string {
	k.t.Helper()
	b, err := os.ReadFile(filepath.Join(k.root, name))
	if err != nil {
		k.t.Fatal(err)
	}
	return string(b)
}

// ADR-176. A plan piped to `mrw write -` that ends with nothing in the tree is
// kept beside the state directory, so the caller fixes one line and reruns its
// own command with the path in place of `-` instead of sending the plan again.
func TestAPlanRefusedFromStandardInputIsKept(t *testing.T) {
	k := newKeptRig(t)
	k.file("f.txt", "a\nb\n")
	readF := func() {
		t.Helper()
		if _, e, c := k.run("", nil, "read", "f.txt"); c != 0 {
			t.Fatalf("read f.txt: exit %d\n%s", c, e)
		}
	}
	const fix = "@@ f.txt 1 replace\nA\n"
	// every exit with nothing in the tree, after the plan was read
	cases := []struct {
		name  string
		plan  string
		env   []string
		args  []string
		setup func() func()
	}{
		{name: "a parse refusal", plan: "@@ f.txt 1 replace body=1\nA\nB\n", args: []string{"--no-check"}},
		{name: "a body file that does not exist", plan: "@@ f.txt 1 replace body=@nope.txt\n", args: []string{"--no-check"}},
		{name: "a validation refusal", plan: "@@ f.txt 1 replace\nA\n", args: []string{"--no-check"}, setup: func() func() {
			k.file("f.txt", "a\nb\n")
			return func() {}
		}},
		{name: "a dry run", plan: fix, args: []string{"--no-check", "--dry-run"}, setup: func() func() { readF(); return func() {} }},
		{name: "a lock timeout", plan: fix, args: []string{"--no-check"}, env: []string{"MRW_WRITE_LOCK_TIMEOUT=0"}, setup: func() func() {
			readF()
			t.Setenv("XDG_STATE_HOME", k.state)
			release, err := seen.LockWrites(k.root)
			if err != nil {
				t.Fatal(err)
			}
			return release
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k.t = t
			k.reset()
			k.file("f.txt", "a\nb\n")
			if c.setup != nil {
				defer c.setup()()
			}
			_, errOut, code := k.run(c.plan, c.env, append(append([]string{"write"}, c.args...), "-")...)
			path, got := k.onlyKept(c.name)
			if got != c.plan {
				t.Errorf("kept bytes %q, want the plan %q", got, c.plan)
			}
			if !strings.Contains(errOut, "the plan from standard input is kept at "+path+" (") {
				t.Errorf("stderr does not name %s:\n%s", path, errOut)
			}
			if k.read("f.txt") != "a\nb\n" {
				t.Errorf("the tree changed: %q", k.read("f.txt"))
			}
			// the same run under --json: the document names the path, the exit is the same
			k.reset()
			js, _, jcode := k.run(c.plan, c.env, append(append([]string{"write", "--json"}, c.args...), "-")...)
			var doc struct {
				RefusedPlan string `json:"refused_plan"`
			}
			if err := json.Unmarshal([]byte(js), &doc); err != nil {
				t.Fatalf("--json: %v\n%s", err, js)
			}
			if names := k.kept(); jcode != code || len(names) != 1 || doc.RefusedPlan != names[0] {
				t.Errorf("--json: exit %d (human %d), refused_plan %q, kept %v", jcode, code, doc.RefusedPlan, names)
			}
		})
	}

	t.Run("a refused plan runs again from its path", func(t *testing.T) {
		k.t = t
		k.reset()
		k.file("f.txt", "a\nb\n")
		_, errOut, code := k.run("@@ f.txt 1 replace body=1\nA\nB\n", nil, "write", "--no-check", "-")
		if code == 0 {
			t.Fatalf("the stray line was accepted:\n%s", errOut)
		}
		path, _ := k.onlyKept("parse")
		if err := os.WriteFile(path, []byte("@@ f.txt 1 replace body=1\nA\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		readF()
		if _, e, c := k.run("", nil, "write", "--no-check", path); c != 0 || k.read("f.txt") != "A\nb\n" {
			t.Errorf("rerun from %s: exit %d, f.txt %q\n%s", path, c, k.read("f.txt"), e)
		}
		if len(k.kept()) != 1 {
			t.Errorf("a rerun removed the kept plan: %v", k.kept())
		}
	})

	t.Run("the kept bytes are the parser's, after the CRLF trim", func(t *testing.T) {
		k.t = t
		k.reset()
		k.file("f.txt", "a\nb\n")
		// PowerShell's CRLF after LF lines is dropped before the parser sees it
		if _, _, c := k.run("@@ f.txt 1 replace body=1\nA\nB\n\r\n", nil, "write", "--no-check", "-"); c == 0 {
			t.Fatal("the stray line was accepted")
		}
		if _, got := k.onlyKept("crlf"); got != "@@ f.txt 1 replace body=1\nA\nB\n" {
			t.Errorf("kept %q, want the plan without PowerShell's CRLF", got)
		}
	})

	t.Run("an apply_patch document is kept as sent and reruns with the same flag", func(t *testing.T) {
		k.t = t
		k.reset()
		k.file("f.txt", "a\nb\n")
		doc := "*** Begin Patch\n*** Update File: f.txt\n@@\n-zz\n+A\n*** End Patch\n"
		if _, e, c := k.run(doc, nil, "write", "--no-check", "--format=apply_patch", "-"); c == 0 {
			t.Fatalf("a patch whose context is not in the file was applied:\n%s", e)
		}
		path, got := k.onlyKept("apply_patch")
		if got != doc {
			t.Errorf("kept %q, want %q", got, doc)
		}
		fixed := strings.Replace(doc, "-zz", "-a", 1)
		if err := os.WriteFile(path, []byte(fixed), 0o600); err != nil {
			t.Fatal(err)
		}
		readF()
		if _, e, c := k.run("", nil, "write", "--no-check", "--format=apply_patch", path); c != 0 || k.read("f.txt") != "A\nb\n" {
			t.Errorf("rerun: exit %d, f.txt %q\n%s", c, k.read("f.txt"), e)
		}
	})

	t.Run("a commit that landed nothing keeps the plan", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("a read-only directory does not stop this user from writing it")
		}
		k.t = t
		k.reset()
		if err := os.MkdirAll(filepath.Join(k.root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		k.file("d/x.txt", "x\n")
		if _, e, c := k.run("", nil, "read", "d/x.txt"); c != 0 {
			t.Fatalf("read: exit %d\n%s", c, e)
		}
		if err := os.Chmod(filepath.Join(k.root, "d"), 0o555); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(filepath.Join(k.root, "d"), 0o755) }()
		plan := "@@ d/x.txt - unlink\n"
		if _, e, c := k.run(plan, nil, "write", "--no-check", "-"); c == 0 {
			t.Fatalf("an unlink in a read-only directory landed:\n%s", e)
		}
		if _, got := k.onlyKept("commit failure"); got != plan {
			t.Errorf("kept %q, want %q", got, plan)
		}
	})

	t.Run("two refusals get two files, 0600 in a 0700 directory", func(t *testing.T) {
		k.t = t
		k.reset()
		for i := 0; i < 2; i++ {
			k.run("@@ f.txt 1 replace body=1\nA\nB\n", nil, "write", "--no-check", "-")
		}
		names := k.kept()
		if len(names) != 2 {
			t.Fatalf("%d kept plans %v, want 2", len(names), names)
		}
		if runtime.GOOS == "windows" {
			return
		}
		if fi, err := os.Stat(k.dir()); err != nil || fi.Mode().Perm() != 0o700 {
			t.Errorf("directory mode %v (%v), want 0700", fi, err)
		}
		for _, n := range names {
			if fi, err := os.Stat(n); err != nil || fi.Mode().Perm() != 0o600 || !strings.HasSuffix(n, ".plan") {
				t.Errorf("%s: %v (%v), want a .plan of mode 0600", n, fi, err)
			}
		}
	})

	t.Run("only the newest 16 stay", func(t *testing.T) {
		k.t = t
		k.reset()
		if err := os.MkdirAll(k.dir(), 0o700); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 16; i++ {
			if err := os.WriteFile(filepath.Join(k.dir(), fmt.Sprintf("20200101T0000%02dZ-1-aaaa.plan", i)), []byte("old\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		k.run("@@ f.txt 1 replace body=1\nA\nB\n", nil, "write", "--no-check", "-")
		names := k.kept()
		if len(names) != 16 {
			t.Fatalf("%d kept plans, want 16", len(names))
		}
		if _, err := os.Stat(filepath.Join(k.dir(), "20200101T000000Z-1-aaaa.plan")); err == nil {
			t.Error("the oldest plan was not removed")
		}
		if _, err := os.Stat(filepath.Join(k.dir(), "20200101T000015Z-1-aaaa.plan")); err != nil {
			t.Errorf("the newest old plan was removed: %v", err)
		}
	})

	t.Run("a plan over 16 MiB is parsed as today and not kept, and says why", func(t *testing.T) {
		k.t = t
		k.reset()
		small := "@@ f.txt 1 replace body=1\nA\nB\n"
		_, _, want := k.run(small, nil, "write", "--no-check", "-")
		k.reset()
		big := strings.Repeat("# "+strings.Repeat("x", 1022)+"\n", 17*1024) + small
		js, errOut, code := k.run(big, nil, "write", "--json", "--no-check", "-")
		if code != want || len(k.kept()) != 0 || !strings.Contains(errOut, "was not kept") || !strings.Contains(errOut, "16 MiB") || strings.Contains(js, "refused_plan") {
			t.Errorf("exit %d (want %d), kept %v, stderr %q, json names a plan: %v", code, want, k.kept(), errOut, strings.Contains(js, "refused_plan"))
		}
	})

	t.Run("an unwritable state home keeps nothing, says why, and changes nothing else", func(t *testing.T) {
		k.t = t
		k.reset()
		small := "@@ f.txt 1 replace body=1\nA\nB\n"
		_, _, want := k.run(small, nil, "write", "--no-check", "-")
		if err := os.RemoveAll(k.dir()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(k.dir(), []byte("in the way"), 0o600); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Remove(k.dir()) }()
		_, errOut, code := k.run(small, nil, "write", "--no-check", "-")
		if code != want || !strings.Contains(errOut, "was not kept") {
			t.Errorf("exit %d (want %d), stderr %q", code, want, errOut)
		}
	})
}

// ADR-176 Decision 1. Anything in the tree keeps nothing, and neither does a
// run that never read a standard-input plan.
func TestAWriteThatLandedKeepsNoPlan(t *testing.T) {
	k := newKeptRig(t)
	none := func(why string) {
		t.Helper()
		if names := k.kept(); len(names) != 0 {
			t.Errorf("%s kept %v", why, names)
		}
	}
	k.file("f.txt", "a\nb\n")
	if _, e, c := k.run("", nil, "read", "f.txt"); c != 0 {
		t.Fatalf("read: exit %d\n%s", c, e)
	}
	if _, e, c := k.run("@@ f.txt 1 replace\nA\n", nil, "write", "--no-check", "-"); c != 0 {
		t.Fatalf("a valid plan: exit %d\n%s", c, e)
	}
	none("a plan that landed (exit 0)")

	if _, _, c := k.run("@@ f.txt 2 replace\nB\n", nil, "write", "--no-check", "--then-sh", "exit 1", "-"); c != 3 {
		t.Errorf("a failing step: exit %d, want 3", c)
	}
	none("a plan that landed and whose step failed (exit 3)")

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		if err := os.MkdirAll(filepath.Join(k.root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		k.file("d/x.txt", "x\n")
		if _, e, c := k.run("", nil, "read", "f.txt", "d/x.txt"); c != 0 {
			t.Fatalf("read: exit %d\n%s", c, e)
		}
		if err := os.Chmod(filepath.Join(k.root, "d"), 0o555); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chmod(filepath.Join(k.root, "d"), 0o755) }()
		out, _, c := k.run("@@ f.txt 1 replace\nZ\n@@ d/x.txt - unlink\n", nil, "write", "--no-check", "-")
		if c != exitUsage || !strings.Contains(out, "PARTIALLY APPLIED") {
			t.Fatalf("exit %d, want %d and PARTIALLY APPLIED:\n%s", c, exitUsage, out)
		}
		none("a partial commit")
	}

	// a plan from a file, --create, --sub and --rewrite are not standard-input plans
	plan := filepath.Join(t.TempDir(), "bad.plan")
	if err := os.WriteFile(plan, []byte("@@ f.txt 1 replace body=1\nA\nB\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, c := k.run("", nil, "write", "--no-check", plan); c == 0 {
		t.Error("a bad plan file was accepted")
	}
	none("a refused plan file")
	if _, _, c := k.run("new\n", nil, "write", "--create", "f.txt"); c == 0 {
		t.Error("--create over an existing file was accepted")
	}
	none("a refused --create")
	if _, _, c := k.run("", nil, "write", "--grep", "nomatchanywhere", "--sub", "y", "."); c == 0 {
		t.Error("a --sub with no match was accepted")
	}
	none("a --sub that matched nothing")
	if _, _, c := k.run("", nil, "write", "--ast-grep", "nomatch($A)", "--rewrite", "y", "."); c == 0 {
		t.Error("a --rewrite with no match was accepted")
	}
	none("a --rewrite that matched nothing")

	// a refusal before the plan is read has nothing to keep
	if _, _, c := k.run("@@ f.txt 1 replace\nA\n", nil, "write", "--format=bogus", "-"); c == 0 {
		t.Error("an unknown format was accepted")
	}
	none("an unknown --format")
}
