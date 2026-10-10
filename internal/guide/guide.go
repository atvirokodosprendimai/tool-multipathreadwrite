// Package guide holds the sentences every mrw surface must teach.
//
// ADR-037: Shared is the contract a caller with only the binary is entitled
// to; a surface may print more after it, never a different version of it.
// ADR-062: Shared's first sentence is always + plan, not a 3+ threshold, and
// CLI() is the effective-use document, not a pamphlet.
package guide

// Shared is the five sentences both the CLI document and the MCP handshake
// must contain verbatim. It is not interpolated: examples stay in the
// surface that executes them.
func Shared() string {
	return shared
}

const shared = `Use mrw always: plan the activity as one read of every site, then one plan, then one write.
A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED).
Read before you write, per line, not per file.
mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.
A refusal names the file, the plan line, and the reason.`

// WhyAllOrNothing is the reason a failed hunk writes nothing. It is not
// Shared: Shared stays the five ADR-037 sentences (contract §75). CLI and
// the MCP handshake print this after Shared.
func WhyAllOrNothing() string {
	return whyAllOrNothing
}

const whyAllOrNothing = `A plan that fails validation writes nothing because a write that changed nothing is invisible.`

// CLI is Shared plus the why, the operator traps that only the shell
// surface can hit, the plan ops a PATH caller needs to drive a write, and
// the read side — address forms and the flags that find sites a caller cannot
// name (ADR-063), since "one read of every site" is unplannable without them.
// Stdout of `mrw instructions` is exactly this string.
func CLI() string {
	return Shared() + `

` + WhyAllOrNothing() + `

Never read an exit code through a pipe: mrw write plan | head returns head's status.
To make one new file without writing a plan: mrw write --create PATH < content. The lines of standard input become the file, each ending in a newline (a missing final newline is added, CRLF becomes LF); it is refused if PATH exists, and takes no PLAN argument.
Exit 3 means the write applied and the check did not pass — it failed, timed out or was interrupted — so the tree is changed and unverified.
--then NAME runs a step declared in .quality-harness.json "steps", and --then-sh 'CMD' an ad-hoc one, after a write that landed and whose check passed — on write, and on check once it passes. They run in command-line order; the first that does not pass stops the rest, which the receipt names not run, and exits 3 (2 if it could not start). A step is POSIX shell on every platform.
could_not_start means mrw could not start the shell; a command the shell cannot find is a step that ran and failed (exit 127). A step runs with MRW_STEP_DEPTH one higher than its caller's, and --then and --then-sh are refused at depth 8, so a step that re-runs mrw with steps stops instead of recursing. A "steps" block is read only when a step is asked for.
A check, like a step, runs with MRW_STEP_DEPTH one higher than mrw's own; at depth 8 mrw starts neither: --then and --then-sh, a write whose check is due, and mrw check are refused, exit 2, before anything is written or run, while a write that starts no check still lands (--no-check writes without it). A command that clears the environment, such as env -i or sudo, restarts the count below it, as setsid leaves the process group. On unix a stopped check, step or ast-grep process group hears SIGTERM first, and whatever ignores it is killed a second later; an mrw killed that way can leave its own check running.
A step runs as written: a step command holding {files}, {dirs} or {packages} is refused, since mrw expands them only in scoped_check. A passing step prints the last line of its output under its verdict.
` + ThenShCaveat() + `
A write exits 1 when a hunk fails validation, or is refused during staging, before the commit loop, for a cause in its target (held, a permission, a name refused, changed since mrw read it), and nothing is written; 2 on a usage or filesystem failure, or a failure in the commit loop.
MSYS rewrites a spec whose file part holds a / before mrw starts. Export MSYS_NO_PATHCONV=1 or MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL.
A shell glob and an address suffix do not mix.
A value with spaces can be double-quoted (anchor="func openTestStore"), single-quoted (anchor='func openTestStore'), or — for anchor= only — left unquoted until the next key=. An unquoted anchor= that contains a double quote is refused; write it as anchor="…".
body= is a line count, not a character count. Python str splits characters; do not use len(body) as body=.
body= goes ON the header, never on a line of its own: @@ a.go 12-14 replace anchor="func A" body=3, then exactly 3 body lines.
body=@path loads the body from a root-relative file.
lines= is a guard on how many lines the ADDRESS covers, and is not body=.
The checkout is named by global -C DIR or --root DIR before the subcommand (mrw -C repo write plan). After read, -C is context lines, not a checkout.
A path with a leading or trailing space goes after --: the argument parser trims a positional before -- (mrw read 'x ' would reach x), so mrw refuses it, exit 2, and names the fix: mrw read -- 'x '. In cmd.exe, which keeps single quotes as characters, the refusal also names the double-quoted form. An attached flag value that ends in whitespace (--files-from='list ') is refused too: pass it as its own argument, --files-from 'list ', which the parser keeps as given.
Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
@@ path 0 create makes a new file; empty is body=0. A new file is not a reason to skip mrw.
A multi-line replace requires anchor= taken from the served first line.
A file mrw just wrote is wholly known: it produced every line, so a chain of edits to it needs no re-read, until something else changes it.
Read every site in one call: mrw read a.go:40-60 'b.go:/func Start/,+12' c.go:$
A spec is a bare path (the whole file) or PATH:RANGE[,RANGE...]. A RANGE is N, N-M, N- (to the end), -M (from the start), A,+N (A plus the N lines after it), $ (the last line), /regexp/ (every matching line; -C N, or --context N, adds lines either side) or /from/,/to/ (to the first match of to at or after from). Quote a spec that contains a space.
A write plan takes N, N-M, N-, $, A,+N, /regexp/ and /from/,/to/, but not -M or a comma list; its start pattern must match exactly once, or occurrence=N names the Nth match once every match before it has been read, and it refuses a relative end past the last line where a read clamps.
To find files you cannot name: --grep PATTERN walks the paths given, or the root when none are, and serves each match as /regexp/ would. --exclude GLOB drops files the walk finds, by root-relative path or basename, and repeats; a bare directory name met below where the walk starts prunes that whole subtree, a path you name is walked even if it matches, and --ast-grep drops excluded hits after the binary runs. Inside a git checkout --grep skips what .gitignore and .git/info/exclude ignore (core.excludesFile is not read), and any --grep walk skips a binary file; the footer says how many, and --no-ignore walks every file; it does not enter a nested repository — a directory below the checkout holding its own .git — as git does not, though one you name is walked; a .git directory the walk meets is skipped, but one you name is walked.
--ast-grep PATTERN is structural search run by the ast-grep binary, which must be on PATH: a missing one exits 2 naming it, and one that hangs is, on unix, sent SIGTERM at 2 s and killed by 3 s if it ignores it; on Windows it is killed at 2 s.
--files-from FILE takes one spec per line, - for stdin: rg -l X . | sed 's|$|:/X/|' | mrw read --files-from - (name rg's path: with none, rg reads a piped stdin and waits)
--stat prints only length, size and sha; --max-lines N caps each spec; --max-cols N (CLI) cuts a line to a window round the match and does not record it as read; --no-numbers drops the numbers a plan addresses by. A read exits 1 when a range cannot be served (a pattern with no match, a start past the end, lines --max-lines withheld) and still prints the rest; an end past the last line is clamped, not an error.
A pattern is a Go regular expression: start it with (?i) to ignore case.
mrw write PLAN reads a plan from a file and mrw write - reads it from standard input.
mrw instructions --core prints the eight rules that matter most.
`
}

// Core is the eight rules whose absence costs a session a turn, the short form
// to put in standing instructions (ADR-141). It is not generated from CLI: the
// choice of rules is the work, and a test holds each to the full text and every
// flag it names to the binary.
func Core() string {
	return `The eight rules that matter most (mrw instructions prints the whole contract):
1. Use mrw for every file read, edit and create: one read of every site, then one plan, then one write (mrw write PLAN, or mrw write - for a plan on standard input).
2. Read before you write, per line: a write to a line mrw has not served you is refused, except in a file mrw just wrote. After a multi-line body, read on past your range until the enclosing structure closes.
3. A plan that fails validation writes nothing: read the refusal, which names the file, the plan line and the reason, and do not reach for --force. A commit that fails reports PARTIALLY APPLIED and names what landed.
4. A multi-line replace needs anchor= copied from the line your read printed; body= is a line count and goes on the header.
5. For a write: exit 0 fine, 1 a hunk failed and nothing was written, 2 usage or filesystem, 3 applied but the check did not pass (it failed, timed out or was interrupted). A read exits 1 when a range cannot be served. Never read an exit code through a pipe: mrw write plan | head returns head's status.
6. Find with mrw read --grep PATTERN [paths]; start the pattern with (?i) to ignore case; --stat lists the matching files; --max-cols N cuts a long line to a window, and a line it cut is unread; --no-numbers drops the numbers.
7. A new file is mrw write --create PATH < content, or @@ path 0 create in a plan.
8. -C DIR or --root DIR comes BEFORE the subcommand; after read, -C N is context lines. A path with a leading or trailing space goes after --.
`
}

// ThenShCaveat is the one sentence every surface carries beside --then-sh
// (ADR-092 Decision 8).
func ThenShCaveat() string {
	return "--then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh."
}
