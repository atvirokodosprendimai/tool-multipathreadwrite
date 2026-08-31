// Command mrw reads and writes many file ranges in one call.
//
// It fills the gap between the two edit primitives an agent already has: Edit
// applies one replacement per call and fails loudly on a bad anchor; Write
// replaces a whole file and cannot tell you which of its intended changes did
// not land. mrw batches N changes across M files with a per-hunk verdict, and
// refuses to write anything unless every hunk passes.
//
//	mrw read internal/plan/plan.go:1-40 internal/apply/apply.go:/func Apply/,/^}/
//	mrw write plan.mrw
//	mrw write --dry-run --json plan.mrw
//
// Exit status is 0 when everything asked for succeeded, 1 when any hunk failed
// or any range could not be served, and 2 for a usage or I/O error. Nothing is
// written on a non-zero write status.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// Exit statuses. They are distinguished because the caller's next move differs:
// nothing-written means fix the plan and retry, a failed check means read the
// test output, and a usage error means neither.
//
// ⚠ exitCheckFailed says a check RAN AND FAILED; it says nothing about whether
// anything was written. After `write --check` the edit IS applied and now
// unverified; after a bare `check` the tree was never touched. Both are 3
// because both mean "go read the output", and neither is exitNotApplied, which
// promises an untouched tree.
const (
	exitNotApplied  = 1
	exitUsage       = 2
	exitCheckFailed = 3
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

// rootCommand builds the CLI. It is a function rather than inline in main so a
// test can run the real command — the --version wiring is otherwise reachable
// only by launching the process.
func rootCommand() *cli.Command {
	return &cli.Command{
		Name:    "mrw",
		Usage:   "read and write many file ranges in one call",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"C"},
				Value:   ".",
				Usage:   "resolve every path relative to `DIR`",
			},
		},
		Commands: []*cli.Command{readCmd(), writeCmd(), checkCmd(), iterCmd()},
	}
}

func main() {
	if err := rootCommand().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "mrw:", err)
		os.Exit(exitCode(err))
	}
}

func readCmd() *cli.Command {
	return &cli.Command{
		Name:      "read",
		Usage:     "print line ranges from one or more files",
		ArgsUsage: "PATH[:RANGE[,RANGE...]] ...",
		Description: `A RANGE is 3-6, 5, 3- (to end of file), -20 (from the start),
/pattern/ (every matching line, with -C context) or /start/,/end/.
Overlapping ranges are merged, so no line is printed — or paid for — twice.

Ranges print as "@@ 3-6", which is exactly the address a write plan takes.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "stat",
				Usage: "print only each file's length, size and sha — no content",
			},
			&cli.BoolFlag{
				Name:    "no-numbers",
				Aliases: []string{"N"},
				Usage:   "omit line numbers (they are the addresses a plan uses, so keep them unless piping)",
			},
			&cli.IntFlag{
				Name:    "context",
				Aliases: []string{"C"},
				Usage:   "lines of context either side of a single-pattern match",
			},
			&cli.IntFlag{
				Name:  "max-lines",
				Usage: "stop after `N` lines per file; whatever is withheld is always reported",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			set, err := iter.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			args := cmd.Args().Slice()
			if len(args) == 0 {
				// No arguments means "what I am working on": the working set
				// was emitted once and costs nothing to name again.
				if len(set.Entries) == 0 {
					return cli.Exit("read needs a PATH[:RANGE] or @N, or a working set (see: mrw iter add)", exitUsage)
				}
				args = set.Entries
				if set.Note != "" {
					fmt.Printf("# iteration: %s\n", set.Note)
				}
			} else if args, err = set.ResolveAll(args); err != nil {
				return cli.Exit(err, exitUsage)
			}
			specs := make([]read.Spec, 0, len(args))
			for _, a := range args {
				sp, err := read.ParseSpec(a)
				if err != nil {
					return cli.Exit(err, 2)
				}
				specs = append(specs, sp)
			}
			out := bufio.NewWriter(os.Stdout)
			defer out.Flush()

			problems := read.Run(out, root, specs, read.Options{
				Numbers:  !cmd.Bool("no-numbers"),
				Stat:     cmd.Bool("stat"),
				Context:  cmd.Int("context"),
				MaxLines: cmd.Int("max-lines"),
			})
			if problems > 0 {
				out.Flush()
				return cli.Exit(fmt.Sprintf("%d range(s) could not be served", problems), 1)
			}
			return nil
		},
	}
}

func writeCmd() *cli.Command {
	return &cli.Command{
		Name:      "write",
		Usage:     "apply an edit plan across one or more files, all or nothing",
		ArgsUsage: "[PLAN|-]",
		Description: `A plan is a sequence of hunks:

  @@ <path> <addr> <op> [sha=… lines=… anchor=… body=…]
  <body lines>

Ops are replace, insert-after, insert-before, delete and create. Addresses are
1-based and inclusive, and every one of them resolves against the ORIGINAL
file — so several hunks in one file need no offset arithmetic.

The optional guards are what make a batch safe to trust: sha= pins the whole
file, lines= asserts how many lines the range covers, anchor= requires a
substring in the range's first line. If any hunk fails, every hunk is reported
and NOTHING is written.

Prefer authoring the plan with your harness's own file tool and passing its
path, rather than piping it in: a plan on disk is a reviewable artifact and is
visible to whatever hooks watch file writes.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "validate and report, write nothing",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "emit the receipt as JSON (for hooks and quality gates)",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "print only failures and the summary line",
			},
			&cli.BoolFlag{
				Name:  "check",
				Usage: "after a successful write, run the project's check scoped to the files it touched",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args().Slice()
			if len(args) > 1 {
				return cli.Exit("write takes at most one plan file", exitUsage)
			}
			src, name := os.Stdin, "<stdin>"
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return cli.Exit(err, 2)
				}
				defer f.Close()
				src, name = f, args[0]
			}

			hunks, err := plan.Parse(src)
			if err != nil {
				return cli.Exit(fmt.Sprintf("%s: %v", name, err), 2)
			}

			root := cmd.Root().String("root")
			set, err := iter.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}

			in := make([]apply.Input, 0, len(hunks))
			for _, h := range hunks {
				// A hunk's path may be a pointer into the working set. It must
				// resolve to exactly one file: "@1-3 replace" would apply the
				// same body to three files, which is never what was meant.
				path := h.Path
				if iter.IsPointer(path) {
					got, err := set.Resolve(path)
					if err != nil {
						return cli.Exit(fmt.Sprintf("%s line %d: %v", name, h.SrcLine, err), exitUsage)
					}
					if len(got) != 1 {
						return cli.Exit(fmt.Sprintf("%s line %d: %s names %d entries; a hunk needs exactly one",
							name, h.SrcLine, path, len(got)), exitUsage)
					}
					path = iter.Path(got[0])
				}
				in = append(in, apply.Input{
					Path: path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
					Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
					SrcLine: h.SrcLine, Index: h.Index,
				})
			}

			res, err := apply.Apply(root, in, cmd.Bool("dry-run"))
			if err != nil {
				return cli.Exit(err, exitUsage)
			}

			// The check runs only on a real, successful write: verifying a tree
			// the plan did not touch would attribute someone else's red suite
			// to this edit.
			receipt := receipt{Result: res}
			if cmd.Bool("check") && res.Applied && res.Failed == 0 {
				var written []string
				for _, f := range res.Files {
					if f.Written {
						written = append(written, f.Path)
					}
				}
				cfg, err := check.Load(root)
				if err != nil {
					return cli.Exit(err, exitUsage)
				}
				cr, err := check.Run(ctx, root, cfg, written)
				if err != nil {
					return cli.Exit(err, exitUsage)
				}
				receipt.Check = &cr
			}

			if cmd.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(receipt); err != nil {
					return cli.Exit(err, exitUsage)
				}
			} else {
				report(os.Stdout, res, cmd.Bool("quiet"))
				reportCheck(os.Stdout, receipt.Check)
			}
			switch {
			case res.Failed > 0:
				return cli.Exit(fmt.Sprintf("%d hunk(s) failed — nothing was written", res.Failed), exitNotApplied)
			case receipt.Check != nil && !receipt.Check.Ran:
				// The write stands. Say so first — the caller's tree changed
				// even though the verification never happened.
				return cli.Exit("the write applied but no check could run: "+receipt.Check.Skipped+
					" — declare one in .quality-harness.json", exitUsage)
			case receipt.Check != nil && !receipt.Check.OK():
				return cli.Exit("the write applied but the check did not pass — the tree is changed and unverified", exitCheckFailed)
			}
			return nil
		},
	}
}

// receipt is what one write produced: the edit and, when asked for, the
// verification of that edit. They travel together because the whole point of
// --check is that the change and the evidence for it are one result, not two
// round trips.
type receipt struct {
	apply.Result
	Check *check.Result `json:"check,omitempty"`
}

// iterCmd manages the working set: the files and ranges this piece of work is
// about. Write it once; every later read and check refers to it for free.
func iterCmd() *cli.Command {
	return &cli.Command{
		Name:      "iter",
		Usage:     "show or edit the working set (" + iter.File + ")",
		ArgsUsage: "[add|rm|clear|note] [SPEC...]",
		Description: `With no arguments, prints the working set.

  mrw iter add internal/apply/apply.go internal/apply/apply_test.go
  mrw iter add internal/read/read.go:120-180
  mrw iter note "scoped check wiring"
  mrw iter rm internal/read/read.go        # drops its ranged entries too

Entries are read specs, so "mrw read" with no arguments returns exactly these
ranges, and "mrw check" runs the project's check scoped to these files.`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			set, err := iter.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			args := cmd.Args().Slice()
			verb := ""
			if len(args) > 0 {
				verb, args = args[0], args[1:]
			}
			switch verb {
			case "":
			case "add":
				if len(args) == 0 {
					return cli.Exit("iter add needs at least one SPEC", exitUsage)
				}
				// Refuse the whole add if any path is missing. An unquoted spec
				// containing a space arrives as several arguments, and the
				// fragments look like plausible relative paths — so the only
				// thing that catches it is asking the filesystem.
				var missing []string
				for _, a := range args {
					if _, err := os.Stat(filepath.Join(root, iter.Path(a))); err != nil {
						missing = append(missing, a)
					}
				}
				if len(missing) > 0 {
					return cli.Exit(fmt.Sprintf("no such file: %s (quote a spec containing spaces)",
						strings.Join(missing, ", ")), exitUsage)
				}
				set.Add(args...)
			case "rm":
				if len(args) == 0 {
					return cli.Exit("iter rm needs at least one SPEC or PATH", exitUsage)
				}
				set.Remove(args...)
			case "clear":
				set = iter.Set{}
			case "note":
				set.Note = strings.Join(args, " ")
			default:
				return cli.Exit(fmt.Sprintf("unknown iter verb %q (want add, rm, clear or note)", verb), exitUsage)
			}
			if verb != "" {
				if err := iter.Save(root, set); err != nil {
					return cli.Exit(err, exitUsage)
				}
			}
			if set.Note != "" {
				fmt.Printf("# %s\n", set.Note)
			}
			// Printed with their pointers, because the number IS the address:
			// later calls say "@3" instead of repeating the path.
			for i, e := range set.Entries {
				fmt.Printf("@%-3d %s\n", i+1, e)
			}
			fmt.Printf("%d entr(ies), %d file(s)\n", len(set.Entries), len(set.Paths()))
			return nil
		},
	}
}

// checkCmd runs the project's own verification, scoped to the working set, with
// no arguments at all. It is the read-side twin of `write --check`.
func checkCmd() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "run the project's check, scoped to the working set or to the given paths",
		ArgsUsage: "[PATH...]",
		Description: `The command comes from .quality-harness.json ("check" for the whole project,
"scoped_check" for a narrow run, with {packages} and {files} expanded). With no
such file, a Go project falls back to "go test ./..." — and the output says the
command was INFERRED, because an inferred check can be red on a tree you never
touched, which is a finding about the machine and not about your change.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "emit the result as JSON"},
			&cli.BoolFlag{Name: "full", Usage: "run the whole-project check, ignoring any scope"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			paths := cmd.Args().Slice()
			if len(paths) == 0 && !cmd.Bool("full") {
				set, err := iter.Load(root)
				if err != nil {
					return cli.Exit(err, exitUsage)
				}
				paths = set.Paths()
			}
			if cmd.Bool("full") {
				paths = nil
			}
			cfg, err := check.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			res, err := check.Run(ctx, root, cfg, paths)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			if cmd.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(res); err != nil {
					return cli.Exit(err, exitUsage)
				}
			} else {
				reportCheck(os.Stdout, &res)
			}
			if !res.Ran {
				// Not exit 3: nothing ran, so nothing failed. Reporting this as
				// a failing check would tell the caller to go read output that
				// does not exist.
				return cli.Exit("no check could run: "+res.Skipped+
					" — declare one in .quality-harness.json", exitUsage)
			}
			if !res.OK() {
				return cli.Exit("check did not pass", exitCheckFailed)
			}
			return nil
		},
	}
}

// reportCheck prints a check result. It always states the command and the real
// exit code: a summary that omits either invites the reader to trust a green
// that was never computed.
func reportCheck(w *os.File, r *check.Result) {
	if r == nil {
		return
	}
	out := bufio.NewWriter(w)
	defer out.Flush()

	if !r.Ran {
		fmt.Fprintf(out, "check SKIPPED: %s\n", r.Skipped)
		return
	}
	origin := "inferred"
	if r.Declared {
		origin = "declared"
	}
	fmt.Fprintf(out, "check (%s): %s\n", origin, r.Command)
	if r.Truncated > 0 {
		fmt.Fprintf(out, "... %d earlier line(s) in %s\n", r.Truncated, r.OutputFile)
	}
	for _, l := range r.Tail {
		fmt.Fprintf(out, "  | %s\n", l)
	}
	verdict := "PASS"
	if !r.OK() {
		verdict = "FAIL"
	}
	if r.Skipped != "" {
		fmt.Fprintf(out, "check %s (exit %d, %dms) — %s\n", verdict, r.ExitCode, r.DurationMS, r.Skipped)
		return
	}
	fmt.Fprintf(out, "check %s (exit %d, %dms) — full output: %s\n", verdict, r.ExitCode, r.DurationMS, r.OutputFile)
}

// report renders the receipt for a human. It is terse on purpose: the caller is
// usually an agent paying for every line it reads back.
func report(w *os.File, res apply.Result, quiet bool) {
	out := bufio.NewWriter(w)
	defer out.Flush()

	for _, h := range res.Hunks {
		switch {
		case h.Status == apply.StatusFailed:
			fmt.Fprintf(out, "FAIL %s %s %s (plan line %d): %s\n", h.Path, h.Addr, h.Op, h.SrcLine, h.Reason)
		case quiet:
		case h.Status == apply.StatusSkipped:
			fmt.Fprintf(out, "skip %s %s %s\n", h.Path, h.Addr, h.Op)
		default:
			fmt.Fprintf(out, "ok   %s %s %s  -%d +%d\n", h.Path, h.Addr, h.Op, h.Removed, h.Added)
		}
	}
	if !quiet {
		for _, f := range res.Files {
			if !f.Written {
				continue
			}
			verb := "wrote"
			if f.Created {
				verb = "created"
			}
			fmt.Fprintf(out, "%s %s  %dL -> %dL  sha %s\n", verb, f.Path, f.LinesFrom, f.LinesTo, short(f.SHAAfter))
		}
	}

	state := "applied"
	switch {
	case res.Failed > 0:
		state = "NOTHING WRITTEN"
	case res.DryRun:
		state = "dry run, nothing written"
	}
	fmt.Fprintf(out, "%d hunk(s), %d file(s), %d failed — %s\n",
		len(res.Hunks), len(res.Files), res.Failed, state)
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// exitCode pulls the status out of a cli.ExitCoder, defaulting to 2 so an
// unexpected error is never mistaken for "some hunks failed" — the two mean
// different things to a caller deciding whether to retry.
func exitCode(err error) int {
	var ec cli.ExitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return 2
}
