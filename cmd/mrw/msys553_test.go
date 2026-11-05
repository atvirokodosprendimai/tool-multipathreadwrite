package main

import (
	"os"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// Issue #553. An exported MSYS_NO_PATHCONV or MSYS2_ARG_CONV_EXCL changes
// every later command in the shell, so a native tool after mrw gets POSIX
// paths MSYS no longer converts for it. Every place that advises the setting
// puts it on the one mrw command instead.
func TestTheMSYSAdvicePutsTheSettingOnTheOneCommand(t *testing.T) {
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	out, _ := runIn(t, t.TempDir(), "read", `cmd\mrw\main.go;C:\Users\me\AppData\Local\Programs\Git\^func main\`)
	for name, text := range map[string]string{
		"AGENTS.md":        read("../../AGENTS.md"),
		"README.md":        read("../../README.md"),
		"CONTRIBUTING.md":  read("../../CONTRIBUTING.md"),
		"mrw instructions": guide.CLI(),
		"the read hint":    out,
	} {
		for _, want := range []string{"MSYS_NO_PATHCONV=1 mrw", "MSYS2_ARG_CONV_EXCL='*' mrw"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not put the setting on the command: no %q", name, want)
			}
		}
		for _, global := range []string{"Export `MSYS", "Export MSYS", "Set MSYS", "export MSYS"} {
			if strings.Contains(text, global) {
				t.Errorf("%s still advises a shell-wide setting: %q", name, global)
			}
		}
	}
}
