package read

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// BuildError is why a plan could not be made from a walk (ADR-161). Usage says
// the caller's request was malformed (the CLI's exit 2); otherwise nothing was
// planned, or a path cannot be carried in a plan (exit 1).
type BuildError struct {
	Msg   string
	Usage bool
}

func (e *BuildError) Error() string { return e.Msg }

func usageError(format string, a ...any) error {
	return &BuildError{Msg: fmt.Sprintf(format, a...), Usage: true}
}

func notApplied(format string, a ...any) error {
	return &BuildError{Msg: fmt.Sprintf(format, a...)}
}

// Built is a plan made from a walk and a replacement (ADR-161), for the CLI's
// `read --sub`, `write --grep --sub` and their --rewrite twins, and for the MCP
// surface's mrw_read and mrw_write alike.
type Built struct {
	Plan     []byte
	Observed map[string]seen.Observation // the planned lines, which are all the plan licenses
	Sites    int
	Files    int
	Problems int // paths refused and ranges not served: a partial plan
	Skipped  WalkSkipped
}

// BuildSub walks posArgs for pattern and plans repl over the lines it finds.
// What the walk could not serve goes to diag, never to the plan. Nothing is
// recorded here: that is the caller's, after the plan has reached its reader.
// The error is a *BuildError.
func BuildSub(root string, posArgs []string, pattern, repl string, excludes []string, noIgnore bool, diag io.Writer) (*Built, error) {
	// The plan's body is lines, and a replacement that makes more than one
	// would turn a one-line hunk into several the header does not count.
	if strings.ContainsAny(repl, "\r\n") {
		return nil, usageError("--sub REPLACEMENT cannot hold a line break: it rewrites one line, as sed does; a change across lines is a plan")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, usageError("--grep %q: %v", pattern, err)
	}
	posArgs, err = RangeFreePaths(root, posArgs, "--grep")
	if err != nil {
		return nil, err
	}
	var skipped WalkSkipped
	specs, refusals, err := Walk(root, posArgs, WalkOptions{Pattern: re, Exclude: excludes, NoIgnore: noIgnore, Skipped: &skipped})
	if err != nil {
		return nil, usageError("%v", err)
	}
	for _, p := range refusals {
		fmt.Fprintf(diag, "==> %s  REFUSED  %s\n", p.Path, p.Reason)
	}
	skipNote := func() {
		if note := SkipNote(skipped, "--no-ignore"); note != "" {
			fmt.Fprintln(diag, note)
		}
	}
	if len(specs) == 0 {
		skipNote()
		return nil, notApplied("%s", NoMatchLine(pattern, len(refusals), ""))
	}
	sub := &Sub{Re: re, Repl: repl}
	observed, problems := Run(diag, root, specs, Options{Sub: sub})
	skipNote()
	if len(sub.Refused) > 0 {
		for _, r := range sub.Refused {
			fmt.Fprintf(diag, "==> %s\n", r)
		}
		return nil, notApplied("incomplete: %d path(s) cannot be written in a plan, so nothing is planned", len(sub.Refused))
	}
	if sub.Sites == 0 {
		return nil, notApplied("incomplete: %d file(s) matched /%s/ and the substitution changed no line", len(specs), OneLine(pattern))
	}
	return &Built{Plan: []byte(sub.Plan.String()), Observed: observed, Sites: sub.Sites, Files: sub.Files, Problems: problems + len(refusals), Skipped: skipped}, nil
}

// BuildRewrite asks ast-grep for its own rewrite of pattern to repl in the
// files under posArgs and plans it (ADR-161). What the walk could not serve
// goes to diag, never to the plan. Nothing is recorded here. The error is a
// *BuildError: a usage mistake or a missing ast-grep is Usage; nothing planned,
// or matches that cannot be written as a plan, is not.
func BuildRewrite(root string, posArgs []string, pattern, repl string, excludes []string, noIgnore bool, diag io.Writer) (*Built, error) {
	posArgs, err := RangeFreePaths(root, posArgs, "--ast-grep")
	if err != nil {
		return nil, err
	}
	var skipped WalkSkipped
	specs, refusals, files, err := AstGrepRewrite(root, posArgs, pattern, repl, excludes, AstGrepOptions{NoIgnore: noIgnore, Skipped: &skipped})
	if errors.Is(err, ErrAstGrepMissing) {
		return nil, usageError("%s", err.Error())
	}
	if err != nil {
		return nil, usageError("%v", err)
	}
	for _, p := range refusals {
		fmt.Fprintf(diag, "==> %s  REFUSED  %s\n", p.Path, p.Reason)
	}
	if note := SkipNote(skipped, "--no-ignore"); note != "" {
		fmt.Fprintln(diag, note)
	}
	if len(specs) == 0 {
		return nil, notApplied("%s", NoMatchLine(pattern, len(refusals), ""))
	}
	var rw Rewrite
	for _, sp := range specs { // sorted by path; a file AstGrep could not read has no entry and is a refusal above
		if f := files[sp.Path]; f != nil {
			rw.File(sp.Path, f.Data, f.Hits)
		}
	}
	if len(rw.Refused) > 0 {
		for _, r := range rw.Refused {
			fmt.Fprintf(diag, "==> %s\n", r)
		}
		return nil, notApplied("incomplete: %d file(s) cannot be rewritten as a plan, so nothing is planned", len(rw.Refused))
	}
	if rw.Sites == 0 {
		return nil, notApplied("incomplete: ast-grep matched /%s/ in %d file(s) and the rewrite changed no line", OneLine(pattern), len(specs))
	}
	return &Built{Plan: []byte(rw.Plan.String()), Observed: rw.Observed, Sites: rw.Sites, Files: rw.Files, Problems: len(refusals), Skipped: skipped}, nil
}

// RangeFreePaths resolves the working-set pointers in posArgs and refuses a
// range: a finder (flag) and a range are two answers to one question. The error
// is a usage *BuildError.
func RangeFreePaths(root string, posArgs []string, flag string) ([]string, error) {
	if len(posArgs) > 0 {
		set, err := iter.Load(root)
		if err != nil {
			return nil, usageError("%v", err)
		}
		if posArgs, err = set.ResolveAll(posArgs); err != nil {
			return nil, usageError("%v", err)
		}
	}
	for _, a := range posArgs {
		sp, err := ParseSpec(a)
		if err != nil {
			return nil, usageError("%v", err)
		}
		if len(sp.Ranges) > 0 {
			return nil, usageError("%s: a range and %s are two answers to one question", a, flag)
		}
	}
	return posArgs, nil
}

// NoMatchLine is the last line of a read whose search found no file (ADR-146).
// It says `no match:` when nothing the caller asked for was refused, and
// `incomplete:` when part of the search was never made because a named path
// was. The pattern is shown on one line, and an ADR-140 hint, when there is
// one, ends the line in both kinds.
func NoMatchLine(pattern string, refused int, hint string) string {
	found := fmt.Sprintf("no file matched /%s/", OneLine(pattern))
	if refused > 0 {
		return fmt.Sprintf("incomplete: %s, and %d path(s) were refused (see the REFUSED lines)%s", found, refused, hint)
	}
	return "no match: " + found + hint
}

// OneLine spells a pattern's line breaks, so the kind that begins the last line
// is on the last line a caller reads.
func OneLine(s string) string {
	return strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(s)
}
