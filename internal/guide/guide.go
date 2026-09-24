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
A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit says PARTIALLY APPLIED.
Read before you write, per line, not per file.
mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.
A refusal names the file, the plan line, and the reason.`

// WhyAllOrNothing is the reason a failed hunk writes nothing. It is not
// Shared: Shared stays the five ADR-037 sentences (contract §75). CLI and
// the MCP handshake print this after Shared.
func WhyAllOrNothing() string {
	return whyAllOrNothing
}

const whyAllOrNothing = `A failed hunk writes nothing because a write that changed nothing is invisible.`

// CLI is Shared plus the why, the operator traps that only the shell
// surface can hit, the plan ops a PATH caller needs to drive a write, and
// the read side — address forms and the flags that find sites a caller cannot
// name (ADR-063), since "one read of every site" is unplannable without them.
// Stdout of `mrw instructions` is exactly this string.
func CLI() string {
	return Shared() + `

` + WhyAllOrNothing() + `

Never read an exit code through a pipe: mrw write plan | head returns head's status.
Exit 3 means the write applied and the check failed, so the tree is changed and unverified.
MSYS rewrites a regex address before mrw starts. Export MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL.
A shell glob and an address suffix do not mix.
A value with spaces can be double-quoted (anchor="func openTestStore"), single-quoted (anchor='func openTestStore'), or — for anchor= only — left unquoted until the next key=. An unquoted anchor= that contains a double quote is refused; write it as anchor="…".
body= is a line count, not a character count. Python str splits characters; do not use len(body) as body=.
body=@path loads the body from a root-relative file.
lines= is a guard on how many lines the ADDRESS covers, and is not body=.
The checkout is named by global -C DIR or --root DIR before the subcommand (mrw -C repo write plan). After read, -C is context lines, not a checkout.
Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
@@ path 0 create makes a new file; empty is body=0. A new file is not a reason to skip mrw.
A multi-line replace requires anchor= taken from the served first line.
Read every site in one call: mrw read a.go:40-60 'b.go:/func Start/,+12' c.go:$
A spec is a bare path (the whole file) or PATH:RANGE[,RANGE...]. A RANGE is N, N-M, N- (to the end), -M (from the start), A,+N (A plus the N lines after it), $ (the last line), /regexp/ (every matching line; -C N, or --context N, adds lines either side) or /from/,/to/ (to the first match of to at or after from). Quote a spec that contains a space.
A write plan takes N, N-M, N-, $, A,+N, /regexp/ and /from/,/to/, but not -M or a comma list; its start pattern must match exactly once, and it refuses a relative end past the last line where a read clamps.
To find files you cannot name: --grep PATTERN walks the paths given, or the root when none are, and serves each match as /regexp/ would. --exclude GLOB drops files the walk finds, by root-relative path or basename, and repeats. --grep does not read .gitignore; a .git directory the walk meets is skipped, but one you name is walked.
--ast-grep PATTERN is structural search run by the ast-grep binary, which must be on PATH: a missing one exits 2 naming it, and one that hangs is killed at 2 s.
--files-from FILE takes one spec per line, - for stdin: rg -l X . | sed 's|$|:/X/|' | mrw read --files-from - (name rg's path: with none, rg reads a piped stdin and waits)
--stat prints only length, size and sha; --max-lines N caps each spec; --no-numbers drops the numbers a plan addresses by. A read exits 1 when a range cannot be served (a pattern with no match, a start past the end, lines --max-lines withheld) and still prints the rest; an end past the last line is clamped, not an error.
`
}
