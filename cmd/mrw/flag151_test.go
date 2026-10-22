package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// planFlagSentence is what ADR-151 appends for a flag that names the plan.
const planFlagSentence = "the plan is an argument, not a flag: mrw write PLAN, or mrw write - to read it from standard input"

// ADR-151 decision 1. A flag a caller types on `mrw write` to name the plan is
// still a usage error, exit 2 with the old prefix, and now ends with the
// argument's spelling. -f and -p read as another tool's flag and are worded as
// before, and read has no plan.
func TestAPlanFlagOnWriteIsAnsweredWithTheArgument(t *testing.T) {
	root := t.TempDir()
	for _, flag := range []string{"plan", "file", "input", "stdin"} {
		out, code := runIn(t, root, "write", "--"+flag, "x.plan")
		if code != 2 || !strings.Contains(out, "flag provided but not defined: --"+flag) || !strings.HasSuffix(strings.TrimSpace(out), "; "+planFlagSentence) {
			t.Errorf("mrw write --%s: exit %d, want 2 with the old prefix and the plan sentence last:\n%s", flag, code, out)
		}
	}
	for _, argv := range [][]string{{"write", "-f", "x.plan"}, {"write", "-p", "x.plan"}, {"read", "--plan", "x"}} {
		out, code := runIn(t, root, argv...)
		if code != 2 || strings.Contains(out, "the plan is an argument") {
			t.Errorf("mrw %s: exit %d, want 2 and no plan sentence:\n%s", strings.Join(argv, " "), code, out)
		}
	}
}

// ADR-151 decisions 2 and 3, through the built binary, since the note is printed
// by main before the parser runs. In a working directory holding a directory 3,
// every spelling of an all-digit short root before the subcommand prints the
// note and serves exactly what --root 3 serves, with the same exit; the long
// spelling, a non-digit value and -C after read print none. With no directory 3
// the refusal and its exit are unchanged and the note is there.
func TestAnAllDigitShortRootBeforeTheVerbIsNamed(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	cwd, state := t.TempDir(), t.TempDir()
	for _, d := range []string{"3", "sub"} {
		if err := os.MkdirAll(filepath.Join(cwd, d), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cwd, d, "f.txt"), []byte("in "+d+"\nneedle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(dir string, args ...string) (stdout, stderr string, code int) {
		c := exec.Command(bin, args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		var o, e strings.Builder
		c.Stdout, c.Stderr = &o, &e
		err := c.Run()
		if ee, ok := err.(*exec.ExitError); ok {
			return o.String(), e.String(), ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), 0
	}
	const note = `-C before the subcommand is --root, so "3" is taken as the root directory`
	wantOut, _, wantCode := run(cwd, "--root", "3", "read", "f.txt")
	if wantCode != 0 || !strings.Contains(wantOut, "in 3") {
		t.Fatalf("--root 3 read f.txt: exit %d, want 0 serving 3/f.txt:\n%s", wantCode, wantOut)
	}
	for _, pre := range [][]string{{"-C", "3"}, {"-C=3"}, {"--C", "3"}, {"--C=3"}, {"-C", "sub", "-C", "3"}} {
		out, errOut, code := run(cwd, append(pre, "read", "f.txt")...)
		if code != wantCode || out != wantOut || strings.Count(errOut, note) != 1 || !strings.Contains(errOut, "mrw read -C 3") {
			t.Errorf("mrw %q read f.txt: exit %d, stdout same as --root 3: %v, want the note once on stderr:\n%s", pre, code, out == wantOut, errOut)
		}
	}
	// The parser passes over an empty token and still takes the root after it
	// (it then prints the help, the empty token standing where the subcommand
	// goes), so the walk passes over it too and the note is given once.
	if _, errOut, _ := run(cwd, "", "-C", "3", "read", "f.txt"); strings.Count(errOut, note) != 1 {
		t.Errorf(`mrw "" -C 3 read f.txt: want the note once on stderr:\n%s`, errOut)
	}
	// No note where the parser did not take an all-digit short root: the long
	// spelling, a non-digit value, a later option that overrides it, and a call
	// the parser or the padding guard refuses before it reaches the root.
	for _, args := range [][]string{
		{"--root", "3", "read", "f.txt"},
		{"--root=3", "read", "f.txt"},
		{"-C", "sub", "read", "f.txt"},
		{"-C", "3a", "read", "f.txt"},
		{"-C", "3", "--root", "sub", "read", "f.txt"},
		{"-C", "3", "-C", "sub", "read", "f.txt"},
		{"--bogus", "-C", "3", "read", "f.txt"},
		{"---C=3", "read", "f.txt"},
		{"--C=3 ", "read", "f.txt"},
		{"-C", "3", "seen", "--root", "sub"}, // the root flag is inherited: a later one replaces 3
		{"-C", "3", "seen", "-C", "sub"},
	} {
		_, errOut, _ := run(cwd, args...)
		if strings.Contains(errOut, "-C before the subcommand") {
			t.Errorf("mrw %q: a note where none is due:\n%s", args, errOut)
		}
	}
	if _, errOut, code := run(cwd, "-C", "sub", "read", "-C", "3", "f.txt:/needle/"); code != 0 || strings.Contains(errOut, "-C before the subcommand") {
		t.Errorf("-C 3 after read is context: exit %d, want 0 and no note:\n%s", code, errOut)
	}
	empty := t.TempDir()
	out, errOut, code := run(empty, "-C", "3", "read", "f.txt")
	if code != 1 || !strings.Contains(out+errOut, "the root 3 does not exist") || !strings.Contains(errOut, note) {
		t.Errorf("-C 3 with no directory 3: exit %d, want 1, the refusal and the note:\n%s%s", code, out, errOut)
	}
}
