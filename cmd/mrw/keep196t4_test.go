package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-196 T4, from the Windows field test of v1.86.0: a plan file or a
// --files-from list named as a device (`write CON` waited on the console) or
// as an NTFS stream (`write p.mrw:alt` ran a plan that `type p.mrw` does not
// show) was opened, while a path inside the root is refused for both (ADR-071,
// ADR-081). The rule is judged for Windows on every platform.
func TestAPlanOrListNamingADeviceOrAStreamIsRefused(t *testing.T) {
	for _, p := range []string{"CON", "con", `d\NUL`, "NUL.txt", "COM1", "lpt9.mrw", "CONIN$", "aux .mrw", `C:\CON`, `C:nul`, `\\.\CON`, `\\.\C:\d\prn`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "device") {
			t.Errorf("%q on windows: %v, want the device refusal", p, err)
		}
	}
	for _, p := range []string{"p.mrw:alt", "p.mrw::$DATA", "p.mrw:alt:$DATA", `sub\x:alt`, `C:\t\list::$DATA`, `\\?\C:\t\p.mrw:alt`, `\\.\C:\t\p.mrw:alt`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "NTFS stream") {
			t.Errorf("%q on windows: %v, want the stream refusal", p, err)
		}
	}
	for _, p := range []string{"p.mrw", `C:\t\p.mrw`, "C:p.mrw", `\\?\C:\t\CON`, `\\?\UNC\srv\share\p.mrw`, `\\.\C:\t\p.mrw`, "console.mrw", "COM10", "LPT0", "nul_x", `\\srv\share\p.mrw`} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	for _, p := range []string{"CON", "p.mrw:alt"} {
		if err := keepsOn("linux", "the plan", p); err != nil {
			t.Errorf("%q off Windows is a name, refused: %v", p, err)
		}
	}
	// The Codex reviews of #630: only the volume is skipped — a slash-form or
	// mixed device prefix, a UNC server and share — and a named pipe, local or
	// remote, is no file, unless a `..` climbs out of the pipe namespace.
	for _, p := range []string{`\\.\pipe\..\NUL`, `//./pipe/../CON`, `\\.\.\pipe\..\NUL`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "device") {
			t.Errorf("%q on windows: %v, want the device refusal", p, err)
		}
	}
	if err := keepsOn("windows", "the plan", `\\.\pipe\..\C:\t\p.mrw:alt`); err == nil || !strings.Contains(err.Error(), "NTFS stream") {
		t.Errorf(`\\.\pipe\..\C:\t\p.mrw:alt on windows: %v, want the stream refusal`, err)
	}
	for _, p := range []string{`\\srv\pipe\CON`, `\\srv\pipe\plan:input`, `\\?\UNC\srv\pipe\x:y`, `\\.\pipe\a\b:c`, `\\.\.\pipe\CON`, `//./././pipe/x:y`} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	for _, p := range []string{"//./CON", `\\srv\share\CON`, `\\.\UNC\srv\share\nul`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "device") {
			t.Errorf("%q on windows: %v, want the device refusal", p, err)
		}
	}
	for _, p := range []string{"//./C:/t/p.mrw:alt", `\\srv\share\p.mrw:alt`, `\\?\UNC\srv\share\p:alt`} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "NTFS stream") {
			t.Errorf("%q on windows: %v, want the stream refusal", p, err)
		}
	}
	for _, p := range []string{"//./C:/t/p.mrw", "//?/C:/t/p.mrw", `\/.\C:\t\p.mrw`, `\\con.example.test\share\p.mrw`, `\\srv\nul\p.mrw`, `\\?\UNC\con\aux\p.mrw`, `\\.\pipe\plan:input`, `\\.\pipe\CON`, `//./PIPE/x`, `\\?\Volume{0d2a}\p.mrw`} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	// On Windows, through the commands. NUL, not CON: a regression on CON
	// would wait on the console until the test binary's deadline.
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"p.mrw": "@@ c.txt 0 create\nmade\n", "list": "a.txt\n", "a.txt": "listed\n"})
	if err := os.WriteFile(filepath.Join(root, "p.mrw:alt"), []byte("@@ s.txt 0 create\nhidden\n"), 0o644); err != nil {
		t.Fatalf("writing the stream: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "list:alt"), []byte("a.txt\n"), 0o644); err != nil {
		t.Fatalf("writing the stream: %v", err)
	}
	t.Chdir(root)
	for _, argv := range [][]string{{"write", "NUL"}, {"read", "--files-from", "NUL"}} {
		if out, code := runIn(t, root, argv...); code != 2 || !strings.Contains(out, "device") {
			t.Errorf("%q: exit %d, want the device refusal:\n%s", argv, code, out)
		}
	}
	if out, code := runIn(t, root, "write", "p.mrw:alt"); code != 2 || !strings.Contains(out, "NTFS stream") {
		t.Errorf("write p.mrw:alt: exit %d, want the stream refusal:\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "s.txt")); err == nil {
		t.Fatal("write p.mrw:alt applied the plan in the stream")
	}
	if out, code := runIn(t, root, "read", "--files-from", "list:alt"); code != 2 || strings.Contains(out, "| listed") {
		t.Errorf("read --files-from list:alt: exit %d, read the stream:\n%s", code, out)
	}
}
