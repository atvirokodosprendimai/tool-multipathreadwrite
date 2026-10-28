// Package guide holds the sentences every mrw surface must teach.
//
// ADR-037: Shared is the contract a caller with only the binary is entitled
// to; a surface may print more after it, never a different version of it.
// ADR-062: Shared's first sentence is always + plan, not a 3+ threshold, and
// CLI() is the effective-use document, not a pamphlet.
package guide

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

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

// Recipes is the block mrw instructions prints right after Shared (ADR-161
// Decision 4): each recipe a numbered label line ending in ":" and its command on
// the next line, indented two spaces, and no prose. It leads because an agent
// that reads only the top of the contract should meet first the forms that keep
// lines it does not need out of its context.
func Recipes() string {
	return `Recipes:
1. the same change at many sites, nothing in your context:
  mrw write --grep '\bOld\b' --sub 'New' .
2. see what it will do first, nothing written (a --sub or --rewrite dry run shows each change):
  mrw write --grep '\bOld\b' --sub 'New' --dry-run .
3. a change by syntax, not text:
  mrw write --ast-grep 'foo($A, $B)' --rewrite 'bar($B, $A)' .
4. look before you change, only at what you will use (count first, then read the matches you need):
  mrw read --grep P --count . && mrw read 'f.go:/P/' -C 3 --max-lines 40
5. different changes at several sites: one read of all of them, then the plan in a file — written with the host's file tool, in a temporary directory you may write — so no shell parser sees its text:
  mrw read 'a.go:/func A/,+12' b.go:40-60 && mrw write "$TMPDIR/plan.txt"
6. a step after the write, in the same call (NAME is a step in .quality-harness.json; a code write already runs the project's check; ` + ThenShCaveat() + `):
  mrw write - --then NAME
7. license lines you already know, without them in your context (a read into /dev/null licenses nothing; a file mrw just wrote needs no re-read):
  mrw read --quiet f.go:40-60
8. a file of long lines (logs, JSONL), each line cut to a window round the match:
  mrw read --grep P --max-cols 200 f.jsonl
9. narrow any command's output without leaving mrw:
  cmd | mrw read --grep P --max-lines 20 -
10. copy lines you have read without typing them again, or move them with a delete of the same range (a copy of lines you were not served is refused, --force or not):
  mrw read a.go:40-52 b.go:9 && printf '%s\n' '@@ b.go 9 insert-after from=a.go:40-52' '@@ a.go 40-52 delete' | mrw write -
11. re-read a file you read earlier, its text only if it changed (the sha is the 8 hex digits its header printed):
  mrw read --if-changed f.go=8dc37337 f.go:40-60
12. run a long command (a build, a test suite, a gate) with its output kept out of your context, one string, instead of cmd | tail, which returns tail's status: a four-line receipt with the command's own exit code, the log's path and a read of it; read the log by pattern in a second call, so the exit code is not lost to it: mrw read --grep FAIL -C 3 - < 'LOG', LOG being the path the receipt printed (` + ThenShCaveat() + `; in Windows PowerShell 5.1 a command holding " reaches mrw as several arguments, so use single quotes inside the command, or pwsh 7):
  mrw run 'go test ./...'
13. a few fields out of a JSON document (cmd is any command that prints one), one line each instead of the lines that hold them (a path address of single-line JSON serves the whole line); the projection licenses nothing, so read the path address (mrw read f.json:$.a.b) before editing the value:
  cmd | mrw read --json-path '$.a,$.b' -
14. a long file's shape before its lines — a .go file's declarations or a .md or .markdown file's headings, each with the lines it spans; it licenses nothing, so read the lines you will edit after it:
  mrw read --outline f.go docs.md
15. matches across many files without the flood: count them, then read the first N (when any remain unshown they are counted, the answer says how many and the read exits 1; a match inside a span already shown counts as shown):
  mrw read --grep P --count . && mrw read --grep P --max-matches 40 .
16. a log or a file of near-identical lines, each run shown once (only its first line is licensed: read the folded lines without --fold to edit them):
  mrw read --fold f.log
17. a Go file without its comment runs (the elided lines are not licensed; trailing comments, directives and the cgo preamble stay):
  mrw read --no-comments f.go
18. a setup that reads long lines: set MRW_MAX_COLS=200 in its environment and every read that serves lines cuts them as --max-cols 200 does, unless the call gives --max-cols itself (--max-cols 0 turns it off; a read that serves a line over 400 columns whole, with no width chosen, says so once on standard error):
  mrw read --grep P .
19. a write receipt of one line per file, every failure and advisory still printed in full:
  mrw write --terse plan.txt
20. answer by reference what you were already served, with --if-served PATH=SHA — only in a session alone in its checkout, whose host runs mrw seen --forget on start, resume, clear and compaction, and whose reads were made under --max-chars below the host's output limit, since a line recorded as shown may never have reached you (anywhere else, recipe 11):
  mrw read --if-served f.go=8dc37337 f.go:40-60`
}

// CLI is Shared plus the why, the operator traps that only the shell
// surface can hit, the plan ops a PATH caller needs to drive a write, and
// the read side — address forms and the flags that find sites a caller cannot
// name (ADR-063), since "one read of every site" is unplannable without them.
// Stdout of `mrw instructions` is exactly this string.
func CLI() string {
	return Shared() + `

` + Recipes() + `

` + WhyAllOrNothing() + `

Never read an exit code through a pipe: mrw write plan | head returns head's status.
To make one new file without writing a plan: mrw write --create PATH < content. The lines of standard input become the file, each ending in a newline (a missing final newline is added, CRLF becomes LF); it is refused if PATH exists, and takes no PLAN argument.
Exit 3 means the write applied and the check did not pass — it failed, timed out or was interrupted — so the tree is changed and unverified.
--then NAME runs a step declared in .quality-harness.json "steps", and --then-sh 'CMD' an ad-hoc one, after a write that landed and whose check passed — on write, and on check once it passes. They run in command-line order; the first that does not pass stops the rest, which the receipt names not run, and exits 3 (2 if it could not start). A step is POSIX shell on every platform.
could_not_start means mrw could not start the shell; a command the shell cannot find is a step that ran and failed (exit 127). A step runs with MRW_STEP_DEPTH one higher than its caller's, and --then and --then-sh are refused at depth 8, so a step that re-runs mrw with steps stops instead of recursing. A "steps" block is read only when a step is asked for.
A check, like a step, runs with MRW_STEP_DEPTH one higher than mrw's own; at depth 8 mrw starts neither: --then and --then-sh, a write whose check is due, and mrw check are refused, exit 2, before anything is written or run, while a write that starts no check still lands (--no-check writes without it). A command that clears the environment, such as env -i or sudo, restarts the count below it, as setsid leaves the process group. On unix a stopped check, step or ast-grep process group hears SIGTERM first, and whatever ignores it is killed a second later; an mrw killed that way can leave its own check running.
A step runs as written: a step command holding {files}, {dirs} or {packages} is refused, since mrw expands them only in scoped_check. A passing step prints the last line of its output under its verdict. mrw write --terse prints one line per file instead of a row per hunk, then the summary line as ever; a failed hunk, its skip rows, the balance and closer rows, the rows of a delete (what it removed) and of a from= copy, a symlink target, the check report and every step verdict still print and the exit code is the same, and it is refused beside --quiet.
mrw run 'CMD' exits with the command's own code, 128+N when a signal N killed it (on Windows, Git for Windows' sh reports N as N<<8, which mrw reads back); 124 (it reached its 24-hour deadline) and 130 (mrw was interrupted) are mrw's. Its own failures exit 2 with a mrw: line and no exit line in the receipt. A log is eligible for removal once it is older than 7 days or beyond the newest 64 completed ones, and a later completed run removes it. At depth 8 mrw run is refused, and its command runs one level deeper.
` + ThenShCaveat() + `
A write exits 1 when a hunk fails validation, or is refused during staging, before the commit loop, for a cause in its target (held, a permission, a name refused, changed since mrw read it), and nothing is written; 2 on a usage or filesystem failure, or a failure in the commit loop.
MSYS rewrites a spec whose file part holds a / before mrw starts. Export MSYS_NO_PATHCONV=1 or MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL.
A shell glob and an address suffix do not mix.
A value with spaces can be double-quoted (anchor="func openTestStore"), single-quoted (anchor='func openTestStore'), or — for anchor= only — left unquoted until the next key=. An unquoted anchor= that contains a double quote is refused; write it as anchor="…".
body= is optional for an ordinary inline body, which runs to the next @@ header or the end of the plan. A count is needed in four cases: an empty create is body=0; a body holding a line that begins @@ needs body=N, and raw=true too when that line is a valid hunk header (below); a body whose first line begins body= needs its count. from= and body=@path take no inline count. Plans mrw compiles from foreign formats keep theirs.
When you write a count it is a line count, not a character count (Python str splits characters; do not use len(body) as body=), and it goes ON the header, never on a line of its own: @@ a.go 12-14 replace anchor="func A" body=3, then exactly 3 body lines. A count catches a body shorter than declared, lost in a channel or in the authoring; it cannot catch a hunk left out whole.
body=@path loads the body from a root-relative file.
A body line that is itself a valid hunk header (@@ b.txt 0 create) is not content: it starts a new hunk, silently. A body that holds one needs body=N and raw=true on its header: @@ a.md 0 create body=2 raw=true.
from=PATH:ADDR is a body copied from lines of a file as it was before the plan (ADDR is N, N-M, A,+N, $ or N-$): it is refused unless you were served them, --force or not, and the receipt names what it copied. With a delete of the same range it moves a block; both address the original.
lines= is a guard on how many lines the ADDRESS covers, and is not body=.
The checkout is named by global -C DIR or --root DIR before the subcommand (mrw -C repo write plan). After read, -C is context lines, not a checkout. An all-digit -C before the subcommand is still a checkout, and mrw says so on stderr.
A path with a leading or trailing space goes after --: the argument parser trims a positional before -- (mrw read 'x ' would reach x), so mrw refuses it, exit 2, and names the fix: mrw read -- 'x '. In cmd.exe, which keeps single quotes as characters, the refusal also names the double-quoted form. An attached flag value that ends in whitespace (--files-from='list ') is refused too: pass it as its own argument, --files-from 'list ', which the parser keeps as given.
Ops: replace, insert-after, insert-before, delete, create, unlink, rename.
@@ path 0 create makes a new file; empty is body=0. A new file is not a reason to skip mrw.
A multi-line replace requires anchor= taken from the served first line. sha= and lines= do not stand in for it.
A file mrw just wrote is wholly known: it produced every line, so a chain of edits to it needs no re-read, until something else changes it.
Read these rules once a session. After a compaction, re-read the lines you will edit (the ledger survives it and still licenses lines you no longer hold, but does not require the re-read); read the rules again only if they are gone from your context.
Read every site in one call: mrw read a.go:40-60 'b.go:/func Start/,+12' c.go:$
A spec is a bare path (the whole file) or PATH:RANGE[,RANGE...]. A RANGE is N, N-M, N- (to the end), -M (from the start), A,+N (A plus the N lines after it), $ (the last line), /regexp/ (every matching line; -C N, or --context N, adds lines either side), /from/,/to/ (to the first match of to at or after from) or, in a JSON file, a path such as $.a.b, $.a[0] or $["k"] (the lines of that one value). A ranged read also serves, and licenses, the line after each range that ends before the file does (ADR-161), so mrw read f.txt:2 prints lines 2 and 3 and a replace of line 2 is not refused for want of the line after it. Quote a spec that contains a space.
A write plan takes N, N-M, N-, $, A,+N, /regexp/ and /from/,/to/, but not -M or a comma list, nor a JSON path; its start pattern must match exactly once, or occurrence=N names the Nth match once every match before it has been read, and it refuses a relative end past the last line where a read clamps.
To find files you cannot name: --grep PATTERN walks the paths given, or the root when none are, and serves each match as /regexp/ would. --exclude GLOB drops files the walk finds, by root-relative path or basename, and repeats; a bare directory name met below where the walk starts prunes that whole subtree, a path you name is walked even if it matches, and --ast-grep drops excluded hits after the binary runs. Inside a git checkout --grep skips what .gitignore and .git/info/exclude ignore (core.excludesFile is not read), and any --grep walk skips a binary file; the footer says how many, and --no-ignore walks every file; it does not enter a nested repository — a directory below the checkout holding its own .git — as git does not, though one you name is walked; a .git directory the walk meets is skipped, but one you name is walked.
--ast-grep PATTERN is structural search run by the ast-grep binary, which must be on PATH: a missing one exits 2 naming it, and one that hangs is, on unix, sent SIGTERM at 2 s and killed by 3 s if it ignores it; on Windows it is killed at 2 s.
--files-from FILE takes one spec per line, - for stdin: rg -l X . | sed 's|$|:/X/|' | mrw read --files-from - (name rg's path: with none, rg reads a piped stdin and waits)
--if-changed PATH=SHA (CLI, repeatable, one per file) answers each spec of a file that still has that sha with one line, ==> PATH  unchanged since sha SHA — N lines, B bytes not served, instead of its text, and serves it as ever when it changed; SHA is a prefix of 8 to 64 hex digits of the file's sha256 (the 8 a header prints suffice), and an unchanged answer records nothing, so it licenses no write.
--stat prints only length, size and sha; --count (with --grep) prints how many lines match in each file and a total, and licenses nothing; --max-lines N caps each spec, and is the way to take the first N lines instead of | head, which loses mrw's exit code; --max-matches N (with --grep or --ast-grep, CLI) shows matching lines across all the files in walk order until N have been shown and counts the rest (a match inside a span already shown, as context or as the line after a range, counts as shown and is served with it, so more than N lines can print, and an ast-grep hit counts once), ending -- max-matches N reached: M more matching line(s) in K file(s) not shown and exiting 1 when some were left, and it licenses only what it showed; --fold (CLI) prints a run of two or more non-blank lines that are identical, or identical but for digits, as its first line and -- ×N: lines A-B like A (with ", differing only in digits" when they are not identical): only the first line is licensed, so read the folded lines without --fold before editing them, and it is refused beside --quiet, --stat, --count, --sub, --rewrite, --max-matches, --outline, --no-comments, --if-served and --json-path; --if-served PATH=SHA (CLI, repeatable, one per file) answers a spec of PATH by -- same as served: lines A-B at sha X, instead of its text, when the file still has the sha you quote (the 8 digits a header printed) and the ledger reports every line of the spec (the line after a range included) shown in that version, recording nothing, a missed range still reported and exit 1; it is for a session alone in its checkout, whose host empties the ledger on start, resume, clear and compaction and whose reads were made under --max-chars below the host's output limit, since a line recorded as shown may never have reached you; mrw seen --forget makes everything served again; it is refused beside --quiet, --stat, --count, --sub, --rewrite, --fold, --no-comments, --outline, --json-path, on -, and with --if-changed on the same file; --outline (CLI) prints each file's structure instead of its lines, one A-B text row per top-level func, type, var and const of a .go file or per heading of a .md or .markdown file outside fenced blocks, so a long file costs its table of contents; it licenses nothing (read the lines before editing), names any other file in one line on standard error and exits 1, and is refused beside every flag that serves, searches or plans lines, --max-lines and --max-cols, on - and with an address, and under --max-chars it names a file whose rows do not fit; --no-comments (CLI) serves a run of two or more whole-line comment lines of a .go file as -- … N comment line(s), A-B and does not license them, keeping trailing comments, //go:, //line and //export directives, the cgo preamble, the line after a range and any line a --grep matched, serves other files and - whole (named once on standard error), and is refused beside --quiet, --stat, --count, --fold, --outline, --if-served, --sub, --rewrite, --max-matches and --json-path; --max-chars N (CLI) bounds the whole answer to N characters in whole lines and lists what it kept back on standard error; --json (CLI) prints the answer and the receipt mrw_read would send for it as one JSON document; --json-path EXPR[,EXPR…] (CLI, repeatable) prints the value at each JSON path of a JSON file or of - as one line, EXPR = VALUE with the value compacted, instead of the lines that hold it, which is the way to take a few fields out of single-line or minified JSON that a path address would serve whole; it licenses and records nothing, so read the path address (f.json:$.a.b) before editing, a miss prints EXPR  NOT FOUND and a file that is not one JSON document prints NOT JSON, either exits 1, and it takes --max-cols and --max-chars, and --json for a named file (on - it is refused), but no --grep, --ast-grep, --sub, --rewrite, --count, --stat, --quiet, -N, -C, --max-lines, --max-matches, --fold, --outline, --no-comments, --if-served, --if-changed or range; --max-cols N (CLI) cuts a line to a window round the match and does not record it as read, and MRW_MAX_COLS=N in the environment is that default when the flag is absent (--max-cols 0 turns it off; --quiet, --sub, --rewrite, --count, --stat and --json-path ignore it), while a read that serves a line over 400 columns whole, with no width chosen, says so once on standard error, unless a width was chosen (--max-cols or MRW_MAX_COLS, 0 included) or standard output is the null device; --no-numbers drops the numbers a plan addresses by; --quiet (CLI) licenses what the read serves and prints only each file's header and -- licensed N line(s) in M file(s), never a line: the way to license lines you already know, since a read sent to the null device licenses nothing; --sub REPLACEMENT (with --grep, CLI) prints a plan instead of lines, one replace hunk per line the Go regexp substitution changes ($1 expands, $$ is a literal dollar, and a reference to a group the pattern does not define is refused), and licenses exactly those lines; mrw write --grep P --sub R applies that plan in one command; --rewrite REPLACEMENT (with --ast-grep, CLI) does the same with ast-grep's own rewrite of each match, one replace hunk per span of changed lines (matches that overlap are refused), and mrw write --ast-grep P --rewrite R applies it in one command. A read exits 1 when a range cannot be served (a pattern with no match, a start past the end, lines --max-lines withheld) and still prints the rest; an end past the last line is clamped, not an error.
A content read naming its specs whose state directory cannot be written (a read-only sandbox) still prints its answer and exits as it would, with mrw: nothing was recorded on standard error (just above the kind line when it exits 1); --quiet, --sub, --rewrite and a read that needs the working set still exit 2. No new licence is recorded, so a write to those lines exits 2 naming the state directory while it stays unwritable, and once it can be written is refused as unread unless an earlier read licensed them. Set XDG_STATE_HOME to a writable absolute directory outside the checkout for the whole session, so every read and write uses one ledger.
A pattern is a Go regular expression: start it with (?i) to ignore case.
mrw write PLAN reads a plan from a file and mrw write - reads it from standard input. mrw read - serves standard input as one unnamed file, with addresses, --grep, -C, --max-lines and --max-cols, and licenses nothing; a file named - is ./-. A lone - may stand before flags: mrw moves it last, so they are read.
Send a plan in a file, written with the host's file tool in a temporary directory you may write; where no such directory is writable a plan file inside the checkout is fine, removed after the write, and where the host has no file tool, mrw write - <<'PLAN' with a quoted delimiter stays the form. A file avoids shell quoting; it does not make the plan complete.
A plan piped to mrw write - that ends with nothing in the tree (a refusal, a dry run, a commit that landed nothing) is kept in <state home>/mrw-refused/, the newest 16, beside mrw's state directory and not in it: stderr names the path, and the same command with that path in place of - runs the edited copy.
mrw instructions --core prints the eight rules that matter most.
mrw skill prints the core and the recipes as an Agent Skills SKILL.md, and mrw skill --dir DIR installs it as DIR/mrw/SKILL.md.
`
}

// Core is the eight rules whose absence costs a session a turn, the short form
// to put in standing instructions (ADR-141). It is not generated from CLI: the
// choice of rules is the work, and a test holds each to the full text and every
// flag it names to the binary.
func Core() string {
	return `The eight rules that matter most; read them once a session, and after a compaction only if lost:
1. Use mrw for every file read, edit and create: one read of every site, one plan, one write (mrw write PLANFILE); the same change at many sites: mrw write --grep P --sub R.
2. Read before you write, per line: a write to a line mrw has not served you is refused, except in a file mrw just wrote. After a multi-line body, read on past your range until the enclosing structure closes.
3. A plan that fails validation writes nothing: read the refusal, which names the file, the plan line and the reason, and do not reach for --force. A commit that fails reports what reached disk.
4. A multi-line replace needs anchor= copied from the line your read printed; omit body= unless mrw instructions names your case.
5. For a write: exit 0 fine, 1 a hunk failed and nothing was written, 2 usage or filesystem, 3 applied but the check did not pass (it failed, timed out or was interrupted). A read exits 1 when a range cannot be served. Never read an exit code through a pipe: mrw write plan | head returns head's status.
6. Find with mrw read --grep PATTERN [paths]; start the pattern with (?i) to ignore case; --stat lists the matching files; --max-cols N cuts a long line to a window, and a line it cut is unread; --no-numbers drops the numbers.
7. A new file is mrw write --create PATH < content, or @@ path 0 create in a plan.
8. -C DIR or --root DIR comes BEFORE the subcommand; after read, -C N is context lines. A path with a leading or trailing space goes after --.
`
}

// ThenShCaveat is the one sentence every surface carries beside --then-sh
// (ADR-092 Decision 8).
func ThenShCaveat() string {
	return "--then-sh and mrw run run any shell command they are given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh and mrw run."
}

// Skill is mrw's Agent Skills SKILL.md (ADR-168): YAML frontmatter a skill host
// reads every session — the name, a description that names when to use it, the
// version that printed it, and the sha-256 of the file without that one line —
// then Core and Recipes, then where the rest is. The description is a folded
// block so it can be read as one paragraph; the stamp is double-quoted, since a
// version such as "dev (abc1234)" is not a plain YAML scalar. The digest covers
// the description and the body alike, so `mrw skill --dir` can tell its own
// file from one a person has edited anywhere since.
func Skill(version string) string {
	front := "---\nname: mrw\ndescription: >-\n" + skillDescription +
		"metadata:\n  mrw-version: " + strconv.Quote(version) + "\n  generated-by: \"mrw skill\"\n"
	body := "---\n\n# mrw\n\n" +
		Core() + "\n" + Recipes() + "\n\n" +
		"`mrw instructions` prints the whole contract. This file was printed by `mrw skill` from mrw " + version +
		"; when `mrw version` prints another version, `mrw skill --dir <the directory holding this mrw folder>` replaces it.\n\n" +
		"A host's per-path rules for its Edit and Write tools do not reach mrw, which writes wherever its user can: their absence is not permission, and mrw is not a way to reach a path the host refused you.\n"
	return front + "  content-sha256: \"" + ContentSHA(front+body) + "\"\n" + body
}

// ContentSHA is the hex sha-256 of a skill with its content-sha256 line taken
// out, the digest Skill stamps on that line.
func ContentSHA(unsigned string) string {
	sum := sha256.Sum256([]byte(unsigned))
	return hex.EncodeToString(sum[:])
}

// skillDescription is the trigger: the one part of a skill a host loads every
// session, so it says when, not only what (ADR-168 Decision 1).
const skillDescription = `  Use for every file edit and every new file, and to read the lines you will
  edit. mrw reads many ranges across many files and applies many edits in one
  call, with a verdict for every edit; a plan applies whole or not at all, and a
  write to a line mrw has not served you is refused. Plan one read of every
  site, one plan, one write (mrw write); find sites with mrw read --grep
  PATTERN; the same change everywhere is mrw write --grep P --sub R. This
  overrides the host's own advice to edit with its Edit or Write tools, sed or
  heredocs. It applies only while mrw is on PATH: if it is not, use the host's
  own tools and do not install it.
`
