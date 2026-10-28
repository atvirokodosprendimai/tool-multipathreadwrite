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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/preview"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/speclist"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/statehint"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

// Exit statuses. They are distinguished because the caller's next move differs:
// nothing-written means fix the plan and retry, a failed check means read the
// test output, and a usage error means neither.
//
// ⚠ exitCheckFailed says a check did not verify the tree: it ran and failed,
// timed out, or was interrupted — before it started, too (ADR-080). It says
// nothing about whether anything was written. After `write --check` the edit IS
// applied and now unverified; after a bare `check` the tree was never touched.
// Both are 3 because both mean "the tree is unverified", and neither is
// exitNotApplied, which promises an untouched tree.
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
	cmd := &cli.Command{
		Name:    "mrw",
		Usage:   "find, read and write many file ranges in one call",
		Version: versionString(),

		// ADR-168 Decision 4: the first line an agent reads tells it to load the skill.
		CustomRootCommandHelpTemplate: agentsHelp + cli.RootCommandHelpTemplate,

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
			// The ledger names the mrw that writes it (ADR-150).
			seen.SetWriter(versionString())
			// version, instructions and skill touch no state at all (ADR-145,
			// ADR-168); the legacy migration runs for every other command, stats included.
			verb := cmd.Args().First()
			if verb != "version" && verb != "instructions" && verb != "skill" {
				migrateLegacyState()
			}
			// version, instructions, skill and stats never read the ledger (ADR-136,
			// ADR-168), and help — asked for by name, or a bare mrw — uses none (ADR-157).
			switch verb {
			case "version", "instructions", "skill", "stats", "help", "h", "":
				return ctx, nil
			}
			// The seen subcommand's flags are parsed before this runs (urfave/cli
			// runs the chain's Before after the leaf's parse), so its parsed --forget
			// is the answer: argv misreads --forget=, --forget --forget=false and
			// -- --forget.
			forgetting := false
			if sub := cmd.Command("seen"); verb == "seen" && sub != nil {
				forgetting = sub.Bool("forget")
			}
			// Looking up the ledger below makes the root's state directory, so a
			// forget over a root that is not there, or not a directory, is refused
			// before it: exit 2, no state directory (ADR-175). A hook's `--root null`,
			// from an input with no cwd, lands here. main prints "mrw: " itself.
			if forgetting {
				if err := forgettableRoot(cmd.String("root")); err != nil {
					return ctx, cli.Exit("seen --forget: "+err.Error(), exitUsage)
				}
			}
			// A newer mrw's ledger is not stale: it is left alone, and the caller
			// is told which binary wrote it (ADR-150).
			if writer, newer, err := seen.Newer(cmd.String("root")); err == nil && newer {
				// A write says it once, in a refusal or after its landing (ADR-157);
				// so does `seen --forget`, whose refusal carries the notice (ADR-175).
				if verb != "write" && !forgetting {
					fmt.Fprintln(os.Stderr, seen.NewerNotice(writer))
				}
			} else if stale, err := seen.IsStale(cmd.String("root")); err == nil && stale {
				fmt.Fprintln(os.Stderr, seen.StaleNotice)
			} else if tellsLedgerDamage(cmd.Args().First()) {
				if notice, err := seen.DamageNotice(cmd.String("root")); err == nil && notice != "" {
					fmt.Fprintln(os.Stderr, notice)
				}
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
		Commands: []*cli.Command{readCmd(), writeCmd(), checkCmd(), runCmd(), iterCmd(), seenCmd(), statsCmd(), mcpCmd(), instructionsCmd(), skillCmd(), versionCmd()},
	}
	installUsageErrors(cmd)
	return cmd
}

// installUsageErrors gives cmd and every subcommand under it one answer to a
// flag the parser rejects: the error, naming the help to read, returned to main
// for stderr — and nothing on stdout. urfave/cli printed the command's whole
// help to stdout, and stdout is `mrw mcp`'s protocol stream: `mrw mcp --bogus`
// in a host's config put help text where the host reads JSON-RPC (ADR-078).
// `--help` itself is unchanged and still prints to stdout.
func installUsageErrors(cmd *cli.Command) {
	cmd.OnUsageError = usageError
	for _, sub := range cmd.Commands {
		installUsageErrors(sub)
	}
}

// usageError is every command's answer to a flag its parser rejects. A flag
// spelled like one of the command's own subcommands — `mrw --instructions` —
// names that subcommand, since the caller most likely met `mrw --version` and
// guessed the shape (ADR-097); every other usage error is worded as before.
func usageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	if flag, sub := subcommandForFlag(cmd, err); sub != "" {
		// ADR-168: a bare --skill IS the skill's spelling, so what went wrong
		// here is the form — one dash, a value, padding — and the refusal says so.
		if sub == "skill" && cmd.Root() == cmd {
			return cli.Exit("only a bare --skill prints the skill — no value, no single dash: write `mrw --skill` or `mrw skill` (see: mrw --help)", exitUsage)
		}
		return cli.Exit(fmt.Sprintf("--%s is not a flag; the subcommand is `%s %s` (see: %s --help)",
			flag, cmd.FullName(), sub, cmd.FullName()), exitUsage)
	}
	// urfave words every refused flag with one dash, `-root`, which is how Go's
	// flag package spells a long flag it parsed and is not what the caller typed
	// or what mrw's help prints; a name of more than one letter is given its
	// second dash back (the v1.71.0 field test: three peers read -root as a
	// different flag). A one-letter flag, -z, is spelled with one.
	text := err.Error()
	if name := refusedFlag(err); len([]rune(name)) > 1 {
		text = "flag provided but not defined: --" + name
	}
	msg := fmt.Sprintf("%s (see: %s --help)", text, cmd.FullName())
	if h := rootFlagHint(cmd, refusedFlag(err)); h != "" {
		msg += "; " + h
	} else if h := foreignFlagHint(cmd.FullName(), refusedFlag(err)); h != "" {
		msg += "; " + h
	}
	// Decision 7 of ADR-177: mrw run --json answers a refusal, this one included,
	// with one {"error": …} document. The parser may stop before it has set
	// --json, so the raw arguments are read too.
	if cmd.Name == "run" && runJSONRequested(cmd.Root().Args().Slice()) {
		emitRunJSON(checkRefusal{Error: msg})
	}
	return cli.Exit(msg, exitUsage)
}

// rootFlagHint is the sentence for a flag the application defines on the root
// command, typed after a subcommand that refused it, or "" (ADR-165 T1). urfave
// makes a root flag persistent, so a subcommand refuses one only when its own
// flags clash with one of the root flag's names — read's -C (context) and
// --root's alias -C — and then urfave skips the root flag whole (flagKinds).
// The flag's own definition supplies its value placeholder, so a root flag
// added later is pointed home with no table to keep.
// applicationFlag says whether f is one mrw defines, not one urfave adds to
// every root command: its help and version flags, whose -h and -v would
// otherwise be pointed home over ADR-164's -v sentence. urfave copies them
// onto the command, so they are known by name, not by identity.
func applicationFlag(f cli.Flag) bool {
	for _, n := range f.Names() {
		if n == "help" || n == "version" {
			return false
		}
	}
	return true
}

func rootFlagHint(cmd *cli.Command, name string) string {
	root := cmd.Root()
	if name == "" || root == cmd {
		return ""
	}
	for _, f := range root.Flags {
		if !applicationFlag(f) || !slices.Contains(f.Names(), name) {
			continue
		}
		long := f.Names()[0]
		value := ""
		if df, ok := f.(cli.DocGenerationFlag); ok && df.TakesValue() {
			value = " VALUE"
			// The placeholder is what the usage text backticks, as urfave
			// renders it in --help: "resolve every path relative to `DIR`".
			if _, rest, ok := strings.Cut(df.GetUsage(), "`"); ok {
				if ph, _, ok := strings.Cut(rest, "`"); ok && ph != "" {
					value = " " + ph
				}
			}
		}
		where := strings.TrimPrefix(cmd.FullName(), root.Name+" ")
		return fmt.Sprintf("--%s goes before the subcommand: %s --%s%s %s …", long, root.Name, long, value, where)
	}
	return ""
}

// subcommandForFlag returns the flag name the parser refused, as the caller
// typed it, and the name of cmd's own subcommand it exactly names, or two empty
// strings. urfave/cli v3.11.0 builds that refusal as plain fmt.Errorf text
// (command_parse.go:208, :219) and reads its own message back by this prefix
// (flagFromError, :14-23), so the text is the only handle; a reworded message
// after an upgrade falls through to the old wording, never a wrong one.
func subcommandForFlag(cmd *cli.Command, err error) (flag, sub string) {
	name, ok := strings.CutPrefix(err.Error(), "flag provided but not defined: -")
	if !ok {
		return "", ""
	}
	child := cmd.Command(name)
	if child == nil {
		return "", ""
	}
	return name, child.Name
}

// declareAdvice ends a no-check-could-run exit with the fix for it: declare a
// check — unless the check could not start for want of a shell, where declaring
// one changes nothing and the report already names what to install (ADR-082).
func declareAdvice(skipped string) string {
	if strings.Contains(skipped, check.NoShell) {
		return ""
	}
	return " — declare one in .quality-harness.json"
}

// stoppedBefore says how a check that did not run was stopped before it
// started, in the words the exit message carries (ADR-080, ADR-126).
func stoppedBefore(r check.Result) string {
	if r.Skipped == check.TimedOutBeforeStart {
		return check.TimedOutBeforeStart
	}
	return "was interrupted before it started"
}

// hasSinglePattern reports whether any spec carries a single /pattern/ range —
// the only range -C widens (read.Options.Context).
func hasSinglePattern(specs []read.Spec) bool {
	for _, sp := range specs {
		for _, r := range sp.Ranges {
			if r.Re != nil && r.ReEnd == nil {
				return true
			}
		}
	}
	return false
}

// cmdExeForm is a refusal's fix spelled for cmd.exe, which keeps single quotes
// as literal characters, so pasting `mrw read -- 'x '` there names a path with
// quotes in it (a Windows cmd.exe session, 2026-09-25). PowerShell and Git Bash
// take the form already printed. Empty outside Windows (ADR-078). It quotes
// val itself: rewriting every ' in the POSIX form turned a name's own
// apostrophe into a quote (the review of #239). A Windows name holds no ".
func cmdExeForm(head, val string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	return " (in cmd.exe: " + head + `"` + val + `")`
}

// posixQuote single-quotes s for a POSIX shell, closing and reopening the quote
// around each ' it holds, so a name like ` O'Brien` survives the paste.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func main() {
	root := rootCommand()
	// ADR-168: a bare --skill is the subcommand's spelling, read as `skill`
	// before any other argv check, so every guard below judges `mrw skill …`.
	argv := skillSpelling(root, os.Args)
	// ADR-161: a lone - moves last, or the parser drops every flag after it.
	args, err := liftLoneDash(root, argv)
	if err != nil {
		args = argv // the refusal is reported below; nothing is run
	}
	// A padded ATTACHED flag value (--root='dir ') is trimmed by the parser
	// before any Action runs, and a root flag never reaches a subcommand's raw
	// tail, so the whole argv is checked here (ADR-069 T5).
	if err == nil {
		err = refusePaddedFlagValues(root, args[1:])
	}
	if v, ok := shortRoot(root, args[1:]); ok {
		noteShortRoot(root, v)
	}
	if err == nil {
		err = root.Run(context.Background(), args)
	}
	// An unknown subcommand cannot be reported by returning an error: the
	// framework's CommandNotFound hook returns nothing, so it leaves its
	// verdict on the command and main is what turns that into a status.
	if err == nil {
		if notFound, ok := root.Metadata["notFound"].(error); ok {
			err = notFound
		}
	}
	if err != nil {
		// A command's own exit (mrw run) carries no message: its receipt is the report.
		if msg := statehint.Hint(err); msg.Error() != "" {
			fmt.Fprintln(os.Stderr, "mrw:", msg)
		}
		os.Exit(exitCode(err))
	}
}

// seenCmd answers "where is my state, and what does mrw think it saw" — the
// mcpCmd starts a Model Context Protocol server on stdio, so an agent that
// speaks MCP reaches the same engine as the CLI without shell access.
//
// ADR-010. The binary is unchanged by this: every subcommand, exit status, plan
// format and ledger entry means what it meant before, and the server is one
// more caller of the same functions rather than a second engine.
func mcpCmd() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "serve mrw over the Model Context Protocol on stdio",
		Description: `Speaks newline-delimited JSON-RPC 2.0 on stdin and stdout, which is
what an MCP host connects to. There is no socket and nothing listens on a port.

Add it to a host's config as the command ` + "`mrw`" + ` with the arguments ` + "`mcp --enable`" + `. The
server shares this checkout's ledger with the CLI, so a file read over MCP may
be edited from a shell and the reverse.

Stdout carries MCP messages and nothing else — diagnostics go to stderr — so
this subcommand is not useful to run by hand.

The tools are OFF until enabled (ADR-170): started without ` + "`--enable`" + ` or MRW_MCP=on this
server lists no tools and refuses every call, because a host with a shell does
better with the CLI. A host with no shell registers ` + "`mrw mcp --enable`" + `, or sets
MRW_MCP=on in the server's environment, and restarts the server.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "enable",
				Usage: "serve the tools mrw_read and mrw_write; without it, or MRW_MCP=on, the server lists none and refuses every call. `--enable=false` turns them off whatever MRW_MCP says, which is read only when the flag is absent",
			},
			&cli.IntFlag{
				Name:  "max-result-chars",
				Value: mcp.DefaultMaxResultChars,
				Usage: "produce no tool result larger than `N` characters, advertised to the host in _meta and enforced on BOTH tools. Zero means zero: omit the flag to take the default. Also settable as MRW_MAX_RESULT_CHARS, which the flag overrides",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			// A host's config names the command `mrw` and the one argument
			// `mcp`; anything after it is a mistake the server would otherwise
			// start past, serving a root nobody questioned (ADR-078).
			if cmd.Args().Len() > 0 {
				return cli.Exit(fmt.Sprintf("mrw mcp takes no arguments, and got %q", cmd.Args().Slice()), exitUsage)
			}
			// A host launches this without --root and expects the project it is
			// working in. The working directory it happens to inherit is not
			// that, so the environment the host sets is consulted first.
			// IsSet, not != ".": `--root .` typed on purpose and `--root`
			// omitted both arrive as "." from the parser, and only IsSet tells
			// them apart. Without it a deliberate `--root .` loses to the
			// host's environment — measured, and the same lesson this file
			// already learned for --grep/--files-from below.
			explicitRoot := ""
			if cmd.Root().IsSet("root") {
				explicitRoot = cmd.Root().String("root")
			}
			root, src := mcp.ResolveRoot(explicitRoot, os.LookupEnv)
			// ADR-018: a root nobody named, that cannot be a project, is
			// refused BEFORE the server starts rather than served correctly.
			// Exit 2 is the usage status: the fix is an argument.
			if err := mcp.CheckRoot(root, src); err != nil {
				return cli.Exit("mrw mcp: "+err.Error(), exitUsage)
			}
			// The ceiling this server advertises and enforces (ADR-032). It
			// was one host's number compiled in; a host with a larger budget
			// could not use it and a host with a smaller one was not
			// protected. Resolved before the announcement, so a bad value is a
			// usage error rather than a line claiming the server started.
			budget, err := resultBudget(cmd, os.LookupEnv)
			if err != nil {
				return cli.Exit("mrw mcp: "+err.Error(), exitUsage)
			}
			mcp.MaxResultChars = budget
			// ADR-170: resolved with the budget, so a bad registration still
			// fails at start with the switch off.
			mcp.ToolsOff = !mcpEnabled(cmd, os.LookupEnv, os.Stderr)
			// Announced on STDERR: the spec forbids anything on stdout that is
			// not an MCP message, and a server silently bound to the wrong tree
			// is this subcommand's worst failure — every refusal it then gives
			// is correct and about a repository nobody asked about.
			if abs, err := filepath.Abs(root); err == nil {
				fmt.Fprintf(os.Stderr, "mrw mcp: serving %s (from %s)\n", abs, src)
			} else {
				fmt.Fprintf(os.Stderr, "mrw mcp: serving %s (from %s)\n", root, src)
			}
			// The wire reports the same version the binary does, so a host
			// showing "mrw 0.0.14" is never talking to a different build.
			mcp.Version = versionString()
			if err := mcp.Serve(os.Stdin, os.Stdout, root); err != nil {
				return exitWith(err)
			}
			return nil
		},
	}
}

// mcpEnabled resolves the ADR-170 switch: --enable decides when it is given
// (so `--enable=false` beats MRW_MCP=on), else MRW_MCP=on turns the tools on;
// off, empty, unset and any other value leave them off, and only another value
// is named on warn, verbatim: the raw value is compared, not a trimmed one. The
// variable is read only when the flag is absent, as
func mcpEnabled(cmd *cli.Command, lookup func(string) (string, bool), warn io.Writer) bool {
	if cmd.IsSet("enable") {
		return cmd.Bool("enable")
	}
	raw, _ := lookup("MRW_MCP")
	switch raw {
	case "on":
		return true
	case "", "off":
	default:
		fmt.Fprintf(warn, "mrw mcp: MRW_MCP=%q is not one of on or off; the tools stay off\n", raw)
	}
	return false
}

// resultBudget resolves the MCP result ceiling: the flag, else the
// environment, else the default.
//
// ⚠ IsSet, not a test for zero. `--max-result-chars 0` is a caller asking for a
// server that may return nothing, and only IsSet tells that from an absent
// flag — the distinction ADR-033 settled for --max-lines, inherited here rather
// than argued again.
//
// A host config is the natural place for this and a host config sets an
// environment, so the variable is not a convenience: for some hosts it is the
// only way to say it.
func resultBudget(cmd *cli.Command, lookup func(string) (string, bool)) (int, error) {
	if cmd.IsSet("max-result-chars") {
		n := cmd.Int("max-result-chars")
		if n < 0 {
			return 0, fmt.Errorf("--max-result-chars %d: a ceiling cannot be negative", n)
		}
		return n, nil
	}
	if raw, ok := lookup("MRW_MAX_RESULT_CHARS"); ok && strings.TrimSpace(raw) != "" {
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || n < 0 {
			return 0, fmt.Errorf("MRW_MAX_RESULT_CHARS=%q: want a whole number of characters, zero or more", raw)
		}
		return n, nil
	}
	return mcp.DefaultMaxResultChars, nil
}

// versionCmd prints the same version string `-v` and `--version` print.
func versionCmd() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print the version string (-v / --version already do this)",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "check-latest",
				Usage: "also ask GitHub for the latest release and say whether this mrw is older: exit 1 if a newer " +
					"release exists, 2 if it could not ask; the only network call mrw makes, and only when asked (ADR-149)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 0 {
				return cli.Exit("version takes no arguments", exitUsage)
			}
			out := cmd.Root().Writer
			if out == nil {
				out = os.Stdout
			}
			if _, err := fmt.Fprintln(out, versionString()); err != nil {
				return err
			}
			if !cmd.Bool("check-latest") {
				return nil
			}
			return checkLatest(ctx, out)
		},
	}
}

// instructionsCmd prints the contract a caller with only this binary is entitled to.
func instructionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "instructions",
		Usage: "print the contract a caller with only this binary is entitled to; --core prints the eight rules that matter most",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "core", Usage: "print the eight rules that matter most instead of the whole contract (ADR-141)"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 0 {
				return cli.Exit("instructions takes no arguments", exitUsage)
			}
			out := cmd.Root().Writer
			if out == nil {
				out = os.Stdout
			}
			if cmd.Bool("core") {
				_, err := fmt.Fprint(out, guide.Core())
				return err
			}
			_, err := fmt.Fprint(out, guide.CLI())
			return err
		},
	}
}

// statsCmd prints what became of the plans this checkout has been given.
//
// ADR-009: mrw's plan format is bespoke and nothing measured whether a caller
// can author it. This is the reader for that measurement — the tally itself is
// written by `write`, and holds counts only.
func statsCmd() *cli.Command {
	return &cli.Command{
		Name:  "stats",
		Usage: "print what became of the plans this checkout has been given",
		Description: `Counts only — no plan text, no paths, no anchors, and nothing
is ever transmitted. The tally lives beside the ledger in the state directory
that ` + "`mrw seen`" + ` names.

A rate is always printed with its denominator: a percentage without its sample
size is the form that gets quoted out of the population it was measured on.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "emit the same numbers as JSON"},
			&cli.BoolFlag{Name: "reset", Usage: "empty the tally, reporting how many records were discarded"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			// The reset reads what it discards under the hold that discards it:
			// read before, a plan recorded in between was discarded unreported
			// (the review of #240).
			if cmd.Bool("reset") {
				discarded, err := authoring.Reset(root)
				if err != nil {
					return exitWith(err)
				}
				// Say what was discarded. A reset that reports nothing is
				// indistinguishable from a reset that did nothing.
				fmt.Printf("tally reset: %d plan(s) discarded\n", discarded.Plans())
				return nil
			}
			t, err := authoring.Load(root)
			if err != nil {
				return exitWith(err)
			}
			if cmd.Bool("json") {
				// ADR-054: every vocabulary key is present, zero included, so
				// a script never has to know that an absent key means zero;
				// and the derived pair is emitted so it is not re-derived
				// with a different denominator.
				counts := map[string]int{}
				for _, name := range authoring.Vocabulary() {
					counts[name] = t[name]
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false) // <, > and & stay themselves: ==> is not a different header to a raw reader (#496)
				return enc.Encode(statsReceipt{Plans: t.Plans(), Counts: counts, Landed: t.Landed(), Failed: t["failed_check"], Pricing: authoring.LoadPricing(root)})
			}
			total := t.Plans()
			if total == 0 {
				// NOT "0%". An empty measurement that reads like a good result
				// is the silent-success shape this project refuses.
				fmt.Println("no plans recorded yet — this says nothing has been MEASURED, not that nothing has failed")
				return nil
			}
			// The whole vocabulary, not t.Names(): a name at zero is the one
			// a reader most needs to see, because its absence reads as "that
			// never happens" rather than "that was never measured" (ADR-054).
			for _, name := range authoring.Vocabulary() {
				n := t[name]
				fmt.Printf("  %-17s %4d of %d plan(s) (%.1f%%)\n", name, n, total, 100*float64(n)/float64(total))
			}
			landed, failed := t.Landed(), t["failed_check"]
			if landed > 0 {
				fmt.Printf("\nlanded writes: %d; failed_check %d of those (%.1f%%). Landed = applied + partially_applied + failed_check + check_not_run:\n"+
					"the tree changed. It is not \"wrote and was checked\" — --no-check and prose-only plans count as applied, and a partial commit ran no check.\n",
					landed, failed, 100*float64(failed)/float64(landed))
			}
			// ADR-055: the window, always, and the pattern line when it holds.
			recent := authoring.Recent(root)
			fmt.Printf("\nrecent: %d write(s) in the window (last %d landed writes; op class and advisory count only)\n", len(recent), authoring.RecentWindow)
			if line := patternLine(recent); line != "" {
				fmt.Println(line)
			}
			// ADR-056: the pricing block, with the bar it is measured against,
			// so the number and the criterion are read together.
			pr := authoring.LoadPricing(root)
			fmt.Printf("\nstrict-balance pricing: %d landed writes with a single-line code replace; the flag would have refused %d — broke %d, held %d, unchecked %d\n",
				pr.Candidates, pr.WouldRefuse, pr.Broke, pr.Held, pr.Unchecked)
			if rate, ok := pr.FalsePositiveRate(); ok {
				fmt.Printf("  false positives %d of %d checked (%.1f%%) — pre-registered bar: under 5%% in every corpus, 50 refusals total (BACKLOG)\n",
					pr.Held, pr.Broke+pr.Held, 100*rate)
			} else {
				fmt.Printf("  no checked refusals yet — pre-registered bar: under 5%% false positives in every corpus, 50 refusals total (BACKLOG)\n")
			}
			fmt.Printf("\n%d plan(s) recorded in this checkout. The rate is valid for THIS population;\n"+
				"a number from one repository and one family of callers is not a general one.\n", total)
			return nil
		},
	}
}

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
changed since mrw last looked, and the next write to it is refused.

--prune removes the state directories whose ` + "`root`" + ` marker names a checkout that
is no longer there, and names each one it removed. It NEVER removes an entry
whose marker is missing, unreadable or not an absolute path — mrw did not write
those, so it does not know what they are — nor the entry for the root you are
running in. --prune --dry-run prints the same list and removes nothing.

--forget empties the read ledger of this checkout, read and written files alike, so
that a write needs a fresh read of the lines it touches. A host hook runs it after
it compacts its context (docs/agents.md); it costs every session in the checkout a
re-read, and it is refused, exit 2, over a ledger a newer mrw wrote.

There is no automatic reaper and there will not be one: a path that is gone may
be a deleted checkout or a volume that is not mounted, and only you can tell.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "prune",
				Usage: "remove the state directories whose checkout no longer exists, naming each",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "with --prune: print what would be removed, and remove nothing",
			},
			&cli.BoolFlag{
				Name:  "forget",
				Usage: "empty this checkout's read ledger: every later write needs a fresh read",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			root := cmd.Root().String("root")
			if cmd.Bool("forget") {
				if cmd.Bool("prune") || cmd.Bool("dry-run") {
					return cli.Exit("mrw seen: --forget empties the ledger and takes neither --prune nor --dry-run", exitUsage)
				}
				return forget(root)
			}
			if cmd.Bool("prune") {
				return prune(root, cmd.Bool("dry-run"))
			}
			// A dry run of the thing that only reads is not a request anyone
			// can mean, so it is a usage error rather than a silent no-op.
			if cmd.Bool("dry-run") {
				return cli.Exit("mrw seen: --dry-run says what --prune would remove, "+
					"and there is no --prune here; add it, or drop --dry-run", exitUsage)
			}
			dir, err := seen.Dir(root)
			if err != nil {
				return exitWith(err)
			}
			// Location first: "where is it" is the question that brought you here.
			fmt.Println(dir)
			// ADR-161 Decision 14: the ledger is the checkout's, the tally and the
			// working set stay this root's own; below a checkout's top they differ.
			if own, err := state.Dir(root); err == nil && own != dir {
				fmt.Printf("# this root's tally and working set: %s\n", own)
			}
			// ⚠ AFTER THE DIRECTORY, NEVER BEFORE IT. `mrw seen | head -1` is
			// the documented way to find the state directory and contract §54
			// reads it that way, so this line goes second or it breaks them.
			// It is a bare count and not a survey, and state.Count is what
			// makes that true. It used to call state.Entries, which opens
			// every entry, reads every marker and STATS EVERY CHECKOUT those
			// markers name — 22,836 of them on the machine that motivated
			// ADR-034, some against unmounted or network paths, on a command
			// whose job is to print one number. The comment here described the
			// cheap behaviour while the line below it did the expensive one.
			if n, err := state.Count(); err == nil {
				fmt.Printf("# %d state director%s under %s\n",
					n, plural(n, "y", "ies"), filepath.Dir(dir))
				if n > 1 {
					fmt.Println("# `mrw seen --prune` removes the ones whose checkout is gone")
				}
			}
			// ADR-157: which mrw wrote the ledger, and whether this one uses it.
			if h, err := seen.Header(root); err == nil && h.Exists {
				switch {
				case h.Newer:
					fmt.Printf("# ledger written by a newer mrw (%s), format v%s: left untouched, licenses nothing\n", h.Writer, h.Format)
				case h.Older:
					fmt.Println("# ledger in an older format: not used; mrw replaces it the next time it records a read or a write")
				default:
					fmt.Printf("# ledger format v%s, written by %s\n", h.Format, h.Writer)
				}
			}

			path, err := seen.ReadPath(root)
			if err != nil {
				return exitWith(err)
			}
			if path != filepath.Join(dir, seen.Name) {
				fmt.Printf("# reading a legacy in-tree ledger: %s\n", path)
			}
			// ADR-079: read under the ledger's lock, as a writer's snapshot is, so
			// `mrw seen` never prints a ledger another process emptied to rewrite.
			ledger, err := seen.Snapshot(root)
			if err != nil {
				return exitWith(err)
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
	// jsonPaths is --json-path (ADR-178): each use kept whole, in command-line order.
	jsonPaths := &jsonPathValue{}
	return &cli.Command{
		Name: "read",
		// ADR-099: urfave adds a `help` subcommand (alias `h`) to every command,
		// and it won over a file named help, even after `--`. The --help flag
		// stays; `mrw help` stays on the root.
		HideHelpCommand: true,
		Usage:           "print line ranges, pattern matches or --grep hits from one or more files",
		ArgsUsage:       "PATH[:RANGE[,RANGE...]] ...",
		Description: `A RANGE is 3-6, 5, 3- (to end of file), -20 (from the start),
A,+N (the line A plus the N lines after it, so 12,+2 is lines 12 through 14),
/pattern/ (every matching line, with -C context) or /start/,/end/.
A relative end takes a single start: 5-7,+3 and a bare ,+3 are refused.
A read clamps a relative end at the last line; a write refuses one past it.
Overlapping ranges are merged, so no line is printed — or paid for — twice.
A ranged read also serves, and licenses, the line after each range that ends before
the file does (ADR-161): f.txt:2 prints lines 2 and 3.

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
				Usage: "stop after `N` lines per SPEC, where zero means zero: omit the flag to ask for no cap. Two specs naming one file get two budgets. Whatever is withheld is always reported",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "print one JSON document: content, the read's standard output exactly as without --json, and receipt, what mrw_read sends as content[1] for the same read (ADR-165)",
			},
			&cli.IntFlag{
				Name:  "max-chars",
				Usage: "write at most `N` characters to standard output, in whole lines: what does not fit is not served, not licensed, and listed on standard error, and the read exits 1 incomplete. Zero means zero; omit it for no budget (ADR-165)",
			},
			&cli.IntFlag{
				Name:  "max-cols",
				Usage: "cut a line longer than `N` characters to a window of N centred on the first match of a pattern (the start otherwise), holding the whole match when it fits, marked `[cols A-B of L]`. A line served cut is NOT recorded as read: a plan that replaces it is refused until it is read whole (ADR-137). Zero or omitted serves lines whole",
			},
			&cli.IntFlag{
				Name:  "max-matches",
				Usage: "with --grep or --ast-grep, show matching lines across all the files in walk order until `N` have been shown (a match inside a span already shown counts as shown, so more than N can print) and count the rest: the answer ends `-- max-matches N reached: M more matching line(s) in K file(s) not shown` and the read exits 1 incomplete when some were left; a match inside a span already shown counts as shown, and only what was shown is licensed (ADR-179)",
			},
			&cli.BoolFlag{
				Name:  "no-comments",
				Usage: "in a .go file, serve a run of two or more whole-line comment lines (// or /* */) as `-- … N comment line(s), A-B` and do not record them as read; trailing comments, //go: and //line directives and the cgo preamble stay, and a file that is not .go (or standard input) is served whole and named in one line on standard error (ADR-179)",
			},
			&cli.BoolFlag{
				Name:  "outline",
				Usage: "print each file's structure instead of its lines: a .go file's top-level func, type, var and const declarations, a .md file's headings outside fenced blocks, each as `A-B text` with the lines it spans; any other file is named in one line on standard error and the read exits 1. It licenses nothing: the file is hashed, no line recorded as read (ADR-179)",
			},
			&cli.BoolFlag{
				Name:  "fold",
				Usage: "print a run of two or more non-blank lines that are identical, or identical but for digits, as its first line and `-- ×N: lines A-B like A`; the folded lines are not shown, so not licensed: a write to one is refused until it is read without --fold (ADR-179)",
			},
			&cli.BoolFlag{
				Name:  "count",
				Usage: "with --grep, print each matching file's header and how many of its lines match, then a total — no line is served, so it licenses no write (ADR-153)",
			},
			&cli.StringFlag{
				Name:  "grep",
				Usage: "serve every range matching `PATTERN` in the files under the given paths (a directory is walked)",
			},
			&cli.StringFlag{
				Name:  "sub",
				Usage: "with --grep, print a PLAN instead of lines: each matching line rewritten by `REPLACEMENT` (Go regexp: $1 and ${name} expand, $$ is a literal dollar, and a reference to a group the pattern does not define is refused), one replace hunk per line that changes, and license exactly those lines. Pipe it to `mrw write -`, or use `mrw write --grep P --sub R`. Not with -C, --count, --stat, --max-cols, --max-lines, -N, --files-from or --ast-grep (ADR-161)",
			},
			&cli.StringFlag{
				Name:  "ast-grep",
				Usage: "serve every range the ast-grep binary reports for `PATTERN` in the files under the given paths (a directory is walked). The binary must be on PATH. A hit in a file whose lines end in \\r alone is reported, not served (ADR-065)",
			},
			&cli.StringFlag{
				Name:  "rewrite",
				Usage: "with --ast-grep, print a PLAN instead of lines: ast-grep's own rewrite of each match to `REPLACEMENT` (its $A metavariables expand; an empty one deletes the match), one replace hunk per span of changed lines, anchored on the span's first line, and license each span and the line after it. Pipe it to `mrw write -`, or use `mrw write --ast-grep P --rewrite R`. Matches that overlap are refused. Not with --grep, --sub, -C, --count, --stat, --quiet, --max-cols, --max-lines, -N, --files-from or a range (ADR-161)",
			},
			&cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "skip paths matching `GLOB`, matched against the root-relative path AND the basename; under --grep a bare directory name met below where the walk starts prunes that whole subtree, while a path you name is walked; --ast-grep drops excluded hits after the binary runs (repeatable)",
			},
			&cli.StringFlag{
				Name:  "files-from",
				Usage: "read specs from `FILE`, or from stdin when it is -",
			},
			&cli.BoolFlag{
				Name:  "no-ignore",
				Usage: "under --grep or --ast-grep, walk every regular file: no .gitignore, no binary skip, nested repositories entered (ADR-116, ADR-122, ADR-130)",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "license what the read serves and print none of it: each file's header and `-- licensed N line(s) in M file(s)`, counting what this read licensed (not what earlier reads did), for lines you already know. A read whose answer goes to the null device licenses nothing; this is the way to license without reading. Not with --count, --stat or --sub (ADR-161)",
			},
			&cli.StringSliceFlag{
				Name:  "if-served",
				Usage: "`PATH=SHA`, repeatable, one per file: answer each spec of PATH by `-- same as served: lines A-B at sha SHA` instead of its text when the file still has that sha and the ledger reports every line of the spec shown to the caller in that version, recording nothing. Opt-in, for a session alone in its checkout whose host empties the ledger on start, resume, clear and compaction (mrw seen --forget) (ADR-179)",
			},
			&cli.StringSliceFlag{
				Name:  "if-changed",
				Usage: "`PATH=SHA`, repeatable, one per file: answer each spec of PATH with one line, `==> PATH  unchanged since sha SHA — N lines, B bytes not served`, instead of its text when the file still has that sha (8 to 64 hex digits of its sha256, the 8 a header prints are enough), and serve it as ever when it changed. An unchanged answer records nothing, so it licenses no write; a range it would not have served is still reported. The flag splits its value on commas, as --exclude does, so a PATH holding one cannot be guarded (ADR-169)",
			},
			jsonPathFlag(jsonPaths),
		},
		Action: func(_ context.Context, cmd *cli.Command) (retErr error) {
			// ADR-165: under --json a refusal before the answer is one document,
			// {"error": …} on standard output, as check --json refuses (ADR-100);
			// the exit code and the stderr line are the plain read's. printed is
			// set once the {"content", "receipt"} document is out.
			printed := false
			if cmd.Bool("json") {
				defer func() {
					if retErr != nil && !printed {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						enc.SetEscapeHTML(false)
						_ = enc.Encode(readRefusal{Error: retErr.Error()})
					}
				}()
			}
			if err := refusePaddedArgs(cmd); err != nil {
				return err
			}
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
			if cmd.IsSet("max-chars") && cmd.Int("max-chars") < 0 {
				return cli.Exit(fmt.Sprintf("--max-chars %d: a budget cannot be negative", cmd.Int("max-chars")), exitUsage)
			}
			if cmd.IsSet("max-chars") && cmd.Bool("stat") {
				return cli.Exit("--stat serves no line, so --max-chars has nothing to bound: drop --max-chars", exitUsage)
			}
			if n := cmd.Int("max-lines"); n < 0 {
				return cli.Exit(fmt.Sprintf("--max-lines %d: a cap cannot be negative", n), exitUsage)
			}
			if n := cmd.Int("max-cols"); n < 0 {
				return cli.Exit(fmt.Sprintf("--max-cols %d: a width cannot be negative", n), exitUsage)
			}
			// ADR-007: --grep and --files-from are two further ways of arriving
			// at the same []Spec read.Run has always served. Their domains are
			// settled HERE, with the other flag domains, because every one of
			// them is a usage error and those preempt everything — the reason
			// is the comment above, and it applies identically.
			// IsSet, not != "": precedence must depend on whether a flag was
			// SUPPLIED, not on whether its value happens to be non-empty.
			// Testing the value let `--grep "" --files-from F` through as a
			// files-from read (the table says two sources is a usage error),
			// let `--files-from "" a.go` through the same way, and made
			// `--grep ""` read the working set instead of walking --root.
			// Found by an independent review, 2026-09-03.
			grepSet := cmd.IsSet("grep")
			astSet := cmd.IsSet("ast-grep")
			filesFromSet := cmd.IsSet("files-from")
			pattern := cmd.String("grep")
			astPattern := cmd.String("ast-grep")
			excludes := cmd.StringSlice("exclude")
			filesFrom := cmd.String("files-from")
			posArgs := cmd.Args().Slice()
			if grepSet {
				if msg := readFlagPatternRefusal(cmd.Flags, pattern); msg != "" {
					return cli.Exit(msg, exitUsage)
				}
			}

			// An empty value is accepted or refused explicitly rather than
			// silently meaning "not given". An empty PATTERN matches every
			// line of every file, which is a real request and not obviously a
			// mistake, so it is allowed; an empty --files-from names no file
			// and cannot be honoured.
			if filesFromSet && filesFrom == "" {
				return cli.Exit("--files-from needs a FILE or -", exitUsage)
			}

			// ADR-178: --json-path prints values and serves no line, so a flag
			// that serves, folds or licenses lines has nothing to act on beside
			// it. Said first and by name, before another flag's own table speaks
			// of it, and with the expressions parsed, so a mistake in either
			// preempts any read.
			var projections []read.Projection
			if cmd.IsSet("json-path") {
				for _, f := range []string{"grep", "ast-grep", "sub", "rewrite", "count", "stat", "quiet", "no-numbers", "context", "max-lines", "if-changed", "max-matches", "fold", "outline", "no-comments", "if-served"} {
					if cmd.IsSet(f) {
						shown := "--" + f
						switch f {
						case "context":
							shown = "-C"
						case "no-numbers":
							shown = "-N"
						}
						return cli.Exit(fmt.Sprintf("--json-path prints values and serves no line, so %s has nothing to act on, or serves lines of its own: drop %s", shown, shown), exitUsage)
					}
				}
				var perr error
				if projections, perr = read.ParseProjections(jsonPaths.vals); perr != nil {
					return cli.Exit("--json-path "+perr.Error(), exitUsage)
				}
			}

			if len(excludes) > 0 && !grepSet && !astSet {
				return cli.Exit("--exclude without --grep: there is nothing to exclude from", exitUsage)
			}
			if cmd.Bool("no-ignore") && !grepSet && !astSet {
				return cli.Exit("--no-ignore without --grep or --ast-grep: it changes what their walk skips", exitUsage)
			}
			if grepSet && astSet {
				return cli.Exit("--grep and --ast-grep are two sources of specs; use one", exitUsage)
			}
			if astSet && filesFromSet {
				return cli.Exit("--ast-grep and --files-from are two sources of specs; use one", exitUsage)
			}
			if grepSet && filesFromSet {
				return cli.Exit("--grep and --files-from are two sources of specs; use one", exitUsage)
			}
			if filesFromSet && len(posArgs) > 0 {
				return cli.Exit("--files-from and positional paths are two sources of specs; use one", exitUsage)
			}
			// ADR-078: an --ast-grep hit is served as the lines it spans, so -C
			// has nothing to widen. Said here, before the finder runs and by
			// name; the general refusal below told the caller to name a
			// /pattern/ (the review of #239).
			if astSet && cmd.IsSet("context") {
				return cli.Exit("-C widens a /pattern/ or --grep match, and an --ast-grep hit is served as the lines it spans: drop -C, or use --grep", exitUsage)
			}
			// ADR-161: --rewrite prints a plan of ast-grep's rewrite of the spans
			// --ast-grep finds, so it needs that finder and a flag that shapes served
			// lines has none. IsSet, not != "": an empty replacement is a real one.
			if cmd.IsSet("rewrite") {
				if !astSet {
					return cli.Exit("--rewrite is ast-grep's rewrite of the matches --ast-grep finds: give --ast-grep PATTERN", exitUsage)
				}
				for _, f := range []string{"sub", "count", "stat", "quiet", "max-cols", "max-lines", "max-chars", "no-numbers", "json"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--rewrite prints a plan and serves no line, so --%s has nothing to act on, or makes a plan of its own: drop --%s", f, f), exitUsage)
					}
				}
			}
			// ADR-161: --sub prints a plan of the lines a --grep matches, so it
			// needs the --grep and a flag that shapes served lines has none.
			if cmd.IsSet("sub") {
				if astSet {
					return cli.Exit("--sub rewrites the lines a --grep matches, and an --ast-grep hit is a span of lines: use --grep", exitUsage)
				}
				if !grepSet {
					return cli.Exit("--sub rewrites the lines a --grep matches: give --grep PATTERN", exitUsage)
				}
				for _, f := range []string{"context", "count", "stat", "quiet", "max-cols", "max-lines", "max-chars", "no-numbers", "files-from", "json"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--sub prints a plan and serves no line, so --%s has nothing to act on: drop --%s", f, f), exitUsage)
					}
				}
			}
			// ADR-153: --count answers a --grep with counts and serves no line,
			// so it needs a --grep, and a flag that acts on served lines has
			// nothing to act on beside it.
			if cmd.Bool("count") {
				if astSet {
					return cli.Exit("--count counts the lines a --grep pattern matches, and an --ast-grep hit is a span of lines: use --grep", exitUsage)
				}
				if !grepSet {
					return cli.Exit("--count counts the lines a --grep pattern matches: give --grep PATTERN", exitUsage)
				}
				for _, f := range []struct{ flag, refusal string }{
					{"stat", "--count's header already carries what --stat prints: drop --stat"},
					{"context", "--count serves no line, so -C has nothing to widen: drop -C"},
					{"max-lines", "--count serves no line, so --max-lines has nothing to cap: drop --max-lines"},
					{"max-chars", "--count serves no line, so --max-chars has nothing to bound: drop --max-chars"},
					{"max-cols", "--count serves no line, so --max-cols has nothing to cut: drop --max-cols"},
					{"no-numbers", "--count serves no line, so -N has no line numbers to drop: drop -N"},
				} {
					if cmd.IsSet(f.flag) {
						return cli.Exit(f.refusal, exitUsage)
					}
				}
			}
			// ADR-179: --max-matches caps what a finder shows across the whole
			// read, so it needs one and has nothing to cap beside a count or a plan.
			// --json-path's list above speaks first when both are given.
			if cmd.IsSet("max-matches") {
				if !grepSet && !astSet {
					return cli.Exit("--max-matches caps the matches a --grep or --ast-grep finds: give --grep PATTERN or --ast-grep PATTERN", exitUsage)
				}
				if cmd.Int("max-matches") < 0 {
					return cli.Exit("--max-matches needs a number of matches, zero or more", exitUsage)
				}
				for _, f := range []string{"count", "stat", "sub", "rewrite"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--max-matches caps the matches a read shows, and --%s shows none or makes a plan of its own: drop one", f), exitUsage)
					}
				}
			}
			// ADR-179: --fold shows a run as one line and licenses only that line,
			// so it has nothing to fold where a read serves no line, prints a plan,
			// counts what it shows, or budgets matches it would hide.
			if cmd.Bool("fold") {
				for _, f := range []string{"quiet", "stat", "count"} {
					if cmd.Bool(f) {
						return cli.Exit(fmt.Sprintf("--fold folds the lines a read serves, and --%s serves no line or licenses differently: drop one", f), exitUsage)
					}
				}
				for _, f := range []string{"sub", "rewrite", "max-matches"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--fold folds the lines a read serves, and --%s prints a plan or counts matches it would hide: drop one", f), exitUsage)
					}
				}
			}
			// ADR-179: --outline prints structure, not lines, so every flag that
			// serves, folds, elides, plans, counts or searches lines has nothing to
			// act on beside it (--json-path's list above speaks first when both are given).
			if cmd.Bool("outline") {
				for _, f := range []string{"quiet", "stat", "count", "fold", "no-comments"} {
					if cmd.Bool(f) {
						return cli.Exit(fmt.Sprintf("--outline prints a file's structure and serves no line, so --%s has nothing to act on: drop one", f), exitUsage)
					}
				}
				for _, f := range []string{"grep", "ast-grep", "sub", "rewrite", "context", "if-changed", "max-matches", "max-lines", "max-cols"} {
					if cmd.IsSet(f) {
						shown := "--" + f
						if f == "context" {
							shown = "-C"
						}
						return cli.Exit(fmt.Sprintf("--outline prints a file's structure and serves no line, so %s has nothing to act on: drop %s", shown, shown), exitUsage)
					}
				}
			}
			// ADR-179: --no-comments elides comment runs from the lines a read serves.
			if cmd.Bool("no-comments") {
				for _, f := range []string{"quiet", "stat", "count"} {
					if cmd.Bool(f) {
						return cli.Exit(fmt.Sprintf("--no-comments elides lines of the answer, and --%s serves no line or licenses differently: drop one", f), exitUsage)
					}
				}
				for _, f := range []string{"fold", "outline"} {
					if cmd.Bool(f) {
						return cli.Exit(fmt.Sprintf("--no-comments elides comment runs from the lines a read serves, and --%s changes those lines in its own way: drop one", f), exitUsage)
					}
				}
				for _, f := range []string{"sub", "rewrite", "max-matches"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--no-comments elides lines of the answer, and --%s prints a plan or counts matches it would hide: drop one", f), exitUsage)
					}
				}
			}
			// ADR-179: --if-served answers by reference the lines a read would serve,
			// so it has nothing to answer where a read serves no line, prints a plan,
			// changes the lines it serves or budgets matches it would not show.
			if cmd.IsSet("if-served") {
				for _, f := range []string{"quiet", "stat", "count", "fold", "no-comments", "outline"} {
					if cmd.Bool(f) {
						return cli.Exit(fmt.Sprintf("--if-served answers by reference the lines a read serves, and --%s serves no line or changes the lines it serves: drop one", f), exitUsage)
					}
				}
				for _, f := range []string{"sub", "rewrite"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--if-served answers by reference the lines a read serves, and --%s prints a plan of its own: drop one", f), exitUsage)
					}
				}
			}
			// ADR-161: --quiet licenses what a read serves; --count and --stat
			// serve no line and --sub licenses a plan's lines, so none of them
			// has anything for it to keep quiet.
			if cmd.Bool("quiet") {
				for _, f := range []string{"count", "stat", "sub"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--quiet licenses the lines a read serves, and --%s serves or licenses differently: drop one", f), exitUsage)
					}
				}
			}
			// ADR-169: a guard judges the bytes of a file a read serves, so the
			// reads that serve no line or exist to license have nothing for it.
			var guards []read.Guard
			if cmd.IsSet("if-changed") {
				for _, f := range []string{"count", "stat", "quiet", "sub", "rewrite"} {
					if cmd.IsSet(f) {
						return cli.Exit(fmt.Sprintf("--if-changed answers a spec whose file did not change, and --%s serves no line or exists to license: drop one", f), exitUsage)
					}
				}
				for _, v := range cmd.StringSlice("if-changed") {
					g, guardErr := read.ParseGuard(v)
					if guardErr != nil {
						return cli.Exit("--if-changed "+guardErr.Error(), exitUsage)
					}
					guards = append(guards, g)
				}
			}
			// ADR-179: --if-served names the sha the caller holds, as --if-changed does.
			var servedGuards []read.Guard
			if cmd.IsSet("if-served") {
				for _, v := range cmd.StringSlice("if-served") {
					g, guardErr := read.ParseGuard(v)
					if guardErr != nil {
						return cli.Exit("--if-served "+guardErr.Error(), exitUsage)
					}
					servedGuards = append(servedGuards, g)
				}
				// Before any spec resolves, whatever a finder matches: one file under two
				// guards would be answered twice over.
				if p, shared := read.SharedGuard(cmd.Root().String("root"), guards, servedGuards); shared {
					return cli.Exit(fmt.Sprintf("--if-served and --if-changed both name %s: drop one", p), exitUsage)
				}
			}
			// A glob that can never match is refused at parse time rather than
			// silently matching nothing: one path.Match rejects, or one spelled
			// so no root-relative path can match it (read.CheckExclude, shared
			// with MCP; ADR-078).
			if err := read.CheckExclude(excludes); err != nil {
				return cli.Exit("--exclude "+err.Error(), exitUsage)
			}
			// ADR-161: `-` is standard input, read as one unnamed file and
			// recorded nowhere. A file named - is ./-.
			if stdinSpec(posArgs) {
				return stdinRead(cmd, pattern, grepSet, posArgs, projections)
			}
			if cmd.IsSet("sub") {
				return subRead(cmd, pattern, excludes, posArgs)
			}
			if cmd.IsSet("rewrite") {
				return rewriteRead(cmd, astPattern, excludes, posArgs)
			}

			root := cmd.Root().String("root")
			var (
				specs    []read.Spec
				refusals []read.Problem
				skipped  read.WalkSkipped
				err      error
				note     string
			)

			// Working-set pointers resolve here too. The ADR's precedence
			// table documents `mrw read --grep P @1 @2` as how a caller
			// asks for both, and without this `@1` was walked as a LITERAL
			// filename — refused as missing, or worse, matching a real file
			// actually named `@1`. Found by an independent review,
			// 2026-09-03.
			//
			// Loaded only when paths were given: `--grep P` with none walks
			// --root and must not consult the working set, which is the row
			// immediately above it in the same table.
			if len(posArgs) > 0 {
				set, err := iter.Load(root)
				if err != nil && !readsWithoutTheSet(err, posArgs) {
					return exitWith(err)
				}
				if posArgs, err = set.ResolveAll(posArgs); err != nil {
					return exitWith(err)
				}
			}
			switch {
			case grepSet:
				for _, a := range posArgs {
					sp, err := read.ParseSpec(a)
					if err != nil {
						return exitWith(err)
					}
					if len(sp.Ranges) > 0 {
						return cli.Exit(fmt.Sprintf("%s: a range and --grep are two answers to one question", a), exitUsage)
					}
				}
				re, compileErr := regexp.Compile(pattern)
				if compileErr != nil {
					return cli.Exit(fmt.Sprintf("--grep %q: %v", pattern, compileErr), exitUsage)
				}
				// Walk starts at the root when given no paths. The working set
				// is deliberately NOT consulted: iter holds read specs and
				// --grep supplies its own, so a caller who wants both writes
				// them out (mrw read --grep P @1 @2).
				specs, refusals, err = read.Walk(root, posArgs, read.WalkOptions{Pattern: re, Exclude: excludes, NoIgnore: cmd.Bool("no-ignore"), Skipped: &skipped})
				if err != nil {
					return exitWith(err)
				}
			case astSet:
				for _, a := range posArgs {
					sp, err := read.ParseSpec(a)
					if err != nil {
						return exitWith(err)
					}
					if len(sp.Ranges) > 0 {
						return cli.Exit(fmt.Sprintf("%s: a range and --ast-grep are two answers to one question", a), exitUsage)
					}
				}
				specs, refusals, err = read.AstGrep(root, posArgs, astPattern, excludes, read.AstGrepOptions{NoIgnore: cmd.Bool("no-ignore"), Skipped: &skipped})
				if errors.Is(err, read.ErrAstGrepMissing) {
					return cli.Exit(err.Error(), exitUsage)
				}
				if err != nil {
					return exitWith(err)
				}
			default:
				args := posArgs
				if filesFromSet {
					list, err := specList(root, filesFrom)
					if err != nil {
						return exitWith(err)
					}
					args = list
				}
				set, err := iter.Load(root)
				if err != nil && !readsWithoutTheSet(err, args) {
					return exitWith(err)
				}
				if len(args) == 0 {
					// No arguments means "what I am working on": the working set
					// was emitted once and costs nothing to name again.
					if len(set.Entries) == 0 {
						return cli.Exit("read needs a PATH[:RANGE] or @N, or a working set (see: mrw iter add)", exitUsage)
					}
					args = set.Entries
					note = set.Note
				} else if args, err = set.ResolveAll(args); err != nil {
					return exitWith(err)
				}
				specs = make([]read.Spec, 0, len(args))
				for _, a := range args {
					sp, err := read.ParseSpec(a)
					if err != nil {
						return exitWith(err)
					}
					if len(projections) > 0 && len(sp.Ranges) > 0 {
						return cli.Exit(fmt.Sprintf("%s: --json-path prints values, and an address names lines: give the file alone", a), exitUsage)
					}
					specs = append(specs, sp)
				}
			}
			var unusedGuards []string // named once below, however many flags carried the same guard (ADR-179 Decision e)
			var ifChanged map[string]string
			if len(guards) > 0 {
				var unused []string
				if ifChanged, unused, err = read.ResolveGuards(root, guards, specs, grepSet || astSet); err != nil {
					return cli.Exit("--if-changed: "+err.Error(), exitUsage)
				}
				unusedGuards = append(unusedGuards, unused...)
			}
			// ADR-179 Decision (d): the sha the caller quotes, and the ledger's word
			// on which lines it was shown. A ledger that cannot be read answers nothing.
			var ifServed map[string]string
			var servedEntry func(string) seen.Observation
			var referred []read.SameAsServed
			if len(servedGuards) > 0 {
				var unused []string
				if ifServed, unused, err = read.ResolveGuards(root, servedGuards, specs, grepSet || astSet); err != nil {
					return cli.Exit("--if-served: "+err.Error(), exitUsage)
				}
				unusedGuards = append(unusedGuards, unused...)
				if ledger, lerr := seen.Snapshot(root); lerr == nil {
					servedEntry = entryFinder(root, ledger)
				}
			}
			if len(unusedGuards) > 0 { // one line for the invocation, naming every file (ADR-179 Decision e)
				seenGuard, named := map[string]bool{}, []string{}
				for _, p := range unusedGuards {
					if !seenGuard[p] {
						seenGuard[p] = true
						named = append(named, p)
					}
				}
				them := "it"
				if len(named) > 1 {
					them = "them"
				}
				fmt.Fprintf(os.Stderr, "-- guard not used: %s (this read did not serve %s)\n", strings.Join(named, ", "), them)
			}
			// ADR-078: -C widens a single /pattern/ match and nothing else, so on
			// a call with none it changed nothing, silently: `mrw read -C 1 a.go:2`
			// served line 2 alone. Refused, as --exclude without --grep is: a flag
			// with nothing to act on is a mistake the caller should hear about.
			if cmd.IsSet("context") && !grepSet && !hasSinglePattern(specs) {
				return cli.Exit("-C widens a /pattern/ match, and no spec here has one: name a pattern (a.go:/func A/) or drop -C", exitUsage)
			}
			// Printed only now, past every refusal, so a usage error leaves stdout
			// empty (ADR-078; the reviews of #239 met the note before -C's).
			// Under --json the answer is collected, to become the document's
			// content once it is whole (ADR-165).
			var content bytes.Buffer
			out := bufio.NewWriter(os.Stdout)
			if cmd.Bool("json") {
				out = bufio.NewWriter(&content)
			}
			// document prints {"content", "receipt"} on standard output.
			// document prints {"content", "receipt"} on standard output. JSON
			// carries text, so content that is not valid UTF-8 would reach the
			// caller changed while its lines were licensed as read: refused, the
			// way mrw_read refuses to checkpoint such lines (ADR-108). A document
			// that could not be written is an answer nobody got, so the caller
			// records nothing after either (Codex review of 0f92edf0).
			document := func(receipt map[string]any) error {
				if !utf8.Valid(content.Bytes()) {
					return cli.Exit("the answer holds bytes that are not valid UTF-8, which a JSON document would change: read it without --json, so nothing was recorded", exitUsage)
				}
				// Set before the write, not after: a write that fails partway
				// has already put part of a document on stdout, and an error
				// document appended to it would be a second, broken one.
				printed = true
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				if err := enc.Encode(readDocument{Content: content.String(), Receipt: receipt}); err != nil {
					return cli.Exit(fmt.Sprintf("the answer could not be written, so nothing was recorded: %v", err), exitUsage)
				}
				return nil
			}
			// ADR-165: --max-chars bounds everything this read writes to standard
			// output — the iteration note, the walk's refusals and skip line, and
			// read.Run's answer — so one budget wraps the stream they all share.
			var stdout io.Writer = out
			var budget *read.Budget
			if cmd.IsSet("max-chars") {
				budget = read.NewBudget(out, cmd.Int("max-chars"))
				stdout = budget
			}
			// What the budget kept back is said on standard error AFTER the answer
			// is flushed, so a host that merges the streams never sees the list
			// land inside a served line (go-reviewer of a043d346).
			var withheld bytes.Buffer
			reached := func() string {
				if budget.Over() {
					return fmt.Sprintf("; --max-chars %d reached", cmd.Int("max-chars"))
				}
				return ""
			}
			if note != "" {
				fmt.Fprintf(stdout, "# iteration: %s\n", note)
			}

			// Every path the walk could not serve reaches the caller with its
			// reason, and counts. Rule 5: one bad path never costs the caller
			// the answers about the good ones, but it is never silent either.
			for _, p := range refusals {
				fmt.Fprintf(read.Reporting(stdout, &withheld), "==> %s  REFUSED  %s\n", p.Path, p.Reason)
			}
			// ADR-116: what the walk skipped is said, never silent, as the
			// read's last line.
			skipNote := func() {
				if note := read.SkipNote(skipped, "--no-ignore"); note != "" {
					fmt.Fprintln(stdout, note)
				}
			}
			// A pattern that matched no file is said out loud, naming the
			// pattern. read.Run over an empty spec list prints nothing at all,
			// which is byte-for-byte the output of a successful read that
			// happened to serve nothing — the one ambiguity worth a line.
			if grepSet && len(specs) == 0 {
				skipNote()
				_ = out.Flush() // exit 1 follows whether or not the refusals reached stdout
				_, _ = os.Stderr.Write(withheld.Bytes())
				if cmd.Bool("json") {
					if err := document(mcp.NoMatchReceipt(len(refusals), skipped)); err != nil {
						return err
					}
				}
				hint := ""
				if h := patternFlagHint(pattern); h != "" {
					hint = " — " + h + " (mrw took the value after --grep as the pattern)"
				}
				return cli.Exit(read.NoMatchLine(pattern, len(refusals), hint)+reached(), 1)
			}
			if astSet && len(specs) == 0 {
				skipNote()
				_ = out.Flush() // exit 1 follows whether or not the refusals reached stdout
				_, _ = os.Stderr.Write(withheld.Bytes())
				if cmd.Bool("json") {
					if err := document(mcp.NoMatchReceipt(len(refusals), skipped)); err != nil {
						return err
					}
				}
				return cli.Exit(read.NoMatchLine(astPattern, len(refusals), "")+reached(), 1)
			}

			cols, colsFixed := effectiveMaxCols(cmd)
			if cmd.Bool("outline") {
				for _, sp := range specs {
					if len(sp.Ranges) > 0 {
						return cli.Exit(fmt.Sprintf("--outline lists a whole file, and %s names lines: drop the address", sp.Raw), exitUsage)
					}
				}
			}
			var unoutlined []string
			var wholeNames []string
			var matchesLeft int
			var long read.LongLines
			var wholeLong *read.LongLines
			if !colsFixed {
				wholeLong = &long
			}
			var counts map[string]int
			var unchanged []read.Unchanged
			if cmd.Bool("count") {
				counts = map[string]int{}
			}
			observed, problems := read.Run(stdout, root, specs, read.Options{
				Numbers:      !cmd.Bool("no-numbers"),
				Stat:         cmd.Bool("stat"),
				Context:      cmd.Int("context"),
				MaxLines:     maxLines(cmd),
				MaxCols:      cols,
				MaxMatches:   maxMatches(cmd),
				Fold:         cmd.Bool("fold"),
				Outline:      cmd.Bool("outline"),
				Unoutlined:   &unoutlined,
				NoComments:   cmd.Bool("no-comments"),
				WholeNames:   &wholeNames,
				Count:        cmd.Bool("count"),
				Counts:       counts,
				Quiet:        cmd.Bool("quiet"),
				NextLine:     true,
				Withheld:     &withheld,
				IfChanged:    ifChanged,
				Unchanged:    &unchanged,
				IfServed:     ifServed,
				Served:       servedEntry,
				SameAsServed: &referred,
				Project:      projections,
				WholeLong:    wholeLong,
				Unshown:      &matchesLeft,
			})
			skipNote()
			// The answer reaches the caller BEFORE anything is recorded (ADR-088).
			// It used to be flushed on return, unchecked, after seen.Record: a read
			// whose output could not be written — a full disk — licensed a write to
			// lines nobody saw, which is ADR-002 inverted.
			if err := out.Flush(); err != nil {
				return cli.Exit(fmt.Sprintf("the answer could not be written, so nothing was recorded: %v", err), exitUsage)
			}
			_, _ = os.Stderr.Write(withheld.Bytes())
			if len(unoutlined) > 0 {
				fmt.Fprintf(os.Stderr, "mrw: --outline reads .go and .md files; no outline for %s\n", namedFiles(unoutlined))
			}
			if len(wholeNames) > 0 {
				fmt.Fprintf(os.Stderr, "mrw: --no-comments reads .go files; served whole: %s\n", namedFiles(wholeNames))
			}
			if hint := longLineHint(long); hint != "" && !toNullDevice(os.Stdout) {
				fmt.Fprintln(os.Stderr, hint)
			}
			if cmd.Bool("json") {
				if err := document(mcp.ReadReceipt(observed, problems+len(refusals), skipped, counts, unchanged, referred)); err != nil {
					return err
				}
			}
			// Reading a file is how mrw learns what it holds; recording that is
			// what lets a later write know whether its picture is still current.
			// ADR-133: an answer sent to the null device reached nobody, so it
			// licenses nothing — the one discard on the CLI that mrw can see.
			if toNullDevice(os.Stdout) {
				// Only a read that served lines has lines to name: a --stat or
				// --max-lines 0 serves none (the stress runs of 25f4f04). Nil
				// Spans is a whole-file read, every line served (seen.Observation).
				served := false
				for _, o := range observed {
					served = served || o.Spans == nil || len(o.Spans) > 0
				}
				if served {
					fmt.Fprintln(os.Stderr, "mrw: the answer went to the null device, so nothing was recorded: a write to these lines needs a read whose answer you see; on unix a stdout closed when mrw starts is the null device too (ADR-133); read again without redirecting, or license lines you already know with `mrw read --quiet SPEC`")
					if line := stillLicensed(root, observed); line != "" {
						fmt.Fprintln(os.Stderr, line)
					}
				}
			} else if err := seen.Record(root, recordedAs(cmd.Bool("quiet"), observed)); err != nil {
				// ADR-166: a content read whose ledger cannot be written has
				// served its answer; it records nothing new and says so, last,
				// with the read's own exit. A --quiet read exists to record, so
				// it still refuses (Decision 3).
				if !statehint.About(err) || cmd.Bool("quiet") {
					return exitWith(err)
				}
				defer fmt.Fprintln(os.Stderr, "mrw: nothing was recorded, so a write to these lines will be refused until a read can record them:", statehint.Hint(err))
			}
			problems += len(refusals)
			// A budget that dropped anything — a served line, or only the walk's
			// own last line — leaves an answer that is not the whole one, and a
			// cut answer must never read as complete (ADR-165).
			if problems > 0 || budget.Over() {
				return cli.Exit(incompleteLine(problems, matchesLeft, maxMatches(cmd), reached()), 1)
			}
			return nil
		},
	}
}

// readsWithoutTheSet says whether a read may go on when the working set could
// not be loaded (ADR-166): only when the failure is mrw's state directory and
// the read names specs, none a working-set pointer, so the set is not needed.
func readsWithoutTheSet(err error, args []string) bool {
	if !statehint.About(err) || len(args) == 0 {
		return false
	}
	for _, a := range args {
		if iter.IsPointer(a) {
			return false
		}
	}
	return true
}

func writeCmd() *cli.Command {
	// asked is the --then / --then-sh list, in command-line order (ADR-092).
	var asked []check.Step
	return &cli.Command{
		Name: "write",
		// ADR-099: a plan file named help is a plan file.
		HideHelpCommand: true,
		Usage:           "apply an edit plan across one or more files; a plan that fails validation writes nothing",
		ArgsUsage:       "[PLAN|-]",
		Description: `A plan is a sequence of hunks:

  @@ <path> <addr> <op> [sha=… lines=… anchor=… body=…]
  <body lines>

Ops are replace, insert-after, insert-before, delete, create, unlink and rename.
unlink and rename take address - (a hyphen, no line number). unlink's body is
empty. rename's destination is the one-line body (not a to= key). Other
addresses are 1-based and inclusive, and every one of them resolves against
the ORIGINAL file — so several hunks in one file need no offset arithmetic.

The guards are what make a batch safe to trust: sha= pins the whole file,
lines= asserts how many lines the range covers, anchor= requires a substring in
the range's first line and is required on a multi-line replace, where sha= and
lines= do not stand in for it. If any hunk fails validation, every hunk is
reported and NOTHING is written. A filesystem failure while committing can leave
some files written: the receipt says PARTIALLY APPLIED and names them.

A value with spaces can be double-quoted (anchor="func openTestStore"),
single-quoted (anchor='func openTestStore'), or — for anchor= only — left
unquoted until the next key= (anchor=func openTestStore body=1).
body= is optional for an ordinary body: it runs to the next @@ header. A count
is needed for an empty create (body=0), a body holding a line that begins @@,
and a body whose first line begins body=; see mrw instructions.
body= is a line count, not a character count. Python str splits characters;
do not use len(body) as body=.
body=@path loads the body from a root-relative file (an empty file is the
same as body=0). An unquoted anchor= that contains a double quote is refused;
write it as anchor="…". A leftover body= names the declared count and how
many extra lines sat before the next @@.
lines= is a guard on how many lines the ADDRESS covers, and is not body=.

The checkout is named by global -C DIR or --root DIR before the subcommand
(mrw -C repo write plan). After read, -C is context lines, not a checkout.

Prefer authoring the plan with your harness's own file tool and passing its
path, rather than piping it in: a plan on disk is a reviewable artifact and is
visible to whatever hooks watch file writes.

--format=apply_patch compiles a Codex apply_patch document (*** Begin Patch)
into the native plan above, then Parse and Apply run unchanged. A git patch
is not an apply_patch; --format=git is usage.
--format=search_replace compiles an Aider SEARCH/REPLACE document
(<<<<<<< SEARCH / ======= / >>>>>>> REPLACE) the same way. Default --format is plan.

--echo-pad N prints N lines after an applied body so a surviving closer is
visible. Default 0. It is not a checker: a closer in the pad does not fail
the hunk. Negative is usage.

After a successful apply the project's check runs by default when a written
path is not prose (.md .markdown .txt .rst .adoc) and a check exists — declared
in .quality-harness.json or inferred from go.mod. A markdown-only plan does
not spawn it; a tree with no check stays exit 0. --no-check opts out. --check
is a demand: it runs on prose too, and is exit 2 when nothing can run. A
failing check is exit 3 and the tree is NOT reverted.

A non-prose hunk whose {} () [] nets differ between the replaced lines and
the body prints a balance row under ok. It does not fail the hunk and it is
not a checker: braces in strings miscount, and a balanced insert landing
inside a function is invisible to it — the check catches that, the row does
not.

The summary line counts those rows — "N hunk(s), M file(s), F failed, A
advisories — applied", zero included — and the JSON receipt carries
"advisories". When three of your last ten landed writes carried one, the
receipt prints a pattern line — read past the range before the next one.
--strict-balance is opt-in: it refuses a single-line replace whose line's
{} () [] do not balance and whose body does not match them (the wrap-tail
shape) as a failed hunk — exit 1, nothing is written. Off by default; braces
inside strings count, so drop it for that plan. Both JSON receipts (--json
and mrw_write) carry "pattern" {advisory_writes, window, fires} on every
write, and mrw stats prints a "strict-balance pricing" block as writes land —
how many the flag would have refused, and whether those writes then broke,
held or went unchecked.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "validate and report, write nothing; with --grep --sub or --ast-grep --rewrite, show each change",
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
				Usage: "print only failed hunks and the summary line, dropping ok and skip rows and file lines; the check's report and every step verdict still print",
			},
			&cli.BoolFlag{
				Name:  "terse",
				Usage: "print one line per file (ok PATH N hunk(s) -A +B sha X; dry under --dry-run) and the summary instead of a row per hunk; every failed hunk, balance and closer row, the check's report and every step verdict still print, and --json is unchanged (ADR-179)",
			},
			&cli.BoolFlag{
				Name: "check",
				Usage: "demand the project's check after a successful write, even on a prose-only plan " +
					"(exit 2 if none can run)",
			},
			&cli.BoolFlag{
				Name:  "no-check",
				Usage: "do not run the project's check after this write (the default runs it when a written path is not prose)",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "plan",
				Usage: "plan (default), apply_patch (Codex *** Begin Patch; a git patch is not one), or search_replace (Aider SEARCH/REPLACE)",
			},
			&cli.StringFlag{
				Name:  "create",
				Usage: "create `PATH` (relative to the root) from standard input, as the create plan of its lines would: each line ends in a newline, so a missing final newline is added and CRLF becomes LF; refused if PATH exists. Takes no PLAN argument and no --format (ADR-139)",
			},
			&cli.StringFlag{
				Name:  "grep",
				Usage: "with --sub, rewrite the lines matching `PATTERN` in the files under the given paths (a directory is walked) instead of applying a PLAN; the paths are given where the PLAN would be (ADR-161)",
			},
			&cli.StringFlag{
				Name:  "sub",
				Usage: "with --grep, rewrite each matching line by `REPLACEMENT` (Go regexp: $1 and ${name} expand, $$ is a literal dollar, and a reference to a group the pattern does not define is refused) and apply that as one plan: a verdict per site, nothing written if any fails or any matching file could not be read; --dry-run also shows each change, and --check and --then work as for any write. Not with a PLAN, - , --format or --create (ADR-161)",
			},
			&cli.StringFlag{
				Name:  "ast-grep",
				Usage: "with --rewrite, rewrite the matches of `PATTERN` the ast-grep binary (on PATH) reports in the files under the given paths (a directory is walked) instead of applying a PLAN (ADR-161)",
			},
			&cli.StringFlag{
				Name:  "rewrite",
				Usage: "with --ast-grep, replace each match by ast-grep's own rewrite to `REPLACEMENT` and apply that as one plan: a verdict per span, nothing written if any fails, if matches overlap, or if any matching file could not be read; --dry-run also shows each change, and --check and --then work as for any write. Not with a PLAN, - , --grep, --sub, --format or --create (ADR-161)",
			},
			&cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "with --grep --sub or --ast-grep --rewrite, skip paths matching `GLOB` as read --grep does: the root-relative path and the basename, a bare directory name prunes its subtree (repeatable; ADR-161)",
			},
			&cli.BoolFlag{
				Name:  "no-ignore",
				Usage: "with --grep --sub or --ast-grep --rewrite, walk every regular file as read --grep --no-ignore does (ADR-161)",
			},
			&cli.IntFlag{
				Name:  "echo-pad",
				Value: 0,
				Usage: "print N lines after an applied body (opt-in pad; default 0; not a checker)",
			},
			&cli.BoolFlag{
				Name: "strict-balance",
				Usage: "refuse a single-line replace whose line's {} () [] do not balance and whose body does not match them " +
					"(the wrap-tail shape); exit 1, nothing written. Off by default",
			},
			thenFlag(&asked),
			thenShFlag(&asked),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// ADR-157: a write under a newer ledger says ADR-150's sentence once,
			// in a refusal that carries it or on stderr, whichever way it returns:
			// a plan that does not parse returns long before the landing (the
			// review of #412).
			newerTold := false
			defer func() {
				if !newerTold {
					if notice := writer.NewerUntold(cmd.Root().String("root"), apply.Result{}); notice != "" {
						fmt.Fprintln(os.Stderr, notice)
					}
				}
			}()
			if err := refusePaddedArgs(cmd); err != nil {
				return err
			}
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
			if cmd.Bool("check") && cmd.Bool("no-check") {
				return cli.Exit("--check and --no-check contradict each other: drop one", exitUsage)
			}
			if cmd.Bool("terse") && cmd.Bool("quiet") {
				return cli.Exit("--terse prints a line per file and --quiet prints none: drop one", exitUsage)
			}
			if cmd.Int("echo-pad") < 0 {
				return cli.Exit("--echo-pad must be >= 0", exitUsage)
			}
			planned := cmd.IsSet("grep") || cmd.IsSet("rewrite") // a plan mrw makes itself (ADR-161)
			if len(args) > 1 && !planned {
				return cli.Exit("write takes at most one plan file", exitUsage)
			}
			if cmd.IsSet("create") {
				if err := ingest.CheckCreatePath(cmd.String("create")); err != nil {
					return cli.Exit(err.Error(), exitUsage)
				}
				if len(args) > 0 {
					return cli.Exit("--create takes the content on standard input: it takes no PLAN argument", exitUsage)
				}
				if cmd.IsSet("format") {
					return cli.Exit("--create and --format contradict each other: drop one", exitUsage)
				}
			}
			if cmd.IsSet("grep") != cmd.IsSet("sub") {
				return cli.Exit("--grep and --sub go together: --sub is the replacement for the lines --grep matches", exitUsage)
			}
			if cmd.IsSet("rewrite") != cmd.IsSet("ast-grep") {
				return cli.Exit("--ast-grep and --rewrite go together: --rewrite is the replacement for the matches --ast-grep finds", exitUsage)
			}
			if cmd.IsSet("rewrite") && cmd.IsSet("grep") {
				return cli.Exit("--ast-grep --rewrite and --grep --sub are two ways to make a plan: use one", exitUsage)
			}
			if !planned && (cmd.IsSet("exclude") || cmd.IsSet("no-ignore")) {
				return cli.Exit("--exclude and --no-ignore shape the walk of --grep --sub or --ast-grep --rewrite: give one of those, or drop them", exitUsage)
			}
			if planned {
				makes := "--grep --sub"
				if cmd.IsSet("rewrite") {
					makes = "--ast-grep --rewrite"
				}
				if cmd.IsSet("create") || cmd.IsSet("format") {
					return cli.Exit(makes+" makes its own plan: drop --create and --format", exitUsage)
				}
				for _, a := range args {
					if a == "-" {
						return cli.Exit(makes+" makes its own plan: it takes paths to walk, not a PLAN or -", exitUsage)
					}
				}
				if err := read.CheckExclude(cmd.StringSlice("exclude")); err != nil {
					return cli.Exit("--exclude "+err.Error(), exitUsage)
				}
			}
			// ADR-072: under --json every refusal after the plan is named is
			// one JSON document — the receipt's shape with an error field — so
			// a consumer that parses stdout never meets text. A refusal after
			// the write landed carries the real result; one before it carries
			// an empty one, with applied false and empty arrays, not null.
			// printed and tallied say the human receipt is on stdout and the
			// landing is counted, so a refusal after either does neither twice.
			// parsed says the plan parsed: a refusal after that and before anything
			// landed is one refused_apply, as mrw_write counts it (ADR-083).
			printed, tallied, parsed := false, false, false
			var createNotes []string // what --create did to its content, told once the create has landed
			emitNotes := func() {
				for _, n := range createNotes {
					fmt.Fprintf(os.Stderr, "mrw: --create %s: %s\n", cmd.String("create"), n)
				}
				createNotes = nil
			}
			// stepsNotRun is the --then list a refusal after the landing carries,
			// every step not_run: the check could not run, or the ledger could not
			// record the landing, so none followed it (ADR-092 T4; the 2026-09-29 gap survey, C3).
			var stepsNotRun *check.StepsResult
			// ADR-176: the standard-input plan the parser read, kept beside the
			// state directory when the run ends with nothing in the tree.
			// keptDone makes it once, whichever exit reaches it first.
			var tee *planTee
			var refusedPlan string
			keptDone := false
			keepRefused := func(res apply.Result) {
				if tee == nil || keptDone || writer.MutationOf(res) != writer.None {
					return
				}
				keptDone = true
				path, line := tee.keep()
				refusedPlan = path
				if line != "" {
					fmt.Fprintln(os.Stderr, line)
				}
			}
			refuseWith := func(res apply.Result, err error) error {
				msg := err.Error()
				keepRefused(res)
				// A refusal after the write landed (the ledger could not be
				// written, the check could not start) still says what landed
				// and counts it: the tree changed either way. In human form the
				// ledger failure printed nothing, and the landing went
				// uncounted (review of #229, advisory 1).
				landed := res.Applied && !res.DryRun
				if landed {
					emitNotes()
				}
				if landed && !tallied {
					_ = authoring.Record(cmd.Root().String("root"), authoring.Applied)
					tallied = true
				}
				if !landed && parsed && !tallied {
					_ = authoring.Record(cmd.Root().String("root"), authoring.RefusedApply)
					tallied = true
				}
				if !cmd.Bool("json") && landed && !printed {
					reportAs(os.Stdout, res, cmd)
					printed = true
				}
				if !cmd.Bool("json") {
					reportSteps(os.Stdout, stepsNotRun)
				}
				if cmd.Bool("json") {
					if res.Files == nil {
						res.Files = []apply.FileResult{}
					}
					if res.Hunks == nil {
						res.Hunks = []apply.HunkResult{}
					}
					root := cmd.Root().String("root")
					if res.Root == "" {
						res.Root = root
					}
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					enc.SetEscapeHTML(false)
					_ = enc.Encode(receipt{Result: res, Pattern: authoring.PatternOf(root), Error: msg, Then: stepsNotRun, RefusedPlan: refusedPlan})
				}
				return exitWith(err)
			}
			refuse := func(msg string) error {
				return refuseWith(apply.Result{DryRun: cmd.Bool("dry-run")}, errors.New(msg))
			}
			// refuseErr keeps err unwrappable, so main can name a state-directory
			// failure under it (stateHint); refuse with err's text would flatten it.
			refuseErr := func(err error) error {
				return refuseWith(apply.Result{DryRun: cmd.Bool("dry-run")}, err)
			}
			if err := askedStepsError(asked); err != nil {
				return refuseErr(err)
			}
			var src io.Reader = os.Stdin
			name := "<stdin>"
			if planned {
				// ADR-161: the plan is made here, from the walk, and goes through
				// the parse and apply every plan does. Recorded as read BEFORE it
				// is applied, as a read records what it printed; a partial plan
				// (a path the walk refused) writes nothing, since a plan that
				// leaves a site out must never exit 0.
				var b *read.Built
				var err error
				if cmd.IsSet("rewrite") {
					b, err = read.BuildRewrite(cmd.Root().String("root"), args, cmd.String("ast-grep"), cmd.String("rewrite"), cmd.StringSlice("exclude"), cmd.Bool("no-ignore"), os.Stderr)
				} else {
					b, err = read.BuildSub(cmd.Root().String("root"), args, cmd.String("grep"), cmd.String("sub"), cmd.StringSlice("exclude"), cmd.Bool("no-ignore"), os.Stderr)
				}
				if err != nil {
					err = exitOf(err)
				} else if b.Problems > 0 {
					err = cli.Exit(fmt.Sprintf("incomplete: %d path(s) or range(s) could not be served, so nothing was written", b.Problems), exitNotApplied)
				}
				if err != nil {
					// ADR-072: under --json a refusal is one JSON document; it keeps
					// its own exit, 1 for nothing planned, 2 for a usage mistake.
					var ec cli.ExitCoder
					if cmd.Bool("json") && errors.As(err, &ec) {
						_ = refuse(ec.Error())
					}
					// err itself, not a cli.Exit rebuilt from its text: the chain exitOf
					// kept is what lets main name a state-directory failure (ADR-164).
					return err
				}
				if err := seen.Record(cmd.Root().String("root"), b.Observed); err != nil {
					return refuseErr(err)
				}
				src, name = bytes.NewReader(b.Plan), "--sub"
				if cmd.IsSet("rewrite") {
					name = "--rewrite"
				}
			} else if len(args) == 1 && args[0] != "-" {
				// ADR-077: the plan is a shell argument and never passed the
				// boundary; a plan naming mrw's ack store had its JSON quoted
				// back as "text before the first @@ header" (Codex review of #238).
				if abs, err := filepath.Abs(args[0]); err == nil && rooted.InStateOrLinked(cmd.String("root"), abs) {
					return refuse(fmt.Sprintf("%s is inside mrw's own state directory, or is a hard link to a file in it; mrw does not read its own files as a plan", args[0]))
				}
				f, err := os.Open(args[0])
				if err != nil {
					// -C moves the paths INSIDE the plan; the plan file itself
					// is a shell argument like any other and resolves against
					// the working directory. That split is defensible and
					// invisible, so a miss says where it looked rather than
					// leaving the caller to doubt the plan instead of the path.
					return refuseErr(planOpenError(args[0], cmd.String("root"), err))
				}
				defer func() { _ = f.Close() }()
				src, name = f, args[0]
			}

			// ADR-161 Decision 13: PowerShell appends CRLF to what it pipes, and after
			// a plan whose lines end in LF that CRLF was a blank last line the write
			// added (found on Windows by a peer session). Dropped by ADR-142's rule as
			// the plan streams past, so an unknown --format and an over-long line are
			// refused as before; --create keeps ADR-142's own notes for its content.
			if src == os.Stdin && !cmd.IsSet("create") {
				piped := ingest.NewPipedCRLFReader(os.Stdin)
				piped.OnTrim = func() {
					fmt.Fprintln(os.Stderr, "mrw: the plan ended in CRLF after lines that end in LF; that CRLF is most likely the one PowerShell appends to what it pipes, so it was dropped")
				}
				tee = &planTee{src: piped}
				src = tee
			}
			var hunks []plan.Hunk
			var err error
			format := cmd.String("format")
			if cmd.IsSet("create") {
				format = "create" // ADR-139: standard input is a file's content, compiled to one create hunk
			}
			switch format {
			case "plan", "":
				// Issue #537: the parser drops every leading mark; the caller is
				// told when that was more than one.
				br := bufio.NewReader(src)
				if n, more := ingest.LeadingMarks(br); n > 1 || more {
					fmt.Fprintln(os.Stderr, "mrw: "+ingest.MarksNote("the plan", n, more))
				}
				hunks, err = plan.Parse(br)
			case "apply_patch":
				raw, rerr := io.ReadAll(src)
				if rerr != nil {
					return refuse(fmt.Sprintf("%s: %v", name, rerr))
				}
				raw, marks := ingest.DropLeadingMarks(raw)
				if marks > 1 {
					fmt.Fprintln(os.Stderr, "mrw: "+ingest.MarksNote("the apply_patch document", marks, false))
				}
				compiled, cerr := ingest.CompileApplyPatch(cmd.Root().String("root"), raw)
				if cerr != nil {
					// The document did not become a plan. Same bucket as a
					// native parse refusal: the FORMAT was the problem.
					_ = authoring.Record(cmd.Root().String("root"), authoring.RefusedParse)
					return refuse(fmt.Sprintf("%s: %v", name, cerr))
				}
				hunks, err = plan.Parse(bytes.NewReader(compiled))
			case "search_replace":
				raw, rerr := io.ReadAll(src)
				if rerr != nil {
					return refuse(fmt.Sprintf("%s: %v", name, rerr))
				}
				raw, marks := ingest.DropLeadingMarks(raw)
				if marks > 1 {
					fmt.Fprintln(os.Stderr, "mrw: "+ingest.MarksNote("the search_replace document", marks, false))
				}
				compiled, cerr := ingest.CompileSearchReplace(cmd.Root().String("root"), raw)
				if cerr != nil {
					_ = authoring.Record(cmd.Root().String("root"), authoring.RefusedParse)
					return refuse(fmt.Sprintf("%s: %v", name, cerr))
				}
				hunks, err = plan.Parse(bytes.NewReader(compiled))
			case "create":
				if !cmd.IsSet("create") {
					return refuse(fmt.Sprintf("unknown --format %q (plan, apply_patch, or search_replace)", cmd.String("format")))
				}
				raw, rerr := io.ReadAll(src)
				if rerr != nil {
					return refuse(fmt.Sprintf("%s: %v", name, rerr))
				}
				raw, notes := ingest.CreateContent(raw)
				createNotes = notes
				compiled, cerr := ingest.CompileCreate(cmd.String("create"), raw)
				if cerr != nil {
					return refuse(fmt.Sprintf("%s: %v", name, cerr)) // a content refusal, not a plan that failed to parse: not counted by ADR-009
				}
				if n := ingest.EndingNote(raw); n != "" {
					createNotes = append(createNotes, n) // told once the create lands, like the others
				}
				// The notes say what was done to content that went into a file, so
				// they are told once it landed (emitNotes): a refused create is told nothing.
				hunks, err = plan.Parse(bytes.NewReader(compiled))
			case "git":
				return refuse("a git patch is not an apply_patch; --format=apply_patch is for *** Begin Patch documents")
			default:
				return refuse(fmt.Sprintf("unknown --format %q (plan, apply_patch, or search_replace)", cmd.String("format")))
			}
			if err != nil {
				// A plan that did not PARSE is the outcome ADR-009 exists to
				// count: the only one that says the FORMAT was the problem
				// rather than the caller's picture of a file. Recorded here,
				// at the site that already decided it, so the tally is a
				// projection of that decision and never a second opinion.
				_ = authoring.Record(cmd.Root().String("root"), authoring.RefusedParse)
				return refuse(fmt.Sprintf("%s: %v", name, err))
			}

			root := cmd.Root().String("root")
			if err := plan.LoadBodyFiles(root, hunks); err != nil {
				_ = authoring.Record(root, authoring.RefusedParse)
				return refuse(fmt.Sprintf("%s: %v", name, err))
			}
			parsed = true
			// ADR-086: a --json receipt is JSON, and encoding/json turns a byte
			// that is not valid UTF-8 into U+FFFD, so the receipt would name
			// another file than the one written. Refused before anything lands.
			if cmd.Bool("json") {
				for _, h := range hunks {
					names := []string{h.Path}
					if string(h.Op) == "rename" && len(h.Body) == 1 {
						names = append(names, h.Body[0])
					}
					if h.BodyFrom != "" { // ADR-167: the receipt names the source too
						names = append(names, h.BodyFrom)
					}
					for _, n := range names {
						if !utf8.ValidString(n) {
							return refuse(fmt.Sprintf("%s line %d: %q is not valid UTF-8, and a --json receipt cannot name it as written; drop --json to write it", name, h.SrcLine, n))
						}
					}
				}
			}
			if cmd.Bool("dry-run") && !cmd.Bool("json") {
				for _, h := range hunks {
					body := fmt.Sprintf("body=%d", len(h.Body))
					if h.BodyFrom != "" {
						body = "from=" + h.BodyFrom + ":" + h.BodyFromAddr.String() // ADR-167
					}
					fmt.Fprintf(os.Stdout, "parsed: %s %s %s %s\n", h.Path, h.Addr.String(), h.Op, body)
				}
			}
			set, err := iter.Load(root)
			if err != nil {
				return refuseErr(err)
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
						return refuse(fmt.Sprintf("%s line %d: %v", name, h.SrcLine, err))
					}
					if len(got) != 1 {
						return refuse(fmt.Sprintf("%s line %d: %s names %d entries; a hunk needs exactly one",
							name, h.SrcLine, path, len(got)))
					}
					path = iter.Path(got[0])
					// ADR-086: the --json check above saw the pointer, not the
					// name it expands to (Codex review of #254).
					if cmd.Bool("json") && !utf8.ValidString(path) {
						return refuse(fmt.Sprintf("%s line %d: %s names %q, which is not valid UTF-8, and a --json receipt cannot name it as written; drop --json to write it", name, h.SrcLine, h.Path, path))
					}
				}
				in = append(in, apply.Input{
					Path: path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
					StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd, Occurrence: h.Occurrence, CountedBody: h.CountedBody,
					Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor,
					SrcLine: h.SrcLine, Index: h.Index,
					From: h.BodyFrom, FromStart: h.BodyFromAddr.Start, FromEnd: h.BodyFromAddr.End, FromRelEnd: h.BodyFromAddr.RelEnd,
				})
			}

			// ADR-113: the gates, the landing's count, the check, the drift,
			// the steps and the pricing are the shared sequence mrw_write runs
			// too; this action renders what it returns. ADR-072: the harness
			// is read BEFORE anything is written, inside Prepare.
			mode := writer.CheckAuto
			switch {
			case cmd.Bool("no-check"):
				mode = writer.CheckOff
			case cmd.Bool("check"):
				mode = writer.CheckDemand
			}
			prep, err := writer.Prepare(writer.Request{
				Root:  root,
				In:    in,
				Check: mode,
				Steps: asked,
				Opts: apply.Options{
					DryRun:        cmd.Bool("dry-run"),
					Force:         cmd.Bool("force"),
					EchoPad:       cmd.Int("echo-pad"),
					Preview:       cmd.Bool("dry-run") && planned, // ADR-164: a plan maker's dry run shows each change
					StrictBalance: cmd.Bool("strict-balance"),
				},
			})
			if err != nil {
				tallied = true // Prepare counted it
				return refuseErr(gateRefusal(err))
			}
			land, err := prep.Land()
			if err != nil {
				tallied = true // Land counted it
				return refuseErr(gateRefusal(err))
			}
			// Land counted the landing, whatever it was.
			tallied = true
			// Past the landing the receipt decides (ADR-157): a refusal in it
			// carries ADR-150's sentence, or it is said here.
			newerTold = true
			if notice := writer.NewerUntold(root, land.Res); notice != "" {
				fmt.Fprintln(os.Stderr, notice)
			}
			if land.Res.Applied && !land.Res.DryRun {
				emitNotes()
			}
			// ADR-132: what the caller is shown spells paths with "/", as
			// mrw_write's receipt does (ADR-091); the engine kept its own.
			res := shown(land.Res)
			keepRefused(res)
			if land.Err != nil {
				// ADR-001 rule 3: every hunk carries its own verdict, and a
				// filesystem failure is not an exception. Apply fills the
				// receipt before returning the error, so render it on whichever
				// surface the caller asked for — a --json caller that got a
				// bare "mrw: …: permission denied" had nothing to parse — and
				// only then exit. The exit code is unchanged: a filesystem
				// failure stays 2, distinct from a failing hunk's 1.
				if cmd.Bool("json") {
					// ADR-072: one document whether or not any hunk has a
					// verdict yet. Rendered only when it had hunks, a failure
					// before the first one — a plan naming a directory —
					// printed nothing under --json (review of #229, B2).
					return refuseWith(res, land.Err)
				}
				if len(res.Hunks) > 0 {
					reportAs(os.Stdout, res, cmd)
				}
				return exitSaying(exitTwoLine(land.Res, land.Err), land.Err)
			}
			// The landing (writer's applyCounted) recorded what the files now hold before it released
			// the write lock. This is why a chain of edits needs no re-read
			// between steps — mrw knows what it just produced — while a change
			// made behind its back still leaves the ledger disagreeing with the
			// disk, and the next write refused. When the ledger could not
			// record the landing, the write is refused with the landed receipt
			// and every step asked for named not_run (ADR-092 Decision 5).
			if land.LedgerErr != nil {
				stepsNotRun = land.Then
				return refuseWith(res, land.LedgerErr)
			}

			receipt := receipt{Result: res, RefusedPlan: refusedPlan}

			// ADR-055: a landed write joined the recent-window ring in Land,
			// BEFORE the receipt is rendered, so the receipt can say "3 of
			// your last 3" — the line has to be on the receipt, read in the
			// turn, not only in stats, which is run after the fact.
			var pattern string
			if res.Applied && !res.DryRun {
				pattern = patternLine(authoring.Recent(root))
			}
			receipt.Pattern = authoring.PatternOf(root)

			// ADR-072: the human receipt is printed, and the landing counted,
			// BEFORE the check runs. Rendered after it, a write killed during
			// its check printed nothing and was missing from stats. --json
			// stays one document, rendered once the check is done; the
			// ledger and the tally carry the landing if mrw is killed first.
			if !cmd.Bool("json") {
				reportAs(os.Stdout, res, cmd)
				printed = true
				if pattern != "" {
					// Not hidden by --quiet: quiet drops ok rows, and this
					// is the opposite of an ok row.
					fmt.Println(pattern)
				}
			}
			// ADR-054: the check runs by default, but only when the plan
			// touched a file a check could plausibly cover AND a command
			// exists; --check demands it. Verify runs it, the drift (ADR-112)
			// and the steps (ADR-092), and moves the count to the verdict.
			v := land.Verify(ctx)
			if v.CheckErr != nil {
				// The write landed and its check could not run: that is
				// check_not_run, not applied (review of #229).
				stepsNotRun = v.Then
				return refuseWith(res, v.CheckErr)
			}
			receipt.Check, receipt.Drift, receipt.DriftWriters, receipt.Then = v.Check, shownPaths(v.Drift), v.DriftWriters, v.Then

			if cmd.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				if err := enc.Encode(receipt); err != nil {
					return exitWith(err)
				}
			} else {
				reportCheck(os.Stdout, receipt.Check)
				reportDrift(os.Stdout, receipt.Drift, receipt.DriftWriters)
				reportSteps(os.Stdout, receipt.Then)
			}
			// ADR-113: counted and priced once the receipt is out, so an
			// output failure above leaves the count where Land put it.
			land.Settle(v)
			switch {
			case res.Failed > 0:
				return cli.Exit(fmt.Sprintf("%d hunk(s) failed — nothing was written", res.Failed), exitNotApplied)
			case receipt.Check != nil && check.StoppedBeforeStart(*receipt.Check):
				// ADR-080, ADR-126: stopped before the check started — by an
				// interrupt or by its deadline — is the same exit as stopped while
				// it ran, since the tree changed and nothing verified it, though
				// the tally files it as check_not_run: nothing ran. Not "declare a
				// check": one was.
				return cli.Exit("the write applied and its check "+stoppedBefore(*receipt.Check)+" — the tree is changed and unverified", exitCheckFailed)
			case receipt.Check != nil && !receipt.Check.Ran:
				// The write stands. Say so first — the caller's tree changed
				// even though the verification never happened.
				return cli.Exit("the write applied but no check could run: "+receipt.Check.Skipped+
					declareAdvice(receipt.Check.Skipped), exitUsage)
			case receipt.Check != nil && !receipt.Check.OK():
				return cli.Exit("the write applied but the check did not pass — the tree is changed and unverified", exitCheckFailed)
			}
			return stepsExit(receipt.Then, "the write applied but ", " — the tree is changed and unverified")
		},
	}
}

// patternLine renders ADR-055's repeat-pattern line, or "" when the ring
// holds fewer than PatternThreshold advisory writes. It counts and points;
// it does not judge the edit — the wording says what was counted.
func patternLine(entries []authoring.RecentEntry) string {
	k, n, fires := authoring.Pattern(entries)
	if !fires {
		return ""
	}
	return fmt.Sprintf("pattern: %d of your last %d writes carried a balance advisory — read past the range before the next one", k, n)
}

// gateRefusal is the CLI's words for a write the shared sequence refused before
// anything was written (ADR-113): the harness, the steps, the depth limit, or
// a ledger that could not be loaded.
func gateRefusal(err error) error {
	var r *writer.Refusal
	if !errors.As(err, &r) {
		return err
	}
	switch r.Stage {
	case writer.StageHarness, writer.StageSteps:
		return fmt.Errorf("%w: nothing was written", r.Err)
	case writer.StageDepth:
		return fmt.Errorf("%w; --no-check writes without it: nothing was written", r.Err)
	}
	return r.Err
}

// receipt is what one write produced: the edit and, when asked for, the
// verification of that edit. They travel together because the whole point of
// --check is that the change and the evidence for it are one result, not two
// round trips.
type receipt struct {
	apply.Result
	Check *check.Result `json:"check,omitempty"`
	// Pattern is the recent-window pattern after this write (ADR-056):
	// always present, so the JSON caller holds the fact the human line
	// prints. Read from the ring after RecordRecent ran.
	Pattern authoring.PatternInfo `json:"pattern"`
	// Error is why a write was refused after its plan was named (ADR-072):
	// under --json every such refusal is this document, not text.
	Error string `json:"error,omitempty"`
	// Then is every --then / --then-sh step's verdict (ADR-092), present
	// whenever one was asked for.
	Then *check.StepsResult `json:"then,omitempty"`
	// Drift is each file the write touched that changed while its check ran
	// (ADR-112): absent when nothing did, and when no check ran. Advisory —
	// the exit code is the check's.
	Drift []string `json:"drift,omitempty"`
	// DriftWriters is how many other writes landed in this checkout while the
	// check ran (ADR-127): absent when none did, and when no check ran.
	// Advisory, as Drift is.
	DriftWriters int `json:"drift_writers,omitempty"`
	// RefusedPlan is where a standard-input plan that ended with nothing in the
	// tree was kept (ADR-176): absent when nothing was kept.
	RefusedPlan string `json:"refused_plan,omitempty"`
}

// checkReceipt is `mrw check --json`: the check's own flat fields, unchanged,
// and the steps beside them (ADR-092).
type checkReceipt struct {
	check.Result
	Then *check.StepsResult `json:"then,omitempty"`
}

// statsReceipt is `mrw stats --json` (ADR-054): every vocabulary key in counts,
// zero included, and the derived pair. Named so ADR-111's test can hold its keys.
type statsReceipt struct {
	Plans   int               `json:"plans"`
	Counts  map[string]int    `json:"counts"`
	Landed  int               `json:"landed"`
	Failed  int               `json:"failed_check_of_landed"`
	Pricing authoring.Pricing `json:"pricing"`
}

// checkRefusal is a `mrw check --json` refusal (ADR-100): the error, and the
// steps it names. Named so ADR-111's test can hold its keys.
// readDocument is `mrw read --json` (ADR-165): the read's standard output as it
// would have been, and the receipt mrw_read sends as content[1] for the same
// read, built by the same function.
type readDocument struct {
	Content string         `json:"content"`
	Receipt map[string]any `json:"receipt"`
}

// readRefusal is a `mrw read --json` refusal before the answer (ADR-165), as
// checkRefusal is check's.
type readRefusal struct {
	Error string `json:"error"`
}

type checkRefusal struct {
	Error string             `json:"error"`
	Then  *check.StepsResult `json:"then,omitempty"`
}

// stepFlag is the value of --then or --then-sh (ADR-092). Both append to one
// list as the parser meets them, so the list is in command-line order, and a
// value is never split — a comma included.
type stepFlag struct {
	list  *[]check.Step
	adhoc bool
	vals  []string
}

// Set appends one step.
func (f *stepFlag) Set(v string) error {
	f.vals = append(f.vals, v)
	if f.adhoc {
		*f.list = append(*f.list, check.Step{Command: v, AdHoc: true})
	} else {
		*f.list = append(*f.list, check.Step{Name: v})
	}
	return nil
}

// Get is the flag itself, so cmd.Generic hands back its values.
func (f *stepFlag) Get() any { return f }

// String is every value this flag was given.
func (f *stepFlag) String() string {
	// Shown, not raw: urfave prints this as the flag's default in --help, so a
	// value holding control bytes reached the terminal there (review of #269).
	vals := make([]string, 0, len(f.vals))
	for _, v := range f.Values() {
		vals = append(vals, writer.Shown(v))
	}
	return strings.Join(vals, " ")
}

// Values is every value this flag was given, in order.
func (f *stepFlag) Values() []string { return f.vals }

// thenFlag is --then NAME: a step the project declares under "steps".
func thenFlag(list *[]check.Step) cli.Flag {
	return &cli.GenericFlag{
		Name:  "then",
		Value: &stepFlag{list: list},
		Usage: "after a landed write and a passing check, run the step `NAME` declared in .quality-harness.json \"steps\" " +
			"(repeatable; runs in command-line order with --then-sh; the first that does not pass stops the rest)",
	}
}

// thenShFlag is --then-sh CMD: an ad-hoc shell step the caller writes.
func thenShFlag(list *[]check.Step) cli.Flag {
	return &cli.GenericFlag{
		Name:  "then-sh",
		Value: &stepFlag{list: list, adhoc: true},
		Usage: "like --then, but run `CMD` with sh -c as given (repeatable). " + guide.ThenShCaveat(),
	}
}

// jsonPathValue is the value of --json-path (ADR-178). Each use is kept whole
// and in command-line order: a comma belongs to the expression grammar, and
// urfave's slice flag would cut at it before read.ParseProjections could honour
// a quoted step.
type jsonPathValue struct{ vals []string }

// Set keeps one use of the flag.
func (f *jsonPathValue) Set(v string) error {
	f.vals = append(f.vals, v)
	return nil
}

// Get is the flag itself, so cmd.Generic hands back its values.
func (f *jsonPathValue) Get() any { return f }

// String is every value this flag was given, shown as stepFlag shows its own.
func (f *jsonPathValue) String() string {
	vals := make([]string, 0, len(f.vals))
	for _, v := range f.vals {
		vals = append(vals, writer.Shown(v))
	}
	return strings.Join(vals, " ")
}

// Values is every value this flag was given, in order.
func (f *jsonPathValue) Values() []string { return f.vals }

// jsonPathFlag is --json-path EXPR[,EXPR…] on read.
func jsonPathFlag(v *jsonPathValue) cli.Flag {
	return &cli.GenericFlag{
		Name:  "json-path",
		Value: v,
		Usage: "print the value at each JSON path `EXPR` (ADR-155's grammar, $.a.b[0] or $[\"a b\"]; comma-separated or repeated) of each JSON file, or of - for standard input, as one line `EXPR = VALUE` with the value compacted, instead of the lines that hold it; " +
			"it licenses and records nothing, so read the path address (f.json:$.a.b) before editing; a miss prints `NOT FOUND`, a file that is not one JSON document `NOT JSON`, and either exits 1. Not with --grep, --ast-grep, --sub, --rewrite, --count, --stat, --quiet, -N, -C, --max-lines, --if-changed or a range",
	}
}

// askedStepsError refuses what can be judged of the steps before anything is
// read: a --then-sh with no command, which would run nothing and pass, a
// --then-sh holding {files} or {packages}, which mrw expands only in
// scoped_check (ADR-094), and a sequence already MaxStepDepth deep, where a
// step that runs mrw again would recurse without end (ADR-092 T5, ADR-095).
func askedStepsError(steps []check.Step) error {
	if len(steps) == 0 {
		return nil
	}
	if err := check.DepthRefusal("--then and --then-sh are"); err != nil {
		return err
	}
	for _, s := range steps {
		if s.AdHoc && strings.TrimSpace(s.Command) == "" {
			return errors.New("--then-sh needs a command: a step with none would run nothing and pass")
		}
		if tok := check.Placeholder(s.Command); s.AdHoc && tok != "" {
			return fmt.Errorf("--then-sh %q: its command holds %s, which mrw expands only in scoped_check; a step runs as written", s.Command, tok)
		}
	}
	return nil
}

// stepsExit is the exit a stopped sequence earns (ADR-092 Decision 6): 2 for a
// step that could not start, 3 for one that ran — or was interrupted — and did
// not pass. nil when every step asked for passed or none ran.
func stepsExit(r *check.StepsResult, lead, tail string) error {
	s, i, ok := writer.StepStop(r)
	if !ok {
		return nil
	}
	which := fmt.Sprintf("step %d (%s)", i+1, stepLabel(s.Step))
	if s.Status == check.StepCouldNotStart {
		return cli.Exit(lead+which+" could not start: "+strings.TrimPrefix(s.Skipped, "could not start: "), exitUsage)
	}
	return cli.Exit(lead+which+" did not pass"+tail, exitCheckFailed)
}

// stepLabel names a step: its declared name, or --then-sh for an ad-hoc one.
func stepLabel(s check.Step) string {
	if s.AdHoc {
		return "--then-sh"
	}
	return writer.Shown(s.Name)
}

// reportSteps prints one line per step after the check's report, with the tail
// of any step that stopped the sequence (ADR-092) and the last non-empty line
// of each step that passed (ADR-094).
func reportSteps(w *os.File, r *check.StepsResult) {
	if r == nil {
		return
	}
	out := bufio.NewWriter(w)
	defer func() { _ = out.Flush() }()
	for i, s := range r.Steps {
		head := fmt.Sprintf("then %d/%d %s: %s", i+1, len(r.Steps), stepLabel(s.Step), writer.Shown(s.Command))
		switch s.Status {
		case check.StepPass:
			fmt.Fprintf(out, "%s — PASS\n", head)
			// A pass shows its last non-empty line, so a masked failure such as
			// `go test ./... || true` shows its FAIL (ADR-094). A passing step
			// whose output ran past the tail keeps its log (ADR-080), so the
			// receipt says where (ADR-092 T4), counting every line above the
			// one shown. One whose tail cut a line keeps it too (ADR-104), and
			// the receipt names it after the cut line: the bytes the marker
			// counts are there.
			last := lastShown(s.Tail)
			if s.Truncated > 0 {
				fmt.Fprintf(out, "... %d earlier line(s) in %s\n", s.Truncated+max(last, 0), s.OutputFile)
			}
			if last >= 0 {
				fmt.Fprintf(out, "  | %s\n", s.Tail[last])
			}
			if s.Truncated == 0 && s.OutputFile != "" {
				fmt.Fprintf(out, "full output: %s\n", s.OutputFile)
			}
			continue
		case check.StepNotRun:
			fmt.Fprintf(out, "%s — NOT RUN\n", head)
			continue
		case check.StepFail:
			fmt.Fprintf(out, "%s — FAIL exit %d\n", head, s.ExitCode)
		default:
			fmt.Fprintf(out, "%s — %s: %s\n", head, strings.ToUpper(strings.ReplaceAll(s.Status, "_", " ")), strings.TrimPrefix(s.Skipped, "could not start: "))
		}
		if s.Truncated > 0 {
			fmt.Fprintf(out, "... %d earlier line(s) in %s\n", s.Truncated, s.OutputFile)
		}
		for _, l := range s.Tail {
			fmt.Fprintf(out, "  | %s\n", l)
		}
		for j := len(s.Tail) - 1; j >= 0; j-- {
			if strings.TrimSpace(s.Tail[j]) != "" {
				fmt.Fprintf(out, "then last: %s\n", s.Tail[j])
				break
			}
		}
		if s.OutputFile != "" && s.Truncated == 0 {
			fmt.Fprintf(out, "full output: %s\n", s.OutputFile)
		}
	}
	if r.Pruned > 0 {
		fmt.Fprintf(out, "removed %d check log(s) older than %d days from %s\n", r.Pruned, int(check.LogRetention.Hours()/24), os.TempDir())
	}
}

// lastShown is the index of tail's last non-empty line, or -1 when it has none.
func lastShown(tail []string) int {
	for j := len(tail) - 1; j >= 0; j-- {
		if strings.TrimSpace(tail[j]) != "" {
			return j
		}
	}
	return -1
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

Entries are read specs, so "mrw read" with no arguments returns these ranges,
each with the line after it as any ranged read serves it (ADR-161), and
"mrw check" runs the project's check scoped to these files.`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			if err := refusePaddedArgs(cmd); err != nil {
				return err
			}
			root := cmd.Root().String("root")
			args := cmd.Args().Slice()
			verb := ""
			if len(args) > 0 {
				verb, args = args[0], args[1:]
			}
			// ADR-079: a verb reads, changes and writes the set under its lock,
			// so two iter adds racing each other both keep their entry.
			change := func(set *iter.Set) error {
				switch verb {
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
					var missing, refused, unstatable []string
					for _, a := range args {
						full, err := rooted.Resolve(root, iter.Path(a))
						if err != nil {
							// The boundary's own reason: a missing root, a
							// directory spelling and a path outside all read as
							// "outside the root" before (review of #237).
							refused = append(refused, err.Error())
							continue
						}
						if _, err := os.Stat(full); err != nil {
							// ADR-096: only a path that is not there is "no
							// such file"; its hint is about word splitting. A
							// link loop or a denied directory is the OS's own
							// sentence, which names the path.
							if errors.Is(err, os.ErrNotExist) {
								missing = append(missing, a)
							} else {
								unstatable = append(unstatable, err.Error())
							}
						}
					}
					if len(refused) > 0 {
						return cli.Exit(fmt.Sprintf("%s — the working set feeds `mrw read` and `mrw check`, which "+
							"refuse such a path, so an entry like this could never be served", strings.Join(refused, "; ")), exitUsage)
					}
					if len(unstatable) > 0 {
						return cli.Exit(strings.Join(unstatable, "; "), exitUsage)
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
					*set = iter.Set{}
				case "note":
					set.Note = strings.Join(args, " ")
				default:
					return cli.Exit(fmt.Sprintf("unknown iter verb %q (want add, rm, clear or note)", verb), exitUsage)
				}
				return nil
			}
			var (
				set iter.Set
				err error
			)
			if verb == "" {
				set, err = iter.Load(root)
			} else {
				set, err = iter.Update(root, change)
			}
			if err != nil {
				var exit cli.ExitCoder
				if errors.As(err, &exit) {
					return err
				}
				return exitWith(err)
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
	var asked []check.Step // --then / --then-sh, in order (ADR-092)
	return &cli.Command{
		Name: "check",
		// ADR-099: a PATH named help is a PATH.
		HideHelpCommand: true,
		Usage:           "run the project's check, scoped to the working set or to the given paths",
		ArgsUsage:       "[PATH...]",
		Description: `The command comes from .quality-harness.json ("check" for the whole project,
"scoped_check" for a narrow run, with {packages}, {files} and {dirs} expanded; a {packages}-only one runs what it can place and names the rest). With no
such file, a Go project falls back to "go test ./..." — and the output says the
command was INFERRED, because an inferred check can be red on a tree you never
touched, which is a finding about the machine and not about your change.`,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "emit the result as JSON"},
			&cli.BoolFlag{Name: "full", Usage: "run the whole-project check, ignoring the working set; it takes no PATH"},
			thenFlag(&asked),
			thenShFlag(&asked),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Under --json every refusal is one document, as write's is
			// (ADR-072, ADR-092 T4, ADR-100): the error, and the steps that
			// document names — none before any was due, every one not_run
			// when the check could not start.
			var then *check.StepsResult
			refuse := func(err error) error {
				if cmd.Bool("json") {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					enc.SetEscapeHTML(false)
					_ = enc.Encode(checkRefusal{Error: err.Error(), Then: then})
				} else {
					reportSteps(os.Stdout, then)
				}
				return exitWith(err)
			}
			// ADR-095: mrw check asks for nothing but a check, so at the depth
			// limit it is refused in every form, before anything runs — and
			// first, before an argument is judged or the working set is read,
			// so a caller fixing the usage does not meet it next (ADR-100).
			if err := check.DepthRefusal("a check is"); err != nil {
				return refuse(err)
			}
			if err := refusePaddedArgs(cmd); err != nil {
				return refuse(err)
			}
			root := cmd.Root().String("root")
			paths := cmd.Args().Slice()
			// ADR-099: --full runs the whole project, so a PATH beside it would
			// be dropped in silence — refused before anything runs, as ADR-096
			// refuses a named path a finder would drop.
			if cmd.Bool("full") && len(paths) > 0 {
				return refuse(errors.New("--full runs the whole project; it takes no PATH"))
			}
			if len(paths) == 0 && !cmd.Bool("full") {
				set, err := iter.Load(root)
				if err != nil {
					return refuse(err)
				}
				paths = set.Paths()
			}
			cfg, err := check.Load(root)
			if err != nil {
				return refuse(err)
			}
			if err := askedStepsError(asked); err != nil {
				return refuse(err)
			}
			if err := writer.ResolveSteps(cfg, asked, ""); err != nil {
				return refuse(err)
			}
			res, err := check.Run(ctx, root, cfg, paths)
			if err != nil {
				// A refused scope comes back before a command is chosen, with no
				// Command, and names no step; so does any refusal with no step
				// asked (ADR-092). The one error after a command is chosen is
				// the check's log that could not be created: the check could not
				// start, so the steps asked for are named not_run (ADR-092
				// Decision 5, "the check could not start"; the 2026-09-29 gap
				// survey, C2). Either way it is one document under --json
				// (ADR-100).
				if res.Command != "" && len(asked) > 0 {
					then = writer.RunSteps(ctx, root, cfg, asked, false)
				}
				return refuse(err)
			}
			if len(asked) > 0 {
				then = writer.RunSteps(ctx, root, cfg, asked, res.OK())
			}
			if cmd.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				if err := enc.Encode(checkReceipt{Result: res, Then: then}); err != nil {
					return exitWith(err)
				}
			} else {
				reportCheck(os.Stdout, &res)
				reportSteps(os.Stdout, then)
			}
			if check.StoppedBeforeStart(res) {
				return cli.Exit("the check "+stoppedBefore(res), exitCheckFailed)
			}
			if !res.Ran {
				// Not exit 3: nothing ran, so nothing failed. Reporting this as
				// a failing check would tell the caller to go read output that
				// does not exist.
				return cli.Exit("no check could run: "+res.Skipped+
					declareAdvice(res.Skipped), exitUsage)
			}
			if !res.OK() {
				return cli.Exit("check did not pass", exitCheckFailed)
			}
			return stepsExit(then, "", "")
		},
	}
}

// reportCheck prints a check result. It always states the command and the real
// exit code: a summary that omits either invites the reader to trust a green
// that was never computed.
// reportDrift names each file that changed while the write's check ran
// (ADR-112). Worded for what is known — the bytes moved — and not for who moved
// them: the check may have done it itself.
func reportDrift(w *os.File, paths []string, others int) {
	for _, p := range paths {
		fmt.Fprintf(w, "drift: %s changed while the check ran\n", p)
	}
	if others > 0 {
		fmt.Fprintf(w, "drift: %d other write(s) landed in this checkout while the check ran\n", others)
	}
}

func reportCheck(w *os.File, r *check.Result) {
	if r == nil {
		return
	}
	out := bufio.NewWriter(w)
	defer func() { _ = out.Flush() }()
	// ADR-152: a slow whole-project check says how to scope it, last, under
	// its verdict; this runs before the flush above. ADR-164: so does the line
	// naming the files a scoped check left out, which a scoped run has instead
	// of a hint.
	defer func() {
		if l := r.OutsideLine(); l != "" {
			fmt.Fprintf(out, "%s\n", l)
		}
		if r.Hint != "" {
			fmt.Fprintf(out, "hint: %s\n", r.Hint)
		}
	}()

	// ADR-008: a delete says what it removed — here the old check logs this
	// run pruned from the temp directory (ADR-080).
	if r.Pruned > 0 {
		fmt.Fprintf(out, "removed %d check log(s) older than %d days from %s\n", r.Pruned, int(check.LogRetention.Hours()/24), os.TempDir())
	}

	if !r.Ran {
		// A log kept on this path — its removal failed — is named too: the
		// JSON carried it and the human report did not (the review of #241).
		named := ""
		if r.OutputFile != "" {
			named = " — full output: " + r.OutputFile
		}
		if w := writer.StopWord(r.Skipped); w != "" {
			fmt.Fprintf(out, "check %s: %s%s\n", w, r.Skipped, named)
			return
		}
		fmt.Fprintf(out, "check SKIPPED: %s%s\n", r.Skipped, named)
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
	if !r.OK() {
		for i := len(r.Tail) - 1; i >= 0; i-- {
			if strings.TrimSpace(r.Tail[i]) != "" {
				fmt.Fprintf(out, "check last: %s\n", r.Tail[i])
				break
			}
		}
	}
	verdict := "PASS"
	if !r.OK() {
		verdict = "FAIL"
	}
	if w := writer.StopWord(r.Skipped); w != "" {
		verdict = w
	}
	if r.Skipped != "" {
		// ADR-080: a timed-out or interrupted check keeps its log, and the
		// report named nowhere to find it.
		named := ""
		if r.OutputFile != "" {
			named = " — full output: " + r.OutputFile
		}
		fmt.Fprintf(out, "check %s (exit %d, %dms) — %s%s\n", verdict, r.ExitCode, r.DurationMS, r.Skipped, named)
		return
	}
	if r.OutputFile == "" {
		// Cleaned up: the check passed and withheld nothing, so there is no
		// file to point at and naming an empty path would read as a bug.
		fmt.Fprintf(out, "check %s (exit %d, %dms)\n", verdict, r.ExitCode, r.DurationMS)
		return
	}
	fmt.Fprintf(out, "check %s (exit %d, %dms) — full output: %s\n", verdict, r.ExitCode, r.DurationMS, r.OutputFile)
}

// report renders the receipt for a human. It is terse on purpose: the caller is
// usually an agent paying for every line it reads back.
func report(w *os.File, res apply.Result, quiet bool) {
	reportMode(w, res, quiet, false)
}

// reportAs prints the receipt the write's flags ask for: the default, --quiet's,
// or --terse's one line per file (ADR-179).
func reportAs(w *os.File, res apply.Result, cmd *cli.Command) {
	if cmd.Bool("terse") {
		reportMode(w, res, false, true)
		return
	}
	report(w, res, cmd.Bool("quiet"))
}

func reportMode(w *os.File, res apply.Result, quiet, terse bool) {
	out := bufio.NewWriter(w)
	defer func() { _ = out.Flush() }()

	for _, h := range res.Hunks {
		switch {
		case h.Status == apply.StatusFailed:
			fmt.Fprintf(out, "FAIL %s %s %s (plan line %d): %s\n", h.Path, h.Addr, h.Op, h.SrcLine, h.Reason)
		case quiet:
		case h.Status == apply.StatusSkipped:
			fmt.Fprintf(out, "skip %s %s %s\n", h.Path, h.Addr, h.Op)
		default:
			// ADR-179: --terse says its line per file below; a hunk keeps its own
			// row for the advisory rows that sit under it, for what a delete removed
			// (ADR-008) and for what a from= copy copied (ADR-167).
			if terse && h.Balance == "" && h.Closer == "" && len(h.Echo) == 0 && h.Op != "delete" && h.BodyFrom == "" {
				continue
			}
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
			// ADR-167: a copy says what it copied, keyed on the form (BodyFrom is
			// set on every from= hunk), so a wrong source range shows here.
			if h.BodyFrom != "" {
				bounds += fmt.Sprintf(" copied %s from %q to %q", h.BodyFrom, h.CopiedFirst, h.CopiedLast)
			}
			fmt.Fprintf(out, "ok   %s %s %s  -%d +%d%s\n", h.Path, h.Addr, h.Op, h.Removed, h.Added, bounds)
			if !terse {
				for _, line := range preview.Lines(h.Before, h.After, h.Start) {
					fmt.Fprintln(out, line)
				}
			}
			if h.Balance != "" {
				// ADR-054: the delta is a row under the ok line, like Echo.
				// It never fails the hunk; it says the closer count moved.
				fmt.Fprintf(out, "     balance %s (delimiters in the replaced lines vs the body; not a checker)\n", h.Balance)
			}
			if h.Closer != "" {
				// ADR-119: advice, like the balance row; the hunk stays ok.
				fmt.Fprintf(out, "     closer %s (the closer the file already had? read past the body)\n", h.Closer)
			}
			for _, line := range h.Echo {
				fmt.Fprintln(out, line)
			}
		}
	}
	// ADR-105: what the run made in the tree and did not take away. Printed
	// even under --quiet: the caller did not ask for it to be there.
	for _, p := range res.LeftBehind {
		fmt.Fprintf(out, "left behind: %s\n", p)
	}
	if !quiet {
		// ADR-076: the directories this plan made, parents first, spelled
		// with the separator that names a directory.
		for _, d := range res.DirsCreated {
			fmt.Fprintf(out, "created %s/\n", d)
		}
		files := res.Files
		if terse {
			terseFiles(out, res)
			files = nil
		}
		for _, f := range files {
			if !f.Written {
				continue
			}
			if f.Removed {
				// A removed path has no sha after it; the one it had is what a
				// caller can check against, and a rename says where it went.
				moved := ""
				if f.RenamedTo != "" {
					moved = ", renamed to " + f.RenamedTo
				}
				fmt.Fprintf(out, "removed %s  %dL -> 0L  was sha %s%s\n", f.Path, f.LinesFrom, short(f.SHABefore), moved)
				continue
			}
			verb := "wrote"
			if f.Created {
				verb = "created"
			}
			through := ""
			if f.Target != "" {
				through = "  target " + f.Target
			}
			fmt.Fprintf(out, "%s %s  %dL -> %dL  sha %s%s\n", verb, f.Path, f.LinesFrom, f.LinesTo, short(f.SHAAfter), through)
		}
	}

	// ADR-066: a plan that wrote some files and then failed a commit step is
	// neither applied nor nothing-written. Tested first, or the partial case
	// would read NOTHING WRITTEN — the opposite lie of the one it replaces.
	wrote := false
	for _, f := range res.Files {
		if f.Written {
			wrote = true
		}
	}
	state := "applied"
	switch {
	case res.Failed > 0 && wrote:
		state = "PARTIALLY APPLIED"
	case res.Failed > 0:
		state = "NOTHING WRITTEN"
	case res.DryRun:
		state = "dry run, nothing written"
	}
	// ADR-055: the advisory count rides on the one line every caller reads,
	// zero included — a column that appears only when non-zero is a column
	// the reader learns does not exist (ADR-054's failed_check lesson).
	// ADR-119: hints ride on the same line but only when there are some, so
	// every summary a caller already parses stays byte-identical.
	hints := ""
	if res.Hints > 0 {
		hints = fmt.Sprintf(", %d %s", res.Hints, plural(res.Hints, "hint", "hints"))
	}
	fmt.Fprintf(out, "%d hunk(s), %d file(s), %d failed, %d %s%s — %s\n",
		len(res.Hunks), len(res.Files), res.Failed, res.Advisories, plural(res.Advisories, "advisory", "advisories"), hints, state)
}

// prune removes the state directories whose checkout is gone, and SAYS what it
// removed — ADR-008's rule, which until ADR-034 applied only to lines in a file.
//
// It reports the kept-but-unidentifiable entries too. An entry mrw declines to
// touch because it cannot tell what it is looks, from outside, exactly like an
// entry mrw never looked at; the only difference a caller can see is this line.
func prune(root string, dry bool) error {
	entries, err := state.Entries()
	if err != nil {
		return exitWith(err)
	}
	removed, err := state.Prune(root, entries, dry)
	if err != nil {
		return exitWith(err)
	}

	// Two words, not one: "removed" prefixes a line about ONE directory and
	// "1 of 4 … would remove" does not parse as English.
	verb, summary := "removed", "removed"
	if dry {
		verb, summary = "would remove", "would be removed"
	}
	var bytes int64
	failed := 0
	for _, e := range removed {
		if e.Err != nil {
			fmt.Printf("FAIL %s  %v\n", e.Dir, e.Err)
			failed++
			continue
		}
		fmt.Printf("%s  %s  %s\n", verb, e.Dir, e.Root)
		bytes += e.Bytes
	}

	unidentified := 0
	for _, e := range entries {
		if !e.Identified {
			unidentified++
		}
	}
	// A run that removed nothing says so. Printing nothing would be
	// indistinguishable from a run that never happened.
	//
	// ⚠ "of file content", not a disk figure. This sums the sizes of the files
	// removed; the space the filesystem actually returns is LARGER, because a
	// 40-byte ledger occupies a whole block and each directory costs one too.
	// Measured on one base: 10.7 MB of content across 65,235 files sat in
	// 256 MB of disk. Block size is not something mrw can portably ask about,
	// so it reports what it can count and names the unit (ADR-034).
	fmt.Printf("%d of %d state director%s %s, %d byte(s) of file content\n",
		len(removed)-failed, len(entries), plural(len(entries), "y", "ies"), summary, bytes)
	if unidentified > 0 {
		fmt.Printf("# %d kept: no readable `root` marker naming an absolute path, so mrw cannot say what %s\n",
			unidentified, plural(unidentified, "it is", "they are"))
	}
	if failed > 0 {
		return cli.Exit(fmt.Sprintf("mrw seen --prune: %d director%s could not be removed",
			failed, plural(failed, "y", "ies")), exitUsage)
	}
	return nil
}

// forget is `mrw seen --forget` (ADR-175): a newer mrw's ledger is refused first,
// so that nothing changes; then the pending MCP acknowledgements go, since an ack
// would otherwise promote one into the ledger just emptied. The pending store is
// per root and the ledger per checkout, so the checkout's top is emptied too: a
// server started there is not the root a hook ran from.
func forget(root string) error {
	if writer, newer, err := seen.Newer(root); err == nil && newer {
		// main prints "mrw: " itself.
		return exitWith(errors.New(strings.TrimPrefix(seen.NewerNotice(writer), "mrw: ")))
	}
	roots := []string{root}
	if top, _ := state.Checkout(root); top != root {
		roots = append(roots, top)
	}
	for _, r := range roots {
		if err := mcp.ForgetPending(r); err != nil {
			return exitWith(err)
		}
	}
	n, err := seen.Forget(root)
	if err != nil {
		return exitWith(err)
	}
	if n == 0 {
		fmt.Println("forgot 0 file(s)")
		return nil
	}
	fmt.Printf("forgot %d file(s): a write now needs a fresh read of the lines it touches\n", n)
	return nil
}

// forgettableRoot is nil when root names an existing directory. rooted.Abs
// answers a missing root, but a root below a regular file gets ENOTDIR from its
// symlink walk, which it lets through, so the resolved root is statted here
// (ADR-175).
func forgettableRoot(root string) error {
	abs, err := rooted.Abs(root)
	if err != nil {
		return err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("the root %s: %w", root, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("the root %s is not a directory", root)
	}
	return nil
}

// plural picks the singular or the plural form for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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

// refusePaddedArgs refuses a positional argument the argument parser trimmed.
// urfave/cli v3.11.0 trims every positional before `--` (command_parse.go:81,
// :117), so `mrw read 'x '` acted on x, a file the caller did not name
// (ADR-069). The untrimmed token survives only in the root's raw tail. A token
// that differs from its trim, and whose trim is one of this command's
// arguments, is refused with the `--` that reaches it.
//
// A flag's own value is skipped: the parser keeps a separate value as given
// (command_parse.go:182), and a "--" consumed as a value is not the terminator,
// which is how `read --grep -- 'x '` got past the first version (Codex review
// of v1.25.0, ADR-069 T5). An attached value is checked by padAttached. A flag
// name is classified TRIMMED, as the parser reads it: looked up as typed, a
// padded boolean name (`'--no-numbers '`) read as value-taking and hid the
// padded path after it (Codex review of PR #222, ADR-069 T6).
func refusePaddedArgs(cmd *cli.Command) error {
	args := cmd.Args().Slice()
	// A note is free text, not a path, so its words are never judged — but
	// the flags beside it still are: `iter note --root='dir ' x` reached dir
	// while the whole exemption returned here (ADR-069 T8).
	noteText := cmd.Name == "iter" && len(args) > 0 && args[0] == "note"
	got := make(map[string]bool, len(args))
	for _, a := range args {
		got[a] = true
	}
	// The iter refusal keeps its verb: `mrw iter -- 'x '` makes the path the verb.
	prefix := cmd.Name
	if cmd.Name == "iter" && len(args) > 0 {
		prefix += " " + args[0]
	}
	raw := cmd.Root().Args().Slice()
	if len(raw) > 0 {
		raw = raw[1:] // the subcommand's own name
	}
	kinds := flagKinds(cmd.Lineage()...)
	positionals := 0
	value := false
	for _, tok := range raw {
		if value {
			value = false
			continue
		}
		role, next := flagRole(tok, kinds)
		if role == roleTerminator {
			break // what follows is kept as given
		}
		if role == roleFlag {
			if err := padAttached(tok); err != nil {
				return err
			}
			value = next
			continue
		}
		// A positional, or the token the parser stops at. Of the two stop
		// tokens only the lone "-" is trimmed and kept, so only it is judged
		// before the walk ends (`write ' - '` read stdin; ADR-069 T9). A
		// single dash before a non-letter is kept as given with everything
		// after it, so judging it against a sibling's trimmed spelling refused
		// `read ' -1= ' '-1='` with both files present (ADR-069 T10).
		if role == roleStop && strings.TrimSpace(tok) != "-" {
			break
		}
		// An iter VERB is not a path either: `iter 'add ' x` names the verb
		// with padding the parser trims, and nothing reaches a file by it
		// (found by the random differential test, ADR-069 T11).
		if cmd.Name == "iter" && positionals == 0 {
			positionals++
		} else if !noteText {
			if t := strings.TrimSpace(tok); t != tok && t != "" && got[t] {
				if cmd.Name == "write" && cmd.String("create") != "" {
					// --create took the "--" as its value; the path is --create's own
					// argument, and a "--" before it drops the flag.
					return cli.Exit(fmt.Sprintf("'%s' has edge whitespace the argument parser strips; give it to --create as its own argument, without a --: mrw write --create %s%s", tok, posixQuote(tok), cmdExeForm("mrw write --create ", tok)), exitUsage)
				}
				head := fmt.Sprintf("mrw %s -- ", prefix)
				fix := head + posixQuote(tok)
				// ADR-164 T5: a padded path after `read --grep -v` is refused here,
				// before the read, so the table's pointer for the flag the parsed
				// pattern is (the last --grep, attached or not) is said here too.
				// read only: the pointers name read's flags.
				pointer := ""
				if cmd.Name == "read" {
					if s := patternFlagHint(cmd.String("grep")); s != "" {
						pointer = " — " + s + " (mrw took the value after --grep as the pattern)"
					}
				}
				return cli.Exit(fmt.Sprintf("'%s' has edge whitespace the argument parser strips; "+
					"put -- before the path: %s%s%s", tok, fix, cmdExeForm(head, tok), pointer), exitUsage)
			}
		}
		if role == roleStop {
			break // what follows is dropped
		}
	}
	return nil
}

// takesValue maps every name of every flag in flags to whether the parser
// consumes the NEXT token as its value: false for a boolean, true otherwise.
// The framework's own --help/-h and --version/-v are booleans it adds itself.
func takesValue(flags []cli.Flag) map[string]bool {
	m := map[string]bool{"help": false, "h": false, "version": false, "v": false}
	for _, f := range flags {
		_, isBool := f.(*cli.BoolFlag)
		for _, n := range f.Names() {
			m[n] = !isBool
		}
	}
	return m
}

// flagKinds is takesValue over every flag the parser accepts for the command
// at lineage[0] (lineage runs from it up to the root, as cmd.Lineage() does):
// its own flags, then each ancestor's non-local ones — an ancestor flag is
// skipped WHOLE when any of its names is already one of the command's own
// (command_parse.go:43-57). That is why `write` and `iter` take --root after
// the verb while `read`, whose -C is context, does not. Neither guard knew
// it, so `write --root -- --root='dir '` ended at the -- the root flag
// consumed, and `iter --root ' x' add x` was falsely refused (Codex review of
// PR #222, second round; ADR-069 T7).
func flagKinds(lineage ...*cli.Command) map[string]bool {
	kinds := takesValue(lineage[0].Flags)
	own := map[string]bool{}
	for _, f := range lineage[0].Flags {
		for _, n := range f.Names() {
			own[n] = true
		}
	}
	for _, anc := range lineage[1:] {
		for _, f := range anc.Flags {
			if lf, ok := f.(cli.LocalFlag); ok && lf.IsLocal() {
				continue
			}
			clash := false
			for _, n := range f.Names() {
				if own[n] {
					clash = true
					break
				}
			}
			if clash {
				continue
			}
			_, isBool := f.(*cli.BoolFlag)
			for _, n := range f.Names() {
				kinds[n] = !isBool
			}
		}
	}
	return kinds
}

// tokenRole is how the parser reads one token (command_parse.go:112-139).
type tokenRole int

const (
	rolePositional tokenRole = iota
	roleFlag                 // begins with "-" after the parser's trim
	roleTerminator           // a bare "--": what follows is positional, kept as given
	roleStop                 // a single "-" before a non-letter: parsing stops, the rest is kept as given
)

// flagRole classifies tok the way the parser does. The parser trims a token
// before it looks at it (command_parse.go:81), so the name is looked up
// trimmed: looked up as typed, a padded boolean name (`'--no-numbers '`) read
// as value-taking and hid the padded path after it (ADR-069 T6). A single
// dash before a non-letter is where the parser stops and keeps every
// remaining token as given, so a file named ` -1= ` is served, not refused
// as an attached value (ADR-069 T7). consumesNext is true for a known flag
// that takes a separate value; an attached value (name=value) consumes
// nothing, and an unknown name is left to the parser, which refuses it by
// name before any Action runs.
func flagRole(tok string, kinds map[string]bool) (role tokenRole, consumesNext bool) {
	t := strings.TrimSpace(tok)
	if t == "--" {
		return roleTerminator, false
	}
	if t == "-" {
		// The parser keeps a lone "-" as a positional and ends its parse
		// there, DROPPING every token after it (command_parse.go:123-125):
		// `write - '--format=plan '` reads stdin with the default format, so
		// the token the guard refused was one the parser never saw
		// (ADR-069 T8).
		return roleStop, false
	}
	if len(t) < 2 || t[0] != '-' {
		return rolePositional, false
	}
	if t[1] != '-' && !unicode.IsLetter(rune(t[1])) {
		return roleStop, false
	}
	name := strings.TrimLeft(t, "-")
	if strings.Contains(name, "=") {
		return roleFlag, false
	}
	return roleFlag, kinds[name]
}

// padAttached refuses an attached flag value that ends in whitespace. urfave
// trims the whole token, so --files-from='list ' opened list (ADR-069 T5). The
// trim is strings.TrimSpace, so ANY trailing whitespace counts: checked for
// space and tab only, --files-from=$'list\n' still opened list (ADR-069 T6).
func padAttached(tok string) error {
	name, val, ok := strings.Cut(tok, "=")
	if !ok || strings.TrimRightFunc(tok, unicode.IsSpace) == tok {
		return nil
	}
	head := name + " "
	fix := head + posixQuote(val)
	return cli.Exit(fmt.Sprintf("'%s' ends in whitespace the argument parser strips; pass the value "+
		"as its own argument: %s%s", tok, fix, cmdExeForm(head, val)), exitUsage)
}

// refusePaddedFlagValues checks every attached flag value before the option
// terminator, root flags included; main calls it because a root flag never
// reaches a subcommand's raw tail (ADR-069 T5). It reads argv as the parser
// does — root flags and their values, the subcommand, its flags and their
// values — so a separate value that looks like a flag (`--files-from
// '--list= '`) is not judged, and a "--" a flag consumed does not end the walk
// (`--root -- --root='dir '` reached dir; Codex review of PR #222, ADR-069 T6).
// The flags in force below a subcommand include its ancestors' persistent
// ones, as the parser's do (ADR-069 T7).
func refusePaddedFlagValues(root *cli.Command, argv []string) error {
	lineage := []*cli.Command{root}
	kinds, cmds := flagKinds(lineage...), root.Commands
	value, optionsOff := false, false
	for _, tok := range argv {
		if value {
			value = false
			continue
		}
		role, next := rolePositional, false
		if !optionsOff {
			role, next = flagRole(tok, kinds)
		}
		switch role {
		case roleStop:
			return nil // the parser ends here; the rest is kept as given, or dropped
		case roleTerminator:
			// Options are over for THIS command only. At one with
			// subcommands the parser still dispatches the first positional,
			// whose own parse starts afresh (command_run.go:282-315), so
			// `mrw -- iter note --root='dir '` reached dir while the walk
			// ended here (Codex review of PR #222, third round; ADR-069 T8).
			if len(cmds) == 0 {
				return nil
			}
			optionsOff = true
			continue
		case roleFlag:
			if err := padAttached(tok); err != nil {
				return err
			}
			value = next
			continue
		}
		// A positional: at the root it names the subcommand whose flags are
		// in force from here, its ancestors' persistent ones included; below
		// one it is a path for that command's own guard. An unknown name is
		// the parser's to refuse. After a "--" the parser hands the token on
		// as given; otherwise it has trimmed it (command_parse.go:81).
		if len(cmds) == 0 {
			continue
		}
		name := tok
		if !optionsOff {
			name = strings.TrimSpace(tok)
		}
		sub := subcommand(cmds, name)
		if sub == nil {
			return nil
		}
		lineage = append([]*cli.Command{sub}, lineage...)
		kinds, cmds, optionsOff = flagKinds(lineage...), sub.Commands, false
	}
	return nil
}

// subcommand finds name among cmds, by name or alias.
func subcommand(cmds []*cli.Command, name string) *cli.Command {
	for _, c := range cmds {
		if c.Name == name {
			return c
		}
		for _, a := range c.Aliases {
			if a == name {
				return c
			}
		}
	}
	return nil
}

// specList reads a spec per line from a file, or from stdin when name is "-".
//
// It exists so any searcher composes with mrw in one call: `rg -l PAT | sed
// 's|$|:/PAT/|' | mrw read --files-from -`. The shell quoting that a composed
// argv needs is the failure this avoids — a newline-separated list from `grep
// -rl` collapses into ONE argument under ordinary word splitting, and mrw then
// faithfully reports a file named "a.go\nb.go" as unreadable.
//
// Blank lines are skipped and a leading '#' is a comment, so a generated list
// can carry provenance without the reader stripping it. Nothing else is
// interpreted: each remaining line is a spec, parsed exactly as an argv one is.
func specList(root, name string) ([]string, error) {
	var r io.Reader
	if name == "-" {
		r = os.Stdin
	} else {
		// ADR-077: the list is a shell argument, resolved against the working
		// directory, and never passed the boundary; a list naming mrw's own
		// ack store had its JSON quoted back in the parse error, checkpoint
		// ids and all (Codex review of #238).
		if abs, err := filepath.Abs(name); err == nil && rooted.InStateOrLinked(root, abs) {
			return nil, fmt.Errorf("--files-from %s is inside mrw's own state directory, or is a hard link to a file in it; mrw does not read its own files as input", name)
		}
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		r = f
	}
	return speclist.Parse(r, "--files-from "+name)
}

// maxLines is the read's per-SPEC cap, or nil when the flag was not given.
//
// ⚠ IsSet is the whole point: without it an absent flag and `--max-lines 0` are
// the same input, which is how zero came to mean unlimited — and why nothing was
// reported withheld for a cap the tool never applied (ADR-033).
func maxLines(cmd *cli.Command) *int {
	if !cmd.IsSet("max-lines") {
		return nil
	}
	n := cmd.Int("max-lines")
	return &n
}

// maxMatches is the read's match budget, or nil when --max-matches was not
// given: IsSet again, since zero is a budget that shows no match (ADR-179).
func maxMatches(cmd *cli.Command) *int {
	if !cmd.IsSet("max-matches") {
		return nil
	}
	n := cmd.Int("max-matches")
	return &n
}

// receiptSep is the separator a receipt's paths are converted from (ADR-132);
// a variable so a test can drive `\` on any platform.
var receiptSep = filepath.Separator

// shown is res as a receipt shows it: every root-relative path with "/".
func shown(res apply.Result) apply.Result { return apply.Slashed(res, receiptSep) }

// shownPaths is paths as a receipt shows them, with "/".
func shownPaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = apply.Slash(p, receiptSep)
	}
	if len(out) == 0 {
		return paths
	}
	return out
}

// exitTwoLine is the line an exit-2 write ends on (ADR-132): the error, with
// the reason a FAIL row already printed replaced by a pointer to that row, so
// a long system error is not printed twice and every other cause it carries —
// the files already written, a ledger failure — stays.
func exitTwoLine(res apply.Result, err error) string {
	msg := err.Error()
	for _, h := range res.Hunks {
		if h.Status == apply.StatusFailed && h.Reason != "" && strings.Contains(msg, h.Reason) {
			line := strings.Replace(msg, h.Reason, "see its FAIL line", 1)
			// A commit's reason can be the whole error, path included: keep
			// the path, so the line still says which file stopped it.
			if !strings.Contains(line, h.Path) {
				line = h.Path + ": " + line
			}
			// Spelled as the receipt above spells it (the Codex review of #347).
			return apply.SlashIn(line, h.Path, receiptSep)
		}
	}
	return msg
}
