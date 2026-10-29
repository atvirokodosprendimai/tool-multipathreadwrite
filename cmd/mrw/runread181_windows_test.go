//go:build windows

package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"unicode/utf16"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-181 Decision 3: each read line a Windows `mrw run` receipt prints serves
// the log's last 40 lines when it is run in the shell its label names. The
// binary and the log sit under a path holding a space and an apostrophe, and the
// log's last line is non-ASCII. Windows PowerShell 5.1, cmd and Git for Windows'
// sh are required, never skipped; pwsh 7 runs when it is on PATH. Each version
// is logged, since the job log is this test's evidence.
func TestEveryPrintedReadLineServesTheLogInItsShell(t *testing.T) {
	sh, pathDir, ok := check.Shell()
	if !ok {
		t.Fatal("no Git for Windows sh: the Git Bash line cannot be run on this runner")
	}
	weird := filepath.Join(t.TempDir(), "it's a dir")
	if err := os.MkdirAll(weird, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(weird, "mrw.exe")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("building mrw: %v\n%s", err, out)
	}
	env := append(os.Environ(), "XDG_STATE_HOME="+weird)
	if pathDir != "" {
		env = append(env, "PATH="+pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	run := exec.Command(exe, "-C", t.TempDir(), "run", `i=1; while [ $i -le 44 ]; do echo line$i; i=$((i+1)); done; printf '\303\244 \342\202\254 \303\274\n'`)
	run.Env = env
	receipt, err := run.Output()
	if err != nil {
		t.Fatalf("mrw run: %v\n%s", err, receipt)
	}
	lines := map[string]string{}
	for _, label := range []string{"PowerShell", "cmd", "Git Bash"} {
		if lines[label] = receiptField(string(receipt), "read ("+label+")"); lines[label] == "" {
			t.Fatalf("the receipt has no read (%s) line:\n%s", label, receipt)
		}
	}

	// The served rows must be lines 6-45 of the log, every one, in order, with
	// the non-ASCII last line intact: the only thing a shell may change is the
	// line ending.
	var want []string
	for i := 6; i <= 44; i++ {
		want = append(want, strconv.Itoa(i)+"| line"+strconv.Itoa(i))
	}
	want = append(want, "45| ä € ü")
	row := regexp.MustCompile(`^ *([0-9]+\| .*)$`)
	serves := func(shell string, c *exec.Cmd) {
		t.Helper()
		c.Env = env
		out, err := c.CombinedOutput()
		var got []string
		for _, l := range strings.Split(string(out), "\n") {
			if m := row.FindStringSubmatch(strings.TrimSuffix(l, "\r")); m != nil {
				got = append(got, m[1])
			}
		}
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: the printed line did not serve lines 6-45 byte for byte (err %v):\n%s", shell, err, out)
		}
	}
	// PowerShell decodes a native command's output with [Console]::OutputEncoding
	// before writing it on; set to UTF-8, what reaches us is what mrw printed. The
	// script is passed encoded, so no quoting stands between it and the parser.
	ps := func(line string) []string {
		script := "[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)\n" + line
		return []string{"-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(utf16le(script))}
	}
	shells := []string{"powershell.exe"}
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		shells = append(shells, "pwsh.exe")
	}
	for _, p := range shells {
		v, _ := exec.Command(p, "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()").Output()
		t.Logf("%s %s", p, strings.TrimSpace(string(v)))
		serves(p, exec.Command(p, ps(lines["PowerShell"])...))
	}

	v, _ := exec.Command("cmd.exe", "/d", "/c", "ver").Output()
	t.Logf("cmd.exe %s", strings.TrimSpace(string(v)))
	c := exec.Command("cmd.exe")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /s /c "` + lines["cmd"] + `"`}
	serves("cmd.exe", c)

	v, _ = exec.Command(sh, "--version").Output()
	t.Logf("%s %s", sh, strings.SplitN(string(v), "\n", 2)[0])
	serves("Git Bash", exec.Command(sh, "-c", lines["Git Bash"]))

	// Why a path holding ! gets no cmd line: under delayed expansion cmd reads
	// !b! inside double quotes as a variable, and the line names another file.
	// The pair: the same command with expansion off serves the file.
	bang := filepath.Join(weird, "a!b!.log")
	if err := os.WriteFile(bang, []byte("only\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inCmd := func(v string) (string, error) {
		c := exec.Command("cmd.exe")
		c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /v:` + v + ` /s /c ""` + exe + `" read -- -:1-$ < "` + bang + `""`}
		c.Env = env
		out, err := c.CombinedOutput()
		if _, exited := err.(*exec.ExitError); err != nil && !exited {
			t.Fatalf("cmd.exe /v:%s did not start: %v", v, err)
		}
		return string(out), err
	}
	if out, err := inCmd("off"); err != nil || !strings.Contains(out, "1| only") {
		t.Errorf("cmd /v:off did not serve a path holding ! (err %v):\n%s", err, out)
	}
	if out, err := inCmd("on"); err == nil || strings.Contains(out, "1| only") {
		t.Errorf("cmd /v:on served a path holding !; ADR-181 leaves its line out for a reason this run does not show (err %v):\n%s", err, out)
	}
	if got := runReadLines("windows", exe, bang, 1); len(got) != 2 {
		t.Errorf("a ! in the path: %q, want no cmd line", got)
	}
}

// utf16le is s as PowerShell's -EncodedCommand takes it.
func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = append(b, byte(u), byte(u>>8))
	}
	return b
}
