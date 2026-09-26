package check

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// tempDirForLogs points os.TempDir at a fresh directory for the test.
func tempDirForLogs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", dir)
		t.Setenv("TEMP", dir)
	}
	return dir
}

// ADR-080. A failing or truncated check keeps its log, since the report points
// at it, and nothing bounded how many accumulated: 3,103 on one machine. A check
// removes its own logs older than a week; a young one, another file and a
// directory that only looks like one are kept, and the count is reported.
func TestACheckRemovesItsOwnLogsOlderThanAWeek(t *testing.T) {
	dir := tempDirForLogs(t)
	old := time.Now().Add(-LogRetention - time.Hour)
	for name, age := range map[string]time.Time{"mrw-check-old.log": old, "mrw-check-young.log": time.Now(), "other-old.log": old} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, age, age); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "mrw-check-dir.log"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(dir, "mrw-check-dir.log"), old, old); err != nil {
		t.Fatal(err)
	}
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "exit 0", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Pruned != 1 {
		t.Errorf("pruned %d, want 1", res.Pruned)
	}
	for name, want := range map[string]bool{"mrw-check-old.log": false, "mrw-check-young.log": true, "other-old.log": true, "mrw-check-dir.log": true} {
		if _, err := os.Stat(filepath.Join(dir, name)); (err == nil) != want {
			t.Errorf("%s: present=%v, want %v", name, err == nil, want)
		}
	}
}

// ADR-080. A signal that landed before the check's process started reported
// "could not start: context canceled" and told the caller to declare a check.
// It says interrupted, and the empty log is removed rather than left unnamed.
func TestACheckCancelledBeforeItStartsSaysInterrupted(t *testing.T) {
	needShell(t)
	dir := tempDirForLogs(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := Run(ctx, t.TempDir(), Config{Check: "exit 0", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran || res.Skipped != Interrupted || res.OutputFile != "" {
		t.Errorf("want an interrupted check that never ran and kept no log: %+v", res)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "mrw-check-*.log")); len(left) != 0 {
		t.Errorf("a check that never started left %v", left)
	}
}

// ADR-080. A timed-out check keeps its log, and the report named nowhere to
// find it. The result keeps the file for the report to name.
func TestATimedOutCheckKeepsItsLog(t *testing.T) {
	tempDirForLogs(t)
	if runtime.GOOS == "windows" {
		t.Skip("the check runs through sh")
	}
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "echo started; sleep 30", TimeoutSeconds: 1, declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Skipped, "timed out") || res.OutputFile == "" {
		t.Fatalf("want a timed-out check that kept its log: %+v", res)
	}
	if _, err := os.Stat(res.OutputFile); err != nil {
		t.Errorf("the kept log is not there: %v", err)
	}
}

// ADR-080, the reviews of #241. The prune globbed, and a glob reads
// metacharacters in the directory too: a TMPDIR of t[x] pruned tx's logs and
// kept its own. It lists the directory it was given.
func TestAPruneListsTheDirectoryItWasGiven(t *testing.T) {
	base := t.TempDir()
	old := time.Now().Add(-LogRetention - time.Hour)
	for _, d := range []string{"t[x]", "tx"} {
		p := filepath.Join(base, d, "mrw-check-old.log")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if n := pruneLogs(filepath.Join(base, "t[x]"), time.Now().Add(-LogRetention)); n != 1 {
		t.Errorf("pruned %d, want 1", n)
	}
	if _, err := os.Stat(filepath.Join(base, "t[x]", "mrw-check-old.log")); err == nil {
		t.Error("the directory's own old log was kept")
	}
	if _, err := os.Stat(filepath.Join(base, "tx", "mrw-check-old.log")); err != nil {
		t.Errorf("a sibling directory's log was removed: %v", err)
	}
}

// ADR-080, the review of #241. A check that could not start because its shell
// is missing, under a context also cancelled, was reported interrupted — exit 3
// for what is a missing check, exit 2. The start's own error decides.
func TestAMissingShellUnderACancelIsStillCouldNotStart(t *testing.T) {
	tempDirForLogs(t)
	t.Setenv("PATH", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := Run(ctx, t.TempDir(), Config{Check: "exit 0", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran || !strings.HasPrefix(res.Skipped, "could not start") {
		t.Errorf("want could not start, not interrupted: %+v", res)
	}
}
