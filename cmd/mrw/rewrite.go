package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// buildRewrite asks ast-grep for its own rewrite of pattern to repl in the
// files under posArgs and plans it (ADR-161), for `read --ast-grep --rewrite`
// and `write --ast-grep --rewrite` alike. What the walk could not serve goes to
// diag, never to the plan. Nothing is recorded here: that is the caller's,
// after the plan has reached its reader. The error is the exit: 2 for a usage
// mistake or a missing ast-grep, 1 when nothing was planned or a file's matches
// cannot be written as a plan.
func buildRewrite(root string, posArgs []string, pattern, repl string, excludes []string, noIgnore bool, diag io.Writer) (*builtSub, error) {
	posArgs, err := rangeFreePaths(root, posArgs, "--ast-grep")
	if err != nil {
		return nil, err
	}
	var skipped read.WalkSkipped
	specs, refusals, files, err := read.AstGrepRewrite(root, posArgs, pattern, repl, excludes, read.AstGrepOptions{NoIgnore: noIgnore, Skipped: &skipped})
	if errors.Is(err, read.ErrAstGrepMissing) {
		return nil, cli.Exit(err.Error(), exitUsage)
	}
	if err != nil {
		return nil, cli.Exit(err, exitUsage)
	}
	for _, p := range refusals {
		fmt.Fprintf(diag, "==> %s  REFUSED  %s\n", p.Path, p.Reason)
	}
	if note := read.SkipNote(skipped, "--no-ignore"); note != "" {
		fmt.Fprintln(diag, note)
	}
	if len(specs) == 0 {
		return nil, cli.Exit(noMatchLine(pattern, len(refusals), ""), exitNotApplied)
	}
	var rw read.Rewrite
	for _, sp := range specs { // sorted by path; a file AstGrep could not read has no entry and is a refusal above
		if f := files[sp.Path]; f != nil {
			rw.File(sp.Path, f.Data, f.Hits)
		}
	}
	if len(rw.Refused) > 0 {
		for _, r := range rw.Refused {
			fmt.Fprintf(diag, "==> %s\n", r)
		}
		return nil, cli.Exit(fmt.Sprintf("incomplete: %d file(s) cannot be rewritten as a plan, so nothing is planned", len(rw.Refused)), exitNotApplied)
	}
	if rw.Sites == 0 {
		return nil, cli.Exit(fmt.Sprintf("incomplete: ast-grep matched /%s/ in %d file(s) and the rewrite changed no line", oneLine(pattern), len(specs)), exitNotApplied)
	}
	return &builtSub{plan: []byte(rw.Plan.String()), observed: rw.Observed, sites: rw.Sites, files: rw.Files, problems: len(refusals)}, nil
}

// rewriteRead is `read --ast-grep P --rewrite R`: the plan on stdout and
// nothing else, the count on stderr, and the planned spans recorded as read.
func rewriteRead(cmd *cli.Command, pattern string, excludes, posArgs []string) error {
	root := cmd.Root().String("root")
	b, err := buildRewrite(root, posArgs, pattern, cmd.String("rewrite"), excludes, cmd.Bool("no-ignore"), os.Stderr)
	if err != nil {
		return err
	}
	return servePlan(root, b)
}
