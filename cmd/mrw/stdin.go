package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// stdinSpec reports whether a read names standard input: `-`, alone or with an
// address (`-:3-5`). A file named - is spelled ./- (ADR-161).
func stdinSpec(posArgs []string) bool {
	for _, a := range posArgs {
		if a == "-" || strings.HasPrefix(a, "-:") {
			return true
		}
	}
	return false
}

// stdinRead is `mrw read [flags] -`: standard input served as one unnamed file
// with every address, --grep, -C, --max-lines and --max-cols, so a command's
// output is narrowed by mrw and not by grep, sed -n or head. It records
// nothing: standard input is not a file a plan can name (ADR-161).
func stdinRead(cmd *cli.Command, pattern string, grepSet bool, posArgs []string, projections []read.Projection) error {
	if len(posArgs) != 1 {
		// `cmd | mrw read --grep -v PATTERN -`: --grep took grep's flag as its
		// pattern and PATTERN became a second thing to read. Say which flag,
		// with what mrw spells instead, as a read of files does (ADR-140,
		// ADR-164). It does not say where the other argument stood: an attached
		// value (--grep=-v) and a flag typed last reach here the same way.
		if hint := stdinPatternFlagHint(pattern); grepSet && hint != "" {
			return cli.Exit(fmt.Sprintf("--grep took %q as its pattern, and a read of standard input names nothing else beside -; %s", pattern, hint), exitUsage)
		}
		return cli.Exit("- reads standard input, and a read of it names nothing else: read the files in another call", exitUsage)
	}
	for _, f := range []string{"sub", "quiet", "stat", "count", "files-from", "ast-grep", "exclude", "no-ignore", "json", "if-changed", "outline", "if-served"} {
		if cmd.IsSet(f) {
			return cli.Exit(fmt.Sprintf("- reads standard input, which licenses nothing and is no tree to walk, so --%s has nothing to act on: drop it, or read a file", f), exitUsage)
		}
	}
	// The address is parsed on a placeholder name: on Windows "-:2" reads as a
	// volume ("-:"), so ParseSpec found no range and served everything.
	sp, err := read.ParseSpec("stdin" + strings.TrimPrefix(posArgs[0], "-"))
	if err != nil {
		return cli.Exit(strings.Replace(err.Error(), "stdin", "-", 1), exitUsage)
	}
	if len(projections) > 0 && len(sp.Ranges) > 0 {
		return cli.Exit(fmt.Sprintf("%s: --json-path prints values, and an address names lines: give - alone", posArgs[0]), exitUsage)
	}
	sp.Raw = posArgs[0]
	// ADR-202: the lines that do not match are chosen once the input is read.
	invert := cmd.Bool("invert")
	var re *regexp.Regexp
	if grepSet {
		if len(sp.Ranges) > 0 {
			return cli.Exit(fmt.Sprintf("%s: a range and --grep are two answers to one question", posArgs[0]), exitUsage)
		}
		if re, err = regexp.Compile(pattern); err != nil {
			return cli.Exit(fmt.Sprintf("--grep %q: %v", pattern, err), exitUsage)
		}
		if !invert {
			sp.Ranges = []read.Range{{Re: re, Text: "/" + pattern + "/"}}
		}
	}
	// ADR-078: -C widens a single /pattern/ match, here as on a file.
	if cmd.IsSet("context") && !grepSet && !hasSinglePattern([]read.Spec{sp}) {
		return cli.Exit("-C widens a /pattern/ match, and no spec here has one: name a pattern (-:/x/) or drop -C", exitUsage)
	}
	// A terminal is no input: reading it would wait for the caller to type.
	if stdinIsTerminal() {
		return cli.Exit("- reads standard input, and it is a terminal here: pipe the input in (cmd | mrw read -)", exitUsage)
	}
	b, err := read.Stdin(os.Stdin)
	if err != nil {
		return cli.Exit(fmt.Sprintf("standard input: %v", err), exitUsage)
	}
	if b == nil {
		b = []byte{}
	}
	// ADR-161 Decision 13: PowerShell's trailing CRLF is not a line of the input.
	if trimmed, cut := ingest.TrimPipedCRLF(b); cut {
		fmt.Fprintln(os.Stderr, "mrw: standard input ended in CRLF after lines that end in LF; that CRLF is most likely the one PowerShell appends to what it pipes, so it was dropped")
		b = trimmed
	}
	if invert {
		ls, _, _ := lines.Split(string(b))
		var runs []read.Range
		for i, l := range ls {
			if re.MatchString(l) {
				continue
			}
			if k := len(runs) - 1; k >= 0 && runs[k].End == i {
				runs[k].End = i + 1
				continue
			}
			runs = append(runs, read.Range{Start: i + 1, End: i + 1})
		}
		if len(runs) == 0 {
			// Decided here: a spec with no range means the whole input.
			return invertedNothingLeft(b, len(ls), pattern)
		}
		for i := range runs {
			runs[i].Text = fmt.Sprintf("%d-%d", runs[i].Start, runs[i].End)
		}
		sp.Ranges = runs
	}
	sp.Path, sp.Data = "-", b
	out := bufio.NewWriter(os.Stdout)
	// ADR-165: --max-chars bounds standard input's answer as it bounds a file's
	// (go-reviewer of a043d346: this path once ignored it), the list of what it
	// kept back written after the answer.
	var stdout io.Writer = out
	var budget *read.Budget
	if cmd.IsSet("max-chars") {
		budget = read.NewBudget(out, cmd.Int("max-chars"))
		stdout = budget
	}
	var withheld bytes.Buffer
	cols, colsFixed := effectiveMaxCols(cmd)
	var wholeNames []string
	var matchesLeft, capped int
	var long read.LongLines
	var wholeLong *read.LongLines
	if !colsFixed {
		wholeLong = &long
	}
	_, problems := read.Run(stdout, cmd.Root().String("root"), []read.Spec{sp}, read.Options{
		Numbers:       !cmd.Bool("no-numbers"),
		Context:       cmd.Int("context"),
		MaxLines:      maxLines(cmd),
		MaxCols:       cols,
		MaxMatches:    maxMatches(cmd),
		Fold:          cmd.Bool("fold"),
		Unshown:       &matchesLeft,
		Capped:        &capped,
		NoComments:    cmd.Bool("no-comments"),
		WholeNames:    &wholeNames,
		NextLine:      !invert, // the line after a run is one the caller asked not to see (ADR-202)
		NoReplaceNote: invert,  // a stream is never written
		Withheld:      &withheld,
		Project:       projections,
		WholeLong:     wholeLong,
	})
	if err := out.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf("the answer could not be written: %v", err), exitUsage)
	}
	_, _ = os.Stderr.Write(withheld.Bytes())
	if len(wholeNames) > 0 {
		fmt.Fprintf(os.Stderr, "mrw: --no-comments reads .go files; served whole: %s\n", namedFiles(wholeNames))
	}
	if hint := longLineHint(long); hint != "" && !toNullDevice(os.Stdout) {
		fmt.Fprintln(os.Stderr, hint)
	}
	if problems == 0 && !budget.Over() {
		return nil
	}
	if problems == capped && !budget.Over() { // ADR-173: only the caller's own caps
		return cli.Exit(cappedLine(capped, matchesLeft, maxMatches(cmd)), 0)
	}
	reached := ""
	if budget.Over() {
		reached = fmt.Sprintf("; --max-chars %d reached", cmd.Int("max-chars"))
	}
	if re != nil && !invert && !anyLineMatches(re, string(b)) {
		return cli.Exit(fmt.Sprintf("no match: no line of standard input matched /%s/%s", read.OneLine(pattern), reached), 1)
	}
	return cli.Exit(incompleteLine(problems, matchesLeft, maxMatches(cmd), reached), 1)
}

// invertedNothingLeft answers an inverted read of standard input in which no
// line is left to serve (ADR-202): the stream's header and one `!!` line, as
// read.Run prints a header, and exit 1 as a search that found nothing does.
func invertedNothingLeft(b []byte, n int, pattern string) error {
	sum := sha256.Sum256(b)
	fmt.Fprintf(os.Stdout, "==> -  %dL  %dB  sha %s\n", n, len(b), hex.EncodeToString(sum[:])[:8])
	if n == 0 {
		fmt.Fprintln(os.Stdout, "!! standard input is empty")
		return cli.Exit("no match: standard input is empty", 1)
	}
	p := read.OneLine(pattern)
	fmt.Fprintf(os.Stdout, "!! every line matches /%s/\n", p)
	return cli.Exit(fmt.Sprintf("no match: every line of standard input matched /%s/", p), 1)
}

// invertAllowed are the flags that may stand beside --invert: --grep, which it
// needs, and the ones that act on the lines served (ADR-202).
var invertAllowed = []string{"invert", "grep", "max-lines", "max-cols", "max-chars", "fold", "no-numbers"}

// invertRefusal is the sentence for an --invert that is not `--grep PATTERN
// --invert -`, or "" when it is that form or --invert is not given. It speaks
// before anything is read or dispatched, in the order of ADR-202 Decision 3.
func invertRefusal(cmd *cli.Command, grepSet bool, posArgs []string) string {
	if !cmd.Bool("invert") {
		return ""
	}
	const form = "--grep PATTERN --invert -"
	if !grepSet {
		return "--invert needs --grep: it serves the lines of standard input that do not match a pattern, as " + form
	}
	if len(posArgs) != 1 || posArgs[0] != "-" {
		for _, a := range posArgs {
			if !stdinSpec([]string{a}) {
				return "--invert is for standard input, and a read of files serves the lines that match: the form is " + form
			}
		}
		return "--invert reads standard input and nothing else, with no address and no second argument: the form is " + form
	}
	for _, f := range cmd.Flags {
		name := f.Names()[0]
		if !cmd.IsSet(name) || slices.Contains(invertAllowed, name) {
			continue
		}
		shown := "--" + name
		if name == "context" {
			shown = "-C"
		}
		return fmt.Sprintf("--invert allows only --grep, --max-lines, --max-cols, --max-chars, --fold and -N beside it, and %s is not one of them: drop %s; the form is %s", shown, shown, form)
	}
	return ""
}

// anyLineMatches reports whether a line of s matches re, the lines split as
// read.Run splits them (internal/lines: CRLF, CR-only, and a mixed file's stray
// CR kept), so no match here is no match there.
func anyLineMatches(re *regexp.Regexp, s string) bool {
	ls, _, _ := lines.Split(s)
	for _, l := range ls {
		if re.MatchString(l) {
			return true
		}
	}
	return false
}

// stdinIsTerminal reports whether standard input is a character device other
// than the null device: a terminal, where a read would wait for typing. The
// null device is a character device too, and is an empty input; toNullDevice
// tells them apart on Windows too, where a console and NUL share a zero file ID
// and os.SameFile calls them one file (Codex review of T8).
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return !toNullDevice(os.Stdin)
}
