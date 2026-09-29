package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-099. A path the caller names reaches the command: `check --full x.go`
// dropped x.go and ran the whole project, and `read help` printed read's help
// instead of the file named help, because urfave adds a `help` subcommand to
// every command.

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestACheckWithFullAndAPathIsRefused(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	writeFiles(t, root, map[string]string{
		".quality-harness.json": `{"check":"touch marker099"}` + "\n",
		"x.go":                  "package x\n",
	})
	marker := filepath.Join(root, "marker099")

	// runSplit keeps stdout apart, so "nothing on stdout" is asserted, and the
	// refusal is compared whole (Codex, review of #285).
	stdout, err := runSplit(t, "-C", root, "check", "--full", "x.go")
	if err == nil || exitCode(err) != exitUsage || err.Error() != "--full runs the whole project; it takes no PATH" || stdout != "" {
		t.Errorf("check --full x.go: err %v (exit %d), stdout %q; want exit %d with the refusal alone", err, exitCode(err), stdout, exitUsage)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("check --full x.go ran the check: the refusal must come before anything runs")
	}
	if out, code := runIn(t, root, "check", "--full"); code != 0 {
		t.Errorf("check --full alone: exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("check --full alone did not run the whole-project check")
	}
}

func TestAFileNamedHelpIsAPath(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"help":                  "the file named help\n",
		"h":                     "the file named h\n",
		".quality-harness.json": `{"check":"true","scoped_check":"echo FILES={files} > scoped099"}` + "\n",
	})
	for _, argv := range [][]string{{"read", "help"}, {"read", "--", "help"}} {
		if out, code := runIn(t, root, argv...); code != 0 || !strings.Contains(out, "the file named help") {
			t.Errorf("%q: exit %d, want the file served:\n%.400s", argv, code, out)
		}
	}
	if out, code := runIn(t, root, "read", "h"); code != 0 || !strings.Contains(out, "the file named h") {
		t.Errorf("read h: exit %d, want the file served:\n%.400s", code, out)
	}
	// write's positional is the plan file, resolved from the working directory,
	// so the run happens where the plan is; the plan's effect is the proof.
	writeFiles(t, root, map[string]string{"help": "@@ made099.txt 0 create\nmade\n"})
	t.Chdir(root)
	if out, code := runIn(t, root, "write", "--no-check", "help"); code != 0 {
		t.Errorf("write help: exit %d:\n%.400s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "made099.txt")); err != nil {
		t.Error("write help did not apply the plan in the file named help")
	}
	// check's positional is a PATH: a scoped check receives it as {files}.
	if out, code := runIn(t, root, "check", "help"); code != 0 {
		t.Errorf("check help: exit %d:\n%.400s", code, out)
	}
	if b, err := os.ReadFile(filepath.Join(root, "scoped099")); err != nil || strings.TrimSpace(string(b)) != "FILES=help" {
		t.Errorf("check help did not scope the check to help: %q, %v", b, err)
	}
	// The flag still prints each command's help, and the root keeps its command.
	for _, argv := range [][]string{{"read", "--help"}, {"read", "-h"}, {"write", "--help"}, {"check", "--help"}} {
		if out, code := runIn(t, root, argv...); code != 0 || !strings.Contains(out, "USAGE") {
			t.Errorf("%q: exit %d, the help flag no longer prints help:\n%.300s", argv, code, out)
		}
	}
	if out, code := runIn(t, root, "help"); code != 0 || !strings.Contains(out, "read") {
		t.Errorf("mrw help: exit %d, no longer lists the commands:\n%.300s", code, out)
	}
}
