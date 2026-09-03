//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A CONTRACT THROUGH A REAL WINDOWS SHELL (issue #48).
//
// scripts/contract.sh asserts mrw's promises across a POSIX shell and says why
// that matters: "it drives the real binary through a real shell, which is how a
// caller actually meets the tool — and exit statuses are half of what is being
// asserted". On Windows the caller meets it through PowerShell, and nothing
// asserted that boundary. The gap produced three real findings in one sitting:
// the MSYS argument mangling (#45), a BOM'd plan being refused (#46), and a
// drive letter's colon read as a range separator (found by CI on #57).
//
// This is deliberately SMALLER than a port of contract.sh. The promises worth
// asserting here are the ones where the SHELL participates — exit status,
// argument parsing, and the encoding of a file the shell itself wrote. Anything
// mrw can be asked about in-process belongs in an ordinary test.
//
// It never SKIPS for a missing shell. A skip here is indistinguishable from a
// pass, and this file exists because an invisible gap cost three defects.

func powershell(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	// Loud, not skipped: this test runs only on Windows, where a shell is not
	// optional. If neither is on PATH the environment is wrong and saying so is
	// the finding.
	t.Fatal("no pwsh.exe or powershell.exe on PATH — a Windows runner without a shell cannot assert the shell boundary")
	return ""
}

// runPS runs one PowerShell command and returns its combined output and the
// exit status the SHELL observed, which is the half a Go-level test cannot see.
func runPS(t *testing.T, dir, script string) (string, int) {
	t.Helper()
	cmd := exec.Command(powershell(t), "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("running powershell failed outside the process's own status: %v", err)
	}
	return string(out), code
}

// mrwExe builds the binary under test once per test, because a shell contract
// must drive the REAL executable rather than call rootCommand in-process — an
// in-process call observes a returned error, not the status a shell sees.
func mrwExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "mrw.exe")
	build := exec.Command("go", "build", "-o", exe, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building mrw for the shell contract failed: %v\n%s", err, out)
	}
	return exe
}

// The four exit statuses are the contract, and a shell is where they are met.
func TestExitStatusesThroughPowerShell(t *testing.T) {
	exe := mrwExe(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		why    string
		script string
		want   int
	}{
		{"a served read exits 0", `& '` + exe + `' -C '` + root + `' read a.txt`, 0},
		{"a missing file exits 1", `& '` + exe + `' -C '` + root + `' read nope.txt`, 1},
		{"a bad range is a usage error", `& '` + exe + `' -C '` + root + `' read 'a.txt:notanumber'`, 2},
	} {
		out, code := runPS(t, root, tc.script+"; exit $LASTEXITCODE")
		if code != tc.want {
			t.Errorf("%s: exit %d, want %d\n%s", tc.why, code, tc.want, out)
		}
	}
}

// A DRIVE-LETTER absolute path, typed the way a Windows user types it. This is
// the shape that broke: ParseSpec looked for the range separator with LastIndex
// over the whole string, so C:\dir\f.txt parsed as the path "C".
func TestADriveLetterPathWorksThroughTheShell(t *testing.T) {
	exe := mrwExe(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	abs := filepath.Join(root, "a.txt")
	if err := os.WriteFile(abs, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runPS(t, root, `& '`+exe+`' -C '`+root+`' read '`+abs+`'; exit $LASTEXITCODE`)
	if code != 0 {
		t.Fatalf("an absolute Windows path was refused: exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "one") {
		t.Errorf("the file was not served:\n%s", out)
	}

	// And with a range attached, which is the form that has a real colon after
	// the volume's.
	out, code = runPS(t, root, `& '`+exe+`' -C '`+root+`' read '`+abs+`:2-3'; exit $LASTEXITCODE`)
	if code != 0 {
		t.Fatalf("an absolute Windows path with a range was refused: exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "two") || strings.Contains(out, "one") {
		t.Errorf("the range was not honoured:\n%s", out)
	}
}

// A plan file WRITTEN BY THE SHELL, which is the case no Go test can construct:
// Windows PowerShell 5.1's `-Encoding utf8` emits a BOM and has no BOM-less
// option. Two fragments concatenated is how a plan gets built there, and that
// combination silently applied one hunk instead of two before it was fixed.
func TestAPlanTheShellWroteApplies(t *testing.T) {
	exe := mrwExe(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, code := runPS(t, root, `& '`+exe+`' -C '`+root+`' read a.txt; exit $LASTEXITCODE`); code != 0 {
		t.Fatalf("the priming read failed: exit %d", code)
	}

	// Set-Content -Encoding utf8 is the BOM-emitting path on 5.1 and the
	// BOM-less one on 7, so this asserts whichever the runner has — both must
	// apply, which is the point.
	script := `
Set-Content -Encoding utf8 -Path frag1.mrw -Value @('@@ a.txt 1 replace','FIRST')
Set-Content -Encoding utf8 -Path frag2.mrw -Value @('@@ a.txt 3 replace','THIRD')
Get-Content frag1.mrw, frag2.mrw | Set-Content -Encoding utf8 -Path plan.mrw
& '` + exe + `' -C '` + root + `' write plan.mrw
exit $LASTEXITCODE`
	out, code := runPS(t, root, script)
	if code != 0 {
		t.Fatalf("a plan the shell wrote was refused: exit %d\n%s", code, out)
	}
	if strings.Count(out, "ok  ") != 2 {
		t.Errorf("want two applied hunks, got:\n%s", out)
	}
	body, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if strings.Contains(got, "@@ a.txt") {
		t.Errorf("a plan header was written into the file — the swallowed-hunk corruption:\n%q", got)
	}
	if !strings.Contains(got, "FIRST") || !strings.Contains(got, "THIRD") {
		t.Errorf("both edits should have landed, got:\n%q", got)
	}
}
