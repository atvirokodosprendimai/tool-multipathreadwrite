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
	"runtime/debug"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
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

// version is stamped at build time with -ldflags "-X main.version=...". A
// LOCAL build gets no stamp and reports "dev", which is why versionString
// exists: two local builds a day apart are otherwise indistinguishable, and a
// stale one silently kept the read-before-modify bypass alive on this machine.
var version = "dev"

// versionString reports the version plus the commit the binary was BUILT from.
//
// Go records vcs.revision, vcs.time and vcs.modified in every build made inside
// a git checkout — no -ldflags needed, and the release stamp is unaffected. mrw
// simply never read them, so `mrw --version` said "dev" whether the binary was
// minutes or a day old. That is not cosmetic: a day-old ./bin/mrw still had the
// bypass ADR-005 exists to prevent, and nothing it printed could say so.
//
// Falls back to the bare version when the information is absent — a binary
// built from a tarball, or with -buildvcs=false, is not a liar, it is just
// unstamped.
func versionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	return versionFrom(version, info.Settings)
}

// versionFrom is the composition, split out because it is the half that can be
// TESTED. A test binary carries no vcs.revision — `go test` does not stamp one
// — so a test that read the real build info could only ever SKIP, which is a
// row that never runs dressed as a row that passes. This one takes the settings
// as an argument and always executes.
func versionFrom(v string, settings []debug.BuildSetting) string {
	var rev, dirty string
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = ", dirty"
			}
		}
	}
	if rev == "" {
		return v
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	return fmt.Sprintf("%s (%s%s)", v, rev, dirty)
}

// rootCommand builds the CLI. It is a function rather than inline in main so a
// test can run the real command — the --version wiring is otherwise reachable
// only by launching the process.
func rootCommand() *cli.Command {
	return &cli.Command{
		Name:    "mrw",
		Usage:   "read and write many file ranges in one call",
		Version: versionString(),
		// The framework's default handler PRINTS the error and calls os.Exit
		// itself, which made main's own reporting dead code — the "mrw:" prefix
		// never appeared, and no test could drive a failing command without
		// taking the test binary down with it. Errors come back to main, which
		// already prints them and maps the status.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "root",
				Aliases: []string{"C"},
				Value:   ".",
				Usage:   "resolve every path relative to `DIR`",
			},
		},
		// An unknown subcommand is a USAGE error. Without this the framework
		// falls through to its help machinery, which returns "No help topic
		// for 'x'" carrying exit 3 — the status this tool documents as "a check
		// ran and did not pass". A hook branching on 3 would read a typo'd
		// command as a landed write with a red suite, which is the most
		// expensive possible misreading (probed 2026-09-01).
		// A ledger this mrw will not trust is announced for the same reason
		// the migration is: the consequence is a refusal ("has not been read")
		// on a file the caller believes they read, and a refusal nobody can
		// account for looks like a bug rather than like the guard working.
		//
		// In Before, not in main, because main runs before --root is parsed —
		// the ledger that matters is the one for the root being operated on,
		// and checking "." reported on whichever directory mrw was launched
		// from. That read the wrong ledger under `-C`, which is precisely how
		// a contract row here first passed for the wrong reason.
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if stale, err := seen.IsStale(cmd.String("root")); err == nil && stale {
				fmt.Fprintln(os.Stderr, seen.StaleNotice)
			}
			return ctx, nil
		},
		CommandNotFound: func(_ context.Context, cmd *cli.Command, name string) {
			names := make([]string, 0, len(cmd.Commands))
			for _, c := range cmd.Commands {
				names = append(names, c.Name)
			}
			cmd.Metadata = map[string]any{"notFound": cli.Exit(
				fmt.Sprintf("unknown command %q (want %s)", name, strings.Join(names, ", ")), exitUsage)}
		},
		Commands: []*cli.Command{readCmd(), writeCmd(), checkCmd(), iterCmd(), seenCmd()},
	}
}

func main() {
	// One-time, additive migration of any pre-ADR-004 in-tree state. Announced
	// on stderr because a tool that quietly moves your files is the sibling of
	// the tool that quietly created them.
	if moved, err := state.Migrate("."); err == nil && len(moved) > 0 {
		if dir, err := state.Dir("."); err == nil {
			fmt.Fprintf(os.Stderr, "mrw: moved %s from ./%s/ to %s — the copy in your working tree is "+
				"untouched and can now be deleted\n", strings.Join(moved, " and "), state.LegacyDir, dir)
		}
	}

	root := rootCommand()
	err := root.Run(context.Background(), os.Args)
	// An unknown subcommand cannot be reported by returning an error: the
	// framework's CommandNotFound hook returns nothing, so it leaves its
	// verdict on the command and main is what turns that into a status.
	if err == nil {
		if notFound, ok := root.Metadata["notFound"].(error); ok {
			err = notFound
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mrw:", err)
		os.Exit(exitCode(err))
	}
}

// seenCmd answers "where is my state, and what does mrw think it saw" — the
// inspectability an in-tree file gave away for free, bought back deliberately.
func seenCmd() *cli.Command {
	return &cli.Command{
		Name:  "seen",
		Usage: "print the state directory and the read-before-modify ledger",
		Description: `mrw keeps per-checkout state OUTSIDE the working tree, under
$XDG_STATE_HOME/mrw/<key>/ (or ~/.local/state/mrw/<key>/). This prints where that
is for the current root, and what the ledger currently holds.

A path listed here is one mrw has seen; one that is not listed must be read
before it can be edited. A sha that disagrees with the file on disk means it
changed since mrw last looked, and the next write to it is refused.`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			dir, err := state.Dir(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			// Location first: "where is it" is the question that brought you here.
			fmt.Println(dir)

			path, err := seen.ReadPath(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			if path != filepath.Join(dir, seen.Name) {
				fmt.Printf("# reading a legacy in-tree ledger: %s\n", path)
			}
			ledger, err := seen.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			paths := make([]string, 0, len(ledger))
			for p := range ledger {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			for _, p := range paths {
				fmt.Printf("%s  %-24s  %s\n", short(ledger[p].SHA), ledger[p].Served(), p)
			}
			fmt.Printf("%d file(s) seen\n", len(paths))
			return nil
		},
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
			// Flag domains are settled FIRST, before the working set is loaded
			// or a single spec is parsed. A usage error means "fix the call"
			// and has to preempt everything, or it preempts inconsistently:
			// placed after the parse, `read -C -1 'a.go:'` reported the bad
			// range and `--max-lines -5 @999` reported the pointer, so which
			// error the caller saw depended on which OTHER mistake they made.
			// iter.Load can also create state before a refusal. The identical
			// defect was fixed once already for `write --dry-run --check`; it
			// was reintroduced here (PR #13 review, Codex).
			//
			// A negative -C printed a REVERSED header — `@@ 5-3`, an address
			// mrw's own parser refuses — with no content at exit 0, against a
			// README that promises the header is exactly what a write plan
			// takes. A negative --max-lines was ignored outright.
			if n := cmd.Int("context"); n < 0 {
				return cli.Exit(fmt.Sprintf("-C %d: context cannot be negative", n), exitUsage)
			}
			if n := cmd.Int("max-lines"); n < 0 {
				return cli.Exit(fmt.Sprintf("--max-lines %d: a cap cannot be negative", n), exitUsage)
			}
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

			observed, problems := read.Run(out, root, specs, read.Options{
				Numbers:  !cmd.Bool("no-numbers"),
				Stat:     cmd.Bool("stat"),
				Context:  cmd.Int("context"),
				MaxLines: cmd.Int("max-lines"),
			})
			// Reading a file is how mrw learns what it holds; recording that is
			// what lets a later write know whether its picture is still current.
			if err := seen.Record(root, observed); err != nil {
				return cli.Exit(err, exitUsage)
			}
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
				Name: "force",
				Usage: "edit files mrw has not read, or that changed since it last saw them " +
					"(the escape hatch, not the habit)",
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
			// Flag contradictions are settled BEFORE the plan is read. A usage
			// error means "fix the call", and it has to preempt everything or
			// it preempts inconsistently: with the check below the parse, an
			// unparseable plan won against this pair while a plan whose HUNK
			// failed lost to it — the caller was told about their flags and
			// never learned their address was out of range, and exit 1, which
			// promises an untouched tree, became exit 2. Both are "your plan is
			// wrong" and they ranked differently only because of where the test
			// sat (PR #11 review, F3).
			if cmd.Bool("check") && cmd.Bool("dry-run") {
				return cli.Exit("--check cannot run under --dry-run: nothing is written, so there is "+
					"nothing to verify. Drop one of the two — a check that did not run is not a pass", exitUsage)
			}
			if len(args) > 1 {
				return cli.Exit("write takes at most one plan file", exitUsage)
			}
			src, name := os.Stdin, "<stdin>"
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					// -C moves the paths INSIDE the plan; the plan file itself
					// is a shell argument like any other and resolves against
					// the working directory. That split is defensible and
					// invisible, so a miss says where it looked rather than
					// leaving the caller to doubt the plan instead of the path.
					return cli.Exit(planOpenError(args[0], cmd.String("root"), err), exitUsage)
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

			ledger, err := seen.Load(root)
			if err != nil {
				return cli.Exit(err, exitUsage)
			}
			res, err := apply.Apply(root, in, apply.Options{
				DryRun: cmd.Bool("dry-run"),
				Seen:   ledger,
				Force:  cmd.Bool("force"),
			})
			if err != nil {
				// ADR-001 rule 3: every hunk carries its own verdict, and a
				// filesystem failure is not an exception. Apply now fills the
				// receipt before returning the error, so render it on whichever
				// surface the caller asked for — a --json caller that got a
				// bare "mrw: …: permission denied" had nothing to parse — and
				// only then exit. The exit code is unchanged: a filesystem
				// failure stays 2, distinct from a failing hunk's 1.
				if len(res.Hunks) > 0 {
					if cmd.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						_ = enc.Encode(receipt{Result: res})
					} else {
						report(os.Stdout, res, cmd.Bool("quiet"))
					}
				}
				return cli.Exit(err, exitUsage)
			}

			// The check runs only on a real, successful write: verifying a tree
			// the plan did not touch would attribute someone else's red suite
			// to this edit.
			// Record what the files now hold. This is why a chain of edits needs
			// no re-read between steps — mrw knows what it just produced — while
			// a change made behind its back still leaves the ledger disagreeing
			// with the disk, and the next write refused.
			if res.Applied {
				// A file mrw just wrote is one it knows WHOLLY: it produced
				// every line, so the observation covers the whole file and a
				// chain of edits needs no re-read between steps.
				wrote := map[string]seen.Observation{}
				for _, f := range res.Files {
					if f.Written {
						wrote[f.Path] = seen.Observation{SHA: f.SHAAfter}
					}
				}
				if err := seen.Record(root, wrote); err != nil {
					return cli.Exit(err, exitUsage)
				}
			}

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
		Usage:     "show or edit the working set (stored outside the tree; see `mrw seen`)",
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
				// Refuse the whole add if any path is missing OR leaves the
				// root. An unquoted spec containing a space arrives as several
				// arguments, and the fragments look like plausible relative
				// paths — so the only thing that catches it is asking the
				// filesystem.
				//
				// rooted.Resolve, not filepath.Join: the working set is the
				// FOURTH way into the tree, after read, write and check, and it
				// was the one that did not enforce the boundary. Join cleans
				// `../outside/x` into a path that exists, so the entry was
				// accepted — and every later `mrw check`, which scopes to the
				// set, then refused at exit 2 until someone removed it. The
				// same Join re-rooted an ABSOLUTE path onto the root, where it
				// did not exist, so those were refused as "no such file": the
				// right answer for the wrong reason, which is why the two
				// spellings disagreed.
				var missing, outside []string
				for _, a := range args {
					full, err := rooted.Resolve(root, iter.Path(a))
					if err != nil {
						outside = append(outside, a)
						continue
					}
					if _, err := os.Stat(full); err != nil {
						missing = append(missing, a)
					}
				}
				if len(outside) > 0 {
					return cli.Exit(fmt.Sprintf("outside the root: %s — the working set feeds "+
						"`mrw read` and `mrw check`, which refuse a path that leaves `--root`, so an "+
						"entry like this could never be served", strings.Join(outside, ", ")), exitUsage)
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
			// A delete says what it removed, and only a delete: it is the
			// only op that CONSUMES A RANGE and can be written without a
			// body. Two ops consume one, and replace refuses an empty body;
			// create and the insertions consume no range to report bounds
			// for. So a delete's plan may carry no account of what it took,
			// and the receipt is then the caller's only account of it, while
			// a replace's body is their own account of those lines — which
			// is why `from "" to ""` elsewhere would be noise on every line
			// of every receipt (ADR-008).
			//
			// The criterion is the PLAN's body over a CONSUMED RANGE, and
			// neither half is spare. Four rewrites of this comment each
			// replaced the previous overstatement and asserted something new
			// that did not survive being run — the third claiming a replace
			// receipt prints the caller's body, when it prints `-2 +2`, the
			// bare count the same sentence had just called delete's alone;
			// the fourth claiming delete is the only bodyless op, when
			// `create` with an empty body exits 0 and writes a zero-byte
			// file. Check a rewrite against what it newly asserts, on all
			// five ops, not against the claim it replaces.
			// Keyed on the op, not on the strings being non-empty: a delete
			// that removes blank lines removed something, and a receipt that
			// went quiet about it would be indistinguishable from one where
			// nothing recorded the bounds at all.
			bounds := ""
			if h.Op == "delete" {
				bounds = fmt.Sprintf(" from %q to %q", h.RemovedFirst, h.RemovedLast)
			}
			fmt.Fprintf(out, "ok   %s %s %s  -%d +%d%s\n", h.Path, h.Addr, h.Op, h.Removed, h.Added, bounds)
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

// planOpenError explains a plan file that could not be opened. A relative path
// resolves against the working directory even when --root points elsewhere, so
// the message names the directory it looked in and says so — the difference
// between "your plan is wrong" and "your plan is somewhere else" is the whole
// of what the caller needs.
//
// It returns an error rather than a string, like every other failure in this
// file: a caller that wants to wrap or inspect it should not have to reach
// back through cli.Exit's conversion to do so. The original is wrapped, so
// errors.Is still sees fs.ErrNotExist.
func planOpenError(path, root string, err error) error {
	if rooted.IsRooted(path) {
		return err
	}
	wd, wdErr := os.Getwd()
	if wdErr != nil {
		return err
	}
	absRoot, rootErr := filepath.Abs(root)
	if rootErr != nil || absRoot == wd {
		return err
	}
	return fmt.Errorf("%w — looked in the working directory %s, because --root (%s) moves the paths "+
		"INSIDE a plan and not the plan file itself", err, wd, absRoot)
}
