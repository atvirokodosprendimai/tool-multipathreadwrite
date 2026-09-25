// Package check runs a project's own verification immediately after an edit,
// scoped to what the edit touched.
//
// The point is round trips. Edit, run the tests, read the output is three calls
// and three result blocks; chaining the check to the write makes it one. The
// same habit that makes `gofmt -l x && go vet ./... && git commit` one call
// applies here — the verification and the thing it gates belong in one output.
//
// Two rules constrain the implementation, and both come from checks that lied:
//
//  1. The exit status is never inferred from what was printed. Output goes to a
//     file, a bounded tail is shown, and the process's real code is reported. A
//     tail in the pipeline would make the pipeline's status the tail's, so a
//     failing suite would surface as a pass.
//  2. A failing check never triggers a revert. The caller is told, loudly and
//     with a distinct status; undoing their edit for them can destroy work they
//     wanted to keep and inspect.
package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// Config is the project's declared verification, read from
// .quality-harness.json at the repository root.
type Config struct {
	// Check is the whole-project command, used when no narrower scope can be
	// derived from the edited paths.
	Check string `json:"check"`
	// ScopedCheck is the narrow command. {packages} expands to the Go packages
	// containing the edited files, {files} to the edited paths themselves.
	//
	// Write the placeholder UNQUOTED. Each value is substituted already quoted
	// as one shell argument (see shellArgs), so quoting it again in the
	// template nests the quotes into the argument instead of ending them.
	ScopedCheck string `json:"scoped_check"`
	// TimeoutSeconds bounds the run. Zero means the built-in default.
	TimeoutSeconds int `json:"timeout_seconds"`
	// FenceTimeout is quality-harness's camelCase alias of TimeoutSeconds.
	// Load copies it into TimeoutSeconds when that field is unset, and
	// refuses when both are set to different values (ADR-059).
	FenceTimeout int `json:"fenceTimeout"`
	// TailLines is how many trailing lines of output to show. Zero means the
	// built-in default; the full output is always kept in a file.
	TailLines int `json:"tail_lines"`

	declared bool
}

// Declared reports whether the project actually stated its check, as opposed to
// this package inferring one. A caller should say which it used: a command
// inferred from the repository shape can be red on an untouched tree, and that
// finding is about the machine rather than about the edit.
func (c Config) Declared() bool { return c.declared }

const (
	defaultTimeout = 5 * time.Minute
	// A day, and far below the overflow point of an int64 nanosecond count.
	// Anything larger is indistinguishable from it for a check a caller is
	// waiting on.
	maxTimeoutSeconds = 24 * 60 * 60
	defaultTail       = 30
)

// Load reads .quality-harness.json from root. A missing file is not an error:
// it falls back to a Go-shaped default when the root has a go.mod, and to no
// check at all otherwise.
func Load(root string) (Config, error) {
	var c Config
	b, err := os.ReadFile(filepath.Join(root, ".quality-harness.json"))
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf(".quality-harness.json: %w", err)
		}
		// A value that is only whitespace is one nobody typed on purpose — a
		// stray space, a truncated edit, a template that expanded to nothing.
		// Untrimmed it read as DECLARED, ran as an empty shell command, exited
		// 0 and reported `check PASS`: a check that did not run reporting a
		// pass, which is precisely what ADR-003 rule 2 refuses. Normalising
		// here makes every `== ""` test downstream mean what it says, and
		// sends this config down the same path an empty value already took.
		c.Check, c.ScopedCheck = strings.TrimSpace(c.Check), strings.TrimSpace(c.ScopedCheck)
		c.declared = c.Check != "" || c.ScopedCheck != ""
		if err := resolveTimeout(&c); err != nil {
			return c, err
		}
	case !os.IsNotExist(err):
		return c, err
	}
	if !c.declared {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			c.Check, c.ScopedCheck = "go test ./...", "go test {packages}"
		}
	}
	return c, nil
}

// resolveTimeout folds fenceTimeout into TimeoutSeconds. Two different
// numbers are a config error, not a preference: silently picking one is
// how a caller keeps believing the other (ADR-059).
func resolveTimeout(c *Config) error {
	if c.FenceTimeout > 0 && c.TimeoutSeconds > 0 && c.FenceTimeout != c.TimeoutSeconds {
		return fmt.Errorf(".quality-harness.json: timeout_seconds (%d) and fenceTimeout (%d) disagree",
			c.TimeoutSeconds, c.FenceTimeout)
	}
	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = c.FenceTimeout
	}
	return nil
}

// Result is what one check run produced.
type Result struct {
	Ran        bool     `json:"ran"`
	Declared   bool     `json:"declared"`
	Skipped    string   `json:"skipped,omitempty"`
	Command    string   `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Tail       []string `json:"tail,omitempty"`
	Truncated  int      `json:"truncated_lines,omitempty"`
}

// OK reports whether the check ran and passed. A check that did not run is not
// a pass — the caller has no evidence either way, and saying so is the whole
// job of this type.
func (r Result) OK() bool { return r.Ran && r.ExitCode == 0 }

// Run picks a command for the edited paths and executes it in root.
//
// A path resolving outside root is refused before anything runs. A path that
// is not there is refused the same way (ADR-042): falling back to the
// whole-project command and calling that PASS hides the miss. A directory of
// prose or testdata still falls back — those paths exist. Outside the root
// the complete run covers nothing the caller named: `mrw check ../other`
// answered PASS at exit 0 while ../other did not compile, and exit 3 when the
// root's own tests went red. read and write refuse such a path (ADR-006);
// reporting a pass for a scope that never ran is the silent pass ADR-003
// rule 2 forbids.
func Run(ctx context.Context, root string, cfg Config, editedPaths []string) (Result, error) {
	if err := confine(root, editedPaths); err != nil {
		// Nothing runs and no fields are filled in: a refusal must not hand
		// back a shape a caller could read a verdict out of.
		return Result{Declared: cfg.declared}, err
	}
	cmdline, scoped := command(root, cfg, editedPaths)
	if cmdline == "" {
		return Result{Declared: cfg.declared, Skipped: "no check declared and no go.mod found"}, nil
	}
	// Ran is NOT set here. It is set once a ProcessState exists, because this
	// type's whole job is to distinguish "no evidence" from "evidence of
	// success" (ADR-003 rule 2) and a process that never STARTED produced
	// neither. Set optimistically, an unresolvable `sh`, an already-cancelled
	// context or an overflowed timeout reported exit 3 — "a check ran and did
	// not pass" — about a process that never existed.
	res := Result{Declared: cfg.declared, Command: cmdline}
	_ = scoped

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		// Bounded before it is multiplied. time.Duration is int64 NANOseconds,
		// so a seconds value above maxTimeoutSeconds overflows: 9999999999
		// wrapped to a deadline ~2.3 million hours in the PAST, exec refused
		// before the process existed, and the run was reported as one that
		// could not start. Worse, it was not monotonic — 99999999999 wrapped
		// back to positive and the check ran normally, so a bigger number
		// meant a working timeout again.
		//
		// This is the same rule as the whitespace check command one file over:
		// a value that does not survive being used is one nobody typed on
		// purpose. Clamped rather than refused, because every value in range
		// here means "do not run forever" and the ceiling still means that.
		secs := cfg.TimeoutSeconds
		// `>` and `>=` are EQUIVALENT here and no test can separate them: at
		// secs == maxTimeoutSeconds, `>` leaves it alone and `>=` assigns the
		// value it already has. A mutation run flagged this boundary as
		// unasserted; it is unassertable, which is a different thing. Recorded
		// so the next sweep does not spend a cycle writing a test that cannot
		// fail — the classic equivalent-mutant case.
		if secs > maxTimeoutSeconds {
			secs = maxTimeoutSeconds
		}
		timeout = time.Duration(secs) * time.Second
	}
	// ADR-072: the check runs in a process group of its own (subproc), so the
	// terminal's ^C and a hangup reach mrw and not the check. While the check
	// runs, an interrupt, terminate or hangup sent to mrw cancels it instead:
	// the group is killed and the receipt says "interrupted". The handler
	// lives exactly as long as the check, so a ^C anywhere else behaves as it
	// always did.
	ctx, stopSignals := subproc.Interruptible(ctx)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The output goes to a file first and is read back from there. Nothing
	// between the process and its exit code may be a pipe.
	f, err := os.CreateTemp("", "mrw-check-*.log")
	if err != nil {
		return res, err
	}
	res.OutputFile = f.Name()

	c := subproc.Command(ctx, "sh", "-c", cmdline)
	c.Dir = root
	c.Stdout, c.Stderr = f, f

	start := time.Now()
	runErr := c.Run()
	res.DurationMS = time.Since(start).Milliseconds()
	f.Close()

	switch {
	case runErr == nil:
		res.Ran, res.ExitCode = true, 0
	case c.ProcessState != nil:
		// It started and exited badly. That is a real verdict, and exit 3's
		// meaning: the tree is changed and unverified.
		res.Ran, res.ExitCode = true, c.ProcessState.ExitCode()
	default:
		// It never started. Ran stays false, so this routes to "no check could
		// run" and exit 2 — the configuration problem ADR-003's table files a
		// missing check under, not a failed verdict.
		res.ExitCode = -1
		res.Skipped = "could not start: " + runErr.Error()
	}
	// Only a run that ended badly is read for why: a check that exited 0 a
	// moment before a deadline or a signal passed, and saying "timed out" or
	// "interrupted" of it would be a verdict the process did not give (review
	// of #229). No test can reach that window; it is microseconds wide.
	switch {
	case runErr != nil && ctx.Err() == context.DeadlineExceeded:
		res.ExitCode = -1
		res.Skipped = fmt.Sprintf("timed out after %s", timeout)
	case runErr != nil && ctx.Err() == context.Canceled && res.Ran:
		// It started and was stopped from outside — an interrupt, or the
		// caller's own cancel. A check that never started stays "could not
		// start" above: nothing ran for an interrupt to stop.
		res.ExitCode = -1
		res.Skipped = "interrupted"
	}

	tail := cfg.TailLines
	if tail <= 0 {
		tail = defaultTail
	}
	res.Tail, res.Truncated = lastLines(res.OutputFile, tail)
	// A PASSING check that withheld nothing has no readers left, so its log is
	// removed and the field cleared rather than left naming a file nobody will
	// open. Every `mrw write --check` was leaving one behind in the system temp
	// directory for the rest of the machine's life.
	//
	// Two conditions, not one. A FAILING check keeps its log because the tail
	// is a summary and the file is the evidence. A truncated one keeps it even
	// on success, because the report says "N earlier line(s) in <file>" and
	// deleting a file the report points at is worse than leaving it.
	if res.Ran && res.ExitCode == 0 && res.Truncated == 0 {
		if err := os.Remove(res.OutputFile); err == nil {
			res.OutputFile = ""
		}
	}
	return res, nil
}

// shellArgs renders values as a single space-separated fragment of a shell
// command line, each one quoted so the shell reads it as exactly one argument.
//
// The scoped command is a shell line, so every character of a value mrw
// substitutes into it is syntax there: a path with a space becomes two
// arguments, and one holding `;`, `$(…)` or a glob is executed. A directory
// named "pkg; true #" turned `go test {packages}` into `go test ./pkg; true #`
// — the package was never tested, sh exited 0, and the report said PASS. That
// is the silent pass ADR-003 rule 2 forbids, reached through the shell.
//
// Only the substituted VALUES are quoted. The template around them is the
// project's own shell — `make verify PKG={packages}`, pipelines, `&&` — and is
// meant to be interpreted, so the placeholder must appear unquoted in it.
func shellArgs(vals []string) string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = shellArg(v)
	}
	return strings.Join(out, " ")
}

// shellArg quotes one value. A value made only of characters no shell treats
// specially is returned unchanged, so an ordinary scope like ./internal/check
// reads the same in the printed command as it always has.
func shellArg(v string) string {
	if v != "" && !strings.ContainsFunc(v, needsQuoting) {
		return v
	}
	// Single quotes suspend every shell meaning except their own, which cannot
	// be escaped inside them — so a literal quote closes, escapes, reopens.
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

// needsQuoting reports whether r is outside the set that is inert in every
// POSIX shell context. The set is deliberately small: anything not certainly
// safe is quoted.
func needsQuoting(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return false
	case r == '.' || r == '_' || r == '-' || r == '/' || r == '=' || r == ':' || r == ',' || r == '+' || r == '@':
		return false
	}
	return true
}

// confine reports the first path that cannot be honoured: outside root, not
// there, or an existing directory that cannot be read. The trailing
// /... of a subtree scope is stripped first, so a scope mrw printed can be
// handed straight back to it — the same normalisation packages does.
//
// rooted.Resolve is the one implementation of the boundary (ADR-006 rule 2);
// this is its third caller, after read and apply.
func confine(root string, paths []string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	for _, p := range paths {
		p = strings.TrimSuffix(p, "/...")
		// An absolute path is JOINED onto the root rather than honoured, which
		// is deliberate and tested. read and apply survive it because the
		// reinterpreted path then fails to exist and they say so; a check has
		// no such tell — the joined path places no package, the run falls back
		// and the answer is a pass. So check refuses what it cannot honour.
		if rooted.IsRooted(p) && !rooted.Contains(absRoot, filepath.Clean(p)) {
			return fmt.Errorf("%s is absolute and would be read relative to the root %s, "+
				"which is not where it points: check it with --root pointed where you mean", p, absRoot)
		}
		full, err := rooted.Resolve(root, p)
		if err != nil {
			return fmt.Errorf("%v: check it with --root pointed where you mean", err)
		}
		// A path that is not there is refused (ADR-042). The fallback used to
		// run the whole project and call that PASS, so a typo read as green on
		// the thing the caller meant to check. Honouring the miss would still
		// run something they did not name; refusing it means nothing ran.
		//
		// A directory that EXISTS and cannot be READ is refused for the same
		// reason: mrw cannot place a package it cannot look at, and
		// holdsPackage reports that as "no package here" — a read error
		// returned as an absence. Only os.IsNotExist is a miss. A directory of
		// prose or testdata is there and still falls back.
		fi, statErr := os.Stat(full)
		switch {
		case statErr == nil && fi.IsDir():
			if _, readErr := os.ReadDir(full); readErr != nil {
				return fmt.Errorf("%s cannot be read (%v), so mrw cannot tell whether it holds a "+
					"package: fix the permissions or scope somewhere readable", p, readErr)
			}
		case os.IsNotExist(statErr):
			return fmt.Errorf("%s is not there, so mrw cannot honour that scope: "+
				"name a path that exists or run the whole-project check", p)
		case statErr != nil:
			return fmt.Errorf("%s cannot be read (%v), so mrw cannot tell whether it holds a "+
				"package: fix the permissions or scope somewhere readable", p, statErr)
		}
	}
	return nil
}

// command chooses between the scoped and whole-project forms.
//
// When every path maps to a Go package, the scoped form runs and both
// {packages} and {files} are substituted.
//
// When the map is empty, a template that contains {files} and not
// {packages} still runs scoped if the path list is non-empty: {files}
// names every path the caller wrote, so it cannot omit the way an empty
// {packages} can (ADR-061). A {packages}-only or mixed template still
// falls back, as does an empty path list (mrw check --full). A scoped
// run that quietly omits a changed file is worse than a slow complete one.
//
// root is needed to tell a directory from a typo; see packages.
func command(root string, cfg Config, paths []string) (cmdline string, scoped bool) {
	if cfg.ScopedCheck == "" {
		return cfg.Check, false
	}
	pkgs := packages(root, paths)
	if len(pkgs) > 0 {
		r := strings.NewReplacer("{packages}", shellArgs(pkgs), "{files}", shellArgs(paths))
		return r.Replace(cfg.ScopedCheck), true
	}
	if len(paths) > 0 && strings.Contains(cfg.ScopedCheck, "{files}") && !strings.Contains(cfg.ScopedCheck, "{packages}") {
		r := strings.NewReplacer("{files}", shellArgs(paths))
		return r.Replace(cfg.ScopedCheck), true
	}
	return cfg.Check, false
}

// packages maps paths to the go patterns that cover them, or returns "" when
// any of them is a path this package cannot place — in which case the caller
// falls back to the full check.
//
// A .go FILE maps to its own directory, ./dir. That is the package the edit
// touched, and nothing below it was touched by naming it.
//
// A DIRECTORY maps to ./dir/..., recursively. Naming one is how a caller asks
// for a subtree — `mrw check .` is the most natural way to say "check
// everything here" — and go's ./dir is the one package at the top: scoping a
// directory that way reported PASS with a failing package one level down,
// which is the same silent omission this fallback exists to prevent. The
// trailing /... is stripped before a path is placed, so a scope mrw printed
// can be handed straight back to it.
//
// Two kinds of path used to share one answer — fall back — and they no longer
// do. A path that is not there is refused by confine before this is reached
// (ADR-042). A directory holding no package go will build — prose, or one
// named testdata, which the ... form excludes by design — still falls back,
// because those paths exist and `go test` on them exits 1 for a reason that
// is not about the code.
//
// A path OUTSIDE the root is not refused here at all — Run refuses it
// before this is reached, because falling back would answer about a tree
// the caller never named.
//
// Paths reaching here from a write are always files, so `write --check` meets
// none of this.
func packages(root string, paths []string) []string {
	trees := map[string]bool{} // a named directory, covered recursively
	pkgs := map[string]bool{}  // the directory of a named .go file
	for _, p := range paths {
		if filepath.Ext(p) == ".go" {
			if !isFile(root, p) {
				return nil
			}
			dir, ok := placed(root, filepath.Dir(p))
			if !ok {
				return nil
			}
			pkgs[dir] = true
			continue
		}
		dir, ok := placed(root, strings.TrimSuffix(p, "/..."))
		if !ok || !holdsPackage(filepath.Join(root, dir)) {
			return nil
		}
		trees[dir] = true
	}
	out := make([]string, 0, len(trees)+len(pkgs))
	for d := range trees {
		if covered(trees, d, false) {
			continue
		}
		out = append(out, treePattern(d))
	}
	for d := range pkgs {
		if covered(trees, d, true) {
			continue
		}
		out = append(out, pkgPattern(d))
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// placed returns p as a cleaned, slash-separated path relative to root, and
// reports whether it stays inside it. Existence is not asked about here: a
// directory's is settled by holdsPackage, which has to walk it anyway, and a
// .go file's by isFile before this is called.
func placed(root, p string) (string, bool) {
	full, err := rooted.Resolve(root, p)
	if err != nil {
		return "", false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	rel, err := filepath.Rel(absRoot, full)
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// isFile reports whether p names an existing regular file inside root.
//
// It is asked of a .go path for the same reason holdsPackage is asked of a
// directory: a path that is not there cannot be placed. Without it a mistyped
// .go name still yields its parent directory, and at a module root that
// directory is the root package — a scope that runs, passes, and covers
// nothing the caller asked about.
//
// Paths reaching packages from a write always exist, since a write creates or
// edits them and `delete` removes lines rather than files, so `write --check`
// is unaffected.
func isFile(root, p string) bool {
	full, err := rooted.Resolve(root, p)
	if err != nil {
		return false
	}
	fi, err := os.Stat(full)
	return err == nil && fi.Mode().IsRegular()
}

// holdsPackage reports whether dir is a directory that `go test ./dir/...`
// would find at least one package in. It is the only question this package
// asks the filesystem about a path, and it is asked to tell a package from a
// typo and from a directory of prose.
//
// The exclusions are go's own, applied to the NAMED directory as well as to
// what is under it: a directory called testdata, or one beginning with "." or
// "_", holds nothing the ... form matches, and neither does a file with those
// prefixes. Verified 2026-09-01: `go test ./testdata/...` exits 1 with
// "matched no packages", so scoping to one fails loudly for a reason that has
// nothing to do with the caller's edit.
func holdsPackage(dir string) bool {
	if excluded(filepath.Base(dir)) {
		return false
	}
	found := false
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			return fs.SkipDir
		}
		if d.IsDir() {
			if p != dir && excluded(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") && !excluded(d.Name()) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// excluded reports whether go's package loader ignores an entry with this name.
func excluded(name string) bool {
	return name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// covered reports whether dir already falls inside one of the recursively
// scoped trees. A tree covers itself only for the pkgs set, where naming a
// file inside a named directory would otherwise print the same package twice.
func covered(trees map[string]bool, dir string, includeSelf bool) bool {
	for t := range trees {
		switch {
		case t == dir:
			if includeSelf {
				return true
			}
		case t == "." || strings.HasPrefix(dir, t+"/"):
			return true
		}
	}
	return false
}

func pkgPattern(dir string) string {
	if dir == "." {
		return "."
	}
	return "./" + dir
}

func treePattern(dir string) string {
	if dir == "." {
		return "./..."
	}
	return "./" + dir + "/..."
}

// lastLines returns the final n lines of a file plus how many it left out, so a
// trimmed report never reads as the whole output.
func lastLines(path string, n int) ([]string, int) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, 0
	}
	if len(lines) <= n {
		return lines, 0
	}
	return lines[len(lines)-n:], len(lines) - n
}

// checkSignals are the signals that stop a running check (ADR-072): an
// interrupt, a terminate and a hangup, less any the process was started with
// ignored. They are subproc's own, since the check listens through
// subproc.Interruptible like every child mrw starts (ADR-074).
func checkSignals() []os.Signal { return subproc.Signals() }
