package main

import (
	"strings"
	"testing"
)

// ADR-196 T6, from the Windows field test of v1.86.3: past a device prefix, a
// drive, a volume, a UNC share or the pipe namespace with nothing after it names
// that object itself. `\\.\C:` and `\\.\Volume{…}` reached os.Open on the raw
// volume, stopped only by "Access is denied" in an unelevated shell, and
// `\\.\pipe` opened the pipe namespace. A file is named past the volume.
func TestAPlanOrListNamingAVolumeAndNoFileIsRefused(t *testing.T) {
	for _, p := range []string{
		`\\.\C:`, `\\?\C:`, `\\?\C:\`, `\??\C:`, `//./c:/`, `\\.\Volume{0d2a}`, `\\?\Volume{0d2a}\`,
		`\\.\pipe`, `\\.\pipe\`, `\\.\UNC\srv`, `\\?\UNC\srv\share`, `\\.\UNC\localhost\pipe\..\..`,
		`\\srv`, `\\srv\share`, `\\srv\pipe`, `\\srv\pipe\`,
	} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "and no file in it") {
			t.Errorf("%q on windows: %v, want the no-file refusal", p, err)
		}
		if pipeOn("windows", p) {
			t.Errorf("pipeOn(windows, %q) = true, want false", p)
		}
	}
	for _, p := range []string{`\\.\C:\p.mrw`, `\\?\Volume{0d2a}\p.mrw`, `\\.\pipe\plan`, `\\.\UNC\srv\share\p.mrw`, `\\srv\share\p.mrw`, `\\srv\pipe\plan`, `C:`, `C:\`} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	if err := keepsOn("linux", "the plan", `\\.\C:`); err != nil {
		t.Errorf("off Windows it is a name, refused: %v", err)
	}
}
