package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-196, from the Codex review of #628: a plan file and a --files-from list
// are shell arguments opened with os.Open, outside rooted.Resolve, so on
// Windows a spelling ending in a dot or a space opened the file without it.
// The rule is judged for Windows on every platform; elsewhere it is no rule.
func TestAWindowsAliasOfAPlanOrListIsRefused(t *testing.T) {
	for _, p := range []string{"p.mrw ", "p.mrw.", `d.\p.mrw`, "d /p.mrw", `C:\d.\p.mrw`, `C:p.mrw `, `\\.\C:\d.\p.mrw`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "Windows does not keep") {
			t.Errorf("%q on windows: %v, want the refusal", p, err)
		}
	}
	for _, p := range []string{"p.mrw", `d\p.mrw`, "./p.mrw", `..\p.mrw`, " p.mrw", `C:.\p.mrw`, `C:..\list`, `c:p.mrw`, `C:\d\p.mrw`, `\\server\share\p.mrw`, `\\?\C:\d.\p.mrw`, `\\?\C:\p.mrw `} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	if err := keepsOn("linux", "the plan", "p.mrw "); err != nil {
		t.Errorf("a trailing space off Windows is a name, refused: %v", err)
	}
	if runtime.GOOS != "windows" {
		return
	}
	// On Windows, through the commands: the alias neither applies the plan nor
	// reads the list it would open; the exact spelling does both.
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"p.mrw": "@@ c.txt 0 create\nmade\n", "list": "a.txt\n", "a.txt": "listed\n"})
	t.Chdir(root)
	if out, code := runIn(t, root, "write", "p.mrw "); code == 0 {
		t.Errorf("write 'p.mrw ' exited 0:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "c.txt")); err == nil {
		t.Fatal("write 'p.mrw ' applied p.mrw")
	}
	for _, argv := range [][]string{{"read", "--files-from", "list "}, {"read", "--files-from=list "}} {
		if out, code := runIn(t, root, argv...); code == 0 || strings.Contains(out, "| listed") {
			t.Errorf("%q: exit %d, read the list it does not name:\n%s", argv, code, out)
		}
	}
	if out, code := runIn(t, root, "write", "p.mrw"); code != 0 {
		t.Fatalf("write p.mrw: exit %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "--files-from", "list"); code != 0 || !strings.Contains(out, "| listed") {
		t.Errorf("read --files-from list: exit %d:\n%s", code, out)
	}
}
