package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// runBin runs the built binary and returns its stdout, stderr and exit code.
func runBin(t *testing.T, bin string, env []string, args ...string) (string, string, int) {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Env = append(os.Environ(), env...)
	var so, se bytes.Buffer
	c.Stdout, c.Stderr = &so, &se
	err := c.Run()
	code := 0
	if err != nil {
		if c.ProcessState == nil {
			t.Fatal(err)
		}
		code = c.ProcessState.ExitCode()
	}
	return so.String(), se.String(), code
}

func buildMrw(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mrw")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// receiptField is the text after "key: " on the receipt line that starts with it.
func receiptField(out, key string) string {
	for _, l := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(l, key+": "); ok {
			return v
		}
	}
	return ""
}

// decodeFirst decodes the first JSON document in out: runIn appends the error
// text of a command that exited non-zero to what it printed.
func decodeFirst(out string, v any) error {
	return json.NewDecoder(strings.NewReader(out)).Decode(v)
}

// ADR-177 Decisions 3 and 5. A 500-line command prints a four-line receipt and
// none of its output; the log holds all of it; the read: line, run as printed
// from a state home whose name holds a space and a quote, serves the last 40.
func TestRunKeepsTheOutputAndPrintsOneReceipt(t *testing.T) {
	needShell(t)
	if runtime.GOOS == "windows" {
		t.Skip("the read: line is PowerShell's there; its text is asserted by TestTheReadLineIsQuotedForThePlatform")
	}
	bin := buildMrw(t)
	state := filepath.Join(t.TempDir(), "sp ace's")
	root := t.TempDir()
	env := []string{"XDG_STATE_HOME=" + state, "PATH=" + filepath.Dir(bin) + string(os.PathListSeparator) + os.Getenv("PATH")}
	cmdline := `i=1; while [ $i -le 500 ]; do echo "line $i"; i=$((i+1)); done; exit 1`
	out, errOut, code := runBin(t, bin, env, "-C", root, "run", cmdline)
	if code != 1 {
		t.Fatalf("exit %d, want the command's 1:\nstdout %s\nstderr %s", code, out, errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("the receipt is %d lines, want 4:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "exit 1 · 500 lines · 4,392 bytes · ") || !strings.HasSuffix(lines[0], " s") {
		t.Errorf("first line %q; want `exit 1 · 500 lines · 4,392 bytes · … s`", lines[0])
	}
	if lines[1] != "last: line 500" {
		t.Errorf("second line %q, want the last line", lines[1])
	}
	if strings.Contains(out, "line 250") {
		t.Errorf("the output reached stdout:\n%s", out)
	}
	if errOut != "" {
		t.Errorf("stderr %q; the command's own exit is not an mrw failure", errOut)
	}
	log := receiptField(out, "log")
	if !strings.HasPrefix(log, filepath.Join(state, "mrw-runs", "run-")) || !strings.HasSuffix(log, ".log") || strings.HasSuffix(log, ".running.log") {
		t.Fatalf("log: %q; want a completed log beside the state directory", log)
	}
	b, err := os.ReadFile(log)
	if err != nil || strings.Count(string(b), "\n") != 500 || !strings.HasPrefix(string(b), "line 1\n") {
		t.Fatalf("the log holds %d bytes, %v; want all 500 lines", len(b), err)
	}
	if fi, _ := os.Stat(log); fi.Mode().Perm() != 0o600 {
		t.Errorf("log mode %v, want 0600", fi.Mode().Perm())
	}
	if fi, _ := os.Stat(filepath.Dir(log)); fi.Mode().Perm() != 0o700 {
		t.Errorf("directory mode %v, want 0700", fi.Mode().Perm())
	}
	read := receiptField(out, "read")
	resolved, _ := filepath.EvalSymlinks(bin)
	if want := posixQuote(resolved) + " read -- '-:461-$' < '"; !strings.HasPrefix(read, want) {
		t.Fatalf("read: %q", read)
	}
	c := exec.Command("sh", "-c", read)
	c.Env = append(os.Environ(), env...)
	c.Dir = root
	got, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("the printed read line failed: %v\n%s", err, got)
	}
	if n := strings.Count(string(got), "| line "); n != 40 || !strings.Contains(string(got), "461| line 461") || !strings.Contains(string(got), "500| line 500") {
		t.Errorf("the printed read line served %d lines, want the last 40:\n%s", n, got)
	}
	if entries, _ := os.ReadDir(filepath.Dir(log)); len(entries) != 1 {
		t.Errorf("the log directory holds %d entries, want the one completed log", len(entries))
	}
}

// The pair for the receipt: 40 lines or fewer are read whole.
func TestTheReadLineIsQuotedForThePlatform(t *testing.T) {
	exe := "/opt/my mrw/it's/mrw"
	if got := runReadLine("linux", exe, "/a b/it's/run.log", 3); got != `'/opt/my mrw/it'\''s/mrw' read -- '-:1-$' < '/a b/it'\''s/run.log'` {
		t.Errorf("unix, 3 lines: %s", got)
	}
	if got := runReadLine("linux", exe, "/x/r.log", 40); !strings.Contains(got, ` read -- '-:1-$' < `) {
		t.Errorf("unix, 40 lines: %s", got)
	}
	if got := runReadLine("linux", exe, "/x/r.log", 41); !strings.Contains(got, ` read -- '-:2-$' < `) {
		t.Errorf("unix, 41 lines: %s", got)
	}
	// Windows (Decision 5, measured on Windows 11 in pwsh 7.6 and Windows
	// PowerShell 5.1): the typed form, a PowerShell double-quoted string, with
	// ` and $ escaped and ' left alone; it is byte-exact where a pipe is not.
	bt := "`"
	want := `cmd /c "` + bt + `"C:\Program Files\mrw\mrw.exe` + bt + `" read -- -:61-` + bt + `$ < ` + bt + `"C:\u's\` + bt + `$x\b` + bt + bt + `c.log` + bt + `""`
	if got := runReadLine("windows", `C:\Program Files\mrw\mrw.exe`, "C:\\u's\\$x\\b`c.log", 100); got != want {
		t.Errorf("windows:\n got %s\nwant %s", got, want)
	}
	// A Windows path cannot hold a double quote, so a line that would need one
	// is left out rather than printed wrong.
	if got := runReadLine("windows", `C:\mrw.exe`, `C:\a"b\r.log`, 5); got != "" {
		t.Errorf("windows, a quote in the path: %q, want no line", got)
	}
	// cmd.exe expands %VAR% inside double quotes, so a path holding % would
	// read another file: no line, and the pair without it still prints one.
	if got := runReadLine("windows", `C:\mrw.exe`, `C:\logs\%USERNAME%\r.log`, 5); got != "" {
		t.Errorf("windows, a %% in the log path: %q, want no line", got)
	}
	if got := runReadLine("windows", `C:\100%\mrw.exe`, `C:\logs\r.log`, 5); got != "" {
		t.Errorf("windows, a %% in the binary path: %q, want no line", got)
	}
	if got := runReadLine("windows", `C:\mrw.exe`, `C:\logs\r.log`, 5); got == "" {
		t.Error("windows, no % or quote: want a line")
	}
}

// ADR-181 Decisions 1-2: on Windows the receipt names a read line for each
// shell, labelled, the PowerShell one ADR-177's byte for byte; unix keeps its
// one read: line. A path holding " or % prints none; one holding ! drops only
// the cmd line, since cmd expands !NAME! under delayed expansion.
func TestTheWindowsRunReceiptLabelsAReadLineForEachShell(t *testing.T) {
	bt := "`"
	exe, log := `C:\Program Files\mrw\mrw.exe`, "C:\\u's\\$x\\b`c.log"
	want := []string{
		`read (PowerShell): cmd /c "` + bt + `"C:\Program Files\mrw\mrw.exe` + bt + `" read -- -:61-` + bt + `$ < ` + bt + `"C:\u's\` + bt + `$x\b` + bt + bt + `c.log` + bt + `""`,
		`read (cmd): "C:\Program Files\mrw\mrw.exe" read -- -:61-$ < "C:\u's\$x\b` + bt + `c.log"`,
		`read (Git Bash): 'C:\Program Files\mrw\mrw.exe' read -- '-:61-$' < 'C:\u'\''s\$x\b` + bt + `c.log'`,
	}
	if got := runReadLines("windows", exe, log, 100); !slices.Equal(got, want) {
		t.Errorf("windows:\n got %q\nwant %q", got, want)
	}
	unix := []string{`read: '/opt/my mrw/mrw' read -- '-:1-$' < '/a b/run.log'`}
	if got := runReadLines("linux", "/opt/my mrw/mrw", "/a b/run.log", 3); !slices.Equal(got, unix) {
		t.Errorf("unix:\n got %q\nwant %q", got, unix)
	}
	for _, p := range []string{`C:\a"b\r.log`, `C:\logs\%USERNAME%\r.log`} {
		if got := runReadLines("windows", exe, p, 5); len(got) != 0 {
			t.Errorf("windows, %s: %q, want no line", p, got)
		}
	}
	got := runReadLines("windows", exe, `C:\logs\wow!\r.log`, 5)
	if len(got) != 2 || !strings.HasPrefix(got[0], "read (PowerShell): ") || !strings.HasPrefix(got[1], "read (Git Bash): ") {
		t.Errorf("windows, a ! in the path: %q, want the PowerShell and Git Bash lines only", got)
	}
}

// F4: the read: line names the binary that wrote the log, not whatever mrw is
// first on PATH.
func TestTheReadLineNamesTheRunningBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if got := runningBinary(); got != exe || !filepath.IsAbs(got) {
		t.Fatalf("runningBinary %q, want the resolved absolute %q", got, exe)
	}
	if got := runReadLine("linux", runningBinary(), "/x/r.log", 3); !strings.HasPrefix(got, posixQuote(exe)+" read -- ") {
		t.Errorf("the line does not start with the running binary: %s", got)
	}
}

func TestRunKeepsTheLogOfAShortOrEmptySuccess(t *testing.T) {
	needShell(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	root := t.TempDir()
	out, code := runIn(t, root, "run", "echo hi")
	if code != 0 || !strings.HasPrefix(out, "exit 0 · 1 lines · 3 bytes · ") {
		t.Fatalf("echo: exit %d:\n%s", code, out)
	}
	if b, err := os.ReadFile(receiptField(out, "log")); err != nil || string(b) != "hi\n" {
		t.Errorf("a one-line passing run lost its log: %q, %v", b, err)
	}
	if receiptField(out, "last") != "hi" {
		t.Errorf("last: %q", receiptField(out, "last"))
	}
	out, code = runIn(t, root, "run", "true")
	if code != 0 || !strings.HasPrefix(out, "exit 0 · 0 lines · 0 bytes · ") {
		t.Fatalf("true: exit %d:\n%s", code, out)
	}
	if fi, err := os.Stat(receiptField(out, "log")); err != nil || fi.Size() != 0 {
		t.Errorf("an empty passing run lost its log: %v", err)
	}
	if receiptField(out, "last") != "" || strings.Contains(out, "last:") || strings.Contains(out, "read:") || strings.Contains(out, "read (") {
		t.Errorf("an empty output has no last: or read line:\n%s", out)
	}
	entries, _ := os.ReadDir(filepath.Join(state, "mrw-runs"))
	if len(entries) != 2 {
		t.Errorf("the directory holds %d logs, want the two runs'", len(entries))
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".running.log") {
			t.Errorf("%s was not renamed when its command exited", e.Name())
		}
	}
	// The last line is cut at 200 characters, and blank lines do not count.
	out, _ = runIn(t, root, "run", `printf 'a%.0s' $(seq 1 300); echo; echo; echo "  "`)
	if want := strings.Repeat("a", 200) + "…"; receiptField(out, "last") != want {
		t.Errorf("last: %q; want 200 characters and …", receiptField(out, "last"))
	}
}

func TestRunExitsWithTheCommandsOwnCode(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, c := range []struct {
		cmd  string
		want int
	}{{"exit 3", 3}, {"exit 7", 7}, {"exit 0", 0}, {"exit 2", 2}} {
		out, code := runIn(t, root, "run", c.cmd)
		if code != c.want || !strings.HasPrefix(out, "exit "+string(rune('0'+c.want))+" · ") {
			t.Errorf("%s: exit %d, receipt %q; want %d and its own first line", c.cmd, code, out, c.want)
		}
	}
	if runtime.GOOS == "windows" {
		return
	}
	out, code := runIn(t, root, "run", "kill -TERM $$")
	if code != 143 || !strings.HasPrefix(out, "exit 143 · ") {
		t.Errorf("killed by TERM: exit %d, receipt %q; want 143", code, out)
	}
	out, code = runIn(t, root, "run", "--json", "kill -KILL $$")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil || code != 137 || doc["signal"] != float64(9) || doc["exit_code"] != float64(137) {
		t.Errorf("--json killed by KILL: exit %d, %v, %s", code, err, out)
	}
}

func TestRunThatCannotStartSaysSoAndExitsTwo(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if runtime.GOOS == "windows" {
		t.Skip("Git for Windows' sh may still be found")
	}
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	out, code := runIn(t, root, "run", "echo hi")
	if code != 2 || strings.HasPrefix(out, "exit ") || strings.Contains(out, "\nexit ") || !strings.Contains(out, "could not start") {
		t.Errorf("no sh: exit %d, %q; want 2, a could-not-start line and no exit receipt", code, out)
	}
	out, code = runIn(t, root, "run", "--json", "echo hi")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if code != 2 || doc["ran"] != false || doc["exit_code"] != float64(-1) || !strings.Contains(doc["skipped"].(string), "could not start") {
		t.Errorf("--json: exit %d %v; want ran false, exit_code -1, skipped naming why", code, doc)
	}
}

func TestRunTakesExactlyOneCommandString(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, args := range [][]string{{"run", "echo", "hi"}, {"run"}} {
		out, code := runIn(t, root, args...)
		if code != 2 || !strings.Contains(out, "mrw run 'CMD'") || strings.Contains(out, "exit ") {
			t.Errorf("%q: exit %d:\n%s; want 2 naming mrw run 'CMD'", args, code, out)
		}
	}
	out, code := runIn(t, root, "run", "--json", "echo", "hi")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil || code != 2 || len(doc) != 1 || !strings.Contains(doc["error"].(string), "mrw run 'CMD'") {
		t.Errorf("--json: exit %d, %v, %s; want one document holding only error", code, err, out)
	}
	needShell(t)
	if _, code := runIn(t, root, "run", "echo hi"); code != 0 {
		t.Errorf("the pair: one string exited %d", code)
	}
}

func TestRunIsRefusedAtTheStepDepthLimit(t *testing.T) {
	needShell(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	root := t.TempDir()
	t.Setenv("MRW_STEP_DEPTH", "8")
	out, code := runIn(t, root, "run", "echo hi > marker")
	if code != 2 || !strings.Contains(out, "MRW_STEP_DEPTH") || strings.Contains(out, "exit 0") {
		t.Errorf("at 8: exit %d:\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); err == nil {
		t.Error("at 8 the command ran")
	}
	if _, err := os.Stat(filepath.Join(state, "mrw-runs")); err == nil {
		t.Error("at 8 a log directory was made")
	}
	out, code = runIn(t, root, "run", "--json", "echo hi")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil || code != 2 || len(doc) != 1 || !strings.Contains(doc["error"].(string), "MRW_STEP_DEPTH") {
		t.Errorf("--json at 8: exit %d, %v, %s", code, err, out)
	}
	t.Setenv("MRW_STEP_DEPTH", "7")
	out, code = runIn(t, root, "run", `echo "depth $MRW_STEP_DEPTH"`)
	if code != 0 || receiptField(out, "last") != "depth 8" {
		t.Errorf("at 7 the command runs one level deeper: exit %d:\n%s", code, out)
	}
}

// An interrupt sent to mrw while the command runs stops the command's group,
// and the run says so and exits 130.
func TestRunStoppedByAnInterruptExits130(t *testing.T) {
	needShell(t)
	if runtime.GOOS == "windows" {
		t.Skip("an interrupt cannot be sent to a process on Windows")
	}
	bin := buildMrw(t)
	root := t.TempDir()
	pidFile := filepath.Join(root, "pid")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, bin, "-C", root, "run", "sleep 30 & echo $! > pid; wait")
	c.Env = append(os.Environ(), "XDG_STATE_HOME="+t.TempDir())
	var so bytes.Buffer
	c.Stdout = &so
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	var pid int
	for i := 0; i < 100 && pid == 0; i++ {
		if b, err := os.ReadFile(pidFile); err == nil && strings.HasSuffix(string(b), "\n") {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("the command never wrote its pid")
	}
	if err := c.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	_ = c.Wait()
	if code := c.ProcessState.ExitCode(); code != 130 || !strings.HasPrefix(so.String(), "exit 130 · interrupted · ") {
		t.Errorf("exit %d, receipt %q; want 130 and `exit 130 · interrupted · …`", code, so.String())
	}
	gone := false
	for i := 0; i < 60 && !gone; i++ {
		p, _ := os.FindProcess(pid)
		gone = p.Signal(syscall.Signal(0)) != nil
		time.Sleep(50 * time.Millisecond)
	}
	if !gone {
		t.Errorf("the command's process %d outlived the interrupt", pid)
	}
}

func TestRunHelpNamesTheShellTrust(t *testing.T) {
	out, code := runIn(t, t.TempDir(), "run", "--help")
	if code != 0 || !strings.Contains(out, guide.ThenShCaveat()) {
		t.Errorf("run --help: exit %d, missing the caveat:\n%s", code, out)
	}
	if !strings.Contains(guide.ThenShCaveat(), "mrw run") || !strings.Contains(guide.ThenShCaveat(), "--then-sh") {
		t.Errorf("the caveat names %q; want both mrw run and --then-sh", guide.ThenShCaveat())
	}
}

// runInCtx is runIn under ctx, for a run whose context is done before it starts.
func runInCtx(ctx context.Context, t *testing.T, root string, argv ...string) (string, int) {
	t.Helper()
	full := append([]string{"mrw", "-C", root}, argv...)
	var sink bytes.Buffer
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(ctx, full)
	})
	code := 0
	if err != nil {
		code = exitCode(err)
		out += err.Error()
	}
	return out + sink.String(), code
}

// An interrupt or a deadline that lands before the command starts is mrw's
// 130 or 124, not a configuration failure's 2 (ADR-177 Decision 6), and no
// command ran and no log is left.
func TestRunStoppedBeforeItStartsExits130Or124(t *testing.T) {
	needShell(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	root := t.TempDir()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancel2 := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel2()
	for _, c := range []struct {
		ctx     context.Context
		want    int
		skipped string
	}{{cancelled, 130, check.Interrupted}, {expired, 124, check.TimedOutBeforeStart}} {
		out, code := runInCtx(c.ctx, t, root, "run", "--json", "echo hi > marker")
		var doc map[string]any
		if err := decodeFirst(out, &doc); err != nil || code != c.want || doc["ran"] != false || doc["skipped"] != c.skipped || doc["exit_code"] != float64(-1) {
			t.Errorf("%s: exit %d, %v, %s; want %d, ran false, exit_code -1", c.skipped, code, err, out, c.want)
		}
		out, code = runInCtx(c.ctx, t, root, "run", "echo hi > marker")
		if code != c.want || strings.HasPrefix(out, "exit ") || !strings.Contains(out, c.skipped) {
			t.Errorf("%s: exit %d, %q; want %d and a line naming it, no receipt", c.skipped, code, out, c.want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); err == nil {
		t.Error("the command ran")
	}
	if entries, _ := os.ReadDir(filepath.Join(state, "mrw-runs")); len(entries) != 0 {
		t.Errorf("a run that never started left %d log(s)", len(entries))
	}
}

// What mrw run exits with, from what stopped the command (Decision 6).
func TestRunExitCodeNamesWhatStoppedTheCommand(t *testing.T) {
	for _, c := range []struct {
		skipped      string
		signal, exit int
		want         int
	}{
		{"timed out after 24h0m0s", 15, -1, 124},
		{check.Interrupted, 15, -1, 130},
		{"", 9, -1, 137},
		{"", 0, 7, 7},
		{"", 0, 0, 0},
	} {
		if got := runExitCode(c.skipped, c.signal, c.exit); got != c.want {
			t.Errorf("skipped %q signal %d exit %d: %d, want %d", c.skipped, c.signal, c.exit, got, c.want)
		}
	}
}

// ADR-177 Decision 6, measured on Windows 11: Git for Windows' sh reports a
// signal death to its Windows parent as signal<<8, so 3840 is TERM, 2304 KILL
// and 512 INT. Only a multiple of 256 up to 64<<8 is read so; 300 and 256*65
// are exit codes. Unix exit codes never exceed 255, so the rule is a value
// rule applied everywhere and no build tag hides it from the Linux run.
func TestAWindowsShellSignalStatusIsTheSignal(t *testing.T) {
	for _, c := range []struct{ exit, sig, code int }{
		{3840, 15, 143}, {2304, 9, 137}, {512, 2, 130}, {256, 1, 129}, {256 * 64, 64, 192},
		{256 * 65, 0, 256 * 65}, {300, 0, 300}, {255, 0, 255}, {7, 0, 7}, {0, 0, 0}, {-1, 0, -1},
	} {
		sig := shellSignal(c.exit)
		if sig != c.sig {
			t.Errorf("shellSignal(%d) = %d, want %d", c.exit, sig, c.sig)
		}
		if got := runExitCode("", sig, c.exit); got != c.code {
			t.Errorf("exit %d: mrw run exits %d, want %d", c.exit, got, c.code)
		}
	}
}

// Decision 7: a flag the parser rejects is a {"error": …} document under
// --json, and plain text without it.
func TestRunUsageErrorUnderJSONIsOneRefusalDocument(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	out, code := runIn(t, root, "run", "--json", "--bogus", "echo hi")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil || code != 2 || len(doc) != 1 || !strings.Contains(doc["error"].(string), "bogus") {
		t.Errorf("--json --bogus: exit %d, %v, %s; want one document holding only error", code, err, out)
	}
	if out, code = runIn(t, root, "run", "--bogus", "echo hi"); code != 2 || strings.HasPrefix(out, "{") {
		t.Errorf("--bogus without --json: exit %d, %q; want 2 and no document", code, out)
	}
}

// A line is blank or not by all of it, not by the first 4,096 bytes kept of it.
func TestRunLastLineIsJudgedWholeNotByItsKeptPrefix(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	out, _ := runIn(t, root, "run", `echo previous; head -c 4096 /dev/zero | tr '\0' ' '; echo FAIL`)
	if got := receiptField(out, "last"); got != "…" {
		t.Errorf("a nonblank line behind 4,096 spaces: last %q, want …", got)
	}
	out, _ = runIn(t, root, "run", `echo previous; head -c 5000 /dev/zero | tr '\0' ' '; echo`)
	if got := receiptField(out, "last"); got != "previous" {
		t.Errorf("the pair: a blank long line: last %q, want previous", got)
	}
}

// Names sort by when the command ENDED, so the newest 64 by name are the 64
// that finished last and a long run's log is not the oldest name.
func TestRunLogIsNamedWhenTheCommandEnds(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	before := time.Now()
	out, code := runIn(t, t.TempDir(), "run", "sleep 1")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	stamp := strings.TrimPrefix(filepath.Base(receiptField(out, "log")), "run-")
	got, err := time.Parse("20060102T150405.000000000Z", stamp[:len("20060102T150405.000000000Z")])
	if err != nil {
		t.Fatal(err)
	}
	if got.Before(before.Add(time.Second - time.Millisecond)) {
		t.Errorf("the log is named %s, from before the command ended (it ran 1 s from %s)", got.Format(time.RFC3339Nano), before.UTC().Format(time.RFC3339Nano))
	}
}

// The --json fallback for a usage error reads the arguments as the parser reads
// a boolean flag: the last occurrence wins, --json and =true/=1 are true,
// =false/=0 are false, and nothing after -- is a flag.
func TestRunJSONFlagIsReadAsTheParserReadsIt(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"--json"}, true},
		{[]string{"-json"}, true},
		{[]string{"--json=true"}, true},
		{[]string{"--json=1"}, true},
		{[]string{"--json="}, true},
		{[]string{"--json=false", "--json="}, true},
		{[]string{"--json=false"}, false},
		{[]string{"--json=0"}, false},
		{[]string{"--json", "--json=false"}, false},
		{[]string{"--json=false", "--json"}, true},
		{[]string{"--bogus", "--json=true", "echo hi"}, true},
		{[]string{"--json", "--", "x"}, true},
		{[]string{"--", "--json"}, false},
		{[]string{"--json=maybe"}, false},
		{[]string{"echo hi"}, false},
		{nil, false},
	} {
		if got := runJSONRequested(c.args); got != c.want {
			t.Errorf("%q: %v, want %v", c.args, got, c.want)
		}
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	out, code := runIn(t, root, "run", "--bogus", "--json=true", "echo hi")
	var doc map[string]any
	if err := decodeFirst(out, &doc); err != nil || code != 2 || len(doc) != 1 {
		t.Errorf("--bogus --json=true: exit %d, %v, %s; want one {error} document", code, err, out)
	}
	if out, code = runIn(t, root, "run", "--json", "--json=false", "--bogus", "echo hi"); code != 2 || strings.HasPrefix(out, "{") {
		t.Errorf("--json --json=false --bogus: exit %d, %q; want 2 and no document", code, out)
	}
}

// A whitespace-only line is blank whole: a multi-byte space (U+2003) split
// across the 64 KiB read boundary must not make it the last line.
func TestRunLastLineBlankIsJudgedAcrossFragmentBoundaries(t *testing.T) {
	needShell(t)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	out, _ := runIn(t, root, "run", `echo previous; head -c 4097 /dev/zero | tr '\0' ' '; awk 'BEGIN{for(i=0;i<21000;i++)printf "\342\200\203"}'; echo`)
	if got := receiptField(out, "last"); got != "previous" {
		t.Errorf("a line of spaces and U+2003 across the 64 KiB boundary: last %q, want previous", got)
	}
	out, _ = runIn(t, root, "run", `echo previous; head -c 4097 /dev/zero | tr '\0' ' '; awk 'BEGIN{for(i=0;i<21000;i++)printf "\342\200\203"}'; echo FAIL`)
	if got := receiptField(out, "last"); got != "…" {
		t.Errorf("the pair: the same line ending in FAIL: last %q, want …", got)
	}
}

// The done rename never replaces a log that is already there.
func TestRunLogIsNeverReplacedByTheRename(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a.running.log"), filepath.Join(dir, "a.log")
	if err := os.WriteFile(from, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(from, to); err == nil {
		t.Error("renamed over an existing log")
	}
	if b, _ := os.ReadFile(to); string(b) != "old\n" {
		t.Errorf("the existing log was replaced: %q", b)
	}
	if _, err := os.Stat(from); err != nil {
		t.Errorf("the refused rename lost the running log: %v", err)
	}
	if err := os.Remove(to); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(from, to); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(to); string(b) != "new\n" {
		t.Errorf("the pair: a free name did not receive the log: %q", b)
	}
	if _, err := os.Stat(from); err == nil {
		t.Error("the running name was left behind")
	}
}
