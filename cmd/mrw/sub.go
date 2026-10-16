package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// builtSub is a plan made from a --grep walk and a replacement (ADR-161).
type builtSub struct {
	plan     []byte
	observed map[string]seen.Observation // the planned lines, which are all the plan licenses
	sites    int
	files    int
	problems int // paths refused and ranges not served: a partial plan
}

// buildSub walks posArgs for pattern and plans repl over the lines it finds,
// for `read --sub` and `write --grep --sub` alike. What the walk could not
// serve goes to diag, never to the plan. Nothing is recorded here: that is the
// caller's, after the plan has reached its reader. The error is the exit: 2 for
// a usage mistake, 1 when nothing was planned or a path cannot be carried.
func buildSub(root string, posArgs []string, pattern, repl string, excludes []string, noIgnore bool, diag io.Writer) (*builtSub, error) {
	// The plan's body is lines, and a replacement that makes more than one
	// would turn a one-line hunk into several the header does not count.
	if strings.ContainsAny(repl, "\r\n") {
		return nil, cli.Exit("--sub REPLACEMENT cannot hold a line break: it rewrites one line, as sed does; a change across lines is a plan", exitUsage)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, cli.Exit(fmt.Sprintf("--grep %q: %v", pattern, err), exitUsage)
	}
	if len(posArgs) > 0 {
		set, err := iter.Load(root)
		if err != nil {
			return nil, cli.Exit(err, exitUsage)
		}
		if posArgs, err = set.ResolveAll(posArgs); err != nil {
			return nil, cli.Exit(err, exitUsage)
		}
	}
	for _, a := range posArgs {
		sp, err := read.ParseSpec(a)
		if err != nil {
			return nil, cli.Exit(err, exitUsage)
		}
		if len(sp.Ranges) > 0 {
			return nil, cli.Exit(fmt.Sprintf("%s: a range and --grep are two answers to one question", a), exitUsage)
		}
	}
	var skipped read.WalkSkipped
	specs, refusals, err := read.Walk(root, posArgs, read.WalkOptions{Pattern: re, Exclude: excludes, NoIgnore: noIgnore, Skipped: &skipped})
	if err != nil {
		return nil, cli.Exit(err, exitUsage)
	}
	for _, p := range refusals {
		fmt.Fprintf(diag, "==> %s  REFUSED  %s\n", p.Path, p.Reason)
	}
	skipNote := func() {
		if note := read.SkipNote(skipped, "--no-ignore"); note != "" {
			fmt.Fprintln(diag, note)
		}
	}
	if len(specs) == 0 {
		skipNote()
		return nil, cli.Exit(noMatchLine(pattern, len(refusals), ""), exitNotApplied)
	}
	sub := &read.Sub{Re: re, Repl: repl}
	observed, problems := read.Run(diag, root, specs, read.Options{Sub: sub})
	skipNote()
	if len(sub.Refused) > 0 {
		for _, r := range sub.Refused {
			fmt.Fprintf(diag, "==> %s\n", r)
		}
		return nil, cli.Exit(fmt.Sprintf("incomplete: %d path(s) cannot be written in a plan, so nothing is planned", len(sub.Refused)), exitNotApplied)
	}
	if sub.Sites == 0 {
		return nil, cli.Exit(fmt.Sprintf("incomplete: %d file(s) matched /%s/ and the substitution changed no line", len(specs), oneLine(pattern)), exitNotApplied)
	}
	return &builtSub{plan: []byte(sub.Plan.String()), observed: observed, sites: sub.Sites, files: sub.Files, problems: problems + len(refusals)}, nil
}

// subRead is `read --grep P --sub R`: the plan on stdout and nothing else,
// the count on stderr, and the planned lines recorded as read.
func subRead(cmd *cli.Command, pattern string, excludes, posArgs []string) error {
	root := cmd.Root().String("root")
	b, err := buildSub(root, posArgs, pattern, cmd.String("sub"), excludes, cmd.Bool("no-ignore"), os.Stderr)
	if err != nil {
		return err
	}
	out := bufio.NewWriter(os.Stdout)
	_, _ = out.Write(b.plan) // a failed write surfaces at Flush
	// The plan reaches the caller BEFORE anything is recorded (ADR-088).
	if err := out.Flush(); err != nil {
		return cli.Exit(fmt.Sprintf("the answer could not be written, so nothing was recorded: %v", err), exitUsage)
	}
	fmt.Fprintf(os.Stderr, "-- planned: %d site(s) in %d file(s)\n", b.sites, b.files)
	if toNullDevice(os.Stdout) {
		fmt.Fprintln(os.Stderr, "mrw: the answer went to the null device, so nothing was recorded: a write to these lines needs a read whose answer you see (ADR-133)")
	} else if err := seen.Record(root, b.observed); err != nil {
		return cli.Exit(err, exitUsage)
	}
	if b.problems > 0 {
		return cli.Exit(fmt.Sprintf("incomplete: %d range(s) could not be served", b.problems), exitNotApplied)
	}
	return nil
}
