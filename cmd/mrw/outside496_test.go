package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// printedCommand is what follows "pointed where you mean: " on a refusal, or "".
func printedCommand(out string) string {
	_, after, ok := strings.Cut(out, "pointed where you mean: ")
	if !ok {
		return ""
	}
	line, _, _ := strings.Cut(after, "\n")
	return line
}

// Issue #496. A subagent told to read a file outside the root was refused with
// "read it with --root pointed where you mean" and fell back to sed: the
// refusal named the flag, not the command. It now prints the command, for an
// absolute path and for ../ alike, quoted for the shell and with -- before the
// name, so a file called --help or " x" is read as a file. On unix the printed
// command is run through sh. The pairs: a path that is not outside, mrw's own
// state (no --root lifts that refusal, ADR-157), and a target nothing can serve
// (missing, or a directory) carry no such command.
func TestAnOutOfRootReadNamesTheCommandThatReadsIt(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, other := filepath.Join(parent, "root"), filepath.Join(parent, "other")
	for _, d := range []string{root, other} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// where is the root word of the printed command: the directory, unquoted on
	// unix, where a temp directory holds nothing a shell reads as syntax.
	where := other
	if runtime.GOOS == "windows" {
		where = "'" + other + "'" // a backslash is quoted
	}
	type target struct{ name, body, word string }
	targets := []target{
		{"hosts", "one hosts\n", "hosts:1"},
		{"my file.txt", "one spaced\n", "'my file.txt:1'"},
		{"--help", "one flag\n", "--help:1"},
		{"-", "one dash\n", "./-:1"}, // a bare - is standard input even after -- (Codex review of ff827462)
	}
	if runtime.GOOS != "windows" { // Windows keeps neither a leading space nor, here, a bare quote
		targets = append(targets,
			target{"it's.txt", "one quoted\n", `'it'\''s.txt:1'`},
			target{" lead", "one padded\n", "' lead:1'"})
	}
	var bin string
	if runtime.GOOS != "windows" {
		bin = filepath.Join(t.TempDir(), "mrw")
		if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
			t.Fatalf("go build: %v\n%s", err, out)
		}
	}
	for _, tg := range targets {
		if err := os.WriteFile(filepath.Join(other, tg.name), []byte(tg.body), 0o644); err != nil {
			t.Fatal(err)
		}
		want := "mrw --root " + where + " read -- " + tg.word
		for how, spec := range map[string]string{
			"an absolute path": filepath.Join(other, tg.name) + ":1",
			"a path with ../":  ".." + string(filepath.Separator) + "other" + string(filepath.Separator) + tg.name + ":1",
		} {
			if tg.name == "--help" && how == "a path with ../" {
				continue // "../other/--help" is not a flag; only the printed name is
			}
			out, code := runIn(t, root, "read", spec)
			got := printedCommand(out)
			if code != 1 || got != want {
				t.Errorf("%s, file %q: exit %d; printed %q, want %q:\n%s", how, tg.name, code, got, want, out)
				continue
			}
			if bin == "" {
				continue
			}
			served, err := exec.Command("sh", "-c", bin+strings.TrimPrefix(got, "mrw")).CombinedOutput()
			if err != nil || !strings.Contains(string(served), strings.TrimSpace(tg.body)) {
				t.Errorf("%s, file %q: the printed command did not serve the file: %v\n%s", how, tg.name, err, served)
			}
		}
	}

	// Nothing to advise: mrw's own state, a missing file, a directory.
	stateDir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(stateDir, "seen")
	if err := os.WriteFile(ledger, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, spec := range map[string]string{
		"mrw's own state, absolute": ledger + ":1",
		"a missing file, absolute":  filepath.Join(other, "nope", "x.txt") + ":3",
		"a missing file, ../":       "../other/missing:1",
		"a directory, ../":          "../other:1",
	} {
		out, code := runIn(t, root, "read", spec)
		if code != 1 || strings.Contains(out, "mrw --root") || !strings.Contains(out, "pointed where you mean") && !strings.Contains(out, "own state") {
			t.Errorf("%s: exit %d; want a refusal naming no command:\n%s", name, code, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("in\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "read", "in.txt"); code != 0 || strings.Contains(out, "mrw --root") {
		t.Errorf("a file inside the root: exit %d:\n%s", code, out)
	}
}
