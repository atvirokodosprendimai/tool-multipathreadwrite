package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
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
		return cli.Exit("- reads standard input, and a read of it names nothing else: read the files in another call", exitUsage)
	}
	for _, f := range []string{"sub", "quiet", "stat", "count", "files-from", "ast-grep", "exclude", "no-ignore", "json", "if-changed"} {
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
	var re *regexp.Regexp
	if grepSet {
		if len(sp.Ranges) > 0 {
			return cli.Exit(fmt.Sprintf("%s: a range and --grep are two answers to one question", posArgs[0]), exitUsage)
		}
		if re, err = regexp.Compile(pattern); err != nil {
			return cli.Exit(fmt.Sprintf("--grep %q: %v", pattern, err), exitUsage)
		}
		sp.Ranges = []read.Range{{Re: re, Text: "/" + pattern + "/"}}
	}
	// ADR-078: -C widens a single /pattern/ match, here as on a file.
	if cmd.IsSet("context") && !grepSet && !hasSinglePattern([]read.Spec{sp}) {
		return cli.Exit("-C widens a /pattern/ match, and no spec here has one: name a pattern (-:/x/) or drop -C", exitUsage)
	}
	// A terminal is no input: reading it would wait for the caller to type.
	if stdinIsTerminal() {
		return cli.Exit("- reads standard input, and it is a terminal here: pipe the input in (cmd | mrw read -)", exitUsage)
	}
	b, err := read.ReadStdin(os.Stdin)
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
	_, problems := read.Run(stdout, cmd.Root().String("root"), []read.Spec{sp}, read.Options{
		Numbers:  !cmd.Bool("no-numbers"),
		Context:  cmd.Int("context"),
		MaxLines: maxLines(cmd),
		MaxCols:  cmd.Int("max-cols"),
		NextLine: true,
		Withheld: &withheld,
		Project:  projections,
	})
	if err := out.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf("the answer could not be written: %v", err), exitUsage)
	}
	_, _ = os.Stderr.Write(withheld.Bytes())
	if problems == 0 && !budget.Over() {
		return nil
	}
	reached := ""
	if budget.Over() {
		reached = fmt.Sprintf("; --max-chars %d reached", cmd.Int("max-chars"))
	}
	if re != nil && !anyLineMatches(re, string(b)) {
		return cli.Exit(fmt.Sprintf("no match: no line of standard input matched /%s/%s", read.OneLine(pattern), reached), 1)
	}
	return cli.Exit(fmt.Sprintf("incomplete: %d range(s) could not be served%s", problems, reached), 1)
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

// liftLoneDash moves a lone `-` to the end of the arguments when only flags
// follow it (ADR-161). urfave/cli stops at a lone - and drops every token after
// it, flags included (command_parse.go in urfave/cli v3), so
// `mrw write - --dry-run` wrote the plan and `mrw write - --then vet` skipped
// vet. Moved last, those flags are parsed and - is still the last positional.
// Anything else after a lone - — a path, a second -, a `--` — would be dropped,
// so it is refused by name, as is a flag left with no value once - moves.
// Tokens are read as the parser reads them (flagRole; flagKinds, so a root flag
// a subcommand inherits counts). A - that is a flag's value, a padded ' - '
// (ADR-069 T9 judges it), a dash before a non-letter (the parser keeps what
// follows it as paths), a token before the subcommand and anything after `--`
// are left as they are. args[0] is the program.
func liftLoneDash(root *cli.Command, args []string) ([]string, error) {
	rootKinds := takesValue(root.Flags)
	i := 1
	for i < len(args) {
		role, next := flagRole(args[i], rootKinds)
		if role == roleTerminator { // ends the root's options only (ADR-069)
			i++
			continue
		}
		if role != roleFlag {
			break
		}
		if next {
			i++
		}
		i++
	}
	if i >= len(args) {
		return args, nil
	}
	sub := root.Command(strings.TrimSpace(args[i]))
	if sub == nil {
		return args, nil
	}
	kinds := flagKinds(sub, root)
	dash := -1
	pending := false // the next token is a flag's value
	for j := i + 1; j < len(args); j++ {
		if pending {
			pending = false
			continue
		}
		role, next := flagRole(args[j], kinds)
		if dash < 0 {
			if role == roleTerminator || (role == roleStop && args[j] != "-") {
				return args, nil
			}
			if args[j] == "-" {
				dash = j
				continue
			}
		} else if role != roleFlag {
			return nil, cli.Exit(fmt.Sprintf("%q after a lone - would be dropped unread: the flag parser stops at -, so give - last and alone", args[j]), exitUsage)
		}
		pending = role == roleFlag && next
	}
	if dash < 0 || dash == len(args)-1 {
		return args, nil
	}
	if pending {
		return nil, cli.Exit(fmt.Sprintf("%s needs a value, and nothing follows it: the - before it is not one", strings.TrimSpace(args[len(args)-1])), exitUsage)
	}
	lifted := append(append([]string{}, args[:dash]...), args[dash+1:]...)
	return append(lifted, "-"), nil
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
