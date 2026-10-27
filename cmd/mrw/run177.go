package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

const (
	// exitRunTimedOut is GNU timeout's code, for a run that reached the ceiling.
	exitRunTimedOut = 124
	// exitRunInterrupted is the shell's 128+SIGINT, for a run mrw stopped on an
	// interrupt, terminate or hangup.
	exitRunInterrupted = 130
	// lastChars is how much of the last non-empty line the receipt shows.
	lastChars = 200
	// lastReadLines is how many trailing lines the printed read command serves.
	lastReadLines = 40
	// lastScanBytes is how much of a line the scan keeps to cut `last` from.
	lastScanBytes = 4096
)

// runReceipt is `mrw run --json` (ADR-177 Decision 7): the human receipt's
// fields, and the run's verdict as the other receipts spell it. exit_code is
// what mrw exits with, -1 when the command never ran (ADR-101).
type runReceipt struct {
	Command    string `json:"command"`
	Ran        bool   `json:"ran"`
	ExitCode   int    `json:"exit_code"`
	Signal     int    `json:"signal,omitempty"`
	Skipped    string `json:"skipped,omitempty"`
	DurationMS int64  `json:"duration_ms"`
	Lines      int    `json:"lines"`
	Bytes      int64  `json:"bytes"`
	Last       string `json:"last,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
	PrunedLogs int    `json:"pruned_logs,omitempty"`
}

// runCmd is `mrw run 'CMD'`: the command's output goes to a log and the caller
// gets four lines (ADR-177).
func runCmd() *cli.Command {
	return &cli.Command{
		Name:            "run",
		HideHelpCommand: true,
		Usage:           "run `CMD` with sh -c, keep its output in a log, print a receipt instead of the output, and exit with its code",
		ArgsUsage:       "CMD",
		Description: `CMD is one string: mrw run 'go test ./...'. Its standard output and standard error,
interleaved, go to a log beside mrw's state directory (<state home>/mrw-runs/), mode 0600, and
the receipt is the exit code with the line and byte counts and the time, the last non-empty line,
the log's path, and a mrw read command that serves the log's last 40 lines. A longer look is
mrw read --grep P -C 3 - < LOG. Standard input is the null device, so a command that waits for
input ends. The deadline is 24 hours, and an interrupt sent to mrw stops the command.

The exit code is the command's own, 128+N when a signal N killed it; 124 when it reached the
deadline and 130 when mrw was interrupted are mrw's. A failure of mrw's own (a usage error, a log
that could not be made, a shell that could not start) exits 2 with a mrw: line and no exit line in
the receipt. Every log is kept; one older than 7 days is eligible for removal, as are all but the
newest 64, and the next completed run removes them. ` + guide.ThenShCaveat(),
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "emit the receipt as JSON"},
		},
		Action: runAction,
	}
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	emit := emitRunJSON
	// A refusal before anything runs is one document under --json, as check's is.
	refuse := func(err error) error {
		if cmd.Bool("json") {
			emit(checkRefusal{Error: err.Error()})
		}
		return exitWith(err)
	}
	// First, before an argument is judged or a log directory made (ADR-095).
	if err := check.DepthRefusal("mrw run is"); err != nil {
		return refuse(err)
	}
	if err := refusePaddedArgs(cmd); err != nil {
		return refuse(err)
	}
	if cmd.Args().Len() != 1 {
		return refuse(errors.New("mrw run takes exactly one argument, the command as one string: mrw run 'CMD', for example mrw run 'go test ./...'"))
	}
	cmdline := cmd.Args().First()

	base, err := state.Base()
	if err != nil {
		return refuse(err)
	}
	dir := filepath.Join(filepath.Dir(base), "mrw-runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return refuse(err)
	}
	running, _ := check.RunLogNames(time.Now(), os.Getpid())
	res, err := check.Exec(ctx, cmd.Root().String("root"), cmdline, check.RunSpec{LogDir: dir, LogName: running})
	if err != nil {
		return refuse(err)
	}
	rec := runReceipt{Command: cmdline, Ran: res.Ran, ExitCode: -1, Skipped: res.Skipped, DurationMS: res.DurationMS}
	if !res.Ran {
		if cmd.Bool("json") {
			emit(rec)
		}
		// An interrupt or a deadline before the command started is still mrw
		// stopped, 130 or 124; only a command that could not start is a 2.
		switch res.Skipped {
		case check.Interrupted:
			return cli.Exit(res.Skipped, exitRunInterrupted)
		case check.TimedOutBeforeStart:
			return cli.Exit(res.Skipped, exitRunTimedOut)
		}
		return cli.Exit(res.Skipped, exitUsage)
	}

	// Named when the command ended, so names sort in the order runs finished
	// and the newest 64 by name are the 64 that finished last.
	_, done := check.RunLogNames(time.Now(), os.Getpid())
	logPath := res.OutputFile
	if final := filepath.Join(dir, done); renameNoReplace(logPath, final) == nil {
		logPath = final
	} else {
		done = ""
		fmt.Fprintf(os.Stderr, "mrw: could not rename the log to %s; it stays %s and is removed after 25 hours\n", final, filepath.Base(logPath))
	}
	code := runExitCode(res.Skipped, res.Signal(), res.ExitCode)
	rec.ExitCode, rec.Signal, rec.OutputFile = code, res.Signal(), logPath
	rec.Lines, rec.Bytes, rec.Last = scanLog(logPath)
	// This run's own log is exempt from the prune, whatever its name sorts as.
	rec.PrunedLogs = check.PruneRunLogs(dir, time.Now(), done)

	if cmd.Bool("json") {
		emit(rec)
	} else {
		printRunReceipt(os.Stdout, rec)
	}
	if code != 0 {
		// Silent: the receipt is the report, and main prints no empty mrw: line.
		return cli.Exit("", code)
	}
	return nil
}

// runExitCode is what mrw run exits with (ADR-177 Decision 6): the command's
// own code, 128+N for signal N, and mrw's 124 and 130 for a run it stopped. A
// stopped command died of the TERM mrw sent, so its signal does not count.
func runExitCode(skipped string, signal, exit int) int {
	switch {
	case strings.HasPrefix(skipped, "timed out"):
		return exitRunTimedOut
	case skipped == check.Interrupted:
		return exitRunInterrupted
	case signal > 0:
		return 128 + signal
	}
	return exit
}

// emitRunJSON writes v as the one indented document mrw run --json prints.
func emitRunJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// printRunReceipt writes the four lines (Decision 5): last and read are left
// out for an empty log, and a pruning count only when it is not zero.
func printRunReceipt(w io.Writer, r runReceipt) {
	head := "exit " + strconv.Itoa(r.ExitCode)
	if r.Skipped != "" {
		head += " · " + r.Skipped
	}
	fmt.Fprintf(w, "%s · %s lines · %s bytes · %.1f s\n", head, commas(int64(r.Lines)), commas(r.Bytes), float64(r.DurationMS)/1000)
	if r.Last != "" {
		fmt.Fprintf(w, "last: %s\n", writer.Shown(r.Last))
	}
	fmt.Fprintf(w, "log: %s\n", r.OutputFile)
	if r.Lines > 0 {
		fmt.Fprintf(w, "read: %s\n", runReadLine(runtime.GOOS, r.OutputFile, r.Lines))
	}
	if r.PrunedLogs > 0 {
		fmt.Fprintf(w, "removed: %d old run log(s)\n", r.PrunedLogs)
	}
}

// runReadLine is a command that serves the last lastReadLines lines of a log
// of the given length: standard input reaches a file outside every checkout,
// licenses nothing and records nothing in the ledger. Flags go before --. On
// Windows it is PowerShell's form, with ' doubled (ADR-177 Decision 5).
func runReadLine(goos, path string, lines int) string {
	addr := "'-:" + strconv.Itoa(max(lines-lastReadLines+1, 1)) + "-$'"
	if goos == "windows" {
		return "Get-Content -Raw -LiteralPath '" + strings.ReplaceAll(path, "'", "''") + "' | mrw read -- " + addr
	}
	return "mrw read -- " + addr + " < " + posixQuote(path)
}

// commas groups n's digits by threes.
func commas(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// scanLog streams a log once and reports how many lines it holds, its size, and
// the last line that is not blank, cut to lastChars characters followed by ….
// A line is counted as the tail reader counts it: split on \n, a final line
// without one included. Nothing is held but one line's first lastScanBytes.
func scanLog(path string) (lines int, size int64, last string) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, ""
	}
	defer func() { _ = f.Close() }()
	r := bufio.NewReaderSize(f, 64<<10)
	var cur []byte
	// nonblank is judged over every byte of the line, not the kept prefix: a
	// line of 4,096 spaces and then FAIL is the last line, shown as ….
	over, nonblank := false, false
	// pend is the start of a multi-byte character cut by a fragment boundary,
	// carried to the next fragment so a U+2003 split in two is still a space.
	var pend []byte
	end := func() {
		lines++
		if len(pend) > 0 {
			nonblank = true // bytes that never became a character are text
		}
		if nonblank {
			if t := strings.TrimRight(string(cur), "\r"); strings.TrimSpace(t) != "" {
				last = t
			} else {
				last = "…"
			}
		}
		cur, over, nonblank, pend = cur[:0], false, false, pend[:0]
	}
	for {
		frag, err := r.ReadSlice('\n')
		size += int64(len(frag))
		body := frag
		if err == nil {
			body = frag[:len(frag)-1]
		}
		take := min(max(lastScanBytes-len(cur), 0), len(body))
		cur = append(cur, body[:take]...)
		over = over || take < len(body)
		if !nonblank {
			var whole []byte
			whole, pend = splitPartialRune(append(append([]byte(nil), pend...), body...))
			nonblank = len(bytes.TrimSpace(whole)) > 0
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err == nil {
			end()
			continue
		}
		if len(frag) > 0 || len(cur) > 0 || over {
			end()
		}
		break
	}
	last = strings.ToValidUTF8(last, "")
	if utf8.RuneCountInString(last) > lastChars {
		last = string([]rune(last)[:lastChars]) + "…"
	}
	return lines, size, last
}

// splitPartialRune cuts a trailing, incomplete multi-byte character off b.
func splitPartialRune(b []byte) (whole, partial []byte) {
	for i := 1; i < utf8.UTFMax && i <= len(b); i++ {
		if c := b[len(b)-i]; utf8.RuneStart(c) {
			if !utf8.FullRune(b[len(b)-i:]) {
				return b[:len(b)-i], b[len(b)-i:]
			}
			break
		}
	}
	return b, nil
}

// runJSONRequested reads --json out of the raw arguments of a run whose parser
// stopped early, as the parser reads a boolean flag: the last occurrence wins,
// a bare --json and =true, =1 are true, =false, =0 are false, a value that is
// none of those leaves the last good one, and nothing after -- is a flag.
func runJSONRequested(args []string) bool {
	on := false
	for _, a := range args {
		if a == "--" {
			break
		}
		name, val, hasVal := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if name != "json" || !strings.HasPrefix(a, "-") {
			continue
		}
		if !hasVal || val == "" { // urfave/cli v3 reads an empty boolean value as true
			on = true
		} else if b, err := strconv.ParseBool(val); err == nil {
			on = b
		}
	}
	return on
}

// renameNoReplace moves from to to and fails if to exists: a hard link is
// refused on an existing name where a rename would replace it, so a log that is
// already there is never lost to another run's (ADR-177 Decision 3).
func renameNoReplace(from, to string) error {
	if err := os.Link(from, to); err != nil {
		return err
	}
	return os.Remove(from)
}
