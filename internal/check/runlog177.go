package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// RunCeiling is how long a command started through Exec may run when its
// caller names no timeout: the runner's own ceiling, a day, and not the check's
// five minutes (ADR-177 Decision 2).
const RunCeiling = maxTimeoutSeconds * time.Second

// RunLogsKept is how many completed run logs PruneRunLogs leaves, newest by
// name (ADR-177 Decision 4).
const RunLogsKept = 64

// runningLogAge is how old a *.running.log must be before its run is certainly
// gone: the ceiling, and an hour more.
const runningLogAge = RunCeiling + time.Hour

// RunSpec is what Exec takes beyond the command: the log it writes, and how
// long the command may run.
type RunSpec struct {
	// LogDir and LogName are the log the command's output goes to, made
	// with mode 0600 and refused if it exists. The directory must exist.
	LogDir, LogName string
	// Timeout is the command's deadline; zero or less means RunCeiling.
	Timeout time.Duration
	// keep is set by Exec: the log of a run that passed is kept, where the
	// check deletes it (ADR-177 Decision 3).
	keep bool
}

// Exec runs cmdline under the shell in root through the check's own runner:
// output to the named log and never a pipe, the process group or job object,
// the verdict from the process, an interrupt, terminate or hangup sent to mrw
// stopping the command, and MRW_STEP_DEPTH one higher than mrw's own
// (ADR-177). Every log is kept, whatever the exit; only one made for a command
// that never started is removed. An error is a log that could not be made.
func Exec(ctx context.Context, root, cmdline string, spec RunSpec) (Result, error) {
	spec.keep = true
	if spec.Timeout <= 0 {
		spec.Timeout = RunCeiling
	}
	ctx, stop := subproc.Interruptible(ctx)
	defer stop()
	return runWith(ctx, root, Config{}, cmdline, deeper(), spec)
}

// Signal is the number of the signal that ended the command, or zero when it
// exited by itself or never started. Its exit code is -1 then (ADR-101), so a
// caller needs this to say 128+N; a command mrw stopped carries the TERM it
// was sent (ADR-177 Decision 6).
func (r Result) Signal() int { return r.sig }

// signalOf reads the signal out of the wait status; zero on Windows, whose
// status carries only an exit code.
func signalOf(ps *os.ProcessState) int {
	if ws, ok := ps.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return int(ws.Signal())
	}
	return 0
}

// openLog makes the log a run writes: the one the caller named, or a check's
// own temporary file.
func openLog(spec RunSpec) (*os.File, error) {
	if spec.LogName == "" {
		return os.CreateTemp("", "mrw-check-*.log")
	}
	return os.OpenFile(filepath.Join(spec.LogDir, spec.LogName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}

const runLogPrefix, runningSuffix, doneSuffix = "run-", ".running.log", ".log"

// RunLogNames are the names of a run's log while it runs and after it exits:
// run-<UTC time to the millisecond>-<pid>.running.log and the same ending
// .log. They sort by time (ADR-177 Decision 3).
func RunLogNames(now time.Time, pid int) (running, done string) {
	stem := fmt.Sprintf("%s%s-%d", runLogPrefix, now.UTC().Format("20060102T150405.000Z"), pid)
	return stem + runningSuffix, stem + doneSuffix
}

func isRunningLog(name string) bool {
	return strings.HasPrefix(name, runLogPrefix) && strings.HasSuffix(name, runningSuffix)
}

// isDoneLog excludes a running log by name: "run-x.running.log" ends in .log.
func isDoneLog(name string) bool {
	return strings.HasPrefix(name, runLogPrefix) && strings.HasSuffix(name, doneSuffix) && !strings.HasSuffix(name, runningSuffix)
}

// PruneRunLogs removes, best effort, completed run logs in dir older than
// LogRetention, then all but the newest RunLogsKept by name, and any
// *.running.log older than 25 hours, whose run is gone. It says how many it
// removed. A file of any other name is left alone (ADR-177 Decision 4).
func PruneRunLogs(dir string, now time.Time) int {
	return pruneMatching(dir, isDoneLog, now.Add(-LogRetention), RunLogsKept) +
		pruneMatching(dir, isRunningLog, now.Add(-runningLogAge), 0)
}
