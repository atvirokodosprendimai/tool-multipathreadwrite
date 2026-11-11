package main

import (
	"strings"
	"testing"
)

// ADR-196 T5, from the Windows re-probe of v1.86.1: past a \\.\ or \\?\ prefix
// a path names an object, not only a file, and a list of device names cannot
// cover them all. `\\.\GLOBALROOT\Device\ConDrv\Console` waited on the console
// as `write CON` did, and `\\.\pipe\..\GLOBALROOT\Device\Null` read the null
// device. Only a drive, a UNC share, a volume GUID or a pipe is accepted there,
// after `.` and `..` are resolved as Windows resolves them.
func TestAPlanOrListNamingAWindowsObjectIsRefused(t *testing.T) {
	for _, p := range []string{
		`\\.\GLOBALROOT\Device\Null`, `\\?\GLOBALROOT\Device\Null`, `\\.\GLOBALROOT\Device\ConDrv\Console`,
		`\\.\pipe\..\GLOBALROOT\Device\Null`, `//./pipe/x/../../GLOBALROOT/Device/Null`, `\\.\C:\..\GLOBALROOT\x`,
		`\\?\GLOBAL??\CON`, `\\.\PhysicalDrive0`, `\\.\`, `\\?\GLOBALROOT\Device\HarddiskVolume3\p.mrw`,
	} {
		if err := keepsOn("windows", "the plan", p); err == nil || !strings.Contains(err.Error(), "not a file on a drive or a share") {
			t.Errorf("%q on windows: %v, want the object refusal", p, err)
		}
	}
	// A `..` that climbs out of a pipe into a drive is judged as that drive's
	// path, and the refusal names the component that is wrong.
	if err := keepsOn("windows", "the plan", `\\.\pipe\x\..\..\C:\t\p.mrw:alt`); err == nil || !strings.Contains(err.Error(), `in "p.mrw:alt"`) {
		t.Errorf("a stream reached through a pipe's ..: %v, want it named", err)
	}
	for _, p := range []string{`\\.\Volume{0d2a}\p.mrw`, `\\?\Volume{0D2A}\d\p.mrw`, `\\.\C:\x\..\p.mrw`, `\\.\pipe\x\..\..\C:\t\p.mrw`} {
		if err := keepsOn("windows", "the plan", p); err != nil {
			t.Errorf("%q on windows refused: %v", p, err)
		}
	}
	// A pipe is no state file, so the state check, which opens the path and
	// would spend the pipe's one instance, is not run for one.
	for p, want := range map[string]bool{
		`\\.\pipe\plan`: true, `//./PIPE/plan`: true, `\\.\.\pipe\plan`: true, `\\srv\pipe\plan`: true, `\\?\UNC\srv\pipe\plan`: true,
		`\\.\pipe\..\NUL`: false, `\\.\pipe\..\GLOBALROOT\Device\Null`: false, `p.mrw`: false, `C:\pipe\plan`: false,
	} {
		if got := pipeOn("windows", p); got != want {
			t.Errorf("pipeOn(windows, %q) = %v, want %v", p, got, want)
		}
	}
	if pipeOn("linux", `\\.\pipe\plan`) {
		t.Error("a pipe spelling off Windows is a name")
	}
	if err := keepsOn("linux", "the plan", `\\.\GLOBALROOT\Device\Null`); err != nil {
		t.Errorf("off Windows it is a name, refused: %v", err)
	}
}
